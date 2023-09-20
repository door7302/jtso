package portal

type (
	TabRtr struct {
		Hostname  string
		Shortname string
		Family    string
		Login     string
	}

	TabAsso struct {
		Shortname string
		Profiles  string
	}

	NewRouter struct {
		Hostname  string `json:"hostname"`
		ShortName string `json:"shortname"`
		Login     string `json:"login"`
		Password  string `json:"password"`
		Family    string `json:"family"`
	}

	DeletedRouter struct {
		Hostname string `json:"hostname"`
	}

	AddProfile struct {
		ShortName string   `json:"shortname"`
		Profiles  []string `json:"profiles"`
	}

	DelProfile struct {
		ShortName string `json:"shortname"`
	}

	Reply struct {
		Status string `json:"status"`
		Msg    string `json:"msg"`
	}
)
