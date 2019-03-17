package main

import (
	"github.com/lnquy/elastic-worker-pool"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"time"
)

func main() {
	logger := logrus.New()
	ewp, _ := elastic_worker_pool.New(elastic_worker_pool.Config{
		MinWorker: 5,
		MaxWorker: 10,
	}, logger)
	ewp.Start()

	for i := 0; i < 100; i++ {
		f := func() {
			time.Sleep(1 *time.Second)
			logrus.Println("jobFunc: do")
		}
		if err := ewp.Enqueue(f); err != nil {
			logrus.Println("main: enqueue error:", err)
		}
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan
	logrus.Println("SIGINT received")

	ewp.Close()
	logrus.Println("ewp exit")
}
