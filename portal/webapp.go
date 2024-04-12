package portal

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"jtso/association"
	"jtso/config"
	"jtso/influx"
	"jtso/logger"
	"jtso/netconf"
	"jtso/parser"
	"jtso/sqlite"
	"jtso/worker"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const PATH_CERT string = "/var/cert/"
const PATH_JTS_VERS string = "/etc/jtso/openjts.version"
const PATH_TELE_VERS string = "/var/metadata/telegraf.version"

type WebApp struct {
	listen string
	app    *echo.Echo
}

type collectInfo struct {
	cfg *config.ConfigContainer
}

var collectCfg *collectInfo

// Define the template registry struct
type TemplateRegistry struct {
	templates *template.Template
}

// Implement e.Renderer interface
func (t *TemplateRegistry) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	err := t.templates.ExecuteTemplate(w, name, data)
	if err != nil {
		logger.Log.Errorf("HTML Rendering error: %v", err)
	}
	return err
}

// Init a new we server
func New(cfg *config.ConfigContainer) *WebApp {
	wapp := echo.New()
	//configure app
	wapp.Use(middleware.Static("html/assets"))
	wapp.Use(middleware.CORS())

	//Templating config
	wapp.Renderer = &TemplateRegistry{
		templates: template.Must(template.ParseGlob("html/templates/*")),
	}

	// configure GET routes
	wapp.GET("/", routeIndex)
	wapp.GET("/index.html", routeIndex)
	wapp.GET("/routers.html", routeRouters)
	wapp.GET("/profiles.html", routeProfiles)
	wapp.GET("/cred.html", routeCred)
	wapp.GET("/doc.html", routeDoc)
	wapp.GET("/browser.html", routeBrowse)
	wapp.GET("/stream", routeStream)

	// configure POST routers
	wapp.POST("/addrouter", routeAddRouter)
	wapp.POST("/delrouter", routeDelRouter)
	wapp.POST("/addprofile", routeAddProfile)
	wapp.POST("/delprofile", routeDelProfile)
	wapp.POST("/updatecred", routeUptCred)
	wapp.POST("/updatedoc", routeUptDoc)
	wapp.POST("/influxmgt", routeInfluxMgt)
	wapp.POST("/searchxpath", routeSearchPath)

	collectCfg = new(collectInfo)
	collectCfg.cfg = cfg

	// return app
	return &WebApp{
		listen: ":" + strconv.Itoa(cfg.Portal.Port),
		app:    wapp,
	}

}

func (w *WebApp) Run() {
	if collectCfg.cfg.Portal.Https {
		if err := w.app.StartTLS(w.listen, PATH_CERT+collectCfg.cfg.Portal.ServerCrt, PATH_CERT+collectCfg.cfg.Portal.ServerKey); err != http.ErrServerClosed {
			logger.Log.Errorf("Unable to start HTTPS server: %v", err)
			panic(err)
		}
	} else {
		if err := w.app.Start(w.listen); err != http.ErrServerClosed {
			logger.Log.Errorf("Unable to start HTTP server: %v", err)
			panic(err)
		}
	}
}

