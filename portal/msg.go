package portal

type (
	TabAsso struct {
		Shortname string `json:"shortname"`
		Profiles  string `json:"profiles"`
	}

	LongRouter struct {
		Hostname  string `json:"hostname"`
		Shortname string `json:"shortname"`
	}

	SearchPath struct {
		Shortname string `json:"shortname"`
		Xpath     string `json:"xpath"`
		Merge     bool   `json:"merge"`
	}

	RouterDetails struct {
		Hostname  string `json:"hostname"`
		Shortname string `json:"shortname"`
		Family    string `json:"family"`
		Model     string `json:"model"`
		Version   string `json:"version"`
	}

	ByShortname []RouterDetails

	AddProfile struct {
		Shortname string   `json:"shortname"`
		Profiles  []string `json:"profiles"`
	}

	DocProfile struct {
		Profile string `json:"profile"`
	}

	DocTree struct {
		Family  string `json:"family"`
		Profile string `json:"profile"`
		Config  string `json:"config"`
	}

	ShortNameRouter struct {
		Shortname string `json:"shortname"`
	}

	Reply struct {
		Status string `json:"status"`
		Msg    string `json:"msg"`
	}

	ReplyStats struct {
		Status string      `json:"status"`
		Msg    string      `json:"msg"`
		Data   interface{} `json:"data,omitempty"`
	}

	ReplyRouter struct {
		Status  string `json:"status"`
		Family  string `json:"family"`
		Model   string `json:"model"`
		Version string `json:"version"`
	}

	ReplyDoc struct {
		Status string `json:"status"`
		Img    string `json:"img"`
		Desc   string `json:"desc"`
		Tele   string `json:"tele"`
		Graf   string `json:"graf"`
		Kapa   string `json:"kapa"`
	}

	ReplyTree struct {
		Status string   `json:"status"`
		Tree   TreeView `json:"tree"`
	}

	Credential struct {
		NetconfUser string `json:"netuser"`
		NetconfPwd  string `json:"netpwd"`
		GnmiUser    string `json:"gnmiuser"`
		GnmiPwd     string `json:"gnmipwd"`
		UseTls      string `json:"usetls"`
		SkipVerify  string `json:"skipverify"`
		ClientTls   string `json:"clienttls"`
	}

	InfluxMgt struct {
		Action string `json:"action"`
		Data   string `json:"data"`
	}

	IntervalMgt struct {
		Action string `json:"action"`
		Data   string `json:"data"`
	}

	ReplyInterval struct {
		Status    string         `json:"status"`
		Intervals []PathInterval `json:"intervals"`
	}

	PathInterval struct {
		Path       string   `json:"path"`
		Default    int      `json:"default-interval"`
		Configured int      `json:"configured-interval"`
		Assigned   []string `json:"assigned"`
	}

	UpdateDebug struct {
		Instance string `json:"instance"`
	}

	TreePath struct {
		Name     string   `json:"name"`
		Interval int      `json:"interval"`
		Aliases  []string `json:"aliases"`
		Origin   string   `json:"origin"`
		Fields   []string `json:"listOfFields"`
	}

	TreeView struct {
		RootName string     `json:"rootName"`
		Paths    []TreePath `json:"listOfPaths"`
	}
)

func (a ByShortname) Len() int           { return len(a) }
func (a ByShortname) Less(i, j int) bool { return a[i].Shortname < a[j].Shortname }
func (a ByShortname) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
