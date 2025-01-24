package maker

import (
	"bytes"
	"encoding/json"
	"errors"
	"jtso/logger"
	"os"
	"text/template"
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

	logger.Log.Infof("Successfully Load JSON template from %s", filePath)
	return &config, nil
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
					logger.Log.Errorf("Unable to generate Gnmi toml payload - err: %v", mustErr)
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
					logger.Log.Errorf("Unable to generate Netconf toml payload - err: %v", mustErr)
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
					logger.Log.Errorf("Unable to generate Influx toml payload - err: %v", mustErr)
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
					logger.Log.Errorf("Unable to generate File toml payload - err: %v", mustErr)
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
					logger.Log.Errorf("Unable to generate Clone toml payload - err: %v", mustErr)
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
					logger.Log.Errorf("Unable to generate Pivot toml payload - err: %v", mustErr)
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
					logger.Log.Errorf("Unable to generate Rename toml payload - err: %v", mustErr)
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
					logger.Log.Errorf("Unable to generate Xreducer toml payload - err: %v", mustErr)
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
				err = tmpl.Execute(&result, config.XreducerList)
				if err != nil {
					logger.Log.Errorf("Unable to generate Converter toml payload - err: %v", mustErr)
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
					logger.Log.Errorf("Unable to generate Enricher toml payload - err: %v", mustErr)
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
					logger.Log.Errorf("Unable to generate Rate toml payload - err: %v", mustErr)
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
					logger.Log.Errorf("Unable to generate Monitoring toml payload - err: %v", mustErr)
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
					logger.Log.Errorf("Unable to generate Filtering toml payload - err: %v", mustErr)
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
					logger.Log.Errorf("Unable to generate Enum toml payload - err: %v", mustErr)
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
					logger.Log.Errorf("Unable to generate Regex toml payload - err: %v", mustErr)
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
					logger.Log.Errorf("Unable to generate Strings toml payload - err: %v", mustErr)
				} else {
					payload += result.String()
				}
			}
		}
	}

	fullConfig := header + payload + footer

	return &fullConfig, nil
}
