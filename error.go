package ewp

import "github.com/pkg/errors"

var (
	// ElasticWorkerPool
	ErrWorkerPoolStopped       = errors.New("worker pool stopped")
	ErrWorkerTimeoutExceeded   = errors.New("all workers are busy, timeout exceeded")
	ErrShutdownTimeoutExceeded = errors.New("shutdown timeout exceeded")

	// LoadLevel
	ErrInvalidLoadLevel = errors.New("workload percentage and growth percentage must >= 0 and <= 1")

	// RigidController
	ErrInvalidMaxChangesPerCycle = errors.New("invalid config: maxChangesPerCycle must >= 0")
)
