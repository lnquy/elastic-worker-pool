package main

import (
	"github.com/lnquy/elastic-worker-pool"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"time"
)

func main() {
	ewpConfig := elastic_worker_pool.Config{
		MinWorker:           5,
		MaxWorker:           10,
		PoolControlInterval: 2 * time.Second,
	}
	ewp, err := elastic_worker_pool.New(ewpConfig, nil, logrus.New())
	if err != nil {
		logrus.Panicf("main: failed to create worker pool: %v", err)
	}
	ewp.Start()

	var isClose bool
	producerStopChan := make(chan struct{})
	go func() {
		defer func() {
			close(producerStopChan)
		}()

		for i := 0; i < 1000; i++ {
			if isClose {
				return
			}
			// i will be reuse after each loop,
			// while jobFunc will be executed later (async), so the value of i when
			// jobFunc is being executed may not correct anymore.
			// ==> Keep the current value of i in local variable (counter) and
			// use it in jobFunc instead.
			counter := i
			jobFunc := func() {
				time.Sleep(1 * time.Second)
				logrus.Printf("jobFunc: do #%d", counter)
			}
			if err := ewp.Enqueue(jobFunc, 100*time.Millisecond); err != nil {
				logrus.Errorln("main: enqueue error:", err)
			}
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan
	logrus.Println("SIGINT received")
	logrus.Infoln("main: closing producer")
	isClose = true     // Notify producer to stop
	<-producerStopChan // Wait until producer exited
	logrus.Infoln("main: producer exited")

	logrus.Infoln("main: closing worker pool")
	ewp.Close()
	logrus.Println("app exit")
}
