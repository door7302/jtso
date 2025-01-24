package maker

// ---------------------------------------------------- //
// ---------------------------------------------------- //
// A CONFIG COLLECTION
// ---------------------------------------------------- //
// ---------------------------------------------------- //

// Map 1th key is the device family mx, ptx etc...
// the nested map key is version
var TelegrafCollection map[string]map[string]*TelegrafConfig

// ---------------------------------------------------- //
// ---------------------------------------------------- //
// A FULL CONFIG
// ---------------------------------------------------- //
// ---------------------------------------------------- //

type TelegrafConfig struct {
	GnmiList       []GnmiInput    `json:"gnmi_inputs"`
	NetconfList    []NetconfInput `json:"netconf_inputs"`
	CloneList      []Clone        `json:"clone_list"`
	PivotList      []Pivot        `json:"pivot_list"`
	RenameList     []Rename       `json:"rename_list"`
	XreducerList   []Xreducer     `json:"xreducer_list"`
	ConverterList  []Converter    `json:"converter_list"`
	EnrichmentList []Enrichment   `json:"enrichment_list"`
	RateList       []Rate         `json:"rate_list"`
	MonitoringList []Monitoring   `json:"monitoring_list"`
	FilteringList  []Filtering    `json:"filtering_list"`
	EnumList       []Enum         `json:"enum_list"`
	RegexList      []Regex        `json:"regrex_list"`
	StringsList    []Strings      `json:"strings_list"`
	FileList       []FileOutput   `json:"file_outputs"`
	InfluxList     []InfluxOutput `json:"influx_outputs"`
}

// ---------------------------------------------------- //
// GNMI Input modelization
// ---------------------------------------------------- //

type Subscription struct {
	Name string `json:"name"`
	Path string `json:"path"`
	// sample or on_change
	Mode string `json:"mode"`
	// in sec
	Interval int `json:"interval"`
}

type Alias struct {
	Name     string   `json:"name"`
	Prefixes []string `json:"prefix_list"`
}
type GnmiInput struct {
	Rtrs         []string
	Username     string
	Password     string
	UseTls       bool
	SkipVerify   bool
	UseTlsClient bool
	Aliases      []Alias        `json:"aliases"`
	Subs         []Subscription `json:"subscriptions"`
}

// Go Template Receive a list of GnmiInput (we should only have one) = GnmiList

const GnmiInputTemplate = `
###############################################################################
#                               GNMI INPUT PLUGIN                             #
###############################################################################

{{range .}}
[[inputs.gnmi]]
 
  addresses = [
      {{- range $index, $name := .Rtrs}}
      {{if $index}},{{end}}"{{$name}}"
      {{- end}}
      ]

  username = "{{.Username}}"
  password = "{{.Password}}"
  
  {{if .UseTls}}
  ## enable client-side TLS and define CA to authenticate the device
  enable_tls = true
  tls_ca = "/var/cert/RootCA.crt"
  ## Minimal TLS version to accept by the client
  # tls_min_version = "TLS12"
  {{if .SkipVerify}}
  ## Use TLS but skip chain & host verification
  insecure_skip_verify = true
  {{end}}
  {{if .UseTlsClient}}
  ## define client-side TLS certificate & key to authenticate to the device
  tls_cert = "/var/cert/client.crt"
  tls_key = "/var/cert/client.key"
  {{end}}
  {{end}}

  encoding = "proto"
  redial = "10s"
  long_tag = true
  long_field = true
  check_jnpr_extension = true

    [inputs.gnmi.aliases]
	  {{range .Aliases}}
      {{.Name}} = [
      {{- range $index, $name := .Prefixes}}
      {{if $index}},{{end}}"{{$name}}"
      {{- end}}
      ]
	  {{end}}

	{{range .Subs}}
    [[inputs.gnmi.subscription]]
      name = " {{.Name}}"
      path = "{{.Path}}"
      subscription_mode = "{{.Mode}}"
      sample_interval = "{{.Interval}}s"
	{{end}}

{{end}}
`

// ---------------------------------------------------- //
// Netconf Input modelization
// ---------------------------------------------------- //

type NetField struct {
	FieldPath string `json:"field_path"`
	FieldType string `json:"field_type"`
}

type NetSubscription struct {
	Name   string     `json:"name"`
	RPC    string     `json:"rpc"`
	Fields []NetField `json:"fields"`
	// in sec
	Interval int `json:"interval"`
}

