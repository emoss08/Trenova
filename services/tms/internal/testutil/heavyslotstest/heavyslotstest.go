package heavyslotstest

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type Gate struct {
	Busy  bool
	Taken atomic.Int64
	Held  atomic.Int64
}

var _ kernelports.HeavyJobSlots = (*Gate)(nil)

func Unbounded() *Gate {
	return &Gate{}
}

func (g *Gate) Hold(context.Context, kernelports.HeavyJob) (func(), error) {
	if g.Busy {
		return nil, errortypes.NewRateLimitError(
			"job",
			"The server is busy with other work of this kind. Try again in a moment.",
		).WithRetryAfter(time.Second)
	}
	return g.take(), nil
}

func (g *Gate) TryHold(context.Context, kernelports.HeavyJob) (func(), bool) {
	if g.Busy {
		return nil, false
	}
	return g.take(), true
}

func (g *Gate) take() func() {
	g.Taken.Add(1)
	g.Held.Add(1)
	var released atomic.Bool
	return func() {
		if released.CompareAndSwap(false, true) {
			g.Held.Add(-1)
		}
	}
}
