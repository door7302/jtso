package association

import (
	"bufio"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"jtso/config"
	"jtso/container"
	"jtso/kapacitor"
	"jtso/logger"
	"jtso/maker"
	"jtso/ondemand"
	"jtso/sqlite"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const (
	TELEGRAF_ROOT_PATH string = "/var/shared/telegraf/"
	PATH_GRAFANA       string = "/var/shared/grafana/dashboards/"
	PROFILES           string = "/var/profiles/"
	ACTIVE_PROFILES    string = "/var/active_profiles/"
	ONDEMAND_DASH      string = "/var/shared/grafana/dashboards/ondemand.json"
)

var PathMap = map[string]string{
	"mx":       "/var/shared/telegraf/mx/telegraf.d/",
	"ptx":      "/var/shared/telegraf/ptx/telegraf.d/",
	"acx":      "/var/shared/telegraf/acx/telegraf.d/",
	"ex":       "/var/shared/telegraf/ex/telegraf.d/",
	"qfx":      "/var/shared/telegraf/qfx/telegraf.d/",
	"srx":      "/var/shared/telegraf/srx/telegraf.d/",
	"crpd":     "/var/shared/telegraf/crpd/telegraf.d/",
	"cptx":     "/var/shared/telegraf/cptx/telegraf.d/",
	"vmx":      "/var/shared/telegraf/vmx/telegraf.d/",
	"vsrx":     "/var/shared/telegraf/vsrx/telegraf.d/",
	"vjunos":   "/var/shared/telegraf/vjunos/telegraf.d/",
	"vevo":     "/var/shared/telegraf/vevo/telegraf.d/",
	"ondemand": "/var/shared/telegraf/ondemand/telegraf.d/",
}

var Collections map[string]map[string]sqlite.Collection

func hashStringFNV(input string) uint32 {
	hasher := fnv.New32a()
	hasher.Write([]byte(input))
	return hasher.Sum32()
}

func changeTelegrafDebug(instance string, debug int) error {
	// for enable debug we need to change telegraf.conf and set debug = false
	filePath := TELEGRAF_ROOT_PATH + instance + "/telegraf.conf"

	// Read the file
	file, err := os.Open(filePath)
	if err != nil {
		logger.Log.Errorf("Unable to open the file %s: %v", filePath, err)
		return err
	}
	defer file.Close()

	debugRegex := regexp.MustCompile(`^\s*debug\s*=\s*(true|false)\s*$`)
	var updatedLines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if debugRegex.MatchString(line) {
			if debug == 1 {
				line = "  debug = true"
			} else {
				line = "  debug = false"
			}
		}
		updatedLines = append(updatedLines, line)
	}

	if err := scanner.Err(); err != nil {
		logger.Log.Errorf("Error while reading the file %s: %v", filePath, err)
		return err
	}

	// Write the updated content back to the file
	err = os.WriteFile(filePath, []byte(strings.Join(updatedLines, "\n")), 0644)
	if err != nil {
		logger.Log.Errorf("Error while saving the file %s: %v", filePath, err)
		return err
	}

	return nil
}

