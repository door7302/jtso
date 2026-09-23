package main

import (
	"context"
	"flag"
	"fmt"
	"jtso/association"
	"jtso/config"
	"jtso/container"
	_ "jtso/gnmicollect"
	"jtso/logger"
	_ "jtso/output"
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

const DBPath = "/etc/jtso/jtso.db"

func main() {
	var err error
	flag.Parse()
	if ConfigFile == "" {
		fmt.Println("Please provide the path of the Yaml configuration file")
		os.Exit(0)
	}
	logger.StartLogger()
	defer logger.HandlePanic()

	logger.Log.Info(banner)
	logger.Log.Infof("JTSO version: %s", config.JtsoVersion)

	// Create New Config container
	Cfg := config.NewConfigContainer(ConfigFile)

	// Create a shared Context with cancel function
	ctx, cancel := context.WithCancel(context.Background())

	// Init the sqliteDB
	//err = sqlite.Init("./jtso.db")
	err = sqlite.Init(DBPath, Cfg.JTT.URL != "")
	if err != nil {
		logger.Log.Errorf("unable to open DB... panic...: %v", err)
		panic(err)
	}
	logger.Log.Info("Sqlite DB file loaded successfully")

	// init the webapp
	webapp := portal.New(Cfg)
	if Cfg.Portal.HTTPS {
		logger.Log.Infof("Start HTTPS Server - listen to %d", Cfg.Portal.Port)
	} else {
		logger.Log.Infof("Start HTTP Server  - listen to %d", Cfg.Portal.Port)
	}
	go webapp.Run()

	// create a ticker to refresh the Enrichment struct
	ticker := time.NewTicker(time.Duration(Cfg.Enricher.Interval) * time.Minute)

	// Create the Thread that periodically refreshes the Enrichment struct
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
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
			case <-ctx.Done():
				return
			case <-ticker2.C:
				association.PeriodicCheck(Cfg)
			}
		}
	}()

	// Clean Active profiles - reset directory
	association.CleanActiveDirectory()

	// Trigger a first run of some background processes
	association.PeriodicCheck(Cfg)

	go worker.Collect(Cfg)
	go association.ConfigueStack(Cfg, "all")

	// create a ticker to refresh the docker statistics
	ticker3 := time.NewTicker(1 * time.Minute)

	// Create the Thread that periodically the docker statistics
	container.Init(1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker3.C:
				container.GetContainerStats()
			}
		}
	}()

	// Waiting exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Bye...
	sig := <-c
	fmt.Println("JTSO - received signal: ", sig)

	// Send Close to all threads
	cancel()

	// Stop tickers
	ticker.Stop()
	ticker2.Stop()
	ticker3.Stop()

	// close DB
	sqlite.CloseDb()

	// close logger
	logger.CloseLogger()
}
