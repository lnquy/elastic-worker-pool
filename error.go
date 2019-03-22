package ewp

import "github.com/pkg/errors"

var (
	// ElasticWorkerPool
	WorkerPoolStoppedErr       = errors.New("worker pool stopped")
	WorkerTimeoutExceededErr   = errors.New("all workers are busy, timeout exceeded")
	ShutdownTimeoutExceededErr = errors.New("shutdown timeout exceeded")

	// LoadLevel
	InvalidLoadLevelErr = errors.New("workload percentage and growth percentage must >= 0 and <= 1")

	// RigidController
	InvalidMaxChangesPerCycleErr = errors.New("invalid config: maxChangesPerCycle must >= 0")
)
