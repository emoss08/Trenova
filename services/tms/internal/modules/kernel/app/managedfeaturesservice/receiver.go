package managedfeaturesservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
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

func (r *Receiver) State(
	ctx context.Context,
	scope managedscope.Scope,
) (*managedfeatures.State, error) {
	return r.features.State(ctx, scope)
}

func (r *Receiver) SetFeatures(
	ctx context.Context,
	scope managedscope.Scope,
	received *managedfeatures.Received,
) error {
	replacement, err := r.features.SetFeatures(ctx, scope, received)
	if err != nil {
		return err
	}
	r.tell(ctx, replacement)
	return nil
}

func (r *Receiver) SetLifecycle(
	ctx context.Context,
	scope managedscope.Scope,
	lifecycle managedfeatures.Lifecycle,
) error {
	replacement, err := r.features.SetLifecycle(ctx, scope, lifecycle)
	if err != nil {
		return err
	}
	r.tell(ctx, replacement)
	return nil
}

func (r *Receiver) RemindPlanEnd(
	ctx context.Context,
) (*managedfeaturesports.PlanEndReminderResult, error) {
	reminded, err := r.features.RemindPlanEnds(ctx)
	for _, replacement := range reminded {
		r.tell(ctx, replacement)
	}
	if err != nil {
		return nil, err
	}
	return &managedfeaturesports.PlanEndReminderResult{Reminded: len(reminded)}, nil
}

func (r *Receiver) tell(ctx context.Context, replacement *managedfeatures.Replacement) {
	if !replacement.Updated() || len(r.observers) == 0 {
		return
	}

	observeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), observerTimeout)
	defer cancel()
	for _, observer := range r.observers {
		if err := observer.PlanChanged(observeCtx, replacement); err != nil {
			r.l.Warn("a plan change observer failed",
				zap.String("scope", replacement.Scope.Key()),
				zap.Error(err),
			)
		}
	}
}
