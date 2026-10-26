package wasmmodule

import (
	"time"

	"go.uber.org/zap"
)

const (
	OutcomeOK           = "ok"
	OutcomeRequestError = "request_error"
	OutcomeTrap         = "trap"
	OutcomeContextDone  = "context_done"
	OutcomeUnavailable  = "unavailable"

	DiscardTrap        = "trap"
	DiscardContextDone = "context_done"
	DiscardMemory      = "memory"

	unknownOpLabel = "unknown"
	maxOpLabelLen  = 64
)

type Observer interface {
	CallFinished(op, outcome string, duration time.Duration)
	InstanceAcquired()
	InstanceReleased()
	InstanceDiscarded(reason string)
}

type nopObserver struct{}

func (nopObserver) CallFinished(string, string, time.Duration) {}
func (nopObserver) InstanceAcquired()                          {}
func (nopObserver) InstanceReleased()                          {}
func (nopObserver) InstanceDiscarded(string)                   {}

type instrumentation struct {
	logger   *zap.Logger
	observer Observer
}

func opLabel(op string) string {
	if op == "" || len(op) > maxOpLabelLen {
		return unknownOpLabel
	}
	dot := false
	for i := range len(op) {
		c := op[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		case c == '.' && i > 0 && i < len(op)-1 && !dot:
			dot = true
		default:
			return unknownOpLabel
		}
	}
	if !dot {
		return unknownOpLabel
	}
	return op
}
