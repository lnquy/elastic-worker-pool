package main

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/lnquy/elastic-worker-pool"
)

func main() {
	ewpConfig := ewp.Config{
		// Minimum worker pool size.
		MinWorker: 5,
		// Maximum worker pool size.
		MaxWorker: 20,
		// Check and determine if it should expand/shrink pool size each 10 seconds.
		PoolControlInterval: 10 * time.Second,
		// The number of items can be pushed/enqueued without blocking
		// (len of buffered channel).
		BufferLength: 1000,
	}

	// Create a pool with default PoolController and logging using logrus.Logger.
	myPool, err := ewp.New(ewpConfig, nil, logrus.New())
	if err != nil {
		logrus.Panicf("main: failed to create worker pool: %v", err)
	}
	logrus.Infof("main: elastic worker pool '%s' created", myPool.Name())
	myPool.Start()

	// Producer pushes/enqueues jobs to pool.
	// Pool's workers will execute the jobs later (async).
	prodStopChan := make(chan struct{})
	jobNum := 100000000
	go func() {
		enqueueErrCounter := 0
		defer func() {
			logrus.Infof("producer: %d enqueue errors", enqueueErrCounter)
			close(prodStopChan) // Notify producer stopped
		}()

		for i := 0; i < jobNum; i++ {
			// i will be reuse after each loop,
			// while jobFunc will be executed later (async), so the value of i when
			// jobFunc is being executed may not correct anymore.
			// ==> Keep the current value of i in local variable (counter) and
			// use it in jobFunc instead.
			counter := i
			jobFunc := func() {
				// time.Sleep(1 * time.Second)
				// logrus.Printf("jobFunc: do #%d", counter)
				counter++
			}

			// Send jobs to pool.
			// jobFunc will be executed later in time (async).
			if err := myPool.Enqueue(jobFunc); err != nil {
				enqueueErrCounter++
			}
		}
	}()

	time.Sleep(100*time.Millisecond) // Wait a little bit for producer to start up

	// *** IMPORTANT ***
	// Race condition may happen if you comments these lines below.
	// As the producer may still running and trying to push jobs to
	// worker pool at the same moment when we close the pool.
	// Try to comment and uncomment these lines of code then run: go run -race main.go
	// to test it yourself.
	// *****************
	<-prodStopChan // Block until producer stopped
	logrus.Infoln("main: producer exited")
	// *****************

	logrus.Infoln("main: closing worker pool")
	// Block until pool gracefully shutdown all its workers or exceed shutdown timeout.
	myPool.Close()
	logrus.Println("app exit")
}
