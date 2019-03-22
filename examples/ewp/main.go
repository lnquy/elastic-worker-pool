package main

import (
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/lnquy/elastic-worker-pool"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU() / 2)
}

func main() {
	ewpConfig := ewp.Config{
		MinWorker:           5,
		MaxWorker:           20,
		PoolControlInterval: 10 * time.Second,
		BufferLength:        1000,
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
	isStopped := &ewp.AtomicBool{}
	prodStopChan := make(chan struct{})
	jobNum := 100000000
	go func() {
		enqueueErrCounter := 0
		defer func() {
			logrus.Infof("producer: %d enqueue errors", enqueueErrCounter)
			close(prodStopChan) // Notify producer stopped
		}()

		for i := 0; i < jobNum; i++ {
			if isStopped.Get() {
				return
			}
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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan
	logrus.Infof("main: SIGINT received. Exiting")
	isStopped.Set(true)

	// *** IMPORTANT ***
	// Race condition may happen if you comments these lines below.
	// As the producer may still running and trying to push jobs to
	// worker pool at the same moment when we close the pool.
	// Try to comment and uncomment these lines of code then run: go run -race main.go
	// to test it yourself.
	// *****************
	logrus.Infoln("main: stop producer")
	<-prodStopChan // Block until producer stopped
	logrus.Infoln("main: producer exited")
	// *****************

	logrus.Infoln("main: stop worker pool")
	// Block until pool gracefully shutdown all its workers or exceed shutdown timeout.
	if err := myPool.Close(); err != nil {
		logrus.Errorf("main: shutdown err: %v", err)
	}
	logrus.Println("main: app exit")
}