func routeIndex(c echo.Context) error {
	grafanaPort := collectCfg.cfg.Grafana.Port
	teleVmx, teleMx, telePtx, teleAcx, influx, grafana, kapacitor, jtso := "f8cecc", "f8cecc", "f8cecc", "f8cecc", "f8cecc", "f8cecc", "f8cecc", "f8cecc"
	// check containers state

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Log.Errorf("Unable to open Docker session: %v", err)

	}
	defer cli.Close()
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		logger.Log.Errorf("Unable to list container state: %v", err)

	}
	for _, container := range containers {
		switch container.Names[0] {
		case "/telegraf_vmx":
			if container.State == "running" {
				teleVmx = "ccffcc"
			}
		case "/telegraf_mx":
			if container.State == "running" {
				teleMx = "ccffcc"
			}
		case "/telegraf_ptx":
			if container.State == "running" {
				telePtx = "ccffcc"
			}
		case "/telegraf_acx":
			if container.State == "running" {
				teleAcx = "ccffcc"
			}
		case "/grafana":
			if container.State == "running" {
				grafana = "ccffcc"
			}
		case "/kapacitor":
			if container.State == "running" {
				kapacitor = "ccffcc"
			}
		case "/influxdb":
			if container.State == "running" {
				influx = "ccffcc"
			}
		case "/jtso":
			if container.State == "running" {
				jtso = "ccffcc"
			}
		}
	}

	// Retrive number of active routers per Telegraf
	numVMX, numMX, numPTX, numACX := 0, 0, 0, 0
	for _, r := range sqlite.RtrList {
		switch r.Family {
		case "vmx":
			if r.Profile == 1 {
				numVMX++
			}
		case "mx":
			if r.Profile == 1 {
				numMX++
			}
		case "ptx":
			if r.Profile == 1 {
				numPTX++
			}
		case "acx":
			if r.Profile == 1 {
				numACX++
			}
		}
	}

	// Retrieve module's version
	jtsoVersion := config.JTSO_VERSION
	jtsVersion := "N/A"
	teleVersion := "N/A"

	// Open the OpenJTS version's file
	file_jts, err := os.Open(PATH_JTS_VERS)
	if err != nil {
		logger.Log.Errorf("Unable to open %s file: %v", PATH_JTS_VERS, err)
	} else {
		defer file_jts.Close()
		scanner := bufio.NewScanner(file_jts)
		if scanner.Scan() {
			jtsVersion = scanner.Text()
		}
		// Check for any errors during scanning
		if err := scanner.Err(); err != nil {
			logger.Log.Errorf("Unable to parse %s file: %v", PATH_JTS_VERS, err)
		}
	}

	// Open the Telegraf version's file
	file_tele, err := os.Open(PATH_TELE_VERS)
	if err != nil {
		logger.Log.Errorf("Unable to open %s file: %v", PATH_TELE_VERS, err)
	} else {
		defer file_tele.Close()
		scanner := bufio.NewScanner(file_tele)
		if scanner.Scan() {
			teleVersion = scanner.Text()
		}
		// Check for any errors during scanning
		if err := scanner.Err(); err != nil {
			logger.Log.Errorf("Unable to parse %s file: %v", PATH_TELE_VERS, err)
		}
	}

	return c.Render(http.StatusOK, "index.html", map[string]interface{}{"TeleVmx": teleVmx, "TeleMx": teleMx, "TelePtx": telePtx, "TeleAcx": teleAcx,
		"Grafana": grafana, "Kapacitor": kapacitor, "Influx": influx, "Jtso": jtso, "NumVMX": numVMX, "NumMX": numMX, "NumPTX": numPTX, "NumACX": numACX,
		"GrafanaPort": grafanaPort, "JTS_VERS": jtsVersion, "JTSO_VERS": jtsoVersion, "JTS_TELE_VERS": teleVersion})
}

func routeRouters(c echo.Context) error {
	grafanaPort := collectCfg.cfg.Grafana.Port

	// Get all routers from db
	var lr []RouterDetails
	lr = make([]RouterDetails, 0)

	for _, r := range sqlite.RtrList {
		lr = append(lr, RouterDetails{Hostname: r.Hostname, Shortname: r.Shortname, Family: r.Family, Model: r.Model, Version: r.Version})
	}
	return c.Render(http.StatusOK, "routers.html", map[string]interface{}{"Rtrs": lr, "GrafanaPort": grafanaPort})
}

func routeCred(c echo.Context) error {
	grafanaPort := collectCfg.cfg.Grafana.Port
	return c.Render(http.StatusOK, "cred.html", map[string]interface{}{"Netuser": sqlite.ActiveCred.NetconfUser, "Netpwd": sqlite.ActiveCred.NetconfPwd, "Gnmiuser": sqlite.ActiveCred.GnmiUser, "Gnmipwd": sqlite.ActiveCred.GnmiPwd, "Usetls": sqlite.ActiveCred.UseTls, "Skipverify": sqlite.ActiveCred.SkipVerify, "Clienttls": sqlite.ActiveCred.ClientTls, "GrafanaPort": grafanaPort})
}

