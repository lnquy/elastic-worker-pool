package elastic_worker_pool

import "sort"

var _ PoolController = &agileController{}

type agileController struct {
	levels LoadLevels
}

func NewAgileController(loadLevels LoadLevels) PoolController {
	if len(loadLevels) == 0 {
		loadLevels = defaultLoadLevels
	}
	sort.Slice(loadLevels, func(i, j int) bool {
		return loadLevels[i].LoadPct < loadLevels[j].LoadPct
	})

	return &agileController{
		levels: loadLevels,
	}
}

// E.g.: levels := []LoadLevel{ {0.1, 0.25}, {0.25, 0.5}, {0.5, 0.75}, {0.75, 1} }
// ==> growthSpace = MaxWorker - MinWorker
//     If loadPercentage < 0.1 => MinWorker
//        loadPercentage >= 0.1 => MinWorker + 0.25*growthSpace
//        loadPercentage >= 0.25 => MinWorker + 0.5*growthSpace
//        loadPercentage >= 0.5 => MinWorker + 0.75*growthSpace
//        loadPercentage >= 0.75 => MinWorker + 1*growthSpace = MaxWorker
func (s *agileController) GetDesiredWorkerNum(stats Statistics) int {
	// Number of jobs currently in queue / total length of queue.
	loadPercentage := float64(stats.EnqueuedJobs-stats.FinishedJobs) / float64(stats.BufferLength)
	growthSpace := stats.MaxWorker - stats.MinWorker

	for i := len(s.levels) - 1; i >= 0; i-- {
		if loadPercentage >= s.levels[i].LoadPct {
			return stats.MinWorker + int(s.levels[i].GrowthPct*float64(growthSpace))
		}
	}
	return stats.MinWorker
}
