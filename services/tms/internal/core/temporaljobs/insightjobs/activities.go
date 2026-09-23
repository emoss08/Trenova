package insightjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/insightservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Insights *insightservice.Service
	Tenants  repositories.TenantSyncRepository
	Logger   *zap.Logger
}

type Activities struct {
	insights *insightservice.Service
	tenants  repositories.TenantSyncRepository
	l        *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		insights: p.Insights,
		tenants:  p.Tenants,
		l:        p.Logger.Named("job.insight-refresh"),
	}
}

// RefreshInsightsActivity recomputes insights for every organization in one
// activity. It serves only sweeps started before the sweep fanned out to a
// child workflow per organization.
//
// One organization's failure is recorded and stepped over, and the run is
// anchored to a single instant captured up front, so every tenant in one sweep
// describes the same window.
func (a *Activities) RefreshInsightsActivity(
	ctx context.Context,
) (*InsightRefreshResult, error) {
	organizations, err := a.tenants.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	now := timeutils.NowUnix()
	result := newInsightRefreshResult()

	for _, org := range organizations {
		activity.RecordHeartbeat(ctx, org.ID.String())

		refreshed, rErr := a.refresh(ctx, pagination.TenantInfo{
			OrgID: org.ID,
			BuID:  org.BusinessUnitID,
		}, now)
		if rErr != nil {
			a.l.Error("insight refresh failed for organization",
				zap.String("organization", org.ID.String()),
				zap.Error(rErr),
			)
			result.FailedOrganizations = append(result.FailedOrganizations, org.ID.String())

			continue
		}
		result.absorb(refreshed)
	}

	a.l.Info("insight refresh sweep complete",
		zap.Int("organizations", result.OrganizationsProcessed),
		zap.Int("created", result.InsightsCreated),
		zap.Strings("failedOrganizations", result.FailedOrganizations),
		zap.Strings("failedDetectors", result.FailedDetectors),
	)

	return result, nil
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
