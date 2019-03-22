package ewp

var defaultLoadLevels = []LoadLevel{{0.1, 0.25}, {0.25, 0.5}, {0.5, 0.75}, {0.75, 1}}

type (
	// PoolController controls the pool size based on current workload.
	PoolController interface {
		// GetDesiredWorkerNum calculates the desired number of workers the pool
		// should have in order to cope with current workload.
		// This function should never return the number of workers smaller
		// than MinWorker or greater than MaxWorker configurations.
		GetDesiredWorkerNum(stats Statistics) int
	}

	// LoadLevel represents a workload level and the growth factor to cope with that
	// workload level.
	LoadLevel struct {
		LoadPct   float64 // Load percentage [0, 1]
		GrowthPct float64 // Growth percentage [0, 1]
	}

	// LoadLevels is the list of LoadLevel.
	LoadLevels []LoadLevel
)

// isValidLoadLevels checks if load levels has valid values or not.
func isValidLoadLevels(loadLevels LoadLevels) bool {
	for _, v := range loadLevels {
		if v.LoadPct < 0 || v.LoadPct > 1 || v.GrowthPct < 0 || v.GrowthPct > 1 {
			return false
		}
	}
	return true
}
