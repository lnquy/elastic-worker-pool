package ewp

import (
	"sort"
)

var _ PoolController = &rigidController{}

type rigidController struct {
	levels             LoadLevels
	maxChangesPerCycle int
}

// NewRigidController returns a RigidController.
//
// RigidController reacts to the work load with slower sensitivity.
// Whenever workload raises over a load level, the growth factor of that
// load level will be applied to change the worker pool size.
// But for each control cycle, the number of worker changes (start new workers or
// kill old ones) is limit by maxChangesPerCycle.
//
// For example:
//   LoadLevels = []LoadLevel{ {0.1, 0.3}, {0.5, 0.5}, {0.75, 1} }
//   MinWorker = 1, MaxWorker = 10, maxChangesPerCycle = 1
//   GrowthSize = (MaxWorker-MinWorker) = 9
//   => - When load percentage (the number of jobs currently in buffer queue / buffer queue length)
//        belows 10%, the worker pool size shrink to MinWorker (1 worker).
//      - When load percentage reaches 10%, the worker pool size expand to
//        MinWorker + 30% of GrowthSize (4 workers).
//        Need 3 cycles to reach size of 4 workers, as each cycle only starts 1 new worker.
//      - When load percentage reaches 50%, the worker pool size expand to
//        MinWorker + 50% of GrowthSize (5 workers).
//        Need 1 cycle to reach size of 5 workers, as each cycle only starts 1 new worker.
//      - When load percentage reaches 75%, the worker pool size expand to
//        MaxWorker (10 workers).
//        Need 5 cycles to reaches size of 10 workers, as each cycle only starts 1 new worker.
func NewRigidController(loadLevels LoadLevels, maxChangesPerCycle int) (PoolController, error) {
	if maxChangesPerCycle < 0 {
		return nil, ErrInvalidMaxChangesPerCycle
	}
	if len(loadLevels) == 0 {
		loadLevels = defaultLoadLevels
	}
	if !isValidLoadLevels(loadLevels) {
		return nil, ErrInvalidLoadLevel
	}

	sort.Slice(loadLevels, func(i, j int) bool {
		return loadLevels[i].LoadPct < loadLevels[j].LoadPct
	})

	return &rigidController{
		levels:             loadLevels,
		maxChangesPerCycle: maxChangesPerCycle,
	}, nil
}

// GetDesiredWorkerNum calculates the desired number of workers the pool
// should have in order to cope with current workload.
//
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
