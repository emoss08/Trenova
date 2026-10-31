package paperworkjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/paperworkreader"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/paperworkjobs/paperworkcontract"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	Reader *paperworkreader.Reader
}

type Activities struct {
	reader *paperworkreader.Reader
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{reader: p.Reader}
}

var Module = fx.Module(
	"paperwork-jobs",
	fx.Provide(NewActivities, paperworkreader.New),
)

func (a *Activities) ReadPaperworkActivity(
	ctx context.Context,
	input *paperworkcontract.ReadInput,
) error {
	return a.reader.Read(ctx, input.DocumentID, input.TenantInfo, input.UserID)
}