func ManageDebug(instance string) error {

	// First retrieve the current debug state of the Instance
	var currentState int
	instance = strings.ToLower(instance)

	switch instance {
	case "mx":
		currentState = sqlite.ActiveAdmin.MXDebug
	case "ptx":
		currentState = sqlite.ActiveAdmin.PTXDebug
	case "acx":
		currentState = sqlite.ActiveAdmin.ACXDebug
	case "ex":
		currentState = sqlite.ActiveAdmin.EXDebug
	case "qfx":
		currentState = sqlite.ActiveAdmin.QFXDebug
	case "srx":
		currentState = sqlite.ActiveAdmin.SRXDebug
	case "crpd":
		currentState = sqlite.ActiveAdmin.CRPDDebug
	case "cptx":
		currentState = sqlite.ActiveAdmin.CPTXDebug
	case "vmx":
		currentState = sqlite.ActiveAdmin.VMXDebug
	case "vsrx":
		currentState = sqlite.ActiveAdmin.VSRXDebug
	case "vjunos":
		currentState = sqlite.ActiveAdmin.VJUNOSDebug
	case "vevo":
		currentState = sqlite.ActiveAdmin.VEVODebug
	case "ondemand":
		currentState = sqlite.ActiveAdmin.ONDEMANDDebug
	default:
		logger.Log.Errorf("Unsupported instance %s", instance)
		return errors.New("ManageDebug error: unsupported instance")
	}

	// Now modify the telegraf main config file of the instance
	if err := changeTelegrafDebug(instance, (currentState+1)%2); err != nil {
		logger.Log.Errorf("Error while changing debug mode in the intance %s", instance)
		return err
	}

	// Check if there is at least one router attached to the given instance
	atLeastOne := false
	// Retrive number of active routers per Telegraf instance
	for _, r := range sqlite.RtrList {
		if r.Family == instance {
			if r.Profile == 1 {
				atLeastOne = true
				break
			}
		}
	}

	if atLeastOne {
		// Now restart container only if there are active routers.
		if err := container.RestartContainer("telegraf_" + instance); err != nil {
			logger.Log.Errorf("Unable to restart containter telegraf_%s: %v", instance, err)
			// revert back to previous state
			changeTelegrafDebug(instance, currentState)
			return err
		}
	}

	// Save new State in DB
	if err := sqlite.UpdateDebugMode(instance, (currentState+1)%2); err != nil {
		logger.Log.Errorf("Unable to change debug state in DB for telegraf_%s: %v", instance, err)
		// revert back to previous state
		changeTelegrafDebug(instance, currentState)
		return err
	}

	if (currentState+1)%2 == 0 {
		logger.Log.Infof("Debug mode has been successfully disabled for the telegraf instance %s", instance)
	} else {
		logger.Log.Infof("Debug mode has been successfully enabled for the telegraf instance %s", instance)
	}

	return nil
}

func CheckVersion(searchVersion string, routerVersion string) bool {
	var operator string

	if len(searchVersion) > 2 {
		// Search operator
		if unicode.IsDigit(rune(searchVersion[0])) && unicode.IsDigit(rune(searchVersion[1])) {
			operator = "=="
		} else {
			operator = searchVersion[0:2]
			searchVersion = searchVersion[2:]
		}
		// PRefer Contains for ==
		// Find out if routerVersion can be reduced
		// r, _ := regexp.Compile(searchVersion + "*")
		// result := r.FindString(routerVersion)
		// if result != "" {
		//	routerVersion = result
		// }
		switch operator {
		case "==":
			if strings.Contains(routerVersion, searchVersion) {
				return true
			}

		case ">>":
			if strings.Compare(routerVersion, searchVersion) > 0 {
				return true
			}
		case "<<":
			if strings.Compare(routerVersion, searchVersion) < 0 {
				return true

			}
		case ">=":
			if strings.Compare(routerVersion, searchVersion) >= 0 {
				return true
			}
		case "<=":
			if strings.Compare(routerVersion, searchVersion) <= 0 {
				return true
			}
		default:
			return false
		}
	}
	return false

}

func getLeaf(path string) string {
	if strings.Contains(path, "/") {
		parts := strings.Split(path, "/")
		return strings.ReplaceAll(parts[len(parts)-1], "-", "_")
	}
	return strings.ReplaceAll(path, "-", "_")
}

func getLastTwoNodes(path string) string {
	if !strings.Contains(path, "/") {
		return strings.ReplaceAll(path, "-", "_")
	}

	parts := strings.Split(path, "/")

	// If only one node after split, return it
	if len(parts) == 1 {
		return strings.ReplaceAll(parts[0], "-", "_")
	}

	// Get the last two nodes
	secondLast := parts[len(parts)-2]
	last := parts[len(parts)-1]

	// Replace hyphens with underscores in both parts
	secondLast = strings.ReplaceAll(secondLast, "-", "_")
	last = strings.ReplaceAll(last, "-", "_")

	return secondLast + "_" + last
}

