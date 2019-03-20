package ewp

import (
	"runtime"
	"testing"
	"time"
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
		expectedMinWorker = 5
		expectedMaxWorker = 10
		bufferLength      = 5
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
	if int(stats.CurrWorker) != expectedMinWorker {
		t.Fatalf("2. Expected pool started %d workers. Got %d workers", expectedMinWorker, stats.CurrWorker)
	}

	prodStopChan := make(chan struct{})
	// Test job enqueuing
	go func() {
		defer close(prodStopChan)

		for i := 0; i < bufferLength*2; i++ {
			slowJobFunc := func() {
				time.Sleep(2 * time.Second)
			}
			if err := myPool.Enqueue(slowJobFunc, 50*time.Millisecond); err != nil {
				t.Fatalf("3. Expected slowJobs enqueued success. Got: %v", err)
			}
		}

		if err := myPool.Enqueue(func() {}, 10*time.Millisecond); err != WorkerTimeoutExceededErr {
			t.Fatalf("4. Expected enqueue failed due to timeout, got: %v", err)
		}
	}()
	<-prodStopChan

	expectedJobs := bufferLength * 2
	time.Sleep(5 * time.Second) // Wait for all published jobs to be executed
	stats = myPool.GetStatistics()
	if int(stats.EnqueuedJobs) != expectedJobs {
		t.Fatalf("5. Expected %d jobs enqueued, got: %d", expectedJobs, stats.EnqueuedJobs)
	}
	if int(stats.FinishedJobs) != expectedJobs {
		t.Fatalf("6. Expected %d jobs finished, got: %d", expectedJobs, stats.FinishedJobs)
	}

	// Test worker pool expanding
	prodStopChan = make(chan struct{})
	go func() {
		defer close(prodStopChan)

		for i := 0; i < bufferLength*3; i++ {
			slowJobFunc := func() {
				time.Sleep(2 * time.Second)
			}
			if err := myPool.Enqueue(slowJobFunc); err != nil {
				t.Fatalf("7. Expected slowJobs enqueued success. Got: %v", err)
			}
		}
	}()

	expectedJobs += bufferLength * 3
	time.Sleep(4 * time.Second) // Wait for worker pool to expand
	stats = myPool.GetStatistics()
	if int(stats.CurrWorker) != expectedMaxWorker {
		t.Fatalf("8. Expected pool to expand to %d workers. Got: %d workers", expectedMaxWorker, stats.CurrWorker)
	}

	// Test worker pool shrinking
	time.Sleep(5 * time.Second)
	stats = myPool.GetStatistics()
	if int(stats.CurrWorker) != expectedMinWorker {
		t.Fatalf("9. Expected pool to shrink to %d workers. Got: %d workers", expectedMinWorker, stats.CurrWorker)
	}

	// Test graceful shutdown
	<-prodStopChan
	myPool.Close()
	stats = myPool.GetStatistics()
	if int(stats.CurrWorker) != 0 {
		t.Fatalf("10. Expected pool closed all workers. Got: %d workers remained", stats.CurrWorker)
	}
	if int(stats.EnqueuedJobs) != expectedJobs {
		t.Fatalf("11. Expected total %d jobs enqueued. Got: %d jobs", expectedJobs, stats.EnqueuedJobs)
	}
	if int(stats.FinishedJobs) != expectedJobs {
		t.Fatalf("12. Expected total %d jobs finished. Got: %d jobs", expectedJobs, stats.FinishedJobs)
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
		MinWorker:           runtime.NumCPU(),
		MaxWorker:           runtime.NumCPU() * 2,
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
	myPool.Close()
	stats := myPool.GetStatistics()
	if int(stats.EnqueuedJobs) != jobNum {
		t.Fatalf("3. Expected total %d jobs enqueued. Got: %d jobs", jobNum, stats.EnqueuedJobs)
	}
	if int(stats.FinishedJobs) != jobNum {
		t.Fatalf("4. Expected total %d jobs finished. Got: %d jobs", jobNum, stats.FinishedJobs)
	}
}