type NetconfInput struct {
	Rtrs     []string
	Username string
	Password string
	Subs     []NetSubscription `json:"subscriptions"`
}

// Go Template Receive a list of NetconfInput (we should only have one) = NetconfList

const NetconfInputTemplate = `
###############################################################################
#                             NETCONF INPUT PLUGIN                            #
###############################################################################

{{range .}}
[[inputs.netconf_junos]]
  ## Address of the Juniper NETCONF server
  addresses = [
      {{- range $index, $name := .Rtrs}}
      {{if $index}},{{end}}"{{$name}}"
      {{- end}}
      ]

  ## define credentials
  username = "{{.Username}}"
  password = "{{.Password}}"

  ## redial in case of failures after
  redial = "10s"

  ## Time Layout for epoch convertion - specify a sample Date/Time layout - default layout is the following:
  time_layout = "2006-01-02 15:04:05 MST"

  {{range .Subs}}
  [[inputs.netconf_junos.subscription]]
    ## Name of the measurement that will be emitted
    name = "{{.Name}}"

    ## the JUNOS RPC to collect 
    junos_rpc = "{{.RPC}}"
  
    ## A list of xpath lite + type to collect / encode 
    ## Each entry in the list is made of: <xpath>:<type>
    ## - xpath lite 
    ## - a type of encoding (supported types : int, float, string, epoch, epoch_ms, epoch_us, epoch_ns)
    ## 
    ## The xpath lite should follow the rpc reply XML document. Optional: you can include btw [] the KEY's name that must use to detect the loop 
    fields = [
      {{- range $index, $field := .Fields}}
      {{if $index}},{{end}}"{{$field.FieldPath}}:{{$field.FieldType}}"
      {{- end}}
    ]

    ## Interval to request the RPC
    sample_interval = "{{.Interval}}"
  {{end}}

{{end}}
`

// ---------------------------------------------------- //
// Processor pivot
// ---------------------------------------------------- //

type Pivot struct {
	Order    int      `json:"order"`
	Namepass []string `json:"namepass"`
	Tag      string   `json:"tag"`
	Field    string   `json:"field"`
}

// Go Template
//  we pass a list of Pivot object= PivotList

const PivotTemplate = `
###############################################################################
#                               PIVOT PLUGIN                                  #
###############################################################################

{{range .}}
[[processors.pivot]]
  order = {{.Order}}
  namepass = [
    {{- range $index, $name := .Namepass}}
    {{if $index}},{{end}}"{{$name}}"
    {{- end}}
  ]
  tag_key = "{{.Tag}}"
  value_key = "{{.Field}}"

{{end}}
`

// ---------------------------------------------------- //
// Processor Rename
// ---------------------------------------------------- //

type EntryRename struct {
	// 0 = tag rename ; 1 = field rename
	TypeRename int    `json:"type"`
	From       string `json:"from"`
	To         string `json:"to"`
}

type Rename struct {
	Order    int           `json:"order"`
	Namepass []string      `json:"namepass"`
	Entries  []EntryRename `json:"entries"`
}

// Go Template
// we pass a list of Rename object = RenameList

const RenameTemplate = `
###############################################################################
#                               RENAME PLUGIN                                 #
###############################################################################

{{range .}}
[[processors.rename]]
  order = {{.Order}}
  namepass = [
  {{- range $index, $name := .Namepass}}
  {{if $index}},{{end}}"{{$name}}"
  {{- end}}
  ]

  {{range .Entries}}
  [[processors.rename.replace]]
    {{if eq .TypeRename 0}}
    tag = "{{.From}}"
    dest = "{{.To}}
    {{else}}
    field = "{{.From}}"
    dest = "{{.To}}"
    {{end}}
  {{end}}
{{end}}
`

// ---------------------------------------------------- //
// Xreducer Processor
// ---------------------------------------------------- //

type Xreducer struct {
	Order    int      `json:"order"`
	Namepass []string `json:"namepass"`
	Tags     []string `json:"tags"`
	Fields   []string `json:"fields"`
}

// Go Template
// we pass a list of Xreducer object = XreducerList

