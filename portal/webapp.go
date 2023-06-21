package portal

import (
	"jtso/logger"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type WebApp struct {
	listen string
	app    *echo.Echo
}

// Init a new we server
func New(p int) *WebApp {
	wapp := echo.New()
	//configure app
	wapp.Use(middleware.Static("/html/static"))

	// configure routes
	wapp.GET("/", routeIndex)
	wapp.GET("/index.html", routeIndex)
	wapp.GET("/config.html", routeConfig)
	wapp.GET("/admin.html", routeAdmin)
	wapp.GET("/monitor.html", routeMonitor)

	// return app
	return &WebApp{
		listen: ":" + strconv.Itoa(p),
		app:    wapp,
	}
}

func (w *WebApp) Run() {
	if err := w.app.Start(w.listen); err != http.ErrServerClosed {
		logger.Log.Fatalf("Web server stopped: %v", err)
	}
}

func routeIndex(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}

func routeMonitor(c echo.Context) error {
	return nil
}

func routeConfig(c echo.Context) error {
	return nil
}

func routeAdmin(c echo.Context) error {
	return nil
}
