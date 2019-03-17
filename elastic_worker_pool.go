package elastic_worker_pool

import (
	"github.com/pkg/errors"
	"runtime"
	"sync"
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
		JobsIn     int
		JobsDone   int
	}

	WorkerPool struct {
		conf Config

		jobChan    chan func()
		poisonChan chan struct{}
		wg         *sync.WaitGroup
		startOnce  sync.Once
		stopOnce   sync.Once
		isStopped  bool
		stats      *Stats
		strategy   Strategy
	}
)

func New(ewpConfig Config) (*WorkerPool, error) {
	if err := validateConfig(ewpConfig); err != nil {
		return nil, errors.Wrapf(err, "invalid config")
	}

	return &WorkerPool{
		conf:       ewpConfig,
		jobChan:    make(chan func(), ewpConfig.BufferSize),
		poisonChan: make(chan struct{}),
		wg:         &sync.WaitGroup{},
		stats: &Stats{
			MinWorker:  ewpConfig.MinWorker,
			MaxWorker:  ewpConfig.MaxWorker,
			BufferSize: ewpConfig.BufferSize,
		},
		strategy: &SimpleStrategy{},
	}, nil
}

func (ewp *WorkerPool) Start() {
	ewp.startOnce.Do(func() {
		go ewp.coordinate()

		for i := 0; i < ewp.conf.MinWorker; i++ {
			go startWorker(ewp.wg, ewp.jobChan, ewp.poisonChan)
		}
	})
}

func (ewp *WorkerPool) Enqueue(jobFunc func(), timeout ...time.Duration) error {
	if ewp.isStopped {
		return errors.New("worker pool stopped")
	}
	if len(timeout) > 0 {
		ewp.jobChan <- jobFunc
		return nil
	}

	select {
	case ewp.jobChan <- jobFunc:
		return nil
	case <-time.After(timeout[0]):
		return errors.New("all workers are busy, timeout exceeded")
	}
}

func (ewp *WorkerPool) Close() {
	ewp.stopOnce.Do(func() {
		ewp.isStopped = true
		shutdownChan := make(chan struct{})

		go func() {
			defer close(shutdownChan)

			close(ewp.jobChan)
			close(ewp.poisonChan)
			ewp.wg.Wait()
		}()

		select {
		case <-shutdownChan:
			return // Graceful shutdown normally
		case <-time.After(ewp.conf.ShutdownTimeout):
			return // Force shutdown after timeout
		}
	})
}

func (ewp *WorkerPool) coordinate() {
	if ewp.conf.MinWorker == ewp.conf.MaxWorker {
		return
	}

	ticker := time.NewTicker(ewp.conf.IntervalCheck)
	for {
		if ewp.isStopped {
			return
		}

		select {
		case <-ticker.C:
			wn := ewp.strategy.CaclWorkerNum(ewp.GetStats())
			if wn == 0 {
				continue
			}
			if wn < 0 { // Shrink
				for i := 0; i > wn; i-- {
					ewp.poisonChan <- struct{}{}
				}
				continue
			}
			for i := 0; i < wn; i++ { // Expand
				go startWorker(ewp.wg, ewp.jobChan, ewp.poisonChan)
			}
		}
	}
}

func (ewp *WorkerPool) GetStats() Stats {
	return *ewp.stats
}

// TODO
func validateConfig(ewpConfig Config) error {
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
		ewpConfig.ShutdownTimeout = time.Duration(defaultIntervalCheck)
	}
	return nil
}
