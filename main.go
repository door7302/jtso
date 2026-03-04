package main

import (
	"context"
	"flag"
	"fmt"
	"jtso/association"
	"jtso/config"
	"jtso/container"
	_ "jtso/gnmicollect"
	"jtso/influx"
	"jtso/kapacitor"
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

const DBPATH = "/etc/jtso/jtso.db"

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
	logger.Log.Infof("JTSO version: %s", config.JTSO_VERSION)

	// Create New Config container
	Cfg := config.NewConfigContainer(ConfigFile)

	// Create a shared Context with cancel function
	ctx, cancel := context.WithCancel(context.Background())

	// Clean all kapacitor tasks
	maxAttempts := Cfg.Kapacitor.BootTimeout
	for i := 1; i <= maxAttempts; i++ {
		if kapacitor.IsKapaRun() {
			logger.Log.Info("Kapacitor module is up and running")
			// Clean all kapacitor tasks
			logger.Log.Info("Start cleaning all active Kapacitor tasks")
			kapacitor.CleanKapa()
			break
		}
		time.Sleep(1 * time.Second)
		if i == maxAttempts {
			logger.Log.Error("Unable to clean Kapacitor tasks. Make sure Kapacitor container is running")
		}
	}

	// Init the sqliteDB
	//err = sqlite.Init("./jtso.db")
	err = sqlite.Init(DBPATH)
	if err != nil {
		logger.Log.Errorf("unable to open DB... panic...: %v", err)
		panic(err)
	}
	logger.Log.Info("Sqlite DB file loaded successfully")

	// init the webapp
	webapp := portal.New(Cfg)
	if Cfg.Portal.Https {
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

	// Check if influxdb retention policy is equal to the default value, if not set it.
	currentRP, _ := influx.GetRetentionPolicyDuration()
	equal, err := influx.RetentionDurationEqual(currentRP, sqlite.ActiveAdmin.RPDuration)
	if err != nil {
		logger.Log.Errorf("Error while comparing influxdb retention policy duration: %v", err)
	}
	if !equal {
		logger.Log.Infof("Change the influxdb retention policy duration from %s to %s", currentRP, sqlite.ActiveAdmin.RPDuration)
		err := influx.AlterRetentionPolicyDuration(sqlite.ActiveAdmin.RPDuration)
		if err != nil {
			logger.Log.Errorf("Error while modifying influxdb retention policy duration: %v", err)
		}
	} else {
		logger.Log.Infof("Retention Policy of influxDB is configured well with duration set to: %s", sqlite.ActiveAdmin.RPDuration)
	}

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
