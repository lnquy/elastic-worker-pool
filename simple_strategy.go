package elastic_worker_pool

type SimpleStrategy struct {
}

func (s *SimpleStrategy) CaclWorkerNum(stats Stats) int {
	inBuff := stats.JobsIn - stats.JobsDone
	buffSize := 1
	if stats.BufferSize > 0 {
		buffSize = stats.BufferSize
	}
	inBuffPerc := float64(inBuff) / float64(buffSize)

	switch {
	case inBuffPerc <= 0.1:
		return stats.MinWorker - stats.CurrWorker // Shrink to min worker
	case inBuffPerc <= 0.25:
		return int(0.25*float64(stats.MaxWorker)) - stats.CurrWorker
	case inBuffPerc <= 0.5:
		return int(0.75*float64(stats.MaxWorker)) - stats.CurrWorker
	case inBuffPerc <= 0.7:
		return stats.MaxWorker - stats.CurrWorker
	}
}
