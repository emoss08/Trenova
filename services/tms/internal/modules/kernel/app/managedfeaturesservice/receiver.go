package managedfeaturesservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
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

var _ managedfeaturesports.PlanEndReminder = (*Receiver)(nil)

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

func (r *Receiver) SetFeatures(ctx context.Context, received *managedfeatures.Received) error {
	replacement, err := r.features.SetFeatures(ctx, received)
	if err != nil {
		return err
	}
	r.tell(ctx, replacement)
	return nil
}

func (r *Receiver) RemindPlanEnd(
	ctx context.Context,
) (*managedfeaturesports.PlanEndReminderResult, error) {
	replacement, err := r.features.RemindPlanEnd(ctx)
	if err != nil {
		return nil, err
	}
	if replacement.PlanEnd == nil {
		return &managedfeaturesports.PlanEndReminderResult{}, nil
	}
	r.tell(ctx, replacement)
	return &managedfeaturesports.PlanEndReminderResult{
		Reminded: true,
		EndsAt:   replacement.PlanEnd.EndsAt,
	}, nil
}

func (r *Receiver) tell(ctx context.Context, replacement *managedfeatures.Replacement) {
	if !replacement.Updated() || len(r.observers) == 0 {
		return
	}

	observeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), observerTimeout)
	defer cancel()
	for _, observer := range r.observers {
		if err := observer.PlanChanged(observeCtx, replacement); err != nil {
			r.l.Warn("a plan change observer failed", zap.Error(err))
		}
	}
}
