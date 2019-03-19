package ewp

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

const (
	defaultBufferSize          = 10
	defaultShutdownTimeout     = 10 * time.Second
	defaultPoolControlInterval = 10 * time.Second
)

type (
	Config struct {
		MinWorker           int           `json:"min_worker"`
		MaxWorker           int           `json:"max_worker"`
		BufferLength        int           `json:"buffer_length"`
		ShutdownTimeout     time.Duration `json:"shutdown_timeout"`
		PoolControlInterval time.Duration `json:"pool_control_interval"`
	}

	Statistics struct {
		MinWorker    int   `json:"min_worker"`
		MaxWorker    int   `json:"max_worker"`
		BufferLength int   `json:"buffer_length"`
		CurrWorker   int32 `json:"curr_worker"`
		EnqueuedJobs int64 `json:"enqueued_jobs"`
		FinishedJobs int64 `json:"finished_jobs"`
	}

	ElasticWorkerPool struct {
		name string
		conf Config
		log  Logger

		jobChan          chan func()
		workerPoisonChan chan struct{}
		stopChan         chan struct{}

		wg           *sync.WaitGroup
		startOnce    sync.Once
		stopOnce     sync.Once
		isStopped    *AtomicBool
		lastWorkerID int

		controller PoolController
		stats      *Statistics
	}
)

func New(conf Config, controller PoolController, logger Logger) (*ElasticWorkerPool, error) {
	if err := validateConfig(&conf); err != nil {
		return nil, errors.Wrapf(err, "invalid config")
	}
	if controller == nil {
		controller = NewAgileController(nil)
	}
	if logger == nil {
		logger = &discardLogger{}
	}

	return &ElasticWorkerPool{
		name: getRandomName(0),
		conf: conf,
		log:  logger,

		jobChan:          make(chan func(), conf.BufferLength),
		workerPoisonChan: make(chan struct{}),
		stopChan:         make(chan struct{}),
		wg:               &sync.WaitGroup{},
		isStopped:        &AtomicBool{},

		controller: controller,
		stats: &Statistics{
			MinWorker:    conf.MinWorker,
			MaxWorker:    conf.MaxWorker,
			BufferLength: conf.BufferLength,
		},
	}, nil
}

func (ewp *ElasticWorkerPool) Start() {
	ewp.startOnce.Do(func() {
		ewp.log.Infof("ewp [%s]: starting worker pool\n", ewp.name)
		go ewp.controlPoolSize()

		ewp.log.Infof("ewp [%s]: starting %d workers\n", ewp.name, ewp.conf.MinWorker)
		for i := 0; i < ewp.conf.MinWorker; i++ {
			ewp.wg.Add(1)
			workerName := fmt.Sprintf("%s-%d", ewp.name, i)
			worker := newWorker(workerName, ewp.wg, ewp.jobChan, ewp.workerPoisonChan, ewp.onWorkerReady, ewp.onWorkerExited, ewp.onWorkerJobDone, ewp.log)
			go worker.do()
		}
		ewp.lastWorkerID = ewp.conf.MinWorker
		ewp.log.Infof("ewp [%s]: worker pool started\n", ewp.name)
	})
}

func (ewp *ElasticWorkerPool) Enqueue(jobFunc func(), timeout ...time.Duration) error {
	if ewp.isStopped.Get() {
		return WorkerPoolStoppedErr
	}

	if len(timeout) == 0 {
		select {
		case <-ewp.stopChan:
			ewp.log.Debugf("ewp [%s]: enqueueStopChan closed. Abort sending job to queue", ewp.name)
		case ewp.jobChan <- jobFunc:
			atomic.AddInt64(&ewp.stats.EnqueuedJobs, 1)
			ewp.log.Debugf("ewp [%s]: job enqueued", ewp.name)
		}
		return nil
	}

	select {
	case ewp.jobChan <- jobFunc:
		ewp.stats.EnqueuedJobs++
		ewp.log.Debugf("ewp [%s]: job enqueued", ewp.name)
		return nil
	case <-time.After(timeout[0]):
		return WorkerTimeoutExceededErr
	}
}

func (ewp *ElasticWorkerPool) GetStatistics() *Statistics {
	return &Statistics{
		MinWorker:    ewp.stats.MinWorker,
		MaxWorker:    ewp.stats.MaxWorker,
		BufferLength: ewp.stats.BufferLength,
		CurrWorker:   atomic.LoadInt32(&ewp.stats.CurrWorker),
		EnqueuedJobs: atomic.LoadInt64(&ewp.stats.EnqueuedJobs),
		FinishedJobs: atomic.LoadInt64(&ewp.stats.FinishedJobs),
	}
}