func routeProfiles(c echo.Context) error {
	grafanaPort := collectCfg.cfg.Grafana.Port
	// Get all routers from db
	var lr []RouterDetails
	var lp []string

	lr = make([]RouterDetails, 0)
	lp = make([]string, 0)

	for _, r := range sqlite.RtrList {
		lr = append(lr, RouterDetails{Hostname: r.Hostname, Shortname: r.Shortname, Family: r.Family, Model: r.Model, Version: r.Version})
	}
	association.ProfileLock.Lock()
	for k, _ := range association.ActiveProfiles {
		lp = append(lp, k)
	}
	association.ProfileLock.Unlock()

	// Get All associations from db
	var la []TabAsso
	la = make([]TabAsso, 0)

	for _, r := range sqlite.AssoList {
		var asso string
		for i, a := range r.Assos {
			if i != len(r.Assos)-1 {
				asso += a + " ; "
			} else {
				asso += a
			}
		}
		la = append(la, TabAsso{Shortname: r.Shortname, Profiles: asso})
	}
	return c.Render(http.StatusOK, "profiles.html", map[string]interface{}{"Rtrs": lr, "Assos": la, "Profiles": lp, "GrafanaPort": grafanaPort})
}

func routeDoc(c echo.Context) error {
	grafanaPort := collectCfg.cfg.Grafana.Port
	// Get all profiles
	var lp []string

	lp = make([]string, 0)

	association.ProfileLock.Lock()
	for k, _ := range association.ActiveProfiles {
		lp = append(lp, k)
	}
	association.ProfileLock.Unlock()

	return c.Render(http.StatusOK, "doc.html", map[string]interface{}{"Profiles": lp, "GrafanaPort": grafanaPort})
}

func routeBrowse(c echo.Context) error {
	grafanaPort := collectCfg.cfg.Grafana.Port

	// Get all routers from db
	var lr []RouterDetails
	lr = make([]RouterDetails, 0)

	for _, r := range sqlite.RtrList {
		lr = append(lr, RouterDetails{Hostname: r.Hostname, Shortname: r.Shortname, Family: r.Family, Model: r.Model, Version: r.Version})
	}

	return c.Render(http.StatusOK, "browser.html", map[string]interface{}{"Rtrs": lr, "GrafanaPort": grafanaPort})
}

func routeAddRouter(c echo.Context) error {
	var err error

	r := new(NewRouter)

	err = c.Bind(r)
	if err != nil {
		logger.Log.Errorf("Unable to parse Post request for creating a new router: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to create the router"})
	}
	// here we need to issue a Netconf request to retrieve model and version
	reply, err := netconf.GetFacts(r.Hostname, sqlite.ActiveCred.NetconfUser, sqlite.ActiveCred.NetconfPwd, collectCfg.cfg.Netconf.Port)
	if err != nil {
		logger.Log.Errorf("Unable to retrieve router facts: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to retrieve router facts"})
	}
	// derive family from model
	var f string
	if strings.ToLower(string(reply.Model[0])) == "m" {
		f = strings.ToLower(string(reply.Model[0:2]))
	} else {
		f = strings.ToLower(string(reply.Model[0:3]))
	}
	err = sqlite.AddRouter(r.Hostname, r.Shortname, f, reply.Model, reply.Ver)
	if err != nil {
		logger.Log.Errorf("Unable to add a new router in DB: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to add router in DB"})
	}
	logger.Log.Infof("Router %s has been successfully added - family %s - model %s - version %s", r.Hostname, f, reply.Model, reply.Ver)
	return c.JSON(http.StatusOK, ReplyRouter{Status: "OK", Family: f, Model: reply.Model, Version: reply.Ver})
}

func routeDelRouter(c echo.Context) error {
	var err error

	r := new(DeletedRouter)

	err = c.Bind(r)
	if err != nil {
		logger.Log.Errorf("Unable to parse Post request for deleting a router: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to delete the router"})
	}
	f, err := sqlite.CheckAsso(r.Shortname)
	if err != nil {
		logger.Log.Errorf("Unable to check router profile in DB: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to check router profile in DB"})
	}
	if f {
		logger.Log.Errorf("Router can't be removed - there is an association: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "You can't remove a router associated to a Profile"})
	}
	// before removing retrieve long name of the router
	ln := ""
	for _, v := range sqlite.RtrList {
		if v.Shortname == r.Shortname {
			ln = v.Hostname
			break
		}
	}
	err = sqlite.DelRouter(r.Shortname)
	if err != nil {
		logger.Log.Errorf("Unable to delete router from DB: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to delete router from DB"})
	}
	if ln != "" {
		err = influx.DropRouter(ln)
		if err != nil {
			logger.Log.Errorf("Unable to delete router from InfluxDB: %v", err)
			return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to delete router from InfluxDB"})
		}
	}
	logger.Log.Infof("Router %s has been successfully removed", r.Shortname)
	return c.JSON(http.StatusOK, Reply{Status: "OK", Msg: "Router deleted"})
}

