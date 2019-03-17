package elastic_worker_pool

type Strategy interface {
	CaclWorkerNum(stats Stats) int
}
