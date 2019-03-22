# Elastic Worker Pool
<p align="left">
  <a href="https://godoc.org/github.com/lnquy/elastic-worker-pool" title="GoDoc Reference" rel="nofollow"><img src="https://img.shields.io/badge/go-documentation-blue.svg?style=flat" alt="GoDoc Reference"></a>
  <a href="https://github.com/github.com/lnquy/elastic-worker-pool/releases/tag/v0.0.1" title="0.0.1 Release" rel="nofollow"><img src="https://img.shields.io/badge/version-0.0.1-blue.svg?style=flat" alt="0.0.1 release"></a>
  <a href="https://goreportcard.com/report/github.com/lnquy/elastic-worker-pool"><img src="https://goreportcard.com/badge/github.com/lnquy/elastic-worker-pool" alt="Code Status" /></a>
  <br />
  <!--<a href="https://travis-ci.org/github-com/lnquy/elastic-worker-pool"><img src="https://travis-ci.org/github-com/lnquy/elastic-worker-pool.svg" alt="Build Status" /></a>-->
  <!--<a href='https://coveralls.io/github/fyne-io/fyne?branch=develop'><img src='https://coveralls.io/repos/github/fyne-io/fyne/badge.svg?branch=develop' alt='Coverage Status' /></a>-->
  <!--<a href='https://sourcegraph.com/github.com/fyne-io/fyne?badge'><img src='https://sourcegraph.com/github.com/fyne-io/fyne/-/badge.svg' alt='Used By' /></a-->
</p>

When creating worker pool in Go, most of time we only create a pool with fixed number of workers.
What if the pool can automatically expand its size by starting new workers when under high load and on the other hand, shrink the size by killing free workers when there's not much works to do?
Elastic Worker Pool (ewp) can do that for you.

## Features

- Control worker pool size automatically.
- Graceful shutdown all workers.
- Can write your own pool size controller by implementing [PoolController](https://github.com/lnquy/elastic-worker-pool/blob/master/controller.go).
- No `mutex` or `context.Context`. Synchronization is done using Go `channel`, `select`, `atomic` and simple callbacks. 

## Install
`ewp` is packed as Go module (Go >= 1.11), but it should also be fine when used with older Go version (<= 1.10).
Simply use `go get` command to get the code.

```shell
$ go get -u -v github.com/lnquy/elastic-worker-pool
```

## Examples
```go
package main

import (
	"log"
	"github.com/lnquy/elastic-worker-pool"
)

func main() {
	// Create a pool with default PoolController and omit all logs.
	myPool, _ := ewp.NewDefault()
	myPool.Start()

	// Producer sends jobs to pool.
	prodStopChan := make(chan struct{})
	go func() {
		defer func() {
			close(prodStopChan) // Notify producer stopped
		}()

		jobFunc := func() {
			log.Println("Hello ewp!")
		}
		for i := 0; i < 10; i++ {
			_ = myPool.Enqueue(jobFunc)
		}
	}()

	<-prodStopChan // Block until producer stopped

	// Block until pool gracefully shutdown all its workers or exceed shutdown timeout.
	_ = myPool.Close()

	// You should see 10 "Hello ewp!" lines printed.
}

```

Please read [godoc](https://godoc.org/github.com/lnquy/elastic-worker-pool) and take a look at [examples](https://github.com/lnquy/elastic-worker-pool/blob/master/examples) for more use cases of `ewp`.  

## Design

`ewp` can control its pool size by monitoring the number of jobs has been sent/pushed to pool's queue and the number of jobs has been processed/executed by pool's workers.  
The workload factor then can be calculated by `(jobsPushed - jobsProcessed) / bufferLength`.  

Pool controller will check the workload periodically (configured by `PoolControlInterval`), for each check, we called it one cycle.

When the workload factor reached a specific load level, then the corresponding growth factor will be applied to calculate the number of workers the pool should have in order to cope with that workload level.  
Based on the number calculated, controller expands or shrinks pool size to match the desired number of workers.

`ewp` currently shipped with two simple pool controllers:

- AgileController: Whenever workload factor raises over a specific load level, the growth factor of that load level will be applied immediately to calculate the number of desired workers. For example:  

```
LoadLevels = []LoadLevel{ {0.1, 0.3}, {0.5, 0.6}, {0.75, 1} }
MinWorker = 1, MaxWorker = 10, GrowthSize = (MaxWorker-MinWorker) = 9
    - When workload factor belows 10%, the worker pool size shrink to MinWorker (1 worker).
    - When workload factor reaches 10%, the worker pool size expand to MinWorker + 30% of GrowthSize (4 workers).
    - When workload factor reaches 50%, the worker pool size expand to MinWorker + 60% of GrowthSize (7 workers).
    - When workload factor reaches 75%, the worker pool size expand to MaxWorker (10 workers).

==> Pool size is changed to reach desired number of workers in just one cycle.
```


- RigidController: For each cycle, the number of workers will be changed (starts new ones or kills old ones) is limit by `maxChangesPerCycle`.


```
LoadLevels = []LoadLevel{ {0.1, 0.3}, {0.5, 0.5}, {0.75, 1} }
MinWorker = 1, MaxWorker = 10, maxChangesPerCycle = 1
GrowthSize = (MaxWorker-MinWorker) = 9
    - When workload factor belows 10%, the worker pool size shrink to MinWorker (1 worker).
    - When workload factor reaches 10%, the worker pool size expand to MinWorker + 30% of GrowthSize (4 workers).
      => Need 3 cycles to reach size of 4 workers, as each cycle only starts 1 new worker.
    - When workload factor reaches 50%, the worker pool size expand to MinWorker + 50% of GrowthSize (5 workers).
      => Need 1 cycle to reach size of 5 workers, as each cycle only starts 1 new worker.
    - When workload factor reaches 75%, the worker pool size expand to MaxWorker (10 workers).
      => Need 5 cycles to reaches size of 10 workers, as each cycle only starts 1 new worker.
```

**Note: You can implement the [PoolController](https://github.com/lnquy/elastic-worker-pool/blob/master/controller.go) interface to provide your own custom pool size control mechanism. For example: Control based on CPU or memory usage...**

## License

This project is under the MIT License. See the [LICENSE](https://github.com/lnquy/elastic-worker-pool/blob/master/LICENSE) file for the full license text.