const XreducerTemplate = `
###############################################################################
#                                XREDUCER PLUGIN                              #
###############################################################################

{{range .}}
[[processors.xreducer]]
  order = {{.Order}}
  namepass = [
  {{- range $index, $name := .Namepass}}
  {{if $index}},{{end}}"{{$name}}"
  {{- end}}
  ]

  {{if .Tags}} 
  [[processors.xreducer.tags]]
    {{range .Tags}}
    key = {{.}}
	{{end}}
  {{end}}

  {{if .Fields}} 
  [[processors.xreducer.fields]]
    {{range .Fields}}
    key = {{.}}
	{{end}}
  {{end}}

{{end}}
`

// ---------------------------------------------------- //
// Converter Processor
// ---------------------------------------------------- //

type Converter struct {
	Order        int      `json:"order"`
	Namepass     []string `json:"namepass"`
	IntegerType  []string `json:"integer_type"`
	TagType      []string `json:"tag_type"`
	FloatType    []string `json:"float_type"`
	StringType   []string `json:"string_type"`
	BoolType     []string `json:"bool_type"`
	UnsignedType []string `json:"unsigned_type"`
}

// Go Template
// we pass a list of Converter object = ConverterList

const ConverterTemplate = `
###############################################################################
#                               CONVERTER PLUGIN                              #
###############################################################################

{{range .}}
[[processors.converter]]
  order = {{.Order}}
  namepass = [
  {{- range $index, $name := .Namepass}}
  {{if $index}},{{end}}"{{$name}}"
  {{- end}}
  ]

  [processors.converter.fields]
    {{if .IntegerType}}
    integer = [
    {{- range $index, $name := .IntegerType}}
    {{if $index}},{{end}}"{{$name}}"
    {{- end}}
    ]
	{{end}}

	{{if .TagType}}
	tag = [
    {{- range $index, $name := .TagType}}
    {{if $index}},{{end}}"{{$name}}"
    {{- end}}
    ]
	{{end}}

    {{if .FloatType}}
	float = [
    {{- range $index, $name := .FloatType}}
    {{if $index}},{{end}}"{{$name}}"
    {{- end}}
    ]
	{{end}}

    {{if .StringType}}    
	string = [
    {{- range $index, $name := .StringType}}
    {{if $index}},{{end}}"{{$name}}"
    {{- end}}
    ]
	{{end}}

    {{if .BoolType}}    
	boolean = [
    {{- range $index, $name := .BoolType}}
    {{if $index}},{{end}}"{{$name}}"
    {{- end}}
    ]
	{{end}}

    {{if .UnsignedType}}    
	unsigned = [
    {{- range $index, $name := .UnsignedType}}
    {{if $index}},{{end}}"{{$name}}"
    {{- end}}
    ]
	{{end}}

{{end}}
`

// ---------------------------------------------------- //
// Enrichment Processor
// ---------------------------------------------------- //

type Enrichment struct {
	Order     int      `json:"order"`
	Namepass  []string `json:"namepass"`
	Family    string   `json:"family"`
	TwoLevels bool     `json:"two_levels"`
	Level1    string   `json:"level1_tag"`
	Level2    []string `json:"level2_tags"`
}

// Go Template
// we pass a list of Enrichment object = EnrichmentList

const EnrichmentTemplate = `
###############################################################################
#                               ENRICHMENT PLUGIN                             #
###############################################################################

{{range .}}
[[processors.enrichment]]
  order = {{.Order}}
  namepass = [
  {{- range $index, $name := .Namepass}}
  {{if $index}},{{end}}"{{$name}}"
  {{- end}}
  ]
  enrichfilepath = "/var/metadata/metadata_{{.Family}}.json"
  {{if .TwoLevels}}
  twolevels = true
  {{else}}
  twolevels = false
  {{end}}
  refreshperiod = 1 
  level1tagkey = "{{Level1}}"
  level2tagkey =  [
  {{- range $index, $name := .Level2}}
  {{if $index}},{{end}}"{{$name}}"
  {{- end}}
  ]

{{end}}
`

// ---------------------------------------------------- //
// Rate Processor
// ---------------------------------------------------- //

type Rate struct {
	Order    int      `json:"order"`
	Namepass []string `json:"namepass"`
	Fields   []string `json:"fields"`
}

// Go Template
// we pass a list of Rate object = RateList

