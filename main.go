package main

import (
	"context"
	"flag"
	"fmt"
	"jtso/association"
	"jtso/config"
	"jtso/container"
	"jtso/kapacitor"
	"jtso/logger"
	_ "jtso/output"
	_ "jtso/parser"
	"jtso/portal"
	"jtso/sqlite"
	"jtso/worker"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	ConfigFile string
)

func init() {
	flag.StringVar(&ConfigFile, "config", "/etc/jtso/config.yml", "YAML configuration file path")
	flag.BoolVar(&logger.Verbose, "verbose", false, "Enable verbose in the console")
}

const banner = `
     ██ ████████ ███████  ██████  
     ██    ██    ██      ██    ██ 
     ██    ██    ███████ ██    ██ 
██   ██    ██         ██ ██    ██ 
 █████     ██    ███████  ██████  
`

func main() {
	var err error
	flag.Parse()
	if ConfigFile == "" {
		fmt.Println("Please provide the path of the Yaml configuration file")
		os.Exit(0)
	}
	logger.StartLogger()
	logger.HandlePanic()

	logger.Log.Info(banner)

	// Create New Config container
	Cfg := config.NewConfigContainer(ConfigFile)

	// Create a shared Context with cancel function
	_, close := context.WithCancel(context.Background())

	// wait 5 seconds to let docker DNS service to start
	time.Sleep(5 * time.Second)

	// Clean all kapacitor tasks
	logger.Log.Info("Start cleaning all active Kapacitor tasks")
	kapacitor.CleanKapa()

	// Init the sqliteDB
	//err = sqlite.Init("./jtso.db")
	err = sqlite.Init("/etc/jtso/jtso.db")
	if err != nil {
		logger.Log.Errorf("unable to open DB... panic...: %v", err)
		panic(err)
	}

	// init the webapp
	webapp := portal.New(Cfg)
	if Cfg.Portal.Https {
		logger.Log.Infof("Start HTTPS Server - listen to %d", Cfg.Portal.Port)
	} else {
		logger.Log.Infof("Start HTTP Server  - listen to %d", Cfg.Portal.Port)
	}
	go webapp.Run()

	// create a ticker to refresh the Enrichment struct
	ticker := time.NewTicker(Cfg.Enricher.Interval)

	// Create the Thread that periodically refreshes the Enrichment struct
	go func() {
		for {
			select {
			case <-ticker.C:
				worker.Collect(Cfg)
			}
		}
	}()

	// create a ticker to refresh the profiles
	ticker2 := time.NewTicker(1 * time.Minute)

	// Create the Thread that periodically refreshes the profiles
	go func() {
		for {
			select {
			case <-ticker2.C:
				association.PeriodicCheck()
			}
		}
	}()

	// Trigger a first run of some background processes
	association.PeriodicCheck()
	go worker.Collect(Cfg)
	go association.ConfigueStack(Cfg, "all")

	// create a ticker to refresh the docker statistics
	ticker3 := time.NewTicker(1 * time.Minute)

	// Create the Thread that periodically the docker statistics
	container.Init(1)
	go func() {
		for {
			select {
			case <-ticker3.C:
				container.GetContainerStats()
			}
		}
	}()

	// Waiting exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Send Close to all threads
	close()

	// close DB
	defer sqlite.CloseDb()

	// Bye...
	fmt.Println("JTSO - received signal: ", <-c)
}