func ConfigureOndemand(cfg *config.ConfigContainer, profile ondemand.RunningProfile) error {
	logger.Log.Infof("Time to reconfigure JTS components for the Ondemand profile %s", profile.Name)

	//ondemand telegraf config
	telegrafOnDemand := maker.TelegrafConfig{
		GnmiList:       make([]maker.GnmiInput, 0),
		RenameList:     make([]maker.Rename, 0),     //Order = 100
		ConverterList:  make([]maker.Converter, 0),  //Order = 200
		EnrichmentList: make([]maker.Enrichment, 0), //Order = 300
		RateList:       make([]maker.Rate, 0),       //Order = 400
		InfluxList:     make([]maker.InfluxOutput, 0),
	}

	// Prepare variables for ondemand grafana dashboard
	grafanaDash := ondemand.Dashboard{
		Variables: make([]ondemand.Variable, 0),
		Paths:     make([]ondemand.PathPanel, 0),
	}

	// parse router and retrieve the family to create the EnrichmentList and fill the routers list in gNMI
	rendRtrs := make([]string, 0)
	familyHandled := make(map[string]struct{})
	index := 0
	for _, r := range profile.RtrList {
		family, hostname, err := sqlite.GetRouterByShort(r)
		if err != nil {
			logger.Log.Errorf("Unable to add router %s: %v", r, err)
			continue
		}
		rendRtrs = append(rendRtrs, hostname+":"+strconv.Itoa(cfg.Gnmi.Port))
		if _, exists := familyHandled[family]; !exists {
			// Family not yet handled
			familyHandled[family] = struct{}{}
			el := maker.Enrichment{
				Order:     300 + index,
				Namepass:  []string{"ONDEMAND"},
				Family:    family,
				TwoLevels: false,
				Level1:    "device",
			}
			telegrafOnDemand.EnrichmentList = append(telegrafOnDemand.EnrichmentList, el)
			index += 1
		}
	}

	// now parse entries and fill the other fields of the telegrafOnDemand
	gnmi := new(maker.GnmiInput)
	converter := new(maker.Converter)
	rate := new(maker.Rate)
	influx := new(maker.InfluxOutput)
	rename := new(maker.Rename)

	// Retrieve some common flags
	tls := false
	skip := false
	clienttls := false
	if sqlite.ActiveCred.UseTls == "yes" {
		tls = true
	}
	if sqlite.ActiveCred.SkipVerify == "yes" {
		skip = true
	}
	if sqlite.ActiveCred.ClientTls == "yes" {
		clienttls = true
	}

	gnmi.Rtrs = rendRtrs
	gnmi.Username = sqlite.ActiveCred.GnmiUser
	gnmi.Password = sqlite.ActiveCred.GnmiPwd
	gnmi.UseTls = tls
	gnmi.UseTlsClient = clienttls
	gnmi.SkipVerify = skip
	gnmi.Subs = make([]maker.Subscription, 0)

	influx.Retention = "autogen"
	influx.Fieldpass = make([]string, 0)

	// Simple set to track duplicated tags / fields
	uniqueTagsGlobal := make(map[string]struct{})
	uniqueField := make(map[string]struct{})

	for _, e := range profile.Entries {
		row := ondemand.PathPanel{
			Name:   e.Path,
			Panels: make([]ondemand.Panel, 0),
		}

		if len(e.Aliases) > 0 {
			a := maker.Alias{
				Name:     "ONDEMAND",
				AliasOf:  strings.TrimSuffix(e.Path, "/"),
				Prefixes: e.Aliases,
			}
			gnmi.Aliases = append(gnmi.Aliases, a)
		}
		sub := maker.Subscription{
			Name:     "ONDEMAND",
			Path:     strings.TrimSuffix(e.Path, "/"),
			Mode:     "sample",
			Interval: e.Interval,
		}
		gnmi.Subs = append(gnmi.Subs, sub)

		for _, f := range e.Fields {
			field := f.Name
			tagsToAlias := ""
			tagCode := ""

			// Process field name
			if strings.HasPrefix(field, "./") {
				field = strings.TrimSuffix(e.Path, "/") + "/" + strings.TrimPrefix(field, "./")
			}

			// Handle conversions
			if f.Convert || f.Rate {
				if f.Convert {
					if converter.Order == 0 {
						converter.Order = 200
						converter.Namepass = []string{"ONDEMAND"}
						converter.FloatType = make([]string, 0)
					}
					converter.FloatType = append(converter.FloatType, field)
				}
				if f.Rate {
					if rate.Order == 0 {
						rate.Order = 400
						rate.Namepass = []string{"ONDEMAND"}
						rate.Fields = make([]string, 0)
					}
					rate.Fields = append(rate.Fields, field)
				}
			}

			// Initialize rename once
			if rename.Order == 0 {
				rename.Order = 100
				rename.Namepass = []string{"ONDEMAND"}
				rename.Entries = make([]maker.EntryRename, 0)
			}

			// Process field rename
			finalField := field
			// Determine finalField based on uniqueness
			if _, exists := uniqueField[getLeaf(field)]; !exists {
				finalField = getLeaf(field)
			} else {
				finalField = getLastTwoNodes(field)
				if _, exists := uniqueField[finalField]; exists {
					// Last resort - full path with slashes replaced
					finalField = strings.ReplaceAll(field, "/", "_")
				}
			}

			// Register the final field name
			uniqueField[finalField] = struct{}{}

			// Create single rename entry
			er := maker.EntryRename{
				TypeRename: 1,
				From:       field,
				To:         finalField,
			}
			rename.Entries = append(rename.Entries, er)
			influx.Fieldpass = append(influx.Fieldpass, finalField)

			// Process InheritTags (merged from both loops)
			for _, t := range f.InheritTags {
				tag := t
				finalTag := tag

				if _, exists := uniqueTagsGlobal[getLeaf(tag)]; !exists {
					finalTag = getLeaf(tag)
				} else {
					finalTag = getLastTwoNodes(tag)
					if _, exists := uniqueTagsGlobal[finalTag]; exists {
						finalTag = strings.ReplaceAll(tag, "/", "_")
					}
				}

				// Update both maps with the same finalTag
				uniqueTagsGlobal[finalTag] = struct{}{}

				// Tag rename entry (from first loop)
				er := maker.EntryRename{
					TypeRename: 0,
					From:       tag,
					To:         finalTag,
				}
				rename.Entries = append(rename.Entries, er)

				// Build tag strings for panel (from second loop)
				tagsToAlias += "$tag_" + finalTag + " - "
				tagCode += `AND \"` + finalTag + `\"=~/^.*${` + finalTag + `:regex}.*$/`
			}

			// Finalize tags string
			if len(tagsToAlias) >= 3 {
				tagsToAlias = tagsToAlias[:len(tagsToAlias)-3]
			}

			// Create panel
			gfnaV := ondemand.Panel{
				Alias:   tagsToAlias,
				Field:   finalField,
				TagCode: tagCode,
				Info:    e.Path,
			}
			row.Panels = append(row.Panels, gfnaV)
		}
		grafanaDash.Paths = append(grafanaDash.Paths, row)
	}

	for k := range uniqueTagsGlobal {
		// Grafana variable
		gfnaV := ondemand.Variable{
			VariableName: k,
			LabelName:    k,
		}
		grafanaDash.Variables = append(grafanaDash.Variables, gfnaV)
	}

	telegrafOnDemand.GnmiList = append(telegrafOnDemand.GnmiList, *gnmi)
	telegrafOnDemand.RenameList = append(telegrafOnDemand.RenameList, *rename)
	telegrafOnDemand.ConverterList = append(telegrafOnDemand.ConverterList, *converter)
	telegrafOnDemand.RateList = append(telegrafOnDemand.RateList, *rate)
	telegrafOnDemand.InfluxList = append(telegrafOnDemand.InfluxList, *influx)

	// render telegraf file
	payload, err := maker.RenderConf(&telegrafOnDemand)
	if err != nil {
		logger.Log.Errorf("Unable to render the Ondemand telegraf config from profile %s: %v", profile.Name, err)
		return err
	}
	savedName := PathMap["ondemand"] + "ondemand_" + profile.Name + ".conf"
	file, err := os.Create(savedName)
	if err != nil {
		logger.Log.Errorf("Unable to open the Ondemand telegraf config %s: %v", savedName, err)
		return err
	}
	// Write text to the file
	_, err = file.WriteString(*payload)
	if err != nil {
		logger.Log.Errorf("Unable to write the Ondemand telegraf config %s: %v", savedName, err)
		file.Close()
		return err
	}
	file.Close()

	// Grafana Dashboard config generation
	grafanaConfig, err := ondemand.RenderDashboard(grafanaDash)
	if err != nil {
		logger.Log.Errorf("Unable to render the grafana ondemand dashboard from profile %s: %v", profile.Name, err)
		return err
	}
	file, err = os.Create(ONDEMAND_DASH)
	if err != nil {
		logger.Log.Errorf("Unable to open the grafana ondemand dashboard file %s: %v", ONDEMAND_DASH, err)
		return err
	}
	defer file.Close()

	// Write text to the file
	_, err = file.WriteString(grafanaConfig)
	if err != nil {
		logger.Log.Errorf("Unable to write the grafana ondemand dashboard file %s: %v", ONDEMAND_DASH, err)
		return err
	}

	// Restart grafana
	container.RestartContainer("grafana")

	// Restart telegraf ondemand instance
	container.RestartContainer("telegraf_ondemand")

	logger.Log.Info("All JTS components reconfigured for ondemand profile")
	return nil
}