const RateTemplate = `
###############################################################################
#                                  RATE PLUGIN                                #
###############################################################################

{{range .}}
[[processors.rate]]
  order = {{.Order}}
  namepass = [
  {{- range $index, $name := .Namepass}}
  {{if $index}},{{end}}"{{$name}}"
  {{- end}}
  ]
  period = "10m"
  suffix = "_rate"
  factor = 1.0
  retention = "1h"
  delta_min = "10s"
  fields = [
  {{- range $index, $name := .Fields}}
  {{if $index}},{{end}}"{{$name}}"
  {{- end}}
  ]

{{end}}
`

// ---------------------------------------------------- //
// Monitoring Processor
// ---------------------------------------------------- //

type Probe struct {
	Name      string   `json:"name"`
	Field     string   `json:"field"`
	ProbeType string   `json:"type"`
	Threshold float32  `json:"threshold"`
	Operator  string   `json:"operator"`
	Tags      []string `json:"tags"`
}

type Monitoring struct {
	Order    int      `json:"order"`
	Namepass []string `json:"namepass"`
	Probes   []Probe  `json:"probes"`
}

// Go Template
// we pass a list of Monitoring object = MonitoringList

const MonitoringTemplate = `
###############################################################################
#                              MONITORING PLUGIN                              #
###############################################################################

{{range .}}
[[processors.monitoring]]
  order = {{.Order}}
  namepass = [
  {{- range $index, $name := .Namepass}}
  {{if $index}},{{end}}"{{$name}}"
  {{- end}}
  ]
  measurement = "ALARMING"
  tag_name = "ALARM_TYPE"
  period = "10m"
  retention = "1h"

  {{range .Entries}}
  [[processors.monitoring.probe]]
    alarm_name = "{{.Name}}"
    field = "{{.Field}}"
    probe_type = "{{.ProbeType}}"
    threshold = {{.Threshold}}
    operator = "{{.Operator}}"
    copy_tag = true
    tags = [
    {{- range $index, $name := .Tags}}
    {{if $index}},{{end}}"{{$name}}"
    {{- end}}
    ]

  {{end}}

{{end}}
`

// ---------------------------------------------------- //
// Filtering Processor
// ---------------------------------------------------- //

type Filter struct {
	// 0 = tag - 1 = field
	FilterType int    `json:"type"`
	Key        string `json:"key"`
	Pattern    string `json:"pattern"`
	Action     string `json:"action"`
}

type Filtering struct {
	Order    int      `json:"order"`
	Namepass []string `json:"namepass"`
	Filters  []Filter `json:"filters"`
}

// Go Template
// we pass a list of Filtering object = FilteringList

const FilteringTemplate = `
###############################################################################
#                                  FILTER PLUGIN                              #
###############################################################################

{{range .}}
[[processors.filtering]]
  order = {{.Order}}
  namepass = [
  {{- range $index, $name := .Namepass}}
  {{if $index}},{{end}}"{{$name}}"
  {{- end}}
  ]

  {{range .Filters}}
  {{if eq .FilterType 0}}
  [[processors.filtering.tags]]
  {{else}}
  [[processors.filtering.fields]]
  {{end}}
  key = "{{.Key}}"
  pattern = "{{.Pattern}}"
  Action = "{{.Action}}"
  {{end}}

{{end}}
`

// ---------------------------------------------------- //
// Enum Processor
// ---------------------------------------------------- //

type Mapping struct {
	In  string `json:"in"`
	Out string `json:"out"`
}
type EnumEntry struct {
	Tag  string    `json:"tag"`
	Dest string    `json:"dest"`
	Maps []Mapping `json:"maps"`
}

type Enum struct {
	Order    int         `json:"order"`
	Namepass []string    `json:"namepass"`
	Entries  []EnumEntry `json:"entries"`
}

// Go Template
// we pass a list of Enum object = EnumList

const EnumTemplate = `
###############################################################################
#                                   ENUM PLUGIN                              #
###############################################################################

{{range .}}
[[processors.enum]]
  order = {{.Order}}
  namepass = [
  {{- range $index, $name := .Namepass}}
  {{if $index}},{{end}}"{{$name}}"
  {{- end}}
  ]
  
  {{range .Entries}}
  [[processors.enum.mapping]]
    tag = "{{.Tag}}"
	dest = "{{.Dest}}"
	{{if .Maps}}  
	[processors.enum.mapping.value_mappings]
      {{range .Maps}}
      "{{.In}}" = "{{.Out}}
      {{end}}
	{{end}}
  
  {{end}}

{{end}}
`

