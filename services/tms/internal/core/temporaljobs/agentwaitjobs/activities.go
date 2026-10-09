package agentwaitjobs

import (
	"context"
	"errors"

	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	Worker Worker
}

type Activities struct {
	worker Worker
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{worker: p.Worker}
}

func (a *Activities) ScheduleWaitActivity(
	ctx context.Context,
	payload *Payload,
) (*Schedule, error) {
	return a.worker.Schedule(ctx, payload.tenant(), payload.WaitID)
}

func (a *Activities) CheckWaitActivity(ctx context.Context, in *CheckInput) (*Check, error) {
	return a.worker.Check(ctx, in.Payload.tenant(), in.Payload.WaitID, in.Now)
}

func (a *Activities) ReconcileOverdueWaitsActivity(ctx context.Context) (int, error) {
	return a.worker.ReconcileOverdue(ctx)
}

func (a *Activities) FinishWaitActivity(ctx context.Context, in *FinishInput) error {
	err := a.worker.Finish(ctx, in)
	if errors.Is(err, ErrConversationBusy) {
		return temporal.NewApplicationError(err.Error(), errTypeBusy)
	}

	return err
}
