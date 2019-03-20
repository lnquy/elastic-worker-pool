// package ewp is the implementation of elastic/dynamic worker pool.
// After started up, worker pool automatically monitors the load factor of
// the pool and based on that to determine if it need to expand or shrink the pool size.
//
// Create default ewp: myPool := ewp.NewDefault()
// or create with full configs:


package ewp
