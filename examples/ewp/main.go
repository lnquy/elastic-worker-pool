package main

import (
	"os"
	"os/signal"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/lnquy/elastic-worker-pool"
)

func main() {
	ewpConfig := ewp.Config{
		MinWorker:           5,
		MaxWorker:           20,
		PoolControlInterval: 10 * time.Second,
		BufferLength:        1000,
	}
	myPool, err := ewp.New(ewpConfig, nil, logrus.New())
	if err != nil {
		logrus.Panicf("main: failed to create worker pool: %v", err)
	}
	myPool.Start()

	isClose := &ewp.AtomicBool{}
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
			if err := myPool.Enqueue(jobFunc); err != nil {
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
	myPool.Close()
	logrus.Println("app exit")
}
