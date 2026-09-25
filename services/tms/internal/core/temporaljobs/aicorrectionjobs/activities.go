package aicorrectionjobs

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

	Corrections services.AICorrectionService
	Tenants     repositories.TenantSyncRepository
	Logger      *zap.Logger
}

type Activities struct {
	corrections services.AICorrectionService
	tenants     repositories.TenantSyncRepository
	l           *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		corrections: p.Corrections,
		tenants:     p.Tenants,
		l:           p.Logger.Named("job.aicorrection-retention"),
	}
}

func (a *Activities) ListAICorrectionOrganizationsActivity(
	ctx context.Context,
	input *ListAICorrectionOrganizationsInput,
) (*temporaljobs.TenantPage, error) {
	organizations, err := a.tenants.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	return temporaljobs.OrganizationPage(organizations, input.After, input.Limit), nil
}

func (a *Activities) PurgeOrganizationAICorrectionsActivity(
	ctx context.Context,
	input *OrganizationAICorrectionInput,
) (*OrganizationAICorrectionResult, error) {
	tenant := input.TenantInfo()

	purged, err := a.corrections.PurgeExpired(ctx, services.PurgeExpiredAICorrectionsRequest{
		TenantInfo: tenant,
		Now:        input.Now,
	})
	if err != nil {
		return nil, fmt.Errorf("purge expired ai corrections: %w", err)
	}

	a.l.Debug("ai corrections purged",
		zap.String("organization", tenant.OrgID.String()),
		zap.Int64("purged", purged),
	)

	return &OrganizationAICorrectionResult{Purged: purged}, nil
}
