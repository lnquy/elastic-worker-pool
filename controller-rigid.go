package ewp

import (
	"sort"
)

var _ PoolController = &rigidController{}

type rigidController struct {
	levels             LoadLevels
	maxChangesPerCycle int
}

func NewRigidController(loadLevels LoadLevels, maxChangesPerCycle int) (PoolController, error) {
	if maxChangesPerCycle < 0 {
		return nil, RigidCtlrInvalidConfigErr
	}

	if len(loadLevels) == 0 {
		loadLevels = defaultLoadLevels
	}
	sort.Slice(loadLevels, func(i, j int) bool {
		return loadLevels[i].LoadPct < loadLevels[j].LoadPct
	})

	return &rigidController{
		levels:             loadLevels,
		maxChangesPerCycle: maxChangesPerCycle,
	}, nil
}

// E.g.: levels := []LoadLevel{ {0.1, 0.25}, {0.25, 0.5}, {0.5, 0.75}, {0.75, 1} }
// ==> growthSpace = MaxWorker - MinWorker
//     If loadPercentage < 0.1   => desiredWorkerNum = MinWorker
//        loadPercentage >= 0.1  => desiredWorkerNum = MinWorker + 0.25*growthSpace
//        loadPercentage >= 0.25 => desiredWorkerNum = MinWorker + 0.5*growthSpace
//        loadPercentage >= 0.5  => desiredWorkerNum = MinWorker + 0.75*growthSpace
//        loadPercentage >= 0.75 => desiredWorkerNum = MinWorker + 1*growthSpace = MaxWorker
//
//     diff = abs(desiredWorkerNum - currentWorkerNum)
//     return min(diff, maxChangesPerCycle) + currentWorkerNum
func (s *rigidController) GetDesiredWorkerNum(stats Statistics) int {
	// Number of jobs currently in queue / total length of queue.
	loadPercentage := float64(stats.EnqueuedJobs-stats.FinishedJobs) / float64(stats.BufferLength)
	growthSpace := stats.MaxWorker - stats.MinWorker

	for i := len(s.levels) - 1; i >= 0; i-- {
		if loadPercentage >= s.levels[i].LoadPct {
			desiredWorkerNum := int(stats.MinWorker) + int(s.levels[i].GrowthPct*float64(growthSpace))
			return s.limitToMaxChangesPerCycle(desiredWorkerNum, int(stats.CurrWorker))
		}
	}
	return s.limitToMaxChangesPerCycle(int(stats.MinWorker), int(stats.CurrWorker))
}

func (s *rigidController) limitToMaxChangesPerCycle(desiredWorkerNum, currentWorkerNum int) int {
	diff := desiredWorkerNum - currentWorkerNum
	growthNum := diff

	if diff > s.maxChangesPerCycle {
		growthNum = s.maxChangesPerCycle
	}
	if diff < -s.maxChangesPerCycle {
		growthNum = -s.maxChangesPerCycle
	}
	return currentWorkerNum + growthNum
}
