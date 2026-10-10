package aicorrectionjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aicorrectionports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionrolloutports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionshadowports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Corrections aicorrectionports.AICorrectionService
	Evaluations extractionevalports.ExtractionEvalRunner          `optional:"true"`
	Shadows     extractionshadowports.ExtractionShadowRetention   `optional:"true"`
	Rollout     extractionrolloutports.ExtractionRolloutRetention `optional:"true"`
	Drift       extractionevalports.ExtractionDriftChecker        `optional:"true"`
	Tenants     tenantports.TenantSyncRepository
	Logger      *zap.Logger
}

type Activities struct {
	corrections aicorrectionports.AICorrectionService
	evaluations extractionevalports.ExtractionEvalRunner
	shadows     extractionshadowports.ExtractionShadowRetention
	rollout     extractionrolloutports.ExtractionRolloutRetention
	drift       extractionevalports.ExtractionDriftChecker
	tenants     tenantports.TenantSyncRepository
	l           *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		corrections: p.Corrections,
		evaluations: p.Evaluations,
		shadows:     p.Shadows,
		rollout:     p.Rollout,
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

	if a.evaluations != nil {
		runs, runErr := a.evaluations.PurgeExpiredRuns(
			ctx,
			kernelports.PurgeExpiredAICorrectionsRequest{
				TenantInfo: tenant,
				Now:        input.Now,
			},
		)
		if runErr != nil {
			return nil, fmt.Errorf("purge expired extraction evaluation runs: %w", runErr)
		}
		purged += runs
	}

	if a.shadows != nil {
		shadows, shadowErr := a.shadows.PurgeExpiredShadows(
			ctx,
			kernelports.PurgeExpiredAICorrectionsRequest{
				TenantInfo: tenant,
				Now:        input.Now,
			},
		)
		if shadowErr != nil {
			return nil, fmt.Errorf("purge expired extraction shadows: %w", shadowErr)
		}
		purged += shadows
	}

	if a.rollout != nil {
		assignments, rolloutErr := a.rollout.PurgeExpiredAssignments(
			ctx,
			kernelports.PurgeExpiredAICorrectionsRequest{
				TenantInfo: tenant,
				Now:        input.Now,
			},
		)
		if rolloutErr != nil {
			return nil, fmt.Errorf("purge expired extraction rollout assignments: %w", rolloutErr)
		}
		purged += assignments
	}

	a.l.Debug("ai corrections purged",
		zap.String("organization", tenant.OrgID.String()),
		zap.Int64("purged", purged),
	)

	return &OrganizationAICorrectionResult{Purged: purged}, nil
}
