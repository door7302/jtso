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

	// init the webapp
	webapp := portal.New(Cfg.Portal.Port)
	go webapp.Run()
	logger.Log.Infof("Start web server, listen to %d", Cfg.Portal.Port)

	// Initialize the MetaData structure
	metaData := output.New()

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
	err := metaData.MarshallMeta(Cfg.Enricher.Folder)
	if err != nil {
		logger.Log.Error("Unexpected error while creating the Json files: ", err)
	}

	// Waiting exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	fmt.Println("received signal", <-c)
	// Send Close to all threads
	close()
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

	// Allocate the number of task to WG. = to number of routers
	wg := &sync.WaitGroup{}
	numTasks := len(cfg.Instances[0].Rtrs) + len(cfg.Instances[1].Rtrs) + len(cfg.Instances[2].Rtrs)
	logger.Log.Infof("Number of routers to collect: %d", numTasks)
	wg.Add(numTasks)
	logger.Log.Info("Start dispatching Jobs")
	// Push tasks to worker pool
	// iter on all the intances
	for i := 0; i < 3; i++ {
		for _, rtr := range cfg.Instances[i].Rtrs {
			p.AddWork(&netconf.RouterTask{
				Name:    rtr,
				User:    cfg.Netconf.User,
				Pwd:     cfg.Netconf.Pwd,
				Profile: cfg.Instances[i].Name,
				Port:    cfg.Netconf.Port,
				Timeout: cfg.Netconf.RpcTimeout,
				Wg:      wg,
				Jsonify: m,
			})
		}
	}
	wg.Wait()
	logger.Log.Info("All jobs done... now sleep")
}
