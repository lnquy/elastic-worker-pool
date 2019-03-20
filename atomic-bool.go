package ewp

import "sync/atomic"

// AtomicBool provides the atomic operations on boolean.
type AtomicBool struct {
	flag int32 // 0=false, 1=true
}

// Get returns the boolean value.
// Safe to call concurrently.
func (ab *AtomicBool) Get() bool {
	return atomic.LoadInt32(&ab.flag) == 1
}

// Set sets the boolean value.
// Safe to call concurrently.
func (ab *AtomicBool) Set(b bool) {
	if b {
		atomic.StoreInt32(&ab.flag, 1)
		return
	}
	atomic.StoreInt32(&ab.flag, 0)
}