func checkRouterSupport(filenames []association.Config, routerVersion string) bool {
	confToApply := ""
	defaultConfig := ""
	for _, c := range filenames {
		// Save all config if present as a fallback solution if specific version not found
		if c.Version == "all" {
			defaultConfig = c.Config
		} else {
			result := association.CheckVersion(c.Version, routerVersion)
			if result && (confToApply == "") {
				return true
			}
		}
	}
	if defaultConfig != "" {
		return true
	}
	return false
}

func routeAddProfile(c echo.Context) error {
	var err error
	var f bool

	r := new(AddProfile)

	err = c.Bind(r)
	if err != nil {
		logger.Log.Errorf("Unable to parse Post request for adding router profile: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to adding the router profile"})
	}
	f, err = sqlite.CheckAsso(r.Shortname)
	if err != nil {
		logger.Log.Errorf("Unable to adding router profile in DB: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to adding router profile in DB"})
	}
	if f {
		logger.Log.Errorf("Router %s is already assigned to a profile", r.Shortname)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Router is already assigned to a profile."})
	}
	// Check if a profile can be attached to a router
	// find out the family of the router
	fam := ""
	version := ""
	for _, i := range sqlite.RtrList {
		if i.Shortname == r.Shortname {
			fam = i.Family
			version = i.Version
			break
		}
	}
	// Now check for each profile there is a given Telegraf config
	valid := false
	errString := ""
	for _, i := range r.Profiles {
		allTele := association.ActiveProfiles[i].Definition.TelCfg
		switch fam {
		case "vmx":
			if len(allTele.VmxCfg) == 0 {
				errString += "There is no Telegraf config for profile " + i + " for the VMX platform.</br>"
			} else {
				if checkRouterSupport(allTele.VmxCfg, version) {
					valid = true
				} else {
					errString += "There is no Telegraf config for profile " + i + " for this VMX version.</br>"
				}
			}
		case "mx":
			if len(allTele.MxCfg) == 0 {
				errString += "There is no Telegraf config for profile " + i + " for the MX platform.</br>"
			} else {
				if checkRouterSupport(allTele.MxCfg, version) {
					valid = true
				} else {
					errString += "There is no Telegraf config for profile " + i + " for this MX version.</br>"
				}
			}
		case "ptx":
			if len(allTele.PtxCfg) == 0 {
				errString += "There is no Telegraf config for profile " + i + " for the PTX platform.</br>"
			} else {
				if checkRouterSupport(allTele.PtxCfg, version) {
					valid = true
				} else {
					errString += "There is no Telegraf config for profile " + i + " for this PTX version.</br>"
				}
			}
		case "acx":
			if len(allTele.AcxCfg) == 0 {
				errString += "There is no Telegraf config for profile " + i + " for the ACX platform.</br>"
			} else {
				if checkRouterSupport(allTele.AcxCfg, version) {
					valid = true
				} else {
					errString += "There is no Telegraf config for profile " + i + " for this ACX version.</br>"
				}
			}
		default:
			errString += "There is no Telegraf config for profile " + i + " for the unknown platform.</br>"
		}
	}

	if !valid {
		logger.Log.Errorf("Router %s is not compatible with one or more profiles", r.Shortname)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Incompatibility issue:</br></br>" + errString + "</br>Check Doc menu for details..."})
	}

	err = sqlite.AddAsso(r.Shortname, r.Profiles)
	if err != nil {
		logger.Log.Errorf("Unable to adding router profile in DB: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to adding router profile in DB"})
	}
	logger.Log.Infof("Profile of router %s has been successfully updated", r.Shortname)
	logger.Log.Info("Force the metadata update")

	go worker.Collect(collectCfg.cfg)
	// update the stack for the right family
	go association.ConfigueStack(collectCfg.cfg, fam)
	return c.JSON(http.StatusOK, Reply{Status: "OK", Msg: "Router Profile updated"})

}

