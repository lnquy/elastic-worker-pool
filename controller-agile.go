package ewp

import "sort"

var _ PoolController = &agileController{}

type agileController struct {
	levels LoadLevels
}

// NewAgileController returns an AgileController.
//
// AgileController reacts to the work load with the highest sensitivity.
// Whenever workload raises over a load level, the growth factor of that
// load level will be applied immediately to change the worker pool size.
//
// For example:
//   LoadLevels = []LoadLevel{ {0.1, 0.3}, {0.5, 0.5}, {0.75, 1} }
//   MinWorker = 1, MaxWorker = 10
//   => - When load percentage (the number of jobs currently in buffer queue / buffer queue length)
//        belows 10%, the worker pool size shrink to MinWorker (1 worker).
//      - When load percentage reaches 10%, the worker pool size expand to
//        30% of MaxWorker (3 workers).
//      - When load percentage reaches 50%, the worker pool size expand to
//        50% of MaxWorker (5 workers).
//      - When load percentage reaches 75%, the worker pool size expand to
//        MaxWorker (10 workers).
func NewAgileController(loadLevels LoadLevels) (PoolController, error) {
	if len(loadLevels) == 0 {
		loadLevels = defaultLoadLevels
	}
	if !isValidLoadLevels(loadLevels) {
		return nil, InvalidLoadLevelErr
	}
	sort.Slice(loadLevels, func(i, j int) bool {
		return loadLevels[i].LoadPct < loadLevels[j].LoadPct
	})

	return &agileController{
		levels: loadLevels,
	}, nil
}

// GetDesiredWorkerNum calculates the desired number of workers the pool
// should have in order to cope with current workload.
//
// E.g.: levels := []LoadLevel{ {0.1, 0.25}, {0.25, 0.5}, {0.5, 0.75}, {0.75, 1} }
// ==> growthSpace = MaxWorker - MinWorker
//     If loadPercentage < 0.1   => MinWorker
//        loadPercentage >= 0.1  => MinWorker + 0.25*growthSpace
//        loadPercentage >= 0.25 => MinWorker + 0.5*growthSpace
//        loadPercentage >= 0.5  => MinWorker + 0.75*growthSpace
//        loadPercentage >= 0.75 => MinWorker + 1*growthSpace = MaxWorker
func (s *agileController) GetDesiredWorkerNum(stats Statistics) int {
	// Number of jobs currently in queue / total length of queue.
	loadPercentage := float64(stats.EnqueuedJobs-stats.FinishedJobs) / float64(stats.BufferLength)
	growthSpace := stats.MaxWorker - stats.MinWorker

	for i := len(s.levels) - 1; i >= 0; i-- {
		if loadPercentage >= s.levels[i].LoadPct {
			return int(stats.MinWorker) + int(s.levels[i].GrowthPct*float64(growthSpace))
		}
	}
	return int(stats.MinWorker)
}
