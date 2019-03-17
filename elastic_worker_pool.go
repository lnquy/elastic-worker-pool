package elastic_worker_pool

import (
	"fmt"
	"github.com/pkg/errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultBufferSize      = 10
	defaultShutdownTimeout = 10 * time.Second
	defaultIntervalCheck   = 10 * time.Second
)

type (
	Config struct {
		MinWorker       int
		MaxWorker       int
		BufferSize      int
		ShutdownTimeout time.Duration
		IntervalCheck   time.Duration
	}

	Stats struct {
		MinWorker  int
		MaxWorker  int
		BufferSize int
		CurrWorker int
		JobsIn     int64
		JobsDone   int64
	}

	WorkerPool struct {
		conf Config
		name string

		jobChan     chan func()
		jobDoneChan chan struct{}
		poisonChan  chan struct{}
		wg          *sync.WaitGroup
		startOnce   sync.Once
		stopOnce    sync.Once
		isStopped   bool
		stats       *Stats
		strategy    Strategy
		log         Logger
		wid         int
	}
)

func New(ewpConfig Config, logger Logger) (*WorkerPool, error) {
	if err := validateConfig(&ewpConfig); err != nil {
		return nil, errors.Wrapf(err, "invalid config")
	}

	return &WorkerPool{
		name: getRandomName(0),
		conf: ewpConfig,

		jobChan:     make(chan func(), ewpConfig.BufferSize),
		jobDoneChan: make(chan struct{}, ewpConfig.BufferSize),
		poisonChan:  make(chan struct{}),
		wg:          &sync.WaitGroup{},
		stats: &Stats{
			MinWorker:  ewpConfig.MinWorker,
			MaxWorker:  ewpConfig.MaxWorker,
			BufferSize: ewpConfig.BufferSize,
		},
		strategy: &SimpleStrategy{},
		log:      logger,
	}, nil
}

func (ewp *WorkerPool) Start() {
	ewp.startOnce.Do(func() {
		ewp.log.Printf("ewp [%s]: starting worker pool\n", ewp.name)
		go ewp.updateWorkerStats()
		go ewp.coordinate()

		ewp.log.Printf("ewp [%s]: starting %d workers\n", ewp.name, ewp.conf.MinWorker)
		for i := 0; i < ewp.conf.MinWorker; i++ {
			ewp.wg.Add(1)
			workerName := fmt.Sprintf("%s:%d", ewp.name, i)
			go startWorker(ewp.wg, ewp.jobChan, ewp.poisonChan, ewp.log, workerName)
		}
		ewp.wid = ewp.conf.MinWorker
		ewp.log.Printf("ewp [%s]: started successfully\n", ewp.name)
	})
}

func (ewp *WorkerPool) Enqueue(jobFunc func(), timeout ...time.Duration) error {
	if ewp.isStopped {
		return errors.New("worker pool stopped")
	}
	if len(timeout) <= 0 {
		ewp.jobChan <- jobFunc
		atomic.AddInt64(&ewp.stats.JobsIn, 1)
		ewp.log.Printf("ewp [%s]: enqueued", ewp.name)
		return nil
	}

	select {
	case ewp.jobChan <- jobFunc:
		ewp.stats.JobsIn++
		ewp.log.Printf("ewp [%s]: enqueued", ewp.name)
		return nil
	case <-time.After(timeout[0]):
		return errors.New("all workers are busy, timeout exceeded")
	}
}

func (ewp *WorkerPool) Close() {
	ewp.stopOnce.Do(func() {
		ewp.log.Printf("ewp [%s]: stopping worker pool\n", ewp.name)
		ewp.isStopped = true
		shutdownChan := make(chan struct{})

		go func() {
			defer close(shutdownChan)

			close(ewp.jobChan)
			close(ewp.jobDoneChan)
			// close(ewp.poisonChan)
			ewp.wg.Wait()
		}()

		select {
		case <-shutdownChan:
			ewp.log.Printf("ewp [%s]: worker pool shutdown gracefully\n", ewp.name)
			return // Graceful shutdown normally
		case <-time.After(ewp.conf.ShutdownTimeout):
			ewp.log.Printf("ewp [%s]: worker pool exceeded shutdown timeout. Force quit\n", ewp.name)
			return // Force shutdown after timeout
		}
	})
}

func (ewp *WorkerPool) coordinate() {
	if ewp.conf.MinWorker == ewp.conf.MaxWorker {
		ewp.log.Printf("ewp [%s]: worker pool's coordinator is not started as minWorker==maxWorker\n")
		return
	}

	defer ewp.log.Printf("ewp [%s]: coordinator stopped", ewp.name)
	ticker := time.NewTicker(ewp.conf.IntervalCheck)
	ewp.log.Printf("ewp [%s]: starting worker pool's coordinator\n", ewp.name)
	for {
		if ewp.isStopped {
			return
		}

		select {
		case <-ticker.C:
			wn := ewp.strategy.CaclWorkerNum(ewp.GetStats())
			if wn == 0 {
				ewp.log.Printf("ewp [%s]: coordinator: no change\n", ewp.name)
				continue
			}
			if wn < 0 { // Shrink
				ewp.log.Printf("ewp [%s]: coordinator: shrink %d to %d workers\n", ewp.name, -wn, ewp.stats.CurrWorker+wn)
				for i := 0; i > wn; i-- {
					ewp.poisonChan <- struct{}{}
				}
				continue
			}
			ewp.log.Printf("ewp [%s]: coordinator: expand %d to %d workers\n", ewp.name, wn, ewp.stats.CurrWorker+wn)
			for i := 0; i < wn; i++ { // Expand
				ewp.wg.Add(1)
				ewp.wid++
				workerName := fmt.Sprintf("%s:%d", ewp.name, ewp.wid)
				go startWorker(ewp.wg, ewp.jobChan, ewp.poisonChan, ewp.log, workerName)
			}
		}
	}
}

func (ewp *WorkerPool) GetStats() Stats {
	return *ewp.stats
}

// TODO
func validateConfig(ewpConfig *Config) error {
	if ewpConfig.MinWorker <= 0 {
		ewpConfig.MinWorker = runtime.NumCPU()
	}
	if ewpConfig.MaxWorker < ewpConfig.MinWorker {
		ewpConfig.MaxWorker = ewpConfig.MinWorker
	}
	if ewpConfig.BufferSize < 1 {
		ewpConfig.BufferSize = defaultBufferSize
	}
	if ewpConfig.ShutdownTimeout == 0 {
		ewpConfig.ShutdownTimeout = time.Duration(defaultShutdownTimeout)
	}
	if ewpConfig.IntervalCheck <= 1 {
		ewpConfig.IntervalCheck = time.Duration(defaultIntervalCheck)
	}
	return nil
}

func (ewp *WorkerPool) updateWorkerStats() {
	for range ewp.jobDoneChan {
		ewp.stats.JobsDone++
	}
}
