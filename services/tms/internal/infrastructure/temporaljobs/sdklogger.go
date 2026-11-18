package temporaljobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/pollwatch"
	temporallog "go.temporal.io/sdk/log"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// sdkLogger writes the Temporal SDK's lines through the process's own logger,
// so they carry the same fields and land in the same place as everything else.
type sdkLogger struct {
	l     *zap.SugaredLogger
	polls *pollwatch.Watch
}

var _ temporallog.Logger = sdkLogger{}

type SDKLoggerParams struct {
	fx.In

	Logger *zap.Logger
	Polls  *pollwatch.Watch
}

// NewSDKLogger is the logger the Temporal SDK writes through. The desk bench
// replaces it to keep the SDK's lines out of its report.
func NewSDKLogger(p SDKLoggerParams) temporallog.Logger {
	return sdkLogger{l: p.Logger.Named("temporal-sdk").Sugar(), polls: p.Polls}
}

func (s sdkLogger) Debug(msg string, keyvals ...any) { s.l.Debugw(msg, keyvals...) }

func (s sdkLogger) Info(msg string, keyvals ...any) { s.l.Infow(msg, keyvals...) }

func (s sdkLogger) Warn(msg string, keyvals ...any) {
	if msg == pollwatch.FailedPoll {
		s.polls.Failed(errorIn(keyvals))
		return
	}
	s.l.Warnw(msg, keyvals...)
}

// sdkActivityError is the line the SDK writes for every failed activity
// attempt, at Error even when a retry follows. The worker's logging interceptor
// reports each failed attempt with the level its outcome deserves, so this
// copy is kept at Debug.
const sdkActivityError = "Activity error."

func (s sdkLogger) Error(msg string, keyvals ...any) {
	if msg == sdkActivityError {
		s.l.Debugw(msg, keyvals...)
		return
	}
	s.l.Errorw(msg, keyvals...)
}

func errorIn(keyvals []any) error {
	for _, v := range keyvals {
		if err, ok := v.(error); ok {
			return err
		}
	}
	return nil
}