func routeSearchPath(c echo.Context) error {
	var err error

	// check if other instance is already running
	if parser.StreamObj.Stream != 0 {
		logger.Log.Errorf("Streaming already running for path %s", parser.StreamObj.Path)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Another instance is currently requesting XPATH search. Retry later..."})
	}
	// change the streamer state to pending stream API request
	parser.StreamObj.Stream = 1

	r := new(SearchPath)
	err = c.Bind(r)

	if err != nil {
		logger.Log.Errorf("Unable to parse Post request for searching XPATH: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to parse Post request for searching XPATH"})
	}
	h := ""
	for _, i := range sqlite.RtrList {
		if i.Shortname == r.Shortname {
			h = i.Hostname
			break
		}
	}
	parser.StreamObj.Router = h
	parser.StreamObj.Port = collectCfg.cfg.Gnmi.Port
	parser.StreamObj.Path = r.Xpath
	parser.StreamObj.Merger = r.Merge
	parser.StreamObj.StopStreaming = make(chan struct{})

	return c.JSON(http.StatusOK, Reply{Status: "OK", Msg: "Streaming well started."})
}

func routeStream(c echo.Context) error {
	// Set the response header for Server-Sent Events
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	// Flush the response buffer
	c.Response().Flush()

	if parser.StreamObj.Stream == 0 {
		logger.Log.Errorf("Bad request - direct access of /stream is not allowed")
		c.JSON(http.StatusBadRequest, Reply{Status: "NOK", Msg: "Bad request - direct access of /stream is not allowed"})
		c.Response().Flush()
		return nil
	} else if parser.StreamObj.Stream == 1 {

		// change the state of the stream to streaming
		parser.StreamObj.Stream = 2
		// Pass the context to parser
		parser.StreamObj.Flusher, _ = c.Response().Writer.(http.Flusher)
		parser.StreamObj.Writer = c.Response().Writer
		parser.StreamObj.Ticker = time.Now()
		parser.StreamObj.ForceFlush = true
		// launch parser
		go parser.LaunchSearch()
		// loop until the end
		for {
			select {
			case <-parser.StreamObj.StopStreaming:
				var jsTree []parser.TreeJs
				// depending on the error report:
				errString := parser.StreamObj.Error.Error()

				// Normal end
				if strings.Contains(errString, "context canceled") {

					parser.StreamData("End of the subscription. Close gNMI session", "OK")
					logger.Log.Info("Generate payload based on the Tree")
					jsTree = make([]parser.TreeJs, 0)
					parser.TraverseTree(parser.StreamObj.Result, "#", &jsTree)
					jsonData, err := json.Marshal(jsTree)
					if err != nil {
						logger.Log.Errorf("Unable to marshall the result: %v", err)
						parser.StreamData(fmt.Sprintf("Unable to marshall the result: %s", err.Error()), "ERROR")
					} else {
						logger.Log.Info("Marshall the result: success")
						// Convert the JSON data to a string
						jsonString := string(jsonData)
						parser.StreamData("End of the collection.", "END", jsonString)
					}
				} else {
					logger.Log.Errorf("Unexpected gnmi error: %v", errString)
					parser.StreamData(fmt.Sprintf("Unexpected gnmi error: %s", errString), "ERROR")
				}

				parser.StreamObj.Stream = 0
				logger.Log.Info("Streaming has been now stopped properly...")
				time.Sleep(500 * time.Millisecond)
				return nil
			}
		}
	}
	return nil

}

func routeDelProfile(c echo.Context) error {
	var err error

	r := new(DelProfile)

	err = c.Bind(r)
	if err != nil {
		logger.Log.Errorf("Unable to parse Post request for deleting router profile: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to delete the router profile"})
	}
	err = sqlite.DelAsso(r.Shortname)
	if err != nil {
		logger.Log.Errorf("Unable to delete router profile in DB: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to delete router profile in DB"})
	}
	logger.Log.Infof("Profile of router %s has been successfully deleted", r.Shortname)
	// find out the family of the router
	fam := "all"
	for _, i := range sqlite.RtrList {
		if i.Shortname == r.Shortname {
			fam = i.Family
			break
		}
	}
	// update the stack for the right family
	go association.ConfigueStack(collectCfg.cfg, fam)
	return c.JSON(http.StatusOK, Reply{Status: "OK", Msg: "Router Profile deleted"})

}

