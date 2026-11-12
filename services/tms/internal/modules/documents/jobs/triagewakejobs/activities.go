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

func (a *Activities) WakeDueTriageActivity(ctx context.Context) (int, error) {
	return a.worker.WakeDue(ctx, timeutils.NowUnix())
}

func (a *Activities) RecomputeTriageDueActivity(
	ctx context.Context,
	payload *RecomputeDuePayload,
) (int64, error) {
	return a.worker.RecomputeDueFor(ctx, payload.Tenant(), payload.Classifications)
}
