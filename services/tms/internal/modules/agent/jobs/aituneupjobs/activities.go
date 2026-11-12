package aituneupjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aituneupports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	TuneUps aituneupports.AITuneUpService
	Tenants tenantports.TenantSyncRepository
	Logger  *zap.Logger
}

type Activities struct {
	tuneUps aituneupports.AITuneUpService
	tenants tenantports.TenantSyncRepository
	l       *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		tuneUps: p.TuneUps,
		tenants: p.Tenants,
		l:       p.Logger.Named("job.ai-tune-ups"),
	}
}

func (a *Activities) ListAITuneUpOrganizationsActivity(
	ctx context.Context,
	input *ListAITuneUpOrganizationsInput,
) (*temporaljobs.TenantPage, error) {
	return temporaljobs.OrganizationPage(ctx, a.tenants, input.After, input.Limit)
}

func (a *Activities) ComputeOrganizationAITuneUpsActivity(
	ctx context.Context,
	input *OrganizationAITuneUpsInput,
) (*OrganizationAITuneUpsResult, error) {
	tenant := input.TenantInfo()
	suggested, err := a.tuneUps.Compute(ctx, tenant)
	if err != nil {
		return nil, fmt.Errorf("compute ai tune-ups: %w", err)
	}

	a.l.Debug("ai tune-ups computed",
		zap.String("organization", tenant.OrgID.String()),
		zap.Int("suggested", suggested),
	)
	return &OrganizationAITuneUpsResult{Suggested: suggested}, nil
}
