package portal

type (
	TabAsso struct {
		Shortname string
		Profiles  string
	}

	NewRouter struct {
		Hostname  string `json:"hostname"`
		Shortname string `json:"shortname"`
		Family    string `json:"family"`
	}

	DeletedRouter struct {
		Shortname string `json:"shortname"`
	}

	AddProfile struct {
		Shortname string   `json:"shortname"`
		Profiles  []string `json:"profiles"`
	}

	DelProfile struct {
		Shortname string `json:"shortname"`
	}

	Reply struct {
		Status string `json:"status"`
		Msg    string `json:"msg"`
	}

	Credential struct {
		NetconfUser string `json:"netuser"`
		NetconfPwd  string `json:"netpwd"`
		GnmiUser    string `json:"gnmiuser"`
		GnmiPwd     string `json:"gnmipwd"`
		UseTls      string `json:"usetls"`
	}
)
