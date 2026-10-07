package aituneupjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	TuneUps services.AITuneUpService
	Tenants repositories.TenantSyncRepository
	Logger  *zap.Logger
}

type Activities struct {
	tuneUps services.AITuneUpService
	tenants repositories.TenantSyncRepository
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
	organizations, err := a.tenants.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	return temporaljobs.OrganizationPage(organizations, input.After, input.Limit), nil
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
