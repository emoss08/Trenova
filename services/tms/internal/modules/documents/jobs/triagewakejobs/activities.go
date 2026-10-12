package triagewakejobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/jobs/triagewakejobs/triagewakecontract"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	Worker triagewakecontract.Worker
}

type Activities struct {
	worker triagewakecontract.Worker
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{worker: p.Worker}
}

func (a *Activities) ScheduleTriageWakeActivity(
	ctx context.Context,
	payload *Payload,
) (*Schedule, error) {
	return a.worker.WakeSchedule(ctx, payload.Tenant(), payload.MessageID)
}

func (a *Activities) WakeTriageMessageActivity(
	ctx context.Context,
	in *WakeInput,
) (*WakeResult, error) {
	return a.worker.WakeIfDue(ctx, in.Payload.Tenant(), in.Payload.MessageID, in.Now)
}

func (a *Activities) ReconcileTriageWakesActivity(ctx context.Context) (int, error) {
	return a.worker.ReconcileOverdueWakes(ctx, timeutils.NowUnix())
}

func (a *Activities) RecomputeTriageDueActivity(
	ctx context.Context,
	payload *RecomputeDuePayload,
) (int64, error) {
	return a.worker.RecomputeDueFor(ctx, payload.Tenant(), payload.Classifications)
}
