package ewp

type (
	// Logger is the logging interface for EWP.
	// EWP logs on two levels: Debug and Info.
	Logger interface {
		Debugf(string, ...interface{})
		Infof(string, ...interface{})
	}

	// Logger to omit all EWP logs
	discardLogger struct{}
)

func (l *discardLogger) Debugf(string, ...interface{}) {
	// Do nothing
}

func (l *discardLogger) Infof(string, ...interface{}) {
	// Do nothing
}
