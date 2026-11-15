package managedfeaturesservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const observerTimeout = 10 * time.Second

type ReceiverParams struct {
	fx.In

	Features  *Service
	Observers []managedfeaturesports.PlanObserver `group:"managed_plan_observers"`
	Logger    *zap.Logger
}

type Receiver struct {
	features  *Service
	observers []managedfeaturesports.PlanObserver
	l         *zap.Logger
}

func NewReceiver(p ReceiverParams) *Receiver {
	return &Receiver{
		features:  p.Features,
		observers: p.Observers,
		l:         p.Logger.Named("managed-features-receiver"),
	}
}

func (r *Receiver) State(ctx context.Context) (*managedfeatures.State, error) {
	return r.features.State(ctx)
}

func (r *Receiver) SetFeatures(ctx context.Context, features []platformcatalog.FeatureKey) error {
	replacement, err := r.features.SetFeatures(ctx, features)
	if err != nil {
		return err
	}
	if !replacement.Changed() || len(r.observers) == 0 {
		return nil
	}

	observeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), observerTimeout)
	defer cancel()
	for _, observer := range r.observers {
		if err = observer.PlanChanged(observeCtx, replacement); err != nil {
			r.l.Warn("a plan change observer failed", zap.Error(err))
		}
	}
	return nil
}
