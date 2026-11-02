package temporaljobs

import (
	temporallog "go.temporal.io/sdk/log"
	"go.uber.org/zap"
)

// sdkLogger writes the Temporal SDK's lines through the process's own logger,
// so they carry the same fields and land in the same place as everything else.
type sdkLogger struct {
	l *zap.SugaredLogger
}

var _ temporallog.Logger = sdkLogger{}

// NewSDKLogger is the logger the Temporal SDK writes through. The desk bench
// replaces it to keep the SDK's lines out of its report.
func NewSDKLogger(l *zap.Logger) temporallog.Logger {
	return sdkLogger{l: l.Named("temporal-sdk").Sugar()}
}

func (s sdkLogger) Debug(msg string, keyvals ...any) { s.l.Debugw(msg, keyvals...) }

func (s sdkLogger) Info(msg string, keyvals ...any) { s.l.Infow(msg, keyvals...) }

func (s sdkLogger) Warn(msg string, keyvals ...any) { s.l.Warnw(msg, keyvals...) }

func (s sdkLogger) Error(msg string, keyvals ...any) { s.l.Errorw(msg, keyvals...) }
