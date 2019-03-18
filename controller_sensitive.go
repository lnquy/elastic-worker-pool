package elastic_worker_pool

import (
	"sort"
)

type sensitiveController struct {
	levels LoadLevels
}

func NewSensitiveController(loadLevels LoadLevels) PoolController {
	if len(loadLevels) == 0 {
		loadLevels = defaultLoadLevels
	}
	sort.Slice(loadLevels, func(i, j int) bool {
		return loadLevels[i].LoadPct < loadLevels[j].LoadPct
	})

	return &sensitiveController{
		levels: loadLevels,
	}
}

// E.g.: levels := []LoadLevel{ {0.1, 0.25}, {0.25, 0.5}, {0.5, 0.75}, {0.75, 1} }
// ==> If loadPercentage < 0.1 => MinWorker
//        loadPercentage >= 0.1 => min(MinWorker, 0.25 * MaxWorker)
//        loadPercentage >= 0.25 => min(MinWorker, 0.5 * MaxWorker)
//        loadPercentage >= 0.5 => min(MinWorker, 0.75 * MaxWorker)
//        loadPercentage >= 0.75 => 1 * MaxWorker
func (s *sensitiveController) GetDesiredWorkerNum(stats Statistics) int {
	// Number of jobs currently in queue / total length of queue.
	loadPercentage := float64(stats.EnqueuedJobs-stats.FinishedJobs) / float64(stats.BufferLength)

	for i := len(s.levels) - 1; i >= 0; i-- {
		if loadPercentage >= s.levels[i].LoadPct {
			wnBasedOnMax := int(s.levels[i].GrowthPct * float64(stats.MaxWorker))
			if wnBasedOnMax < stats.MinWorker {
				return stats.MinWorker
			}
			return wnBasedOnMax
		}
	}
	return stats.MinWorker
}
