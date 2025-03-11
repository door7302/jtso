package maker

import (
	"bytes"
	"encoding/json"
	"errors"
	"jtso/logger"
	"os"
	"reflect"
	"strings"
	"text/template"
)

const (
	CLONE_ORDER     int = 1
	PIVOT_ORDER     int = 10
	RENAME_ORDER    int = 100
	PROCESSOR_ORDER int = 200
)

func LoadConfig(filePath string) (*TelegrafConfig, error) {

	// First load JSON file
	file, err := os.Open(filePath)
	if err != nil {
		logger.Log.Errorf("Error opening file %s: %v", filePath, err)
		return nil, err
	}
	defer file.Close()

	// Unmarshall JSON in structure
	var config TelegrafConfig
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		logger.Log.Errorf("Error unmarshaling JSON file %s: %v", filePath, err)
		return nil, err
	}

	logger.Log.Debugf("Successfully Load JSON template from %s", filePath)
	return &config, nil
}

func mergeUniqueInPlaceString(a *[]string, b []string) {
	unique := make(map[string]struct{}) // Track unique values

	// Preserve existing elements from A
	for _, val := range *a {
		unique[val] = struct{}{}
	}

	// Add elements from B if they are not already in A
	for _, val := range b {
		if _, exists := unique[val]; !exists {
			unique[val] = struct{}{}
			*a = append(*a, val) // Append directly to A
		}
	}
}

func mergeNetFieldsInPlaceNetField(a *[]NetField, b []NetField) {
	uniqueFields := make(map[string]int)

	// Track existing elements in A
	for i, field := range *a {
		uniqueFields[field.FieldPath] = i
	}

	// Merge B into A
	for _, field := range b {
		if index, exists := uniqueFields[field.FieldPath]; exists {
			// Overwrite existing entry
			(*a)[index] = field
		} else {
			// Append new entry and update map
			uniqueFields[field.FieldPath] = len(*a)
			*a = append(*a, field)
		}
	}
}

func mergeInPlaceStruct(a interface{}, b interface{}) {
	// Ensure A and B are slices
	aVal := reflect.ValueOf(a)
	bVal := reflect.ValueOf(b)

	if aVal.Kind() != reflect.Ptr || aVal.Elem().Kind() != reflect.Slice {
		logger.Log.Error("Error in mergeInPlaceStruct: A should be a pointer to a slice")
	}
	if bVal.Kind() != reflect.Slice {
		logger.Log.Error("Error in mergeInPlaceStruct: B should be a slice")
	}

	// Iterate over B and append each element to A
	for i := 0; i < bVal.Len(); i++ {
		aVal.Elem().Set(reflect.Append(aVal.Elem(), bVal.Index(i)))
	}

}

func findShortestSubstring(A, B string) (string, string) {
	if strings.Contains(A, B) {
		return B, "B" // B is a substring of A, return the shorter one (B)
	}
	if strings.Contains(B, A) {
		return A, "A" // A is a substring of B, return the shorter one (A)
	}
	return "", "" // No match
}

