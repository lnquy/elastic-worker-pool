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
	defaultBufferLength        = 10
	defaultShutdownTimeout     = 10 * time.Second
	defaultPoolControlInterval = 10 * time.Second
)

type (
	// Config defines all EWP configurations.
	Config struct {
		// Minimum number of workers in worker pool.
		// EWP controller may shrink the pool size to this value at free time.
		MinWorker int32 `json:"min_worker"`

		// Maximum number of workers allowed in worker pool.
		// EWP controller may expand the pool size to this value at high load.
		MaxWorker int32 `json:"max_worker"`

		// The number of jobs allowed to be enqueued to EWP without blocking.
		// If the number of jobs in queue reaches this value, all other enqueuing jobs
		// will be blocked until an already enqueued job be processed.
		BufferLength int32 `json:"buffer_length"`

		// Time to wait for EWP to finish all remaining jobs (graceful shutdown)
		// before terminating it.
		ShutdownTimeout time.Duration `json:"shutdown_timeout"`

		// Interval duration for pool controller to run.
		PoolControlInterval time.Duration `json:"pool_control_interval"`
	}

	// Statistics holds the EWP internal information.
	Statistics struct {
		MinWorker    int32 `json:"min_worker"`
		MaxWorker    int32 `json:"max_worker"`
		BufferLength int32 `json:"buffer_length"`

		// Number of workers currently running in EWP.
		CurrWorker int32 `json:"curr_worker"`
		// Total number of jobs has been enqueued into the EWP.
		EnqueuedJobs int64 `json:"enqueued_jobs"`
		// Total number of jobs has been processed by the EWP workers.
		FinishedJobs int64 `json:"finished_jobs"`
	}

	// ElasticWorkerPool represents the worker pool that can
	// automatically expand/shrink its size.
	ElasticWorkerPool struct {
		name string // Human-readable name
		conf Config
		log  Logger

		// All jobs enqueued to EWP will be pushed to this buffered channel.
		// EWP also distributes jobs to its workers via this channel.
		jobChan chan func()

		// Channel to allow EWP controller to kill the workers.
		workerPoisonChan chan struct{}

		// Channel to notify all waiting/background operations to stop.
		stopChan chan struct{}

		wg           *sync.WaitGroup
		startOnce    sync.Once
		stopOnce     sync.Once
		isStopped    *AtomicBool // Current running state of EWP
		lastWorkerID int         // The ID of last started worker

		// Controller will decide how it should react (expand/shrink pool size)
		// with the current workload level of the EWP.
		controller PoolController

		// Store the internal statistics of EWP for monitoring.
		stats *Statistics
	}
)

// New returns an Elastic Worker Pool (EWP) with provided configurations.
// Should use NewDefault if you want to create an EWP with default configurations instead.
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

// NewDefault creates an EWP with default configurations.
// The AgileController with default load factor levels will be used as the pool controller.
// All EWP logs will be discarded.
func NewDefault() (*ElasticWorkerPool, error) {
	return New(Config{}, nil, nil)
}

// Name returns the human-readable name of current EWP.
func (ewp *ElasticWorkerPool) Name() string {
	return ewp.name
}

// Start starts the EWP controller and all its workers.
// Non-blocking, safe to (mistakenly) call multiple times.
//
// Start must be called before sending jobs to EWP (via Enqueue()).
func (ewp *ElasticWorkerPool) Start() {
	ewp.startOnce.Do(func() {
		ewp.log.Infof("ewp [%s]: starting worker pool\n", ewp.name)
		go ewp.controlPoolSize() // Controller

		// Workers
		ewp.log.Infof("ewp [%s]: starting %d workers\n", ewp.name, ewp.conf.MinWorker)
		for i := int32(0); i < ewp.conf.MinWorker; i++ {
			ewp.wg.Add(1)
			workerName := fmt.Sprintf("%s-%d", ewp.name, i)
			worker := newWorker(workerName, ewp.wg, ewp.jobChan, ewp.workerPoisonChan, ewp.onWorkerReady, ewp.onWorkerExited, ewp.onWorkerJobDone, ewp.log)
			go worker.do()
		}
		ewp.lastWorkerID = int(ewp.conf.MinWorker)
		ewp.log.Infof("ewp [%s]: worker pool started\n", ewp.name)
	})
}

