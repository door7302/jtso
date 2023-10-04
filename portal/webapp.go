package portal

import (
	"context"
	"html/template"
	"io"
	"jtso/association"
	"jtso/config"
	"jtso/logger"
	"jtso/netconf"
	"jtso/sqlite"
	"jtso/worker"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const PATH_CERT string = "/var/cert/"

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

	// configure POST routers
	wapp.POST("/addrouter", routeAddRouter)
	wapp.POST("/delrouter", routeDelRouter)
	wapp.POST("/addprofile", routeAddProfile)
	wapp.POST("/delprofile", routeDelProfile)
	wapp.POST("/updatecred", routeUptCred)
	wapp.POST("/updatedoc", routeUptDoc)

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
	teleMx, telePtx, teleAcx, influx, grafana, kapacitor, jtso := "f8cecc", "f8cecc", "f8cecc", "f8cecc", "f8cecc", "f8cecc", "f8cecc"
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
	numMX, numPTX, numACX := 0, 0, 0
	for _, r := range sqlite.RtrList {
		switch r.Family {
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

	return c.Render(http.StatusOK, "index.html", map[string]interface{}{"TeleMx": teleMx, "TelePtx": telePtx, "TeleAcx": teleAcx, "Grafana": grafana, "Kapacitor": kapacitor, "Influx": influx, "Jtso": jtso, "NumMX": numMX, "NumPTX": numPTX, "NumACX": numACX, "GrafanaPort": grafanaPort})
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
	return c.Render(http.StatusOK, "cred.html", map[string]interface{}{"Netuser": sqlite.ActiveCred.NetconfUser, "Netpwd": sqlite.ActiveCred.NetconfPwd, "Gnmiuser": sqlite.ActiveCred.GnmiUser, "Gnmipwd": sqlite.ActiveCred.GnmiPwd, "Usetls": sqlite.ActiveCred.UseTls, "GrafanaPort": grafanaPort})
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

func routeAddRouter(c echo.Context) error {
	var err error

	r := new(NewRouter)

	err = c.Bind(r)
	if err != nil {
		logger.Log.Errorf("Unable to parse Post request for creating a new router: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to create the router"})
	}
	// here we need to issue a Netconf request to retrieve model and version
	reply, err := netconf.GetFacts(r.Hostname, sqlite.ActiveCred.NetconfUser, sqlite.ActiveCred.NetconfPwd, collectCfg.cfg.Enricher.Port)
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
	err = sqlite.DelRouter(r.Shortname)
	if err != nil {
		logger.Log.Errorf("Unable to delete router from DB: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to delete router from DB"})
	}
	logger.Log.Infof("Router %s has been successfully removed", r.Shortname)
	return c.JSON(http.StatusOK, Reply{Status: "OK", Msg: "Router deleted"})
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
	err = sqlite.AddAsso(r.Shortname, r.Profiles)
	if err != nil {
		logger.Log.Errorf("Unable to adding router profile in DB: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to adding router profile in DB"})
	}
	logger.Log.Infof("Profile of router %s has been successfully updated", r.Shortname)
	logger.Log.Info("Force the metadata update")

	go worker.Collect(collectCfg.cfg)
	go association.ConfigueStack(collectCfg.cfg, "all")
	return c.JSON(http.StatusOK, Reply{Status: "OK", Msg: "Router Profile updated"})

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
	err = sqlite.UpdateCredentials(r.NetconfUser, r.NetconfPwd, r.GnmiUser, r.GnmiPwd, r.UseTls)
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
	if p.Definition.TelCfg.MxCfg != "" {
		tele = tele + "&nbsp; &nbsp;" + p.Definition.TelCfg.MxCfg
	}
	if p.Definition.TelCfg.PtxCfg != "" {
		tele = tele + "&nbsp; &nbsp;" + p.Definition.TelCfg.PtxCfg
	}

	if p.Definition.TelCfg.AcxCfg != "" {
		tele = tele + "&nbsp; &nbsp;" + p.Definition.TelCfg.AcxCfg
	}
	if tele == "" {
		tele = "No Telegraf configuration attached to this profile"
	}

	kapa := ""
	for _, v := range p.Definition.KapaCfg {
		kapa = kapa + "&nbsp; &nbsp;" + v
	}
	if kapa == "" {
		kapa = "No Kapacitor script attached to this profile"
	}

	graf := ""
	for _, v := range p.Definition.GrafaCfg {
		graf = graf + "&nbsp; &nbsp;" + v
	}
	if graf == "" {
		graf = "No Grafana Dasboards attached to this profile"
	}

	logger.Log.Infof("Documentation have been successfully updated")
	return c.JSON(http.StatusOK, ReplyDoc{Status: "OK", Img: p.Definition.Cheatsheet, Desc: p.Definition.Description, Tele: tele, Graf: graf, Kapa: kapa})
}