// ---------------------------------------------------- //
// Regex Processor
// ---------------------------------------------------- //

type RegEntry struct {
	RegType int `json:"type"`
	// 0 = tag - 1 = field
	Pattern     string `json:"pattern"`
	Replacement string `json:"replacement"`
}

type Regex struct {
	Order    int        `json:"order"`
	Namepass []string   `json:"namepass"`
	Entries  []RegEntry `json:"entries"`
}

// Go Template
// we pass a list of Regex object = RegexList

const RegexTemplate = `
###############################################################################
#                                   REGEX PLUGIN                              #
###############################################################################

{{range .}}
[[processors.regex]]
  order = {{.Order}}
  namepass = [
  {{- range $index, $name := .Namepass}}
  {{if $index}},{{end}}"{{$name}}"
  {{- end}}
  ]
  
  {{range .Entries}}
  {{if eq .RegType 0}}
  [[processors.regex.tag_rename]]
  {{else}}
  [[processors.regex.field_rename]]
  {{end}}
    pattern = "{{.Pattern}}"
    replacement = "{{.Replacement}}"
  {{end}}

{{end}}
`

// ---------------------------------------------------- //
// String Processor
// ---------------------------------------------------- //

type StrEntry struct {
	RegType int `json:"type"`
	// 0 = tag - 1 = field
	Method int `json:"method"`
	// 0 = lowercase , 1 = uppercase
	Data string `json:"data"`
}

type Strings struct {
	Order    int        `json:"order"`
	Namepass []string   `json:"namepass"`
	Entries  []RegEntry `json:"entries"`
}

// Go Template
// we pass a list of Strings object = StringsList

const StringTemplate = `
###############################################################################
#                                  STRING PLUGIN                              #
###############################################################################

{{range .}}
[[processors.strings]]
  order = {{.Order}}
  namepass = [
  {{- range $index, $name := .Namepass}}
  {{if $index}},{{end}}"{{$name}}"
  {{- end}}
  ]
  
  {{range .Entries}}
    {{if eq .Method 0}}
  [[processors.strings.lowercase]]
    {{else}}
  [[processors.strings.uppercase]]
    {{end}}
    {{if eq .RegType 0}}
    tag = "{{.Data}}"    
	{{else}}
    field = "{{.Data}}"   
    {{end}}
  {{end}}

{{end}}
`

// ---------------------------------------------------- //
// Clone Processor
// ---------------------------------------------------- //

type Clone struct {
	Order    int      `json:"order"`
	Namepass []string `json:"namepass"`
	Override string   `json:"override"`
}

// Go Template
// we pass a list of Clone object = CloneList

const CloneTemplate = `
###############################################################################
#                                  CLONE PLUGIN                               #
###############################################################################

{{range .}}
[[processors.clone]]
  order = {{.Order}}
  namepass = [
  {{- range $index, $name := .Namepass}}
  {{if $index}},{{end}}"{{$name}}"
  {{- end}}
  ]
  name_override = "{{.Override}}"

{{end}}
`

// ---------------------------------------------------- //
// Influx Output plugin
// ---------------------------------------------------- //

type InfluxOutput struct {
	Retention string
	Fieldpass []string `json:"fieldpass"`
}

// Go Template Receive a list of InfluxOutput (we should only have one) = InfluxList

const InfluxTemplate = `
###############################################################################
#                              INFLUX OUTPUT PLUGIN                           #
###############################################################################

{{range .}}
[[outputs.influxdb]]
  database="jtsdb"
  urls = ["http://influxdb:8086"]
  retention_policy = "{{.Retention}}"
  fieldpass = [
  {{- range $index, $name := .Fieldpass}}
  {{if $index}},{{end}}"{{$name}}"
  {{- end}}
  ]
{{end}}
`

// ---------------------------------------------------- //
// File Output plugin
// ---------------------------------------------------- //

type FileOutput struct {
	Filename string `json:"filename"`
	Format   string `json:"fieldpass"`
}

// Go Template Receive a list of FileOutput (we should only have one) = FileList

const FileTemplate = `
###############################################################################
#                               FILE OUTPUT PLUGIN                            #
###############################################################################

{{range .}}
[[outputs.file]]
  files = ["/var/log/{{.Filename}}"]
  data_format = "{{.Format}}"
{{end}}
`
