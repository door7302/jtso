package portal

type (
	Entry struct {
		Hostname  string
		Shortname string
		Family    string
		Login     string
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

	UpdateProfle struct {
		Hostname string `json:"hostname"`
		Profile  string `json:"profile"`
	}

	Reply struct {
		Status string `json:"status"`
		Msg    string `json:"msg"`
	}
)
