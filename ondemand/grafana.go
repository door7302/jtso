package ondemand

import (
	"bytes"
	"fmt"
	"jtso/logger"
	"text/template"
)

type Variable struct {
	VariableName string
	LabelName    string
}

type Panel struct {
	Alias string
	Field string
	Info  string
}

type Dashboard struct {
	Variables []Variable
	Panels    []Panel
}

const GrafanaVariable = `
{{range $index, $element := .}}{{if $index}},{{end}}
			{
				"current": {},
				"datasource": {
					"type": "influxdb",
					"uid": "jtsuid"
				},
				"definition": "show tag values from ONDEMAND with key=\"{{$element.VariableName}}\" where \"device\" = '$device'",
				"hide": 0,
				"includeAll": true,
				"label": "{{$element.LabelName}}",
				"multi": true,
				"name": "{{$element.VariableName}}",
				"options": [],
				"query": "show tag values from ONDEMAND with key=\"{{$element.VariableName}}\" where \"device\" = '$device'",
				"refresh": 1,
				"regex": "",
				"skipUrlSync": false,
				"sort": 0,
				"type": "query"
			}
{{end}}
`

const GrafanaPanel = `
{{range $index, $element := .}}{{if $index}},{{end}}
{
            "datasource": {
                "type": "influxdb",
                "uid": "jtsuid"
            },
            "description": "",
            "fieldConfig": {
                "defaults": {
                    "color": {
                        "mode": "palette-classic"
                    },
                    "custom": {
                        "axisBorderShow": false,
                        "axisCenteredZero": false,
                        "axisColorMode": "text",
                        "axisLabel": "",
                        "axisPlacement": "auto",
                        "barAlignment": 0,
                        "drawStyle": "line",
                        "fillOpacity": 14,
                        "gradientMode": "hue",
                        "hideFrom": {
                            "legend": false,
                            "tooltip": false,
                            "viz": false
                        },
                        "insertNulls": false,
                        "lineInterpolation": "linear",
                        "lineWidth": 1,
                        "pointSize": 5,
                        "scaleDistribution": {
                            "type": "linear"
                        },
                        "showPoints": "never",
                        "spanNulls": false,
                        "stacking": {
                            "group": "A",
                            "mode": "none"
                        },
                        "thresholdsStyle": {
                            "mode": "off"
                        }
                    },
                    "mappings": [],
                    "thresholds": {
                        "mode": "absolute",
                        "steps": [
                            {
                                "color": "green",
                                "value": null
                            }
                        ]
                    },
                    "unit": "none",
                    "unitScale": true
                },
                "overrides": []
            },
            "gridPos": {
                "h": 13,
                "w": 24,
                "x": 0,
                "y": 0
            },
            "id": 1,
            "maxPerRow": 2,
            "options": {
                "legend": {
                    "calcs": [
                        "lastNotNull",
                        "max",
                        "mean"
                    ],
                    "displayMode": "table",
                    "placement": "right",
                    "showLegend": true
                },
                "tooltip": {
                    "mode": "single",
                    "sort": "none"
                }
            },
            "repeat": "device",
            "repeatDirection": "h",
            "targets": [
                {
                    "alias": "{{$element.Alias}}",
                    "datasource": {
                        "type": "influxdb",
                        "uid": "jtsuid"
                    },
                    "groupBy": [],
                    "measurement": "ONDEMAND",
                    "orderByTime": "ASC",
                    "policy": "default",
                    "query": "SELECT \"{{$element.Field}}\" FROM \"ONDEMAND\" WHERE \"device\"=~ /^$device$/ AND $timeFilter GROUP BY  \"*\"",
                    "rawQuery": true,
                    "refId": "A",
                    "resultFormat": "time_series",
                    "select": [],
                    "tags": []
                }
            ],
            "title": "$device - {{$element.Info}}",
            "type": "timeseries"
        }
{{end}}
`

const GrafanaSection1 = `
{
    "annotations": {
        "list": [
            {
                "builtIn": 1,
                "datasource": {
                    "type": "grafana",
                    "uid": "-- Grafana --"
                },
                "enable": true,
                "hide": true,
                "iconColor": "rgba(0, 211, 255, 1)",
                "name": "Annotations & Alerts",
                "type": "dashboard"
            }
        ]
    },
    "editable": true,
    "fiscalYearStartMonth": 0,
    "graphTooltip": 1,
    "links": [],
    "liveNow": false,
    "panels": [

`

const GrafanaSection2 = `
],
    "refresh": "1m",
    "schemaVersion": 39,
    "tags": [],
    "templating": {
        "list": [
            {
                "current": {},
                "datasource": {
                    "type": "influxdb",
                    "uid": "jtsuid"
                },
                "definition": "show tag values from ONDEMAND with key=\"device\"",
                "hide": 0,
                "includeAll": true,
                "label": "Router",
                "multi": true,
                "name": "device",
                "options": [],
                "query": "show tag values from ONDEMAND with key=\"device\"",
                "refresh": 1,
                "regex": "",
                "skipUrlSync": false,
                "sort": 0,
                "type": "query"
            },

`

const GrafanaSection3 = `
        ]
    },
    "time": {
        "from": "now-1h",
        "to": "now"
    },
    "timepicker": {},
    "timezone": "",
    "uid": "ondemanddashboard",
    "title": "ONDEMAND Dashboard",
    "version": 0,
    "weekStart": ""
}
`

// Typical OnDemand Dashboard is
// Section 1 + Panel(s) + Section2 + Variable(s) + Section3

func RenderDashboard(dash Dashboard) (string, error) {
	var tmpl *template.Template
	var mustErr error
	var allFine bool

	allFine = false
	grafanaCode := GrafanaSection1
	if len(dash.Panels) > 0 {
		t, err := template.New("panelTemplate").Parse(GrafanaPanel)
		if err != nil {
			logger.Log.Errorf("Error parsing Grafana Panel template: %v", err)
		} else {
			tmpl = template.Must(t, mustErr)
			if mustErr != nil {
				logger.Log.Errorf("Unable to render Grafana Panel template - err: %v", mustErr)
			} else {
				var result bytes.Buffer
				err = tmpl.Execute(&result, dash.Panels)
				if err != nil {
					logger.Log.Errorf("Unable to generate Grafana Panel payload - err: %v", err)
				} else {
					grafanaCode += result.String()
					allFine = true
				}
			}
		}
	}
	if !allFine {
		return "", fmt.Errorf("Unable to render panel part of Grafana template")

	}

	grafanaCode += GrafanaSection2
	allFine = false

	if len(dash.Variables) > 0 {
		t, err := template.New("variableTemplate").Parse(GrafanaPanel)
		if err != nil {
			logger.Log.Errorf("Error parsing Grafana Variable template: %v", err)
		} else {
			tmpl = template.Must(t, mustErr)
			if mustErr != nil {
				logger.Log.Errorf("Unable to render Grafana Variable template - err: %v", mustErr)
			} else {
				var result bytes.Buffer
				err = tmpl.Execute(&result, dash.Variables)
				if err != nil {
					logger.Log.Errorf("Unable to generate Grafana Variable payload - err: %v", err)
				} else {
					grafanaCode += result.String()
					allFine = true
				}
			}
		}
	}
	if !allFine {
		return "", fmt.Errorf("Unable to render variable part of Grafana template")

	}

	grafanaCode += GrafanaSection3

	return grafanaCode, nil

}
