package aifeedbackjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aifeedbackports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Maintenance aifeedbackports.AIFeedbackMaintenance
	Tenants     tenantports.TenantSyncRepository
	Logger      *zap.Logger
}

type Activities struct {
	maintenance aifeedbackports.AIFeedbackMaintenance
	tenants     tenantports.TenantSyncRepository
	l           *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		maintenance: p.Maintenance,
		tenants:     p.Tenants,
		l:           p.Logger.Named("job.aifeedback-maintenance"),
	}
}

func (a *Activities) ListAIFeedbackOrganizationsActivity(
	ctx context.Context,
	input *ListAIFeedbackOrganizationsInput,
) (*temporaljobs.TenantPage, error) {
	organizations, err := a.tenants.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	return temporaljobs.OrganizationPage(organizations, input.After, input.Limit), nil
}

func (a *Activities) MaintainOrganizationAIFeedbackActivity(
	ctx context.Context,
	input *OrganizationAIFeedbackInput,
) (*OrganizationAIFeedbackResult, error) {
	tenant := input.TenantInfo()

	purged, err := a.maintenance.PurgeExpired(ctx, aifeedbackports.PurgeExpiredAIFeedbackRequest{
		TenantInfo: tenant,
		Now:        input.Now,
	})
	if err != nil {
		return nil, fmt.Errorf("purge expired ai feedback: %w", err)
	}

	suggested, err := a.maintenance.SuggestMemories(
		ctx,
		aifeedbackports.SuggestAgentMemoriesRequest{
			TenantInfo: tenant,
			Now:        input.Now,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("suggest agent memories: %w", err)
	}

	a.l.Debug("ai feedback maintained",
		zap.String("organization", tenant.OrgID.String()),
		zap.Int64("purged", purged),
		zap.Int("suggested", suggested.Suggested),
	)

	return &OrganizationAIFeedbackResult{
		Purged:     purged,
		Suggested:  suggested.Suggested,
		Covered:    suggested.Covered,
		BelowFloor: suggested.BelowFloor,
	}, nil
}
