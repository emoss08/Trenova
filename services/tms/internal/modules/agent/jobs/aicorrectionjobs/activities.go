package aicorrectionjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aicorrectionports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionaccuracyports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Corrections aicorrectionports.AICorrectionService
	Drift       extractionaccuracyports.ExtractionDriftChecker
	Tenants     tenantports.TenantSyncRepository
	Logger      *zap.Logger
}

type Activities struct {
	corrections aicorrectionports.AICorrectionService
	drift       extractionaccuracyports.ExtractionDriftChecker
	tenants     tenantports.TenantSyncRepository
	l           *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		corrections: p.Corrections,
		drift:       p.Drift,
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

	purged, err := a.corrections.PurgeExpired(ctx, kernelports.PurgeExpiredAICorrectionsRequest{
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