func ConfigueStack(cfg *config.ConfigContainer, family string) error {

	logger.Log.Infof("Time to reconfigure JTS components for family %s", family)

	//var temp *template.Template
	var families []string

	profileSetToRouters := make(map[string][]*sqlite.RtrEntry)
	profileSetToProfilesFilename := make(map[string][]string)
	profileSetToProfilesName := make(map[string][]string)
	profileSetIndex := make(map[string]uint32)

	// Map to store collections (family → collection → Collection struct)
	Collections = make(map[string]map[string]sqlite.Collection)

	// create the slice for which families we have to reconfigure the stack
	if family == "all" {
		families = make([]string, 13)

		families[0] = "mx"
		families[1] = "ptx"
		families[2] = "acx"
		families[3] = "ex"
		families[4] = "qfx"
		families[5] = "srx"

		families[6] = "crpd"
		families[7] = "cptx"

		families[8] = "vmx"
		families[9] = "vsrx"
		families[10] = "vjunos"
		families[11] = "vevo"
		families[12] = "ondemand"

	} else {
		families = make([]string, 1)
		families[0] = family
	}

	// -----------------------------------------------------------------------------------------------------
	// Build a lookup map for router profiles from AssoList
	// -----------------------------------------------------------------------------------------------------
	routerProfiles := make(map[string][]string) // key: Shortname → value: Profile List
	for _, asso := range sqlite.AssoList {
		// Create a new slice with the same length as asso.Assos
		assosCopy := make([]string, len(asso.Assos))
		// Copy the contents of asso.Assos into the new slice
		copy(assosCopy, asso.Assos)
		routerProfiles[asso.Shortname] = assosCopy
	}

	// -----------------------------------------------------------------------------------------------------
	// Create the collection - based on Routers which are associated to profiles
	// -----------------------------------------------------------------------------------------------------
	for _, rtr := range sqlite.RtrList {

		// Ignore routers with Profile = 0
		if rtr.Profile == 0 {
			continue
		}

		// Get the profiles from routerProfiles map
		profileKeys, exists := routerProfiles[rtr.Shortname]
		if !exists {
			continue // Skip if no profile association is found
		}

		// check if version is assigned to a profile and save file name
		profilesFilename := make([]string, len(profileKeys))
		profilesName := make([]string, len(profileKeys))
		for i, p := range profileKeys {

			// bypass unknown profile
			_, ok := ActiveProfiles[p]
			if !ok {
				logger.Log.Errorf("Collection issue - Unknown profile detected: %s - skip it", p)
				continue
			}
			profilesName[i] = p

			var filenameList []Config
			switch rtr.Family {
			case "mx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.MxCfg
			case "ptx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.PtxCfg
			case "acx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.AcxCfg
			case "ex":
				filenameList = ActiveProfiles[p].Definition.TelCfg.ExCfg
			case "qfx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.QfxCfg
			case "srx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.SrxCfg
			case "crpd":
				filenameList = ActiveProfiles[p].Definition.TelCfg.CrpdCfg
			case "cptx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.CptxCfg
			case "vmx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.VmxCfg
			case "vsrx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.VsrxCfg
			case "vjunos":
				filenameList = ActiveProfiles[p].Definition.TelCfg.VjunosCfg
			case "vevo":
				filenameList = ActiveProfiles[p].Definition.TelCfg.VevoCfg
			}

			// Check if a profile has as specific version for the given rtr
			savedVersion := ""
			profilesFilename[i] = ""

			for _, c := range filenameList {
				// Save all config if present as a fallback solution if specific version not found
				if c.Version == "all" && savedVersion == "" {
					profilesFilename[i] = c.Config
					savedVersion = "all"
				} else {
					result := CheckVersion(c.Version, rtr.Version)
					if result && (savedVersion == "" || savedVersion == "all") {
						profilesFilename[i] = c.Config
						savedVersion = c.Version
					}
				}
			}

			if savedVersion != "" {
				profileKeys[i] = p + "_" + savedVersion
			} else {
				// Reset entry if there is no filename found
				profileKeys[i] = ""
			}
		}

		// Sort profiles for uniqueness
		sort.Strings(profileKeys)

		// Create a unique profile key (string format for map indexing)
		profileKey := fmt.Sprintf("%s_%v", rtr.Family, profileKeys)

		// Store profile slice before hashing to prevent later issues
		if _, exists := profileSetIndex[profileKey]; !exists {
			profileSetIndex[profileKey] = hashStringFNV(profileKey)
			profileSetToProfilesFilename[profileKey] = profilesFilename
			profileSetToProfilesName[profileKey] = profilesName
		}

		// Store the router in the corresponding profile set
		profileSetToRouters[profileKey] = append(profileSetToRouters[profileKey], rtr)
	}

	// Finally the construction of  the collections map
	for profileKey, routers := range profileSetToRouters {
		// create the unit name
		collectionID := fmt.Sprintf("collection_%d", profileSetIndex[profileKey])

		// Retrieve original profile slice
		profilesFilename := profileSetToProfilesFilename[profileKey]
		profilesName := profileSetToProfilesName[profileKey]

		// Get the family of the first router (all in the same family)
		family := routers[0].Family

		// Ensure family exists in the collections map
		if _, exists := Collections[family]; !exists {
			Collections[family] = make(map[string]sqlite.Collection)
		}

		// Assign to the collections map
		Collections[family][collectionID] = sqlite.Collection{
			ProfilesName: profilesName,
			ProfilesConf: profilesFilename,
			Routers:      routers,
		}

	}

	for family, familyCollections := range Collections {
		logger.Log.Info("Update collections of Telegraf configs:")
		logger.Log.Infof(" Family: %s", family)
		for collectionID, collection := range familyCollections {
			logger.Log.Infof("  ID: %s:", collectionID)
			logger.Log.Info("     Profiles [files]:")
			for i, p := range collection.ProfilesConf {
				logger.Log.Infof("       -%s [%s]", collection.ProfilesName[i], p)
			}
			logger.Log.Info("     Routers part of the collection:")
			for _, r := range collection.Routers {
				logger.Log.Infof("       -%s", r.Hostname)
			}
		}
	}

	// -----------------------------------------------------------------------------------------------------
	// Now for each requested family - create the Telegraf optmized config
	// -----------------------------------------------------------------------------------------------------
	var telegrafCfgList []*maker.TelegrafConfig
	var readDirectory *os.File
	var err error
	for _, f := range families {
		path, exists := PathMap[f]
		if !exists {
			logger.Log.Errorf("Unknown router family: %s", f)
			continue
		}

		readDirectory, err = os.Open(path)
		if err != nil {
			logger.Log.Errorf("Unable to parse the folder %s: %v", path, err)
			continue
		}
		// clean the right directory only if there are files
		allFiles, _ := readDirectory.Readdir(0)

		for f := range allFiles {
			file := allFiles[f]

			fileName := file.Name()
			filePath := path + fileName

			err := os.Remove(filePath)
			if err != nil {
				logger.Log.Errorf("Unable to clean the file %s: %v", filePath, err)
			}
		}

		// For each collection
		for id, collection := range Collections[f] {
			// create a new collection of config before optimisation
			telegrafCfgList = make([]*maker.TelegrafConfig, 0)
			for index, file := range collection.ProfilesConf {
				fullPath := ACTIVE_PROFILES + collection.ProfilesName[index] + "/" + file
				newCfg, err := maker.LoadConfig(fullPath)
				if err != nil {
					continue
				}
				// Override gNMI subscription intervals if user changed them in the DB
				for k1 := range newCfg.GnmiList {
					subs := newCfg.GnmiList[k1].Subs
					for k2 := range subs {
						// Just do this for "sample" subscriptions
						if subs[k2].Mode == "sample" {
							// if found therefore override the default interval
							ci, found, _ := sqlite.GetTelegrafInterval(collection.ProfilesName[index], subs[k2].Path)
							if found {
								subs[k2].Interval = ci
							}
						}
					}
					newCfg.GnmiList[k1].Subs = subs
				}

				telegrafCfgList = append(telegrafCfgList, newCfg)
			}

			// Create one unique config based on the list of configs
			mergedCfg := maker.OptimizeConf(telegrafCfgList)

			// Retrieve some common flags
			tls := false
			skip := false
			clienttls := false
			if sqlite.ActiveCred.UseTls == "yes" {
				tls = true
			}
			if sqlite.ActiveCred.SkipVerify == "yes" {
				skip = true
			}
			if sqlite.ActiveCred.ClientTls == "yes" {
				clienttls = true
			}

			// create the list of rtr with port
			rendRtrs := make([]string, 0)
			rendRtrsNet := make([]string, 0)
			for _, r := range collection.Routers {
				rendRtrs = append(rendRtrs, r.Hostname+":"+strconv.Itoa(cfg.Gnmi.Port))
				rendRtrsNet = append(rendRtrsNet, r.Hostname)
			}

			// Fill missing data
			if len(mergedCfg.GnmiList) > 0 {
				for i := range mergedCfg.GnmiList {
					mergedCfg.GnmiList[i].Rtrs = rendRtrs
					mergedCfg.GnmiList[i].Username = sqlite.ActiveCred.GnmiUser
					mergedCfg.GnmiList[i].Password = sqlite.ActiveCred.GnmiPwd
					mergedCfg.GnmiList[i].UseTls = tls
					mergedCfg.GnmiList[i].UseTlsClient = clienttls
					mergedCfg.GnmiList[i].SkipVerify = skip
				}
			}
			if len(mergedCfg.NetconfList) > 0 {
				for i := range mergedCfg.NetconfList {
					mergedCfg.NetconfList[i].Rtrs = rendRtrsNet
					mergedCfg.NetconfList[i].Username = sqlite.ActiveCred.NetconfUser
					mergedCfg.NetconfList[i].Password = sqlite.ActiveCred.NetconfPwd
				}
			}
			// render file
			payload, err := maker.RenderConf(mergedCfg)
			if err != nil {
				continue
			}

			savedName := path + f + "_" + id + ".conf"
			file, err := os.Create(savedName)
			if err != nil {
				logger.Log.Errorf("Unable to open the target rendering file %s - err: %v", savedName, err)
				continue
			}
			defer file.Close()

			// Write text to the file
			_, err = file.WriteString(*payload)
			if err != nil {
				logger.Log.Errorf("Error writing to file %s: %v", savedName, err)
				continue
			}

		}

	}

	// -----------------------------------------------------------------------------------------------------
	// create the list of active profile dashboard name and copy the new version of each dashboard
	// -----------------------------------------------------------------------------------------------------
	var excludeDash []string
	excludeDash = make([]string, 0)
	excludeDash = append(excludeDash, "home.json")
	excludeDash = append(excludeDash, "ondemand.json")
	for _, v := range Collections {
		for _, c := range v {
			for _, p := range c.ProfilesName {
				// bypass unknown profile
				_, ok := ActiveProfiles[p]
				if !ok {
					logger.Log.Errorf("Grafana update - Unknown profile detected: %s - skip it", p)
					continue
				}
				for _, d := range ActiveProfiles[p].Definition.GrafaCfg {
					excludeDash = append(excludeDash, d)
					source, err := os.Open(ACTIVE_PROFILES + p + "/" + d) //open the source file
					if err != nil {
						logger.Log.Errorf("Unable to open the source dashboard %s - err: %v", d, err)
						continue
					}
					defer source.Close()
					destination, err := os.Create(PATH_GRAFANA + d) //create the destination file
					if err != nil {
						logger.Log.Errorf("Unable to open the destination dashboard %s - err: %v", d, err)
						continue
					}
					defer destination.Close()
					_, err = io.Copy(destination, source) //copy the contents of source to destination file
					if err != nil {
						logger.Log.Errorf("Unable to update the dashboard %s - err: %v", d, err)
						continue
					}
					logger.Log.Infof("Active dashboard %s for profile %s", d, p)
				}
			}
		}
	}

	// Now clean grafana dashbord directory and keep only dashbords related to active profiles
	readDirectory, _ = os.Open(PATH_GRAFANA)
	allFiles, _ := readDirectory.Readdir(0)
	for f := range allFiles {
		file := allFiles[f]

		fileName := file.Name()
		filePath := PATH_GRAFANA + fileName

		// exclude home dashboard and active dashboards profile
		found := false
		for _, v := range excludeDash {
			if v == fileName {
				found = true
				break
			}
		}
		if !found {
			err := os.Remove(filePath)
			if err != nil {
				continue
			}
		}
	}

	// -----------------------------------------------------------------------------------------------------
	// Create the list of Active Kapacitor script
	// -----------------------------------------------------------------------------------------------------
	var kapaStart, kapaStop, kapaAll []string
	kapaStart = make([]string, 0)
	kapaStop = make([]string, 0)
	kapaAll = make([]string, 0)
	for _, v := range Collections {
		for _, c := range v {
			for _, p := range c.ProfilesName {
				// bypass unknown profile
				_, ok := ActiveProfiles[p]
				if !ok {
					logger.Log.Errorf("Kapacitor update - Unknown profile detected: %s - skip it", p)
					continue
				}
				for _, d := range ActiveProfiles[p].Definition.KapaCfg {
					fileKapa := ACTIVE_PROFILES + p + "/" + d
					to_add := true
					for _, a := range kapaAll {
						if a == fileKapa {
							to_add = false
							break
						}
					}
					if to_add {
						// kapaAll is to compare with ActiveTick later to delete unwanted tick scripts
						kapaAll = append(kapaAll, fileKapa)
					}
					found := false
					for i, _ := range kapacitor.ActiveTick {
						if i == fileKapa {
							found = true
							break
						}
					}
					// if kapa script not already active
					if !found {
						kapaStart = append(kapaStart, fileKapa)
					}
				}
			}
		}
	}
	// check now those that need to be deleted
	for i, _ := range kapacitor.ActiveTick {
		found := false
		for _, v := range kapaAll {
			if i == v {
				found = true
				break
			}
		}
		if !found {
			kapaStop = append(kapaStop, i)
		}
	}

	// remove non active Kapascript
	if len(kapaStop) > 0 {
		kapacitor.DeleteTick(kapaStop)
	}

	// Enable active scripts
	if len(kapaStart) > 0 {
		kapacitor.StartTick(kapaStart)
	}

	// Restart grafana
	container.RestartContainer("grafana")

	// -----------------------------------------------------------------------------------------------------
	// Restart telegraf instance(s) : only for the affected families
	// -----------------------------------------------------------------------------------------------------
	for _, f := range families {
		cntr := 0
		for _, c := range Collections[f] {
			cntr += len(c.Routers)
		}

		// if cntr == 0 prefer shutdown the telegraf container
		if cntr == 0 {
			container.StopContainer("telegraf_" + f)
		} else {
			container.RestartContainer("telegraf_" + f)
		}
	}

	logger.Log.Infof("All JTS components reconfigured for family %s", family)
	return nil
}
