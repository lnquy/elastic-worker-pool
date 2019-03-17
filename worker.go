package elastic_worker_pool

import "sync"

func startWorker(wg *sync.WaitGroup, jobChan <-chan func(), poisonChan <-chan struct{}, logger Logger, name string) {
	defer wg.Done()

	logger.Printf("  > %s: starting worker", name)
	for {
		select {
		case jobFunc, ok := <-jobChan:
			if !ok {
				logger.Printf("  > %s: jobChan closed. exit", name)
				return
			}
			logger.Printf("  > %s: executing job", name)
			jobFunc()
			logger.Printf("  > %s: job executed", name)
		case <-poisonChan:
			logger.Printf("  > %s: poison pill received. exit", name)
			return
		}
	}
}
