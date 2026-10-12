package triageworkjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/app/triageworkservice"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/triageworkjobs/triageworkcontract"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	Work *triageworkservice.Service
}

type Activities struct {
	work *triageworkservice.Service
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{work: p.Work}
}

func (a *Activities) DraftTriageMessageActivity(
	ctx context.Context,
	payload *triageworkcontract.DraftOnePayload,
) (bool, error) {
	return a.work.DraftOne(ctx, payload)
}

func (a *Activities) RecordTriageSplitActivity(
	ctx context.Context,
	payload *triageworkcontract.RecordSplitPayload,
) error {
	return a.work.RecordSplit(ctx, payload)
}