func OptimizeConf(listOfConf []*TelegrafConfig) *TelegrafConfig {
	// keep consistent order
	//var order int
	// Target config
	var config TelegrafConfig
	var keepOrder int

	for _, entry := range listOfConf {

		//---------------------------------------------------------------
		// Optimise GNMI input plugin
		//---------------------------------------------------------------
		if len(entry.GnmiList) > 0 {
			if len(config.GnmiList) == 0 {
				config.GnmiList = append([]GnmiInput{}, entry.GnmiList...)
			} else {
				// Merge Alias first - Today we support only one gNMI INPUT - this explain [0]
				lenAlias := len(config.GnmiList[0].Aliases)
				for _, newEntry := range entry.GnmiList[0].Aliases {
					match := false
					for i := 0; i < lenAlias; i++ {
						if newEntry.Name == config.GnmiList[0].Aliases[i].Name {
							mergeUniqueInPlaceString(&config.GnmiList[0].Aliases[i].Prefixes, newEntry.Prefixes)
							match = true
							break
						}
					}
					if !match {
						config.GnmiList[0].Aliases = append(config.GnmiList[0].Aliases, newEntry)
					}
				}

				// Then Merge all subscriptions - optimisation will be done later
				mergeInPlaceStruct(&config.GnmiList[0].Subs, entry.GnmiList[0].Subs)
			}
		}

		//---------------------------------------------------------------
		// Optimise Netconf input plugin
		//---------------------------------------------------------------
		if len(entry.NetconfList) > 0 {
			if len(config.NetconfList) == 0 {
				config.NetconfList = append([]NetconfInput{}, entry.NetconfList...)
			} else {
				// Merge Subscriptions
				lenSubs := len(config.NetconfList[0].Subs)
				for _, newEntry := range entry.NetconfList[0].Subs {
					match := false
					for i := 0; i < lenSubs; i++ {
						// First check if same MEASUREMENT NAME and same RPC
						if newEntry.Name == config.NetconfList[0].Subs[i].Name && newEntry.RPC == config.NetconfList[0].Subs[i].RPC {
							mergeNetFieldsInPlaceNetField(&config.NetconfList[0].Subs[i].Fields, newEntry.Fields)
							match = true
							break
						}
					}
					if !match {
						config.NetconfList[0].Subs = append(config.NetconfList[0].Subs, newEntry)
					}
				}
			}
		}

		//---------------------------------------------------------------
		// Optimise Clone plugin: No optimisation
		//---------------------------------------------------------------
		// Save smallest order
		if len(entry.CloneList) > 0 {

			if len(config.CloneList) == 0 {
				config.CloneList = append([]Clone{}, entry.CloneList...)
			} else {
				// We merge both list of clone
				mergeInPlaceStruct(&config.CloneList, entry.CloneList)
			}
			// now we reallocate the order
			for i := 0; i < len(config.CloneList); i++ {
				config.CloneList[i].Order = CLONE_ORDER + i
			}
		}

		//---------------------------------------------------------------
		// Optimise PIVOT plugin
		//---------------------------------------------------------------
		// Save smallest order
		if len(entry.PivotList) > 0 {
			if len(config.PivotList) == 0 {
				config.PivotList = append([]Pivot{}, entry.PivotList...)
			} else {
				for _, e := range entry.PivotList {
					match := false
					lenEntry := len(config.PivotList)
					for i := 0; i < lenEntry; i++ {
						if config.PivotList[i].Tag == e.Tag && config.PivotList[i].Field == e.Field {
							// here we have similar pivot - merge namepass
							mergeUniqueInPlaceString(&config.PivotList[i].Namepass, e.Namepass)
							match = true
							break
						}
					}
					if !match {
						// Unknown Pivot add to the List
						config.PivotList = append(config.PivotList, e)
					}
				}
			}
			// now we reallocate the order
			for i := 0; i < len(config.PivotList); i++ {
				config.PivotList[i].Order = PIVOT_ORDER + i
			}
		}

		//---------------------------------------------------------------
		// Optimise Rename plugin: No optimisation
		//---------------------------------------------------------------
		// Save smallest order
		if len(entry.RenameList) > 0 {
			if len(config.RenameList) == 0 {
				config.RenameList = append([]Rename{}, entry.RenameList...)
			} else {
				// We merge both list of RenameList
				mergeInPlaceStruct(&config.RenameList, entry.RenameList)
			}

			// now we reallocate the order
			for i := 0; i < len(config.RenameList); i++ {
				config.RenameList[i].Order = RENAME_ORDER + i
			}
		}

		//---------------------------------------------------------------
		// Optimise Enrichment plugin
		//---------------------------------------------------------------
		// Save smallest order
		if len(entry.EnrichmentList) > 0 {
			keepOrder = 0
			if len(config.EnrichmentList) == 0 {
				config.EnrichmentList = append([]Enrichment{}, entry.EnrichmentList...)
			} else {
				for _, e := range entry.EnrichmentList {
					match := false
					lenEntry := len(config.EnrichmentList)
					for i := 0; i < lenEntry; i++ {
						if config.EnrichmentList[i].Level1 == e.Level1 && config.EnrichmentList[i].Family == e.Family {
							// here we have similar enrichment - merge namepass
							mergeUniqueInPlaceString(&config.EnrichmentList[i].Namepass, e.Namepass)
							// then check if we have level2 tag in entry if yes merge with existing l2 tag and override twolevel flag
							mergeUniqueInPlaceString(&config.EnrichmentList[i].Level2, e.Level2)
							config.EnrichmentList[i].TwoLevels = true
							match = true
							break
						}
					}
					if !match {
						// Unknown Pivot add to the List
						config.EnrichmentList = append(config.EnrichmentList, e)
					}
				}
			}

			for _, e := range config.EnrichmentList {
				if e.Order < keepOrder || keepOrder == 0 {
					keepOrder = e.Order
				}
			}
			// now we reallocate the order
			for i := 0; i < len(config.EnrichmentList); i++ {
				config.EnrichmentList[i].Order = PROCESSOR_ORDER + keepOrder + i
			}
		}

		//--------------------------------------------------------------------
		// Optimise rate plugin: keep only one
		//--------------------------------------------------------------------
		// Init with one empty Rate opbject
		if len(entry.RateList) > 0 {
			if len(config.RateList) == 0 {
				config.RateList = append(config.RateList, Rate{
					Order:    0,
					Namepass: []string{},
					Fields:   []string{},
				})
			}
			keepOrder = config.RateList[0].Order

			for _, e := range entry.RateList {
				if e.Order < keepOrder || keepOrder == 0 {
					keepOrder = e.Order
				}
				mergeUniqueInPlaceString(&config.RateList[0].Namepass, e.Namepass)
				mergeUniqueInPlaceString(&config.RateList[0].Fields, e.Fields)
			}
			config.RateList[0].Order = PROCESSOR_ORDER + keepOrder
		}

		//---------------------------------------------------------------
		// Optimise Xreducer plugin: No optimisation
		//---------------------------------------------------------------
		// Save smallest order
		if len(entry.XreducerList) > 0 {
			keepOrder = 0
			if len(config.XreducerList) == 0 {
				config.XreducerList = append([]Xreducer{}, entry.XreducerList...)
			} else {
				// We merge both list of XreducerList
				mergeInPlaceStruct(&config.XreducerList, entry.XreducerList)
			}

			for _, e := range entry.XreducerList {
				if e.Order < keepOrder || keepOrder == 0 {
					keepOrder = e.Order
				}
			}
			// now we reallocate the order
			for i := 0; i < len(config.XreducerList); i++ {
				config.XreducerList[i].Order = PROCESSOR_ORDER + keepOrder + i
			}
		}

		//---------------------------------------------------------------
		// Optimise Converter plugin - keep only one
		//---------------------------------------------------------------
		// Init with one empty Rate opbject
		if len(entry.ConverterList) > 0 {
			if len(config.ConverterList) == 0 {
				config.ConverterList = append(config.ConverterList, Converter{
					Order:        0,
					Namepass:     []string{},
					IntegerType:  []string{},
					TagType:      []string{},
					FloatType:    []string{},
					StringType:   []string{},
					BoolType:     []string{},
					UnsignedType: []string{},
				})
			}
			keepOrder = config.ConverterList[0].Order

			for _, e := range entry.ConverterList {
				if e.Order < keepOrder || keepOrder == 0 {
					keepOrder = e.Order
				}
				mergeUniqueInPlaceString(&config.ConverterList[0].Namepass, e.Namepass)
				mergeUniqueInPlaceString(&config.ConverterList[0].IntegerType, e.IntegerType)
				mergeUniqueInPlaceString(&config.ConverterList[0].TagType, e.TagType)
				mergeUniqueInPlaceString(&config.ConverterList[0].FloatType, e.FloatType)
				mergeUniqueInPlaceString(&config.ConverterList[0].StringType, e.StringType)
				mergeUniqueInPlaceString(&config.ConverterList[0].BoolType, e.BoolType)
				mergeUniqueInPlaceString(&config.ConverterList[0].UnsignedType, e.UnsignedType)
			}
			config.ConverterList[0].Order = PROCESSOR_ORDER + keepOrder
		}

		//---------------------------------------------------------------
		// Optimise filtering plugin: keep one
		//---------------------------------------------------------------
		// Init with one empty Filter  opbject
		if len(entry.FilteringList) > 0 {
			if len(config.FilteringList) == 0 {
				config.FilteringList = append(config.FilteringList, Filtering{
					Order:    0,
					Namepass: []string{},
					Filters:  []Filter{},
				})
			}
			keepOrder = config.FilteringList[0].Order

			for _, e := range entry.FilteringList {
				if e.Order < keepOrder || keepOrder == 0 {
					keepOrder = e.Order
				}
				mergeUniqueInPlaceString(&config.FilteringList[0].Namepass, e.Namepass)
				for _, f := range e.Filters {
					lenEntry := len(config.FilteringList[0].Filters)
					match := false
					for i := 0; i < lenEntry; i++ {
						if f.Key == config.FilteringList[0].Filters[i].Key && f.Pattern == config.FilteringList[0].Filters[i].Pattern &&
							f.Action == config.FilteringList[0].Filters[i].Action && f.FilterType == config.FilteringList[0].Filters[i].FilterType {
							// existing entry - do nothing
							match = true
							break
						}
					}
					if !match {
						config.FilteringList[0].Filters = append(config.FilteringList[0].Filters, f)
					}
				}
			}
			config.FilteringList[0].Order = PROCESSOR_ORDER + keepOrder
		}

		//---------------------------------------------------------------
		// Optimise Converter plugin - keep only one
		//---------------------------------------------------------------
		// Init with one empty Rate opbject
		if len(entry.ConverterList) > 0 {
			if len(config.ConverterList) == 0 {
				config.ConverterList = append(config.ConverterList, Converter{
					Order:        0,
					Namepass:     []string{},
					IntegerType:  []string{},
					TagType:      []string{},
					FloatType:    []string{},
					StringType:   []string{},
					BoolType:     []string{},
					UnsignedType: []string{},
				})
			}
			keepOrder = config.ConverterList[0].Order

			for _, e := range entry.ConverterList {
				if e.Order < keepOrder || keepOrder == 0 {
					keepOrder = e.Order
				}
				mergeUniqueInPlaceString(&config.ConverterList[0].Namepass, e.Namepass)
				mergeUniqueInPlaceString(&config.ConverterList[0].IntegerType, e.IntegerType)
				mergeUniqueInPlaceString(&config.ConverterList[0].TagType, e.TagType)
				mergeUniqueInPlaceString(&config.ConverterList[0].FloatType, e.FloatType)
				mergeUniqueInPlaceString(&config.ConverterList[0].StringType, e.StringType)
				mergeUniqueInPlaceString(&config.ConverterList[0].BoolType, e.BoolType)
				mergeUniqueInPlaceString(&config.ConverterList[0].UnsignedType, e.UnsignedType)
			}
			config.ConverterList[0].Order = PROCESSOR_ORDER + keepOrder
		}

		//---------------------------------------------------------------
		// Optimise Enum plugin: No optimisation
		//---------------------------------------------------------------
		// Save smallest order
		if len(entry.EnumList) > 0 {
			keepOrder = 0
			if len(config.EnumList) == 0 {
				config.EnumList = append([]Enum{}, entry.EnumList...)
			} else {
				// We merge both list of EnumList
				mergeInPlaceStruct(&config.EnumList, entry.EnumList)
			}

			for _, e := range config.EnumList {
				if e.Order < keepOrder || keepOrder == 0 {
					keepOrder = e.Order
				}
			}
			// now we reallocate the order
			for i := 0; i < len(config.EnumList); i++ {
				config.EnumList[i].Order = PROCESSOR_ORDER + keepOrder + i
			}
		}

		//---------------------------------------------------------------
		// Optimise regex plugin: keep one
		//---------------------------------------------------------------
		// Init with one empty Filter  opbject
		if len(entry.RegexList) > 0 {
			if len(config.RegexList) == 0 {
				config.RegexList = append(config.RegexList, Regex{
					Order:    0,
					Namepass: []string{},
					Entries:  []RegEntry{},
				})
			}
			keepOrder = config.RegexList[0].Order

			for _, e := range entry.RegexList {
				if e.Order < keepOrder || keepOrder == 0 {
					keepOrder = e.Order
				}
				mergeUniqueInPlaceString(&config.RegexList[0].Namepass, e.Namepass)
				for _, f := range e.Entries {
					lenEntry := len(config.RegexList[0].Entries)
					match := false
					for i := 0; i < lenEntry; i++ {
						if f.RegType == config.RegexList[0].Entries[i].RegType && f.Pattern == config.RegexList[0].Entries[i].Pattern &&
							f.Replacement == config.RegexList[0].Entries[i].Replacement {
							// existing entry - do nothing
							match = true
							break
						}
					}
					if !match {
						config.RegexList[0].Entries = append(config.RegexList[0].Entries, f)
					}
				}
			}
			config.RegexList[0].Order = PROCESSOR_ORDER + keepOrder
		}

		//---------------------------------------------------------------
		// Optimise string plugin: keep one
		//---------------------------------------------------------------
		// Init with one empty Filter  opbject
		if len(entry.StringsList) > 0 {
			if len(config.StringsList) == 0 {
				config.StringsList = append(config.StringsList, Strings{
					Order:    0,
					Namepass: []string{},
					Entries:  []StrEntry{},
				})
			}
			keepOrder = config.StringsList[0].Order

			for _, e := range entry.StringsList {
				if e.Order < keepOrder || keepOrder == 0 {
					keepOrder = e.Order
				}
				mergeUniqueInPlaceString(&config.StringsList[0].Namepass, e.Namepass)
				for _, f := range e.Entries {
					lenEntry := len(config.StringsList[0].Entries)
					match := false
					for i := 0; i < lenEntry; i++ {
						if f.StrType == config.StringsList[0].Entries[i].StrType && f.Method == config.StringsList[0].Entries[i].Method &&
							f.Data == config.StringsList[0].Entries[i].Data {
							// existing entry - do nothing
							match = true
							break
						}
					}
					if !match {
						config.StringsList[0].Entries = append(config.StringsList[0].Entries, f)
					}
				}
			}
			config.StringsList[0].Order = PROCESSOR_ORDER + keepOrder
		}

		//---------------------------------------------------------------
		// Optimise monitoring plugin: keep one
		//---------------------------------------------------------------
		// Init with one empty Filter  opbject
		if len(entry.MonitoringList) > 0 {
			if len(config.MonitoringList) == 0 {
				config.MonitoringList = append(config.MonitoringList, Monitoring{
					Order:    0,
					Namepass: []string{},
					Probes:   []Probe{},
				})
			}
			keepOrder = config.MonitoringList[0].Order

			for _, e := range entry.MonitoringList {
				if e.Order < keepOrder || keepOrder == 0 {
					keepOrder = e.Order
				}
				mergeUniqueInPlaceString(&config.MonitoringList[0].Namepass, e.Namepass)
				for _, f := range e.Probes {
					lenEntry := len(config.MonitoringList[0].Probes)
					match := false
					for i := 0; i < lenEntry; i++ {
						if f.Name == config.MonitoringList[0].Probes[i].Name && f.Field == config.MonitoringList[0].Probes[i].Field &&
							f.ProbeType == config.MonitoringList[0].Probes[i].ProbeType && f.Threshold == config.MonitoringList[0].Probes[i].Threshold &&
							f.Operator == config.MonitoringList[0].Probes[i].Operator {
							// existing entry - merge tags
							mergeUniqueInPlaceString(&config.MonitoringList[0].Probes[i].Tags, f.Tags)
							match = true
							break
						}
					}
					if !match {
						config.MonitoringList[0].Probes = append(config.MonitoringList[0].Probes, f)
					}
				}
			}
			config.MonitoringList[0].Order = PROCESSOR_ORDER + keepOrder
		}

		//---------------------------------------------------------------
		// Optimise Influx output plugin
		//---------------------------------------------------------------
		if len(entry.InfluxList) > 0 {
			if len(config.InfluxList) == 0 {
				config.InfluxList = append([]InfluxOutput{}, entry.InfluxList...)
			} else {
				// We merge fieldpass - we support today only one Influx Output that explains the [0]
				mergeUniqueInPlaceString(&config.InfluxList[0].Fieldpass, entry.InfluxList[0].Fieldpass)
			}
		}

		//---------------------------------------------------------------
		// Optimise File output plugin : no optimization
		//---------------------------------------------------------------
		if len(entry.FileList) > 0 {
			if len(config.FileList) == 0 {
				config.FileList = append([]FileOutput{}, entry.FileList...)
			} else {
				// We merge both list of FileList
				mergeInPlaceStruct(&config.FileList, entry.FileList)
			}
		}
	}

	// Last step is to optimize Gnmi subscriptions
	if len(config.GnmiList) > 0 {
		newSubs := config.GnmiList[0].Subs[:0] // Reuse the existing slice memory

		for i := 0; i < len(config.GnmiList[0].Subs); i++ {
			remove := false
			for j := 0; j < len(config.GnmiList[0].Subs); j++ {
				if i != j {
					shortestPath, who := findShortestSubstring(config.GnmiList[0].Subs[i].Path, config.GnmiList[0].Subs[j].Path)
					if shortestPath != "" {
						if who == "B" {
							// Keep lowest interval
							if config.GnmiList[0].Subs[i].Interval < config.GnmiList[0].Subs[j].Interval {
								config.GnmiList[0].Subs[j].Interval = config.GnmiList[0].Subs[i].Interval
							}
							// Mark i for removal
							remove = true
							break
						} else {
							// Keep lowest interval
							if config.GnmiList[0].Subs[j].Interval < config.GnmiList[0].Subs[i].Interval {
								config.GnmiList[0].Subs[i].Interval = config.GnmiList[0].Subs[j].Interval
							}
							// Mark j for removal
							config.GnmiList[0].Subs[j].Path = "" // Mark for removal later
						}
					}
				}
			}

			if !remove && config.GnmiList[0].Subs[i].Path != "" {
				newSubs = append(newSubs, config.GnmiList[0].Subs[i])
			}
		}

		config.GnmiList[0].Subs = newSubs
	}

	return &config
}

