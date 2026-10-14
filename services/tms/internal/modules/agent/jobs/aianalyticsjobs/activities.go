package aianalyticsjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Analytics aianalyticsports.AnalyticsService
	Tenants   tenantports.TenantSyncRepository
	Logger    *zap.Logger
}

type Activities struct {
	analytics aianalyticsports.AnalyticsService
	tenants   tenantports.TenantSyncRepository
	l         *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		analytics: p.Analytics,
		tenants:   p.Tenants,
		l:         p.Logger.Named("job.ai-analytics"),
	}
}

func (a *Activities) ListAIAnalyticsOrganizationsActivity(
	ctx context.Context,
	input *ListAIAnalyticsOrganizationsInput,
) (*temporaljobs.TenantPage, error) {
	organizations, err := a.tenants.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	return temporaljobs.OrganizationPage(organizations, input.After, input.Limit), nil
}

func (a *Activities) RefreshOrganizationAIAnalyticsActivity(
	ctx context.Context,
	input *OrganizationAIAnalyticsInput,
) (*OrganizationAIAnalyticsResult, error) {
	tenant := input.TenantInfo()
	refreshed, err := a.analytics.Refresh(ctx, tenant)
	if err != nil {
		return nil, fmt.Errorf("refresh ai analytics: %w", err)
	}

	a.l.Debug("ai analytics refreshed",
		zap.String("organization", tenant.OrgID.String()),
		zap.Int("units", refreshed.Units),
		zap.Bool("caughtUp", refreshed.CaughtUp),
	)
	return &OrganizationAIAnalyticsResult{
		Units:    refreshed.Units,
		Hours:    refreshed.Hours,
		CaughtUp: refreshed.CaughtUp,
	}, nil
}
