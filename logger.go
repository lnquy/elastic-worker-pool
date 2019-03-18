package elastic_worker_pool

type (
	Logger interface {
		Debugf(string, ...interface{})
		Infof(string, ...interface{})
	}

	discardLogger struct{}
)

func (l *discardLogger) Debugf(string, ...interface{}) {
	// Do nothing
}

func (l *discardLogger) Infof(string, ...interface{}) {
	// Do nothing
}
