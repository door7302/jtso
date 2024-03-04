package portal

type (
	TabAsso struct {
		Shortname string
		Profiles  string
	}

	NewRouter struct {
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

	DeletedRouter struct {
		Shortname string `json:"shortname"`
	}

	AddProfile struct {
		Shortname string   `json:"shortname"`
		Profiles  []string `json:"profiles"`
	}

	UpdateDoc struct {
		Profile string `json:"profile"`
	}

	DelProfile struct {
		Shortname string `json:"shortname"`
	}

	Reply struct {
		Status string `json:"status"`
		Msg    string `json:"msg"`
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
)