func RenderConf(config *TelegrafConfig) (*string, error) {
	var header string
	var payload string
	var footer string
	var mustErr error
	var tmpl *template.Template

	// Check a config has at least one Input and one Ouput
	hasInput, hasOutput := false, false

	// Manage Gnmi Input
	if len(config.GnmiList) > 0 {
		t, err := template.New("gnmiTemplate").Parse(GnmiInputTemplate)
		if err != nil {
			logger.Log.Errorf("Error parsing Gnmi template: %v", err)
		} else {
			tmpl = template.Must(t, mustErr)
			if mustErr != nil {
				logger.Log.Errorf("Unable to render Gnmi json template - err: %v", mustErr)
			} else {
				var result bytes.Buffer
				err = tmpl.Execute(&result, config.GnmiList)
				if err != nil {
					logger.Log.Errorf("Unable to generate Gnmi toml payload - err: %v", err)
				} else {
					header += result.String()
					hasInput = true
				}
			}
		}
	}
	// Manage Netconf Input
	if len(config.NetconfList) > 0 {
		t, err := template.New("netconfTemplate").Parse(NetconfInputTemplate)
		if err != nil {
			logger.Log.Errorf("Error parsing Netconf template: %v", err)
		} else {
			tmpl = template.Must(t, mustErr)
			if mustErr != nil {
				logger.Log.Errorf("Unable to render Netconf json template - err: %v", mustErr)
			} else {
				var result bytes.Buffer
				err = tmpl.Execute(&result, config.NetconfList)
				if err != nil {
					logger.Log.Errorf("Unable to generate Netconf toml payload - err: %v", err)
				} else {
					header += result.String()
					hasInput = true
				}
			}
		}
	}
	// Manage Influx Output
	if len(config.InfluxList) > 0 {
		t, err := template.New("influxTemplate").Parse(InfluxTemplate)
		if err != nil {
			logger.Log.Errorf("Error parsing Influx template: %v", err)
		} else {
			tmpl = template.Must(t, mustErr)
			if mustErr != nil {
				logger.Log.Errorf("Unable to render Influx json template - err: %v", mustErr)
			} else {
				var result bytes.Buffer
				err = tmpl.Execute(&result, config.InfluxList)
				if err != nil {
					logger.Log.Errorf("Unable to generate Influx toml payload - err: %v", err)
				} else {
					footer += result.String()
					hasOutput = true
				}
			}
		}
	}
	// Manage File Output
	if len(config.FileList) > 0 {
		t, err := template.New("fileTemplate").Parse(FileTemplate)
		if err != nil {
			logger.Log.Errorf("Error parsing File template: %v", err)
		} else {
			tmpl = template.Must(t, mustErr)
			if mustErr != nil {
				logger.Log.Errorf("Unable to render File json template - err: %v", mustErr)
			} else {
				var result bytes.Buffer
				err = tmpl.Execute(&result, config.FileList)
				if err != nil {
					logger.Log.Errorf("Unable to generate File toml payload - err: %v", err)
				} else {
					footer += result.String()
					hasOutput = true
				}
			}
		}
	}

	// Stop if no input or output have been generated
	if !hasInput || !hasOutput {
		logger.Log.Error("Unable to continue - no Input and Output plugins found or generated")
		return nil, errors.New("Unable to continue - no Input and Output plugins found or generated")
	}

	// Manage Clone Processor
	if len(config.CloneList) > 0 {
		t, err := template.New("cloneTemplate").Parse(CloneTemplate)
		if err != nil {
			logger.Log.Errorf("Error parsing Clone template: %v", err)
		} else {
			tmpl = template.Must(t, mustErr)
			if mustErr != nil {
				logger.Log.Errorf("Unable to render Clone json template - err: %v", mustErr)
			} else {
				var result bytes.Buffer
				err = tmpl.Execute(&result, config.CloneList)
				if err != nil {
					logger.Log.Errorf("Unable to generate Clone toml payload - err: %v", err)
				} else {
					payload += result.String()
				}
			}
		}
	}

	// Manage Pivot Processor
	if len(config.PivotList) > 0 {
		t, err := template.New("pivotTemplate").Parse(PivotTemplate)
		if err != nil {
			logger.Log.Errorf("Error parsing Pivot template: %v", err)
		} else {
			tmpl = template.Must(t, mustErr)
			if mustErr != nil {
				logger.Log.Errorf("Unable to render Pivot json template - err: %v", mustErr)
			} else {
				var result bytes.Buffer
				err = tmpl.Execute(&result, config.PivotList)
				if err != nil {
					logger.Log.Errorf("Unable to generate Pivot toml payload - err: %v", err)
				} else {
					payload += result.String()
				}
			}
		}
	}

	// Manage Rename Processor
	if len(config.RenameList) > 0 {
		t, err := template.New("renameTemplate").Parse(RenameTemplate)
		if err != nil {
			logger.Log.Errorf("Error parsing Rename template: %v", err)
		} else {
			tmpl = template.Must(t, mustErr)
			if mustErr != nil {
				logger.Log.Errorf("Unable to render Rename json template - err: %v", mustErr)
			} else {
				var result bytes.Buffer
				err = tmpl.Execute(&result, config.RenameList)
				if err != nil {
					logger.Log.Errorf("Unable to generate Rename toml payload - err: %v", err)
				} else {
					payload += result.String()
				}
			}
		}
	}

	// Manage Xreducer Processor
	if len(config.XreducerList) > 0 {
		t, err := template.New("xreducerTemplate").Parse(XreducerTemplate)
		if err != nil {
			logger.Log.Errorf("Error parsing Xreducer template: %v", err)
		} else {
			tmpl = template.Must(t, mustErr)
			if mustErr != nil {
				logger.Log.Errorf("Unable to render Xreducer json template - err: %v", mustErr)
			} else {
				var result bytes.Buffer
				err = tmpl.Execute(&result, config.XreducerList)
				if err != nil {
					logger.Log.Errorf("Unable to generate Xreducer toml payload - err: %v", err)
				} else {
					payload += result.String()
				}
			}
		}
	}

	// Manage Filtering Processor
	if len(config.FilteringList) > 0 {
		t, err := template.New("filteringTemplate").Parse(FilteringTemplate)
		if err != nil {
			logger.Log.Errorf("Error parsing Filtering template: %v", err)
		} else {
			tmpl = template.Must(t, mustErr)
			if mustErr != nil {
				logger.Log.Errorf("Unable to render Filtering json template - err: %v", mustErr)
			} else {
				var result bytes.Buffer
				err = tmpl.Execute(&result, config.FilteringList)
				if err != nil {
					logger.Log.Errorf("Unable to generate Filtering toml payload - err: %v", err)
				} else {
					payload += result.String()
				}
			}
		}
	}

	// Manage Converter Processor
	if len(config.ConverterList) > 0 {
		t, err := template.New("converterTemplate").Parse(ConverterTemplate)
		if err != nil {
			logger.Log.Errorf("Error parsing Converter template: %v", err)
		} else {
			tmpl = template.Must(t, mustErr)
			if mustErr != nil {
				logger.Log.Errorf("Unable to render Converter json template - err: %v", mustErr)
			} else {
				var result bytes.Buffer
				err = tmpl.Execute(&result, config.ConverterList)
				if err != nil {
					logger.Log.Errorf("Unable to generate Converter toml payload - err: %v", err)
				} else {
					payload += result.String()
				}
			}
		}
	}

	// Manage Enrichement Processor
	if len(config.EnrichmentList) > 0 {
		t, err := template.New("enricherTemplate").Parse(EnrichmentTemplate)
		if err != nil {
			logger.Log.Errorf("Error parsing Enricher template: %v", err)
		} else {
			tmpl = template.Must(t, mustErr)
			if mustErr != nil {
				logger.Log.Errorf("Unable to render Enricher json template - err: %v", mustErr)
			} else {
				var result bytes.Buffer
				err = tmpl.Execute(&result, config.EnrichmentList)
				if err != nil {
					logger.Log.Errorf("Unable to generate Enricher toml payload - err: %v", err)
				} else {
					payload += result.String()
				}
			}
		}
	}

	// Manage Rate Processor
	if len(config.RateList) > 0 {
		t, err := template.New("rateTemplate").Parse(RateTemplate)
		if err != nil {
			logger.Log.Errorf("Error parsing Rate template: %v", err)
		} else {
			tmpl = template.Must(t, mustErr)
			if mustErr != nil {
				logger.Log.Errorf("Unable to render Rate json template - err: %v", mustErr)
			} else {
				var result bytes.Buffer
				err = tmpl.Execute(&result, config.RateList)
				if err != nil {
					logger.Log.Errorf("Unable to generate Rate toml payload - err: %v", err)
				} else {
					payload += result.String()
				}
			}
		}
	}

	// Manage Enum Processor
	if len(config.EnumList) > 0 {
		t, err := template.New("enumTemplate").Parse(EnumTemplate)
		if err != nil {
			logger.Log.Errorf("Error parsing Enum template: %v", err)
		} else {
			tmpl = template.Must(t, mustErr)
			if mustErr != nil {
				logger.Log.Errorf("Unable to render Enum json template - err: %v", mustErr)
			} else {
				var result bytes.Buffer
				err = tmpl.Execute(&result, config.EnumList)
				if err != nil {
					logger.Log.Errorf("Unable to generate Enum toml payload - err: %v", err)
				} else {
					payload += result.String()
				}
			}
		}
	}

	// Manage Regex Processor
	if len(config.RegexList) > 0 {
		t, err := template.New("regexTemplate").Parse(RegexTemplate)
		if err != nil {
			logger.Log.Errorf("Error parsing Regex template: %v", err)
		} else {
			tmpl = template.Must(t, mustErr)
			if mustErr != nil {
				logger.Log.Errorf("Unable to render Regex json template - err: %v", mustErr)
			} else {
				var result bytes.Buffer
				err = tmpl.Execute(&result, config.RegexList)
				if err != nil {
					logger.Log.Errorf("Unable to generate Regex toml payload - err: %v", err)
				} else {
					payload += result.String()
				}
			}
		}
	}

	// Manage Strings Processor
	if len(config.StringsList) > 0 {
		t, err := template.New("stringsTemplate").Parse(StringTemplate)
		if err != nil {
			logger.Log.Errorf("Error parsing Strings template: %v", err)
		} else {
			tmpl = template.Must(t, mustErr)
			if mustErr != nil {
				logger.Log.Errorf("Unable to render Strings json template - err: %v", mustErr)
			} else {
				var result bytes.Buffer
				err = tmpl.Execute(&result, config.StringsList)
				if err != nil {
					logger.Log.Errorf("Unable to generate Strings toml payload - err: %v", err)
				} else {
					payload += result.String()
				}
			}
		}
	}

	// Manage Monitoring Processor
	if len(config.MonitoringList) > 0 {
		t, err := template.New("monitoringTemplate").Parse(MonitoringTemplate)
		if err != nil {
			logger.Log.Errorf("Error parsing Monitoring template: %v", err)
		} else {
			tmpl = template.Must(t, mustErr)
			if mustErr != nil {
				logger.Log.Errorf("Unable to render Monitoring json template - err: %v", mustErr)
			} else {
				var result bytes.Buffer
				err = tmpl.Execute(&result, config.MonitoringList)
				if err != nil {
					logger.Log.Errorf("Unable to generate Monitoring toml payload - err: %v", err)
				} else {
					payload += result.String()
				}
			}
		}
	}

	fullConfig := header + payload + footer

	return &fullConfig, nil
}
