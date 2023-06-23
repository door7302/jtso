package main

import (
	"context"
	"flag"
	"fmt"
	"jtso/config"
	"jtso/logger"
	"jtso/netconf"
	"jtso/output"
	"jtso/portal"
	"jtso/sqlite"
	"jtso/worker"
	"os"
	"os/signal"
	"sync"
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
	ctx, close := context.WithCancel(context.Background())

	// Init the sqliteDB
	err = sqlite.Init("./mydb.db")
	if err != nil {
		logger.Log.Errorf("unable to open DB... panic...: %v", err)
		panic(err)
	}

	// init the webapp
	webapp := portal.New(Cfg.Portal.Port)
	go webapp.Run()
	logger.Log.Infof("Start web server, listen to %d", Cfg.Portal.Port)

	// Initialize the MetaData structure
	metaData := output.New()

	fmt.Println(Cfg.Enricher.Interval)
	// create a ticker to refresh the Enrichment struct
	ticker := time.NewTicker(Cfg.Enricher.Interval)

	// Create the Thread that periodically refreshes the Enrichment struct
	go func() {
		for {
			select {
			case <-ticker.C:
				collect(Cfg, ctx, metaData)
				err := metaData.MarshallMeta(Cfg.Enricher.Folder)
				if err != nil {
					logger.Log.Error("Unexpected error while creating the Json files: ", err)
				}
			}
		}
	}()

	// Trigger a first run
	collect(Cfg, ctx, metaData)
	// Create the json file of each instance
	err = metaData.MarshallMeta(Cfg.Enricher.Folder)
	if err != nil {
		logger.Log.Error("Unexpected error while creating the Json files: ", err)
	}

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

func collect(cfg *config.ConfigContainer, ctx context.Context, m *output.Metadata) {
	// create the pooler
	p, err := worker.NewSimplePool(cfg.Enricher.Workers, 0, ctx)
	if err != nil {
		logger.Log.Errorf("Unable to create worker pool... panic...: %v", err)
		panic(err)
	}
	// Start worker pool
	p.Start()
	defer p.Stop()

	// count the number of router with a profile assigned
	numTasks := 0
	for _, rtr := range sqlite.RtrList {
		if rtr.Profile != "" {
			numTasks++
		}
	}
	if numTasks > 0 {
		// Allocate the number of task to WG. = to number of routers
		wg := &sync.WaitGroup{}
		logger.Log.Infof("Number of routers to collect: %d", numTasks)
		wg.Add(numTasks)
		logger.Log.Info("Start dispatching Jobs")
		// Push tasks to worker pool
		// iter on all the intances
		for _, rtr := range sqlite.RtrList {
			// only for routers with a Profile assigned
			if rtr.Profile != "" {
				p.AddWork(&netconf.RouterTask{
					Name:    rtr.Hostname,
					User:    rtr.Login,
					Pwd:     rtr.Pwd,
					Family:  rtr.Family,
					Port:    cfg.Netconf.Port,
					Timeout: cfg.Netconf.RpcTimeout,
					Wg:      wg,
					Jsonify: m,
				})
			}
		}
		wg.Wait()
		logger.Log.Info("All jobs done... now sleep")
	} else {
		logger.Log.Info("No enrichment job to do")
	}
}
