package ewp

import (
	"log"
	"runtime"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU() / 2)
}

func TestElasticWorkerPool(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Fatalf("Expected no race condition. Got: %v", err)
		}
	}()

	var (
		expectedMinWorker int32 = 5
		expectedMaxWorker int32 = 10
		bufferLength      int32 = 5
	)

	ewpConfig := Config{
		MinWorker:           expectedMinWorker,
		MaxWorker:           expectedMaxWorker,
		PoolControlInterval: 2 * time.Second,
		BufferLength:        bufferLength,
	}
	myPool, err := New(ewpConfig, nil, nil)
	if err != nil {
		t.Fatalf("1. Expected no error. Got: %v", err)
	}

	// Test worker pool start up
	myPool.Start()
	time.Sleep(500 * time.Millisecond) // Wait for all workers to start up
	stats := myPool.GetStatistics()
	if stats.CurrWorker != expectedMinWorker {
		t.Fatalf("2. Expected pool started %d workers. Got %d workers", expectedMinWorker, stats.CurrWorker)
	}

	prodStopChan := make(chan struct{})
	// Test job enqueuing
	go func() {
		defer close(prodStopChan)

		for i := int32(0); i < bufferLength*2; i++ {
			slowJobFunc := func() {
				time.Sleep(2 * time.Second)
			}
			if err := myPool.Enqueue(slowJobFunc, 50*time.Millisecond); err != nil {
				t.Fatalf("3. Expected slowJobs enqueued success. Got: %v", err)
			}
		}

		if err := myPool.Enqueue(func() {}, 10*time.Millisecond); err != ErrWorkerTimeoutExceeded {
			t.Fatalf("4. Expected enqueue failed due to timeout, got: %v", err)
		}
	}()
	<-prodStopChan

	expectedJobs := int64(bufferLength * 2)
	time.Sleep(5 * time.Second) // Wait for all published jobs to be executed
	stats = myPool.GetStatistics()
	if stats.EnqueuedJobs != expectedJobs {
		t.Fatalf("5. Expected %d jobs enqueued, got: %d", expectedJobs, stats.EnqueuedJobs)
	}
	if stats.FinishedJobs != expectedJobs {
		t.Fatalf("6. Expected %d jobs finished, got: %d", expectedJobs, stats.FinishedJobs)
	}

	// Test worker pool expanding
	prodStopChan = make(chan struct{})
	go func() {
		defer close(prodStopChan)

		for i := int32(0); i < bufferLength*3; i++ {
			slowJobFunc := func() {
				time.Sleep(2 * time.Second)
			}
			if err := myPool.Enqueue(slowJobFunc); err != nil {
				t.Fatalf("7. Expected slowJobs enqueued success. Got: %v", err)
			}
		}
	}()

	expectedJobs += int64(bufferLength * 3)
	time.Sleep(4 * time.Second) // Wait for worker pool to expand
	stats = myPool.GetStatistics()
	if stats.CurrWorker != expectedMaxWorker {
		t.Fatalf("8. Expected pool to expand to %d workers. Got: %d workers", expectedMaxWorker, stats.CurrWorker)
	}

	// Test worker pool shrinking
	time.Sleep(5 * time.Second)
	stats = myPool.GetStatistics()
	if stats.CurrWorker != expectedMinWorker {
		t.Fatalf("9. Expected pool to shrink to %d workers. Got: %d workers", expectedMinWorker, stats.CurrWorker)
	}

	// Test graceful shutdown
	<-prodStopChan
	if err := myPool.Close(); err != nil {
		t.Fatalf("10. Expected shutdown gracfully. Got: %v", err)
	}
	stats = myPool.GetStatistics()
	if int(stats.CurrWorker) != 0 {
		t.Fatalf("11. Expected pool closed all workers. Got: %d workers remained", stats.CurrWorker)
	}
	if stats.EnqueuedJobs != expectedJobs {
		t.Fatalf("12. Expected total %d jobs enqueued. Got: %d jobs", expectedJobs, stats.EnqueuedJobs)
	}
	if stats.FinishedJobs != expectedJobs {
		t.Fatalf("13. Expected total %d jobs finished. Got: %d jobs", expectedJobs, stats.FinishedJobs)
	}
}

func TestElasticWorkerPool_HighLoad(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Fatalf("Expected no race condition. Got: %v", err)
		}
	}()

	jobNum := 500000

	ewpConfig := Config{
		MinWorker:           int32(runtime.NumCPU()),
		MaxWorker:           int32(runtime.NumCPU() * 2),
		PoolControlInterval: 2 * time.Second,
		BufferLength:        1000,
	}
	myPool, err := New(ewpConfig, nil, nil)
	if err != nil {
		t.Fatalf("1. Expected no error. Got: %v", err)
	}
	myPool.Start()

	prodStopChan := make(chan struct{})
	go func() {
		defer close(prodStopChan)

		for i := 0; i < jobNum; i++ {
			i := i
			stupidJobFunc := func() {
				i++
			}
			if err := myPool.Enqueue(stupidJobFunc); err != nil {
				t.Fatalf("2. Expected stupidJobs enqueued success. Got: %v", err)
			}
		}
	}()

	<-prodStopChan
	if err := myPool.Close(); err != nil {
		t.Fatalf("3. Expected shutdown gracefully. Got: %v", err)
	}
	stats := myPool.GetStatistics()
	if int(stats.EnqueuedJobs) != jobNum {
		t.Fatalf("4. Expected total %d jobs enqueued. Got: %d jobs", jobNum, stats.EnqueuedJobs)
	}
	if int(stats.FinishedJobs) != jobNum {
		t.Fatalf("5. Expected total %d jobs finished. Got: %d jobs", jobNum, stats.FinishedJobs)
	}
}

func ExampleElasticWorkerPool() {
	myPool, _ := NewDefault() // New EWP with default configs
	myPool.Start()            // Start EWP controller and workers

	_ = myPool.Enqueue(func() {
		log.Println("job executed")
	}) // Send job to EWP

	_ = myPool.Close() // Graceful shutdown EWP
}

func ExampleElasticWorkerPool_second() {
	ewpConfig := Config{
		MinWorker:           1,
		MaxWorker:           5,
		PoolControlInterval: 5 * time.Second,
		BufferLength:        10,
	}
	// Create a pool with default PoolController and logging using logrus.Logger.
	myPool, _ := New(ewpConfig, nil, logrus.New())
	myPool.Start()

	// Producer pushes/enqueues jobs to pool.
	prodStopChan, jobNum := make(chan struct{}), 100
	go func() {
		defer func() {
			close(prodStopChan) // Notify producer stopped
		}()

		for i := 0; i < jobNum; i++ {
			i := i
			jobFunc := func() {
				i++
			}
			// Send jobs to pool.
			_ = myPool.Enqueue(jobFunc)
		}
	}()

	time.Sleep(1 * time.Second)
	<-prodStopChan // Block until producer stopped

	// Block until pool gracefully shutdown all its workers or exceed shutdown timeout.
	_ = myPool.Close()
}
