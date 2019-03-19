package ewp

type (
	PoolController interface {
		GetDesiredWorkerNum(stats Statistics) int
	}

	LoadLevel struct {
		LoadPct   float64
		GrowthPct float64
	}

	LoadLevels []LoadLevel
)

var defaultLoadLevels = []LoadLevel{{0.1, 0.25}, {0.25, 0.5}, {0.5, 0.75}, {0.75, 1}}
