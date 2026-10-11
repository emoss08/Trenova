package agentwaitjobs

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentwaitjobs/agentwaitcontract"

	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	Worker agentwaitcontract.Worker
}

type Activities struct {
	worker agentwaitcontract.Worker
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{worker: p.Worker}
}

func (a *Activities) ScheduleWaitActivity(
	ctx context.Context,
	payload *agentwaitcontract.Payload,
) (*agentwaitcontract.Schedule, error) {
	return a.worker.Schedule(ctx, payload.Tenant(), payload.WaitID)
}

func (a *Activities) CheckWaitActivity(
	ctx context.Context,
	in *CheckInput,
) (*agentwaitcontract.Check, error) {
	return a.worker.Check(ctx, in.Payload.Tenant(), in.Payload.WaitID, in.Now)
}

func (a *Activities) ReconcileOverdueWaitsActivity(ctx context.Context) (int, error) {
	return a.worker.ReconcileOverdue(ctx)
}

func (a *Activities) FinishWaitActivity(
	ctx context.Context,
	in *agentwaitcontract.FinishInput,
) error {
	err := a.worker.Finish(ctx, in)
	if errors.Is(err, agentwaitcontract.ErrConversationBusy) {
		return temporal.NewApplicationError(err.Error(), errTypeBusy)
	}

	return err
}
