package portal

import (
	"html/template"
	"io"
	"jtso/logger"
	"jtso/sqlite"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type WebApp struct {
	listen string
	app    *echo.Echo
}

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
func New(p int) *WebApp {
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

	// configure POST routers
	wapp.POST("/addrouter", routeAddRouter)
	wapp.POST("/delrouter", routeDelRouter)
	wapp.POST("/addprofile", routeAddProfile)
	wapp.POST("/delprofile", routeDelProfile)

	// return app
	return &WebApp{
		listen: ":" + strconv.Itoa(p),
		app:    wapp,
	}
}

func (w *WebApp) Run() {
	if err := w.app.Start(w.listen); err != http.ErrServerClosed {
		logger.Log.Errorf("Web server stopped: %v", err)
	}
}

func routeIndex(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}

func routeRouters(c echo.Context) error {
	var lr []TabRtr
	lr = make([]TabRtr, 0)

	for _, r := range sqlite.RtrList {
		lr = append(lr, TabRtr{Hostname: r.Hostname, Shortname: r.Shortname, Family: r.Family, Login: r.Login})
	}
	return c.Render(http.StatusOK, "routers.html", map[string]interface{}{"Rtrs": lr})
}

func routeProfiles(c echo.Context) error {
	var lr []TabAsso
	lr = make([]TabAsso, 0)

	for _, r := range sqlite.AssoList {
		var asso string
		for i, a := range r.Assos {
			if i != len(r.Assos)-1 {
				asso += a + " ; "
			} else {
				asso += a
			}
		}
		lr = append(lr, TabAsso{Shortname: r.Shortname, Profiles: asso})
	}
	return c.Render(http.StatusOK, "profiles.html", map[string]interface{}{"Assos": lr})
}

func routeAddRouter(c echo.Context) error {
	var err error

	r := new(NewRouter)

	err = c.Bind(r)
	if err != nil {
		logger.Log.Errorf("Unable to parse Post request for creating a new router: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to create the router"})
	}
	err = sqlite.AddRouter(r.Hostname, r.ShortName, r.Login, r.Password, r.Family)
	if err != nil {
		logger.Log.Errorf("Unable to add a new router in DB: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to add router in DB"})
	}
	logger.Log.Infof("Router %s has been successfully added", r.Hostname)
	return c.JSON(http.StatusOK, Reply{Status: "OK", Msg: "Router added"})
}

func routeDelRouter(c echo.Context) error {
	var err error

	r := new(DeletedRouter)

	err = c.Bind(r)
	if err != nil {
		logger.Log.Errorf("Unable to parse Post request for deleting a router: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to delete the router"})
	}
	err = sqlite.DelRouter(r.Hostname)
	if err != nil {
		logger.Log.Errorf("Unable to delete router from DB: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to delete router from DB"})
	}
	logger.Log.Infof("Router %s has been successfully removed", r.Hostname)
	return c.JSON(http.StatusOK, Reply{Status: "OK", Msg: "Router deleted"})
}

func routeAddProfile(c echo.Context) error {
	var err error

	r := new(AddProfile)

	err = c.Bind(r)
	if err != nil {
		logger.Log.Errorf("Unable to parse Post request for adding router profile: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to adding the router profile"})
	}
	err = sqlite.AddAsso(r.ShortName, r.Profiles)
	if err != nil {
		logger.Log.Errorf("Unable to adding router profile in DB: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to adding router profile in DB"})
	}
	logger.Log.Infof("Profile of router %s has been successfully updated", r.ShortName)
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
	err = sqlite.DelAsso(r.ShortName)
	if err != nil {
		logger.Log.Errorf("Unable to delete router profile in DB: %v", err)
		return c.JSON(http.StatusOK, Reply{Status: "NOK", Msg: "Unable to delete router profile in DB"})
	}
	logger.Log.Infof("Profile of router %s has been successfully deleted", r.ShortName)
	return c.JSON(http.StatusOK, Reply{Status: "OK", Msg: "Router Profile deleted"})
}
