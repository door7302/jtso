package ondemand

import (
	"bytes"
	"fmt"
	"jtso/logger"
	"regexp"
	"text/template"
)

// Same sanitization rules as telegraf outputs.prometheus_client (metric_version = 2)
var (
	promInvalidMetricChars = regexp.MustCompile(`[^a-zA-Z0-9_:]`)
	promInvalidLabelChars  = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

// PromMetricName returns the Prometheus metric name telegraf produces for a measurement field
func PromMetricName(measurement, field string) string {
	return promInvalidMetricChars.ReplaceAllString(measurement+"_"+field, "_")
}

// PromLabelName returns the Prometheus label name telegraf produces for a tag
func PromLabelName(tag string) string {
	return promInvalidLabelChars.ReplaceAllString(tag, "_")
}

type Variable struct {
	VariableName string
	LabelName    string
}

type Panel struct {
	Alias   string
	Field   string
	Metric  string
	Unit    string
	TagCode string
	Info    string
}

type PathPanel struct {
	Name   string
	Panels []Panel
}

type Dashboard struct {
	Variables []Variable
	Paths     []PathPanel
}

const GrafanaTemplate = `
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
  "id": 305,
  "links": [],
  "liveNow": false,
  "panels": [
  {{range $index, $element := .Paths}}{{if $index}},{{end}}
    {
      "collapsed": true,
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 0
      },
      "id": {{$index}},
      "panels": [ 
	  {{range $index2, $element2 := $element.Panels}}{{if $index2}},{{end}}
        {
          "datasource": {
            "type": "prometheus",
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
              "min": 0,
              "thresholds": {
                "mode": "absolute",
                "steps": [
                  {
                    "color": "green",
                    "value": null
                  }
                ]
              },
              "unit": "{{$element2.Unit}}",
              "unitScale": true
            },
            "overrides": []
          },
          "gridPos": {
            "h": 13,
            "w": 24,
            "x": 0,
            "y": 2
          },
          "id": {{$index2}},
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
              "datasource": {
                "type": "prometheus",
                "uid": "jtsuid"
              },
              "editorMode": "code",
              "expr": "{{$element2.Metric}}{device=~\"$device\"{{$element2.TagCode}}}",
              "legendFormat": "{{if $element2.Alias}}{{$element2.Alias}}{{else}}__auto{{end}}",
              "range": true,
              "refId": "A"
            }
          ],
          "title": "ROUTER: $device - FIELD: {{$element2.Field}}",
          "type": "timeseries"
        }
    {{end}}
      ],
      "title": "MONITORED PATH: {{$element.Name}}",
      "type": "row"
    }
  {{end}} 
  ],
  "refresh": "1m",
  "schemaVersion": 39,
  "tags": [],
  "templating": {
    "list": [
      {
        "current": {},
        "datasource": {
          "type": "prometheus",
          "uid": "jtsuid"
        },
        "definition": "label_values({__name__=~\"ONDEMAND_.*\"}, device)",
        "hide": 0,
        "includeAll": true,
        "label": "Router",
        "multi": true,
        "name": "device",
        "options": [],
        "query": {
          "qryType": 1,
          "query": "label_values({__name__=~\"ONDEMAND_.*\"}, device)",
          "refId": "PrometheusVariableQueryEditor-VariableQuery"
        },
        "refresh": 2,
        "regex": "",
        "skipUrlSync": false,
        "sort": 1,
        "type": "query"
      },
      {{range $index3, $element3 := .Variables}}{{if $index3}},{{end}}
      {
        "current": {},
        "datasource": {
          "type": "prometheus",
          "uid": "jtsuid"
        },
        "definition": "label_values({__name__=~\"ONDEMAND_.*\", device=~\"$device\"}, {{$element3.VariableName}})",
        "hide": 0,
        "includeAll": true,
        "label": "{{$element3.LabelName}}",
        "multi": true,
        "name":  "{{$element3.VariableName}}",
        "options": [],
        "query": {
          "qryType": 1,
          "query": "label_values({__name__=~\"ONDEMAND_.*\", device=~\"$device\"}, {{$element3.VariableName}})",
          "refId": "PrometheusVariableQueryEditor-VariableQuery"
        },
        "refresh": 2,
        "regex": "",
        "skipUrlSync": false,
        "sort": 1,
        "type": "query"
      }
    {{end}}
    ]
  },
  "time": {
    "from": "now-1h",
    "to": "now"
  },
  "timepicker": {},
  "timezone": "",
  "title": "ONDEMAND Dashboard",
  "uid": "ondemanddashboard",
  "version": 3,
  "weekStart": ""
}
`

// Typical OnDemand Dashboard i
func RenderDashboard(dash Dashboard) (string, error) {
	var tmpl *template.Template
	var mustErr error
	var allFine bool

	allFine = false
	grafanaCode := ""

	t, err := template.New("grafanaTemplate").Parse(GrafanaTemplate)
	if err != nil {
		logger.Log.Errorf("Error parsing Grafana Panel template: %v", err)
	} else {
		tmpl = template.Must(t, mustErr)
		if mustErr != nil {
			logger.Log.Errorf("Unable to render Grafana Panel template - err: %v", mustErr)
		} else {
			var result bytes.Buffer
			err = tmpl.Execute(&result, dash)
			if err != nil {
				logger.Log.Errorf("Unable to generate Grafana Panel payload - err: %v", err)
			} else {
				grafanaCode += result.String()
				allFine = true
			}
		}
	}

	if !allFine {
		return "", fmt.Errorf("Unable to render  Grafana template")

	}

	return grafanaCode, nil

}
