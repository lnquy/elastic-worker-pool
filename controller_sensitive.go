package elastic_worker_pool

import (
	"github.com/sirupsen/logrus"
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
		return loadLevels[j].LoadPct > loadLevels[i].LoadPct
	})

	return &sensitiveController{
		levels: loadLevels,
	}
}

// E.g.: levels := []LoadLevel{ {0.1, 0.25}, {0.25, 0.5}, {0.5, 0.75}, {0.75, 1} }
// ==> If loadPercentage < 0.1 => MinWorker
//        loadPercentage >= 0.1 => 0.25 * MaxWorker
//        loadPercentage >= 0.25 => 0.5 * MaxWorker
//        loadPercentage >= 0.5 => 0.75 * MaxWorker
//        loadPercentage >= 0.75 => 1 * MaxWorker
func (s *sensitiveController) GetDesiredWorkerNum(stats Statistics) int {
	// Number of jobs currently in queue / total length of queue.
	loadPercentage := float64(stats.EnqueuedJobs-stats.FinishedJobs) / float64(stats.BufferLength)
	logrus.Infof(">>> %v", stats)
	logrus.Infof(">>> loadPCT: %f", loadPercentage)

	for i := len(s.levels) - 1; i >= 0; i-- {
		if loadPercentage >= s.levels[i].LoadPct {
			t := int(s.levels[i].GrowthPct * float64(stats.MaxWorker))
			return t
		}
	}
	return stats.MinWorker
}
