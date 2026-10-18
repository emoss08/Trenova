package insightjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/app/insightservice"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Insights *insightservice.Service
	Tenants  tenantports.TenantSyncRepository
	Logger   *zap.Logger
}

type Activities struct {
	insights *insightservice.Service
	tenants  tenantports.TenantSyncRepository
	l        *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		insights: p.Insights,
		tenants:  p.Tenants,
		l:        p.Logger.Named("job.insight-refresh"),
	}
}

// ListOrganizationsActivity lists one page of organizations to refresh.
func (a *Activities) ListOrganizationsActivity(
	ctx context.Context,
	input *ListOrganizationsInput,
) (*temporaljobs.TenantPage, error) {
	organizations, err := a.tenants.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	return temporaljobs.OrganizationPage(organizations, input.After, input.Limit), nil
}

// RefreshOrganizationInsightsActivity recomputes one organization's insights
// as of the sweep's instant. It heartbeats on a timer, because a narration
// call can outlast the heartbeat timeout without producing anything.
func (a *Activities) RefreshOrganizationInsightsActivity(
	ctx context.Context,
	input *OrganizationInsightsInput,
) (*OrganizationInsightsResult, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	return a.refresh(ctx, input.TenantInfo(), input.Now)
}

func (a *Activities) refresh(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	now int64,
) (*OrganizationInsightsResult, error) {
	refresh, err := a.insights.Refresh(ctx, insightservice.RefreshRequest{
		TenantInfo: tenantInfo,
		Now:        now,
	})
	if err != nil {
		return nil, err
	}

	return &OrganizationInsightsResult{
		Created:    refresh.Created,
		Resolved:   refresh.Resolved,
		Suppressed: refresh.Suppressed,
		Narrated:   refresh.Narrated,
		Failed:     refresh.Failed,
	}, nil
}
