package managedusageservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ReceiverParams struct {
	fx.In

	Usage     *Service
	Observers []managedusageports.StateObserver `group:"managed_state_observers"`
	Logger    *zap.Logger
}

type Receiver struct {
	usage     *Service
	observers []managedusageports.StateObserver
	l         *zap.Logger
}

func NewReceiver(p ReceiverParams) *Receiver {
	return &Receiver{
		usage:     p.Usage,
		observers: p.Observers,
		l:         p.Logger.Named("managed-usage-receiver"),
	}
}

func (r *Receiver) Readings(ctx context.Context, since int64) ([]Reading, error) {
	return r.usage.Readings(ctx, since)
}

func (r *Receiver) SetState(ctx context.Context, state *managedusage.State) error {
	change, err := r.usage.SetState(ctx, state)
	if err != nil {
		return err
	}
	for _, o := range r.observers {
		if err = o.StateChanged(ctx, change); err != nil {
			r.l.Warn("a usage state observer failed", zap.Error(err))
		}
	}
	return nil
}
