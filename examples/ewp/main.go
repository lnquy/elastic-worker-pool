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
		MaxWorker:           20,
		PoolControlInterval: 10 * time.Second,
		BufferLength:        1000,
	}
	ewp, err := elastic_worker_pool.New(ewpConfig, nil, logrus.New())
	if err != nil {
		logrus.Panicf("main: failed to create worker pool: %v", err)
	}
	ewp.Start()

	isClose := &elastic_worker_pool.AtomicBool{}
	producerStopChan := make(chan struct{})
	go func() {
		defer func() {
			close(producerStopChan)
		}()

		for i := 0; i < 10000000000; i++ {
			if isClose.Get() {
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
			if err := ewp.Enqueue(jobFunc); err != nil {
				logrus.Errorln("main: enqueue error:", err)
			}
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan
	logrus.Println("SIGINT received")
	logrus.Infoln("main: closing producer")
	isClose.Set(true)  // Notify producer to stop
	<-producerStopChan // Wait until producer exited
	logrus.Infoln("main: producer exited")

	logrus.Infoln("main: closing worker pool")
	ewp.Close()
	logrus.Println("app exit")
}