// Enqueue pushes the job to EWP queue, job will be processed later in time (async)
// by EWP workers.
// Safe to call concurrently.
//
// If the EWP buffer still have available slot(s) for new job to be enqueued (BufferLength config) then this method is non-blocking.
//
// In case the EWP buffer is already full, then this method is blocked.
// The blocking time is non-deterministic so caller can provide an optional timeout duration.
// A timeout exceed error will be returned if this method failed to enqueue job during
// the timeout duration.
func (ewp *ElasticWorkerPool) Enqueue(jobFunc func(), timeout ...time.Duration) error {
	if ewp.isStopped.Get() {
		return WorkerPoolStoppedErr
	}

	if len(timeout) == 0 {
		select {
		case <-ewp.stopChan:
			ewp.log.Debugf("ewp [%s]: stopChan closed. Abort sending job to queue", ewp.name)
		case ewp.jobChan <- jobFunc:
			atomic.AddInt64(&ewp.stats.EnqueuedJobs, 1)
			ewp.log.Debugf("ewp [%s]: job enqueued", ewp.name)
		}
		return nil
	}

	select {
	case <-ewp.stopChan:
		ewp.log.Debugf("ewp [%s]: stopChan closed. Abort sending job to queue", ewp.name)
	case ewp.jobChan <- jobFunc:
		atomic.AddInt64(&ewp.stats.EnqueuedJobs, 1)
		ewp.log.Debugf("ewp [%s]: job enqueued", ewp.name)
	case <-time.After(timeout[0]):
		return WorkerTimeoutExceededErr
	}
	return nil
}

// Close gracefully stops the EWP controller and all its workers.
// Blocking at max ShutdownTimeout, safe to (mistakenly) call multiple times.
//
// Close graceful shutdown logic:
//    - Mark the EWP as stopped, won't accept more jobs to be enqueued.
//    - Close stopChan to notify controller to stop. Also abort any jobs waiting
//      to be enqueued to jobChan.
//    - Close jobChan to notify workers to stop and wait until all workers stopped gracefully
//      or ShutdownTimeout exceeded:
//       + Workers will continue to process any enqueued jobs in jobChan.
//         When all jobs has been processed, then workers stop gracefully.
//       + If ShutdownTimeout exceeded before all workers returned,
//         close returns immediately with exceed timeout error.
//
//
// *** IMPORTANT ***
//
// Race condition can happen on this method if producer(s) still running and trying to
// push jobs to ewp via Enqueue().
//
// ==> EWP only guarantee graceful shutdown for its workers.
// It is caller's responsibility to safely stop all producer(s) first, before stopping
// the EWP.
// Otherwise, race condition may happen!
//
// See the examples/ewp/main.go for the example of possible race condition.
// Or take a look on code comments of this method for more detail.
func (ewp *ElasticWorkerPool) Close() (err error) {
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

			// *** IMPORTANT ***
			// Race condition can exist here if producer(s) still running and trying to
			// push jobs to ewp via Enqueue().
			//
			// Explanation: The operation of sending job to channel is not
			// an atomic operation.
			// So there's a case when job is sending (not blocking/waiting but actually
			// doing the send operation) to the channel, and we closes the channel
			// at the same time, that cause race condition.
			//
			// Solution: We can use sync.Mutex to guard the jobChan whenever we're sending
			// jobs to that channel.
			// But it will slowdown the whole worker pool, as every Enqueue() call now will
			// have to wait for each others to acquire the mutex lock.
			// That's what I don't want to.
			//
			// ==> So EWP only guarantee graceful shutdown for its workers.
			// It is caller's responsibility to safely stop all producer(s) first,
			// before stopping the worker pool.
			// Otherwise, race condition may happen!
			//
			// See the examples/ewp/main.go for the example of possible race condition.
			// *****************
			ewp.log.Debugf("ewp [%s]: closing jobChan", ewp.name)
			close(ewp.jobChan)
			ewp.log.Debugf("ewp [%s]: closed jobChan", ewp.name)
			// *****************

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
			err = ShutdownTimeoutExceededErr
		}
	})

	return err
}

// GetStatistics returns the internal statistics of the EWP.
// Non-blocking, safe to call concurrently.
func (ewp *ElasticWorkerPool) GetStatistics() *Statistics {
	return &Statistics{
		MinWorker:    atomic.LoadInt32(&ewp.stats.MinWorker),
		MaxWorker:    atomic.LoadInt32(&ewp.stats.MaxWorker),
		BufferLength: ewp.stats.BufferLength,
		CurrWorker:   atomic.LoadInt32(&ewp.stats.CurrWorker),
		EnqueuedJobs: atomic.LoadInt64(&ewp.stats.EnqueuedJobs),
		FinishedJobs: atomic.LoadInt64(&ewp.stats.FinishedJobs),
	}
}