func routeUptCred(c echo.Context) error {
	var err error

	r := new(Credential)

	err = c.Bind(r)
	if err != nil {
		logger.Log.Errorf("Unable to parse Post request for updating credentials: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to update credentials"})
	}
	err = sqlite.UpdateCredentials(r.NetconfUser, r.NetconfPwd, r.GnmiUser, r.GnmiPwd, r.UseTls, r.SkipVerify, r.ClientTls)
	if err != nil {
		logger.Log.Errorf("Unable to update credentials: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to update credentials"})
	}
	logger.Log.Infof("Credentials have been successfully deleted")
	return c.JSON(http.StatusOK, Reply{Status: "OK", Msg: "Credentials have been updated"})

}

func routeUptDoc(c echo.Context) error {
	var err error

	r := new(UpdateDoc)

	err = c.Bind(r)
	if err != nil {
		logger.Log.Errorf("Unable to parse Post request for updating documentation: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to update documentation"})
	}
	association.ProfileLock.Lock()
	p, ok := association.ActiveProfiles[r.Profile]
	association.ProfileLock.Unlock()
	if !ok {
		logger.Log.Errorf("Unable to update documentation: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to update documentation"})
	}

	tele := ""
	for _, v := range p.Definition.TelCfg.VmxCfg {
		if v.Version == "all" {
			tele += "For VMX version " + v.Version + ": " + v.Config + "</br>"
		} else {
			tele += "For VMX version <=" + v.Version + ": " + v.Config + "</br>"
		}
	}
	for _, v := range p.Definition.TelCfg.MxCfg {
		if v.Version == "all" {
			tele += "For MX version " + v.Version + ": " + v.Config + "</br>"
		} else {
			tele += "For MX version <=" + v.Version + ": " + v.Config + "</br>"
		}
	}
	for _, v := range p.Definition.TelCfg.PtxCfg {
		if v.Version == "all" {
			tele += "For PTX version " + v.Version + ": " + v.Config + "</br>"
		} else {
			tele += "For PTX version <=" + v.Version + ": " + v.Config + "</br>"
		}
	}
	for i, v := range p.Definition.TelCfg.AcxCfg {
		if i == len(p.Definition.TelCfg.AcxCfg)-1 {
			if v.Version == "all" {
				tele += "For ACX version " + v.Version + ": " + v.Config
			} else {
				tele += "For ACX version <=" + v.Version + ": " + v.Config + "</br>"
			}
		} else {
			if v.Version == "all" {
				tele += "For ACX version " + v.Version + ": " + v.Config + "</br>"
			} else {
				tele += "For ACX version <=" + v.Version + ": " + v.Config + "</br>"
			}
		}

	}
	if tele == "" {
		tele = "No Telegraf configuration attached to this profile"
	}

	kapa := ""
	for i, v := range p.Definition.KapaCfg {
		if i == len(p.Definition.KapaCfg)-1 {
			kapa += "Script: " + v
		} else {
			kapa += "Script: " + v + "</br>"
		}

	}
	if kapa == "" {
		kapa = "No Kapacitor script attached to this profile"
	}

	graf := ""
	for i, v := range p.Definition.GrafaCfg {
		if i == len(p.Definition.GrafaCfg)-1 {
			graf += "Dashboard: " + v
		} else {
			graf += "Dashboard: " + v + "</br>"
		}

	}
	if graf == "" {
		graf = "No Grafana Dasboards attached to this profile"
	}

	logger.Log.Infof("Documentation have been successfully updated")
	return c.JSON(http.StatusOK, ReplyDoc{Status: "OK", Img: p.Definition.Cheatsheet, Desc: p.Definition.Description, Tele: tele, Graf: graf, Kapa: kapa})
}

func routeInfluxMgt(c echo.Context) error {
	var err error

	r := new(InfluxMgt)

	err = c.Bind(r)
	if err != nil {
		logger.Log.Errorf("Unable to parse Post request for managing influxDB: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to parse the data"})
	}

	switch r.Action {
	case "emptydb":
		err = influx.EmptyDB()
		if err != nil {
			logger.Log.Errorf("Unable to empty the database: %v", err)
			return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to empty the database"})
		}
		logger.Log.Infof("InfluxDB has been successfully empty")
		return c.JSON(http.StatusOK, Reply{Status: "OK", Msg: "InfluxDB has been successfully empty"})
	default:
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unknown InfluxDB action"})
	}
}