func (ewp *ElasticWorkerPool) Close() {
	ewp.stopOnce.Do(func() {
		ewp.log.Infof("ewp [%s]: stopping worker pool\n", ewp.name)
		start := time.Now()
		ewp.isStopped.Set(true)

		shutdownChan := make(chan struct{})
		go func() {
			defer close(shutdownChan)

			// Abort jobs are currently waiting to be enqueued and stop the controller first.
			ewp.log.Debugf("ewp [%s]: closing stopChan", ewp.name)
			close(ewp.stopChan)
			ewp.log.Debugf("ewp [%s]: closed stopChan", ewp.name)
			// Then notify workers that the input channel is closed.
			// Workers must try to finish all remaining jobs in the jobChan before exited.
			ewp.log.Debugf("ewp [%s]: closing jobChan", ewp.name)
			close(ewp.jobChan)
			ewp.log.Debugf("ewp [%s]: closed jobChan", ewp.name)
			// Wait until all workers closed gracefully.
			ewp.wg.Wait()
			ewp.log.Infof("ewp [%s]: all workers stopped", ewp.name)
			close(ewp.workerPoisonChan)
		}()

		select {
		case <-shutdownChan: // Graceful shutdown normally
			ewp.log.Infof("ewp [%s]: worker pool shutdown gracefully in %v\n", ewp.name, time.Since(start))
		case <-time.After(ewp.conf.ShutdownTimeout): // Force shutdown after timeout
			ewp.log.Infof("ewp [%s]: worker pool exceeded shutdown timeout. Force quit\n", ewp.name)
		}
	})
}

func (ewp *ElasticWorkerPool) controlPoolSize() {
	if ewp.conf.MinWorker == ewp.conf.MaxWorker {
		ewp.log.Infof("ewp [%s]: worker pool controller was not started as pool has fixed size (%d)\n", ewp.name, ewp.conf.MinWorker)
		return
	}

	defer ewp.log.Infof("ewp [%s]: worker pool controller stopped", ewp.name)
	ticker := time.NewTicker(ewp.conf.PoolControlInterval)
	ewp.log.Infof("ewp [%s]: starting worker pool controller\n", ewp.name)

	for {
		select {
		case <-ticker.C:
			stats := ewp.GetStatistics()
			desiredWorkerNum := ewp.controller.GetDesiredWorkerNum(*stats)
			diff := desiredWorkerNum - int(stats.CurrWorker)

			if diff == 0 {
				ewp.log.Infof("ewp [%s]: controller: pool size remains stable: %d\n", ewp.name, stats.CurrWorker)
				continue
			}

			if diff < 0 { // Shrink
				ewp.log.Infof("ewp [%s]: controller: shrink worker pool: %d -> %d\n", ewp.name, stats.CurrWorker, desiredWorkerNum)
				for i := 0; i > diff; i-- {
					ewp.workerPoisonChan <- struct{}{}
				}
				continue
			}

			// Expand
			ewp.log.Infof("ewp [%s]: controller: expand worker pool: %d -> %d\n", ewp.name, stats.CurrWorker, desiredWorkerNum)
			for i := 0; i < diff; i++ {
				ewp.wg.Add(1)
				ewp.lastWorkerID++
				workerName := fmt.Sprintf("%s-%d", ewp.name, ewp.lastWorkerID)
				worker := newWorker(workerName, ewp.wg, ewp.jobChan, ewp.workerPoisonChan, ewp.onWorkerReady, ewp.onWorkerExited, ewp.onWorkerJobDone, ewp.log)
				go worker.do()
			}

		case <-ewp.stopChan:
			return
		}
	}
}

func (ewp *ElasticWorkerPool) onWorkerReady(workerName string) {
	ewp.log.Debugf("ewp [%s]: worker %s started", ewp.name, workerName)
	atomic.AddInt32(&ewp.stats.CurrWorker, 1)
}

func (ewp *ElasticWorkerPool) onWorkerExited(workerName string) {
	ewp.log.Debugf("ewp [%s]: worker %s exited", ewp.name, workerName)
	atomic.AddInt32(&ewp.stats.CurrWorker, -1)
}

func (ewp *ElasticWorkerPool) onWorkerJobDone(workerName string) {
	ewp.log.Debugf("ewp [%s]: job done on worker %s", ewp.name, workerName)
	atomic.AddInt64(&ewp.stats.FinishedJobs, 1)
}

func validateConfig(ewpConfig *Config) error {
	if ewpConfig.MinWorker <= 0 {
		ewpConfig.MinWorker = runtime.NumCPU()
	}
	if ewpConfig.MaxWorker < ewpConfig.MinWorker {
		ewpConfig.MaxWorker = ewpConfig.MinWorker
	}
	if ewpConfig.BufferLength < 1 {
		ewpConfig.BufferLength = defaultBufferSize
	}
	if ewpConfig.ShutdownTimeout == 0 {
		ewpConfig.ShutdownTimeout = time.Duration(defaultShutdownTimeout)
	}
	if ewpConfig.PoolControlInterval <= time.Second {
		ewpConfig.PoolControlInterval = time.Duration(defaultPoolControlInterval)
	}
	return nil
}
