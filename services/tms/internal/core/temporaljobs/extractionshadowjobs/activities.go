package extractionshadowjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionshadowports"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	Runner extractionshadowports.ExtractionShadowRunner
}

type Activities struct {
	runner extractionshadowports.ExtractionShadowRunner
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{runner: p.Runner}
}

func (a *Activities) RunExtractionShadowActivity(ctx context.Context, input *ShadowPayload) error {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	return a.runner.RunShadow(ctx, &extractionshadowports.RunExtractionShadowRequest{
		TenantInfo:   input.tenant(),
		ResultID:     input.ResultID,
		FinalAttempt: modelcall.FinalAttempt(ctx, runAttempts),
	})
}

func (a *Activities) FailExtractionShadowActivity(ctx context.Context, input *FailInput) error {
	if err := a.runner.FailShadow(ctx, extractionshadowports.GetExtractionShadowResultRequest{
		TenantInfo: input.tenant(),
		ID:         input.ResultID,
	}, input.Message); err != nil {
		return fmt.Errorf("fail extraction shadow: %w", err)
	}

	return nil
}
