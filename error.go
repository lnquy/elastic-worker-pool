package ewp

import "github.com/pkg/errors"

var (
	// ElasticWorkerPool
	WorkerPoolStoppedErr       = errors.New("worker pool stopped")
	WorkerTimeoutExceededErr   = errors.New("all workers are busy, timeout exceeded")
	ShutdownTimeoutExceededErr = errors.New("shutdown timeout exceeded")

	// RigidController
	RigidCtlrInvalidConfigErr = errors.New("invalid config: maxChangesPerCycle must >= 0")
)
