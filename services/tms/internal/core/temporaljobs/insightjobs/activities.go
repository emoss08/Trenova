package insightjobs

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/insightservice"
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

// RefreshInsightsActivity recomputes insights for every organization.
//
// One organization's failure is recorded and stepped over. A tenant with corrupt
// distance data or a detention table mid-migration must not take every other
// tenant's home screen with it, and the next run will pick that tenant up again
// without anyone intervening.
//
// The run is anchored to a single instant captured up front, so every tenant in
// one sweep describes the same window. Reading the clock per tenant would make a
// long sweep report subtly different periods at its start and end.
func (a *Activities) RefreshInsightsActivity(
	ctx context.Context,
) (*InsightRefreshResult, error) {
	organizations, err := a.tenants.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	now := timeutils.NowUnix()
	result := &InsightRefreshResult{
		FailedOrganizations: make([]string, 0),
		FailedDetectors:     make([]string, 0),
	}

	for _, org := range organizations {
		// Heartbeating keeps a sweep over many tenants from looking hung to
		// Temporal, and lets a retry see how far the previous attempt reached.
		activity.RecordHeartbeat(ctx, org.ID.String())

		refresh, rErr := a.insights.Refresh(ctx, insightservice.RefreshRequest{
			TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID},
			Now:        now,
		})
		if rErr != nil {
			a.l.Error("insight refresh failed for organization",
				zap.String("organization", org.ID.String()),
				zap.Error(rErr),
			)
			result.FailedOrganizations = append(result.FailedOrganizations, org.ID.String())

			continue
		}

		result.OrganizationsProcessed++
		result.InsightsCreated += refresh.Created
		result.InsightsResolved += refresh.Resolved
		result.InsightsSuppressed += refresh.Suppressed
		result.InsightsNarrated += refresh.Narrated

		for _, failed := range refresh.Failed {
			// A detector broken by a schema change fails for every tenant. One
			// line naming it is actionable; five hundred identical lines are not.
			if !slices.Contains(result.FailedDetectors, failed) {
				result.FailedDetectors = append(result.FailedDetectors, failed)
			}
		}
	}

	a.l.Info("insight refresh sweep complete",
		zap.Int("organizations", result.OrganizationsProcessed),
		zap.Int("created", result.InsightsCreated),
		zap.Int("narrated", result.InsightsNarrated),
		zap.Strings("failedDetectors", result.FailedDetectors),
	)

	return result, nil
}
