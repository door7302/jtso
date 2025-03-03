package worker

import (
	"context"
	"jtso/config"
	"jtso/logger"
	"jtso/netconf"
	"jtso/output"
	"jtso/sqlite"
	"strings"
	"sync"
)

func Collect(cfg *config.ConfigContainer) {

	ctx := context.Background()

	// create the pooler
	p, err := NewSimplePool(cfg.Enricher.Workers, 0, ctx)
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
		if rtr.Profile == 1 {
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
			if rtr.Profile == 1 {
				p.AddWork(&netconf.RouterTask{
					Name:    strings.TrimSpace(rtr.Hostname),
					User:    sqlite.ActiveCred.NetconfUser,
					Pwd:     sqlite.ActiveCred.NetconfPwd,
					Family:  rtr.Family,
					Port:    cfg.Netconf.Port,
					Timeout: cfg.Netconf.RpcTimeout,
					Wg:      wg,
					Jsonify: output.MyMeta,
				})
			}
		}
		wg.Wait()
		err := output.MyMeta.MarshallMeta(cfg.Enricher.Folder)
		if err != nil {
			logger.Log.Error("Unexpected error while creating the Json files: ", err)
		}
		logger.Log.Info("Workers have done all their jobs")
	} else {
		logger.Log.Info("No enrichment job to do")
	}
}
