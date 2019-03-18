package elastic_worker_pool

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorker_Do(t *testing.T) {
	var (
		jobNum     = 30
		poisonPill = 3

		wg         = &sync.WaitGroup{}
		jobChan    = make(chan func(), jobNum)
		poisonChan = make(chan struct{})

		readyCounter int32
		readyHook    = func(name string) {
			atomic.AddInt32(&readyCounter, 1)
		}
		exitCounter int32
		exitHook    = func(name string) {
			atomic.AddInt32(&exitCounter, 1)
		}
		jobDoneCounter int32
		jobDoneHook    = func(name string) {
			atomic.AddInt32(&jobDoneCounter, 1)
		}

		expectedInitWorker        = 10
		expectedReady             = 10
		expectedExitAfterPoisoned = 3
		expectedExit              = 10
		expectedJobDone           = 30
	)

	// Send job to jobChan, non-blocking as len(jobChan) == jobNum
	for i := 0; i < jobNum; i++ {
		i := i
		f := func() {
			i++
		}
		jobChan <- f
	}

	for i := 0; i < expectedInitWorker; i++ {
		wg.Add(1)
		worker := newWorker(strconv.Itoa(i), wg, jobChan, poisonChan, readyHook, exitHook, jobDoneHook, &discardLogger{})
		go worker.do()
	}
	time.Sleep(2 * time.Second) // Wait for all workers to start up
	if int(readyCounter) != expectedReady {
		t.Fatalf("1. Expected all workers to start and call readyHook normally. Expected %d hooks, got %d hooks", expectedReady, readyCounter)
	}

	for i := 0; i < poisonPill; i++ {
		poisonChan <- struct{}{}
	}
	time.Sleep(2 * time.Second) // Wait for all poisoned workers to exit
	if int(exitCounter) != expectedExitAfterPoisoned {
		t.Fatalf("2. Expected all poinsoned workers to stop and call exitHook normally. Expected %d hooks, got %d hooks", expectedExitAfterPoisoned, exitCounter)
	}

	close(jobChan) // Notify all workers to stop
	wg.Wait()

	if int(exitCounter) != expectedExit {
		t.Fatalf("3. Expected all workers to stop and call exitHook normally. Expected %d hooks, got %d hooks", expectedExit, exitCounter)
	}

	if int(jobDoneCounter) != expectedJobDone {
		t.Fatalf("4. Expected all jobs must be executed before exited. Expected %d jobs done, got %d jobs done", expectedJobDone, jobDoneCounter)
	}
}
