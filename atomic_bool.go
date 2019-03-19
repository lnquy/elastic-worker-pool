package elastic_worker_pool

import "sync/atomic"

type AtomicBool struct {
	flag int32 // 0=false, 1=true
}

func (ab *AtomicBool) Get() bool {
	return atomic.LoadInt32(&ab.flag) == 1
}

func (ab *AtomicBool) Set(b bool) {
	if b {
		atomic.StoreInt32(&ab.flag, 1)
		return
	}
	atomic.StoreInt32(&ab.flag, 0)
}
