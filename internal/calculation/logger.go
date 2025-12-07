package calculation

// Logger is a minimal logging interface for the calculation engine.
// Implementations should be fast; the default is a no-op.
type Logger interface {
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}

// NopLogger implements Logger with no output.
type NopLogger struct{}

func (NopLogger) Debugf(format string, args ...any) {}
func (NopLogger) Infof(format string, args ...any)  {}
func (NopLogger) Warnf(format string, args ...any)  {}
func (NopLogger) Errorf(format string, args ...any) {}
