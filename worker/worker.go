package worker

import (
	"context"
	"fmt"
	"jtso/logger"
	"jtso/netconf"
	"sync"
)

type Pool interface {
	Start()
	Stop()
	AddWork(Task)
}

type Task interface {
	Work() error
}

type SimplePool struct {
	numWorkers int
	tasks      chan Task
	start      sync.Once
	stop       sync.Once
	quit       chan struct{}
	ctx        context.Context
}

var _ Pool = (*SimplePool)(nil)

var ErrNoWorkers = fmt.Errorf("Attempting to create worker pool with less than 1 worker")
var ErrNegativeChannelSize = fmt.Errorf("Attempting to create worker pool with a negative channel size")

func NewSimplePool(numWorkers int, channelSize int, ctx context.Context) (Pool, error) {
	logger.HandlePanic()
	if numWorkers <= 0 {
		return nil, ErrNoWorkers
	}
	if channelSize < 0 {
		return nil, ErrNegativeChannelSize
	}

	tasks := make(chan Task, channelSize)

	return &SimplePool{
		numWorkers: numWorkers,
		tasks:      tasks,
		start:      sync.Once{},
		stop:       sync.Once{},
		quit:       make(chan struct{}),
		ctx:        ctx,
	}, nil
}

func (p *SimplePool) Start() {
	p.start.Do(func() {
		p.startWorkers()
	})
}

func (p *SimplePool) Stop() {
	p.stop.Do(func() {
		logger.Log.Infof("Stopping the worker pool")
		close(p.quit)
	})
}

func (p *SimplePool) AddWork(t Task) {
	logger.HandlePanic()
	select {
	case p.tasks <- t:
	case <-p.ctx.Done():
		logger.Log.Infof("End Signal Received... Stop working")
	case <-p.quit:

	}
}

func (p *SimplePool) startWorkers() {
	logger.HandlePanic()
	for i := 0; i < p.numWorkers; i++ {
		logger.Log.Debugf("Start worker thread number %d", i)
		go func(i int) {
			for {
				select {
				case <-p.ctx.Done():
					logger.Log.Infof("End Signal Received... Stop Worker %d", i)
					return
				case <-p.quit:
					logger.Log.Infof("Stop Worker %d", i)
					return
				case task, ok := <-p.tasks:
					t, isOk := task.(*netconf.RouterTask)
					if !isOk {
						logger.Log.Errorf("Worker %d experienced an issue when casting task", i)
						return
					}
					logger.Log.Debugf("Worker %d receives a JOB for router %s", i, t.Name)
					if !ok {
						logger.Log.Errorf("Worker %d experienced an issue when receiving task", i)
						return
					}
					if err := task.Work(); err != nil {
						logger.Log.Errorf("Worker %d experienced an issue after executing its task: %v", i, err)
					}
					logger.Log.Debugf("Worker %d Job done", i)
				}
			}
		}(i)
	}
}
