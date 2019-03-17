package elastic_worker_pool

type Logger interface {
	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})
}
