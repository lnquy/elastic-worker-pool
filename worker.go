package elastic_worker_pool

import "sync"

func startWorker(wg *sync.WaitGroup, jobChan <-chan func(), poisonChan <-chan struct{}) {
	defer wg.Done()

	for {
		select {
		case jobFunc, ok := <-jobChan:
			if !ok {
				return
			}
			jobFunc()
		case <-poisonChan:
			return
		}
	}
}
