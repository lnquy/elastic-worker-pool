package elastic_worker_pool

import "sync"

type worker struct {
	name string
	log  Logger

	wg         *sync.WaitGroup
	jobChan    <-chan func()
	poisonChan <-chan struct{}

	readyHook   func(workerName string)
	exitHook    func(workerName string)
	jobDoneHook func()
}

func newWorker(name string,
	wg *sync.WaitGroup,
	jobChan <-chan func(),
	poisonChan <-chan struct{},
	logger Logger,
	readyHook func(string),
	exitHook func(string),
	jobDoneHook func()) *worker {

	return &worker{
		name:        name,
		log:         logger,
		wg:          wg,
		jobChan:     jobChan,
		poisonChan:  poisonChan,
		readyHook:   readyHook,
		exitHook:    exitHook,
		jobDoneHook: jobDoneHook,
	}
}

func (w *worker) do() {
	defer func() {
		w.exitHook(w.name)
		w.wg.Done()
	}()

	w.log.Debugf("  > %s: starting worker", w.name)
	w.readyHook(w.name)

	for {
		select {
		case jobFunc, ok := <-w.jobChan:
			if !ok {
				w.log.Debugf("  > %s: jobChan closed. exit", w.name)
				return
			}
			jobFunc() // Execute job
			w.jobDoneHook()
		case <-w.poisonChan:
			w.log.Debugf("  > %s: poison pill received. exit", w.name)
			return
		}
	}
}