// controlPoolSize intervally grabs the EWP statistics then determines the reaction
// should be made (expand/shrink pool size) based on current statistics.
// Each pool controller has its own way of dealing with the same workload level.
func (ewp *ElasticWorkerPool) controlPoolSize() {
	// if ewp.conf.MinWorker == ewp.conf.MaxWorker {
	// 	ewp.log.Infof("ewp [%s]: worker pool controller was not started as pool has fixed size (%d)\n", ewp.name, ewp.conf.MinWorker)
	// 	return
	// }

	defer ewp.log.Infof("ewp [%s]: worker pool controller stopped", ewp.name)
	ticker := time.NewTicker(ewp.conf.PoolControlInterval)
	ewp.log.Infof("ewp [%s]: starting worker pool controller\n", ewp.name)

	for {
		select {
		case <-ticker.C:
			stats := ewp.GetStatistics()
			// Skip checking if buffer length too small or pool size is fixed
			if stats.BufferLength <= 1 || stats.MinWorker == stats.MaxWorker {
				continue
			}
			desiredWorkerNum := ewp.controller.GetDesiredWorkerNum(*stats)
			diff := desiredWorkerNum - int(stats.CurrWorker)

			if diff == 0 {
				ewp.log.Infof("ewp [%s]: controller: pool size remains stable: %d\n", ewp.name, stats.CurrWorker)
				continue
			}

			// Shrink
			if diff < 0 {
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

// SetMinWorker allows to update the minimum number of workers in worker pool.
// If the input number is bigger than current maximum number of workers then no change
// will be applied.
func (ewp *ElasticWorkerPool) SetMinWorker(workerNum int32) {
	max := atomic.LoadInt32(&ewp.stats.MaxWorker)
	if workerNum == 0 || workerNum > max {
		return
	}
	atomic.StoreInt32(&ewp.stats.MinWorker, workerNum)
	atomic.StoreInt32(&ewp.conf.MinWorker, workerNum)
}

// SetMaxWorker allows to update the maximum number of workers in worker pool.
// If the input number is smaller than current minimum number of workers then no change
// will be applied.
func (ewp *ElasticWorkerPool) SetMaxWorker(workerNum int32) {
	min := atomic.LoadInt32(&ewp.stats.MinWorker)
	if workerNum == 0 || workerNum < min {
		return
	}
	atomic.StoreInt32(&ewp.stats.MaxWorker, workerNum)
	atomic.StoreInt32(&ewp.conf.MaxWorker, workerNum)
}

// onWorkerReady is a hook for workers to callback whenever a new worker started up
// and ready to serve works.
// Must be non-blocking and safe to call concurrently.
func (ewp *ElasticWorkerPool) onWorkerReady(workerName string) {
	ewp.log.Debugf("ewp [%s]: worker %s started", ewp.name, workerName)
	atomic.AddInt32(&ewp.stats.CurrWorker, 1)
}

// onWorkerExited is a hook for workers to callback whenever a worker stopped gracefully.
// Must be non-blocking and safe to call concurrently.
func (ewp *ElasticWorkerPool) onWorkerExited(workerName string) {
	ewp.log.Debugf("ewp [%s]: worker %s exited", ewp.name, workerName)
	atomic.AddInt32(&ewp.stats.CurrWorker, -1)
}

// onWorkerJobDone is a hook for workers to callback whenever a job has been processed.
// Must be non-blocking and safe to call concurrently.
func (ewp *ElasticWorkerPool) onWorkerJobDone(workerName string) {
	ewp.log.Debugf("ewp [%s]: job done on worker %s", ewp.name, workerName)
	atomic.AddInt64(&ewp.stats.FinishedJobs, 1)
}

// validateConfig tries to set invalid configurations to default values.
func validateConfig(ewpConfig *Config) error {
	if ewpConfig.MinWorker <= 0 {
		ewpConfig.MinWorker = int32(runtime.NumCPU())
	}
	if ewpConfig.MaxWorker < ewpConfig.MinWorker {
		ewpConfig.MaxWorker = ewpConfig.MinWorker
	}
	if ewpConfig.BufferLength < 1 {
		ewpConfig.BufferLength = defaultBufferLength
	}
	if ewpConfig.ShutdownTimeout == 0 {
		ewpConfig.ShutdownTimeout = time.Duration(defaultShutdownTimeout)
	}
	if ewpConfig.PoolControlInterval <= time.Second {
		ewpConfig.PoolControlInterval = time.Duration(defaultPoolControlInterval)
	}
	return nil
}
