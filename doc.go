// Package ewp is the implementation of elastic/dynamic worker pool (EWP) with graceful shutdown.
//
// After started up, EWP controller automatically monitors the workload level of
// the pool and based on that to determine the pool size need to be
// expanded (spawn more workers) or shrunk (kill free workers).
//
// Create EWP with default configs:
// 	   myPool, _ := ewp.NewDefault()
// 	   myPool.Start()
//
// Or create EWP with full customized configs:
// 		ewpConfig := ewp.Config{
// 			MinWorker: 5,
// 			MaxWorker: 20,
// 			PoolControlInterval: 10 * time.Second,
// 			BufferLength: 1000,
// 		}
// 		controller = ewp.NewAgileController(nil)
// 		myPool, _ := ewp.New(ewpConfig, controller, logrus.New())
// 		myPool.Start()
//
// Or create a fixed traditional worker pool:
// 		ewpConfig := ewp.Config{
// 			MinWorker: 5,
// 			MaxWorker: 5,
// 		}
// 		myPool, _ := ewp.New(ewpConfig, nil, nil)
// 		myPool.Start()
//
//
// Send jobs to EWP to be processed later:
// 		for i := 0; i < 10; i++ {
// 			i := i
// 			jobFunc := func() {
// 				// Your job logic go here
// 				log.Printf("job #%d", i)
// 			}
// 			_ = myPool.Enqueue(jobFunc)
// 		}
//
// Send jobs with timeout:
// 		for i := 0; i < 10; i++ {
// 			i := i
// 			jobFunc := func() {
// 				// Your job logic go here
// 				log.Printf("job #%d", i)
// 			}
// 			_ = myPool.Enqueue(jobFunc, 2*time.Second)
// 		}
//
// Gracefully shutdown the EWP:
// 		_ = myPool.Close()
package ewp
