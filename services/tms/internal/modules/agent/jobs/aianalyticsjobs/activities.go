package aianalyticsjobs

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Analytics aianalyticsports.AnalyticsService
	Active    aianalyticsports.ActiveTenantLister
	Logger    *zap.Logger
}

type Activities struct {
	analytics aianalyticsports.AnalyticsService
	active    aianalyticsports.ActiveTenantLister
	l         *zap.Logger
	now       func() int64
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		analytics: p.Analytics,
		active:    p.Active,
		l:         p.Logger.Named("job.ai-analytics"),
		now:       timeutils.NowUnix,
	}
}

// ListAIAnalyticsOrganizationsActivity lists one page of the organizations
// with AI activity in the last activeWindow. A refresh starts from the
// tenant's own watermark, so an organization that has gone quiet loses
// nothing by being left out: its next activity brings it back, and the
// refresh then covers everything since its watermark.
func (a *Activities) ListAIAnalyticsOrganizationsActivity(
	ctx context.Context,
	input *ListAIAnalyticsOrganizationsInput,
) (*temporaljobs.TenantPage, error) {
	since := a.now() - int64(activeWindow.Seconds())
	tenants, err := a.active.ListActiveTenants(ctx, since, maxActiveTenants)
	if err != nil {
		return nil, fmt.Errorf("list organizations with ai activity: %w", err)
	}

	return pageAfter(tenants, input.After, input.Limit), nil
}

// pageAfter is one page of tenants, already in organization id order, after
// the cursor.
func pageAfter(
	tenants []pagination.TenantInfo,
	after *temporaljobs.TenantWorkItem,
	limit int,
) *temporaljobs.TenantPage {
	limit = temporaljobs.NormalizeLimit(limit, temporaljobs.DefaultOrganizationPageSize)
	start := 0
	if after != nil {
		cursor := after.OrganizationID.String()
		start, _ = slices.BinarySearchFunc(tenants, cursor,
			func(tenant pagination.TenantInfo, id string) int {
				return strings.Compare(tenant.OrgID.String(), id)
			})
		for start < len(tenants) && tenants[start].OrgID.String() <= cursor {
			start++
		}
	}

	end := min(start+limit, len(tenants))
	items := make([]temporaljobs.TenantWorkItem, 0, end-start)
	for _, tenant := range tenants[start:end] {
		items = append(items, temporaljobs.NewTenantWorkItem(tenant, 1))
	}

	return &temporaljobs.TenantPage{Tenants: items, HasMore: end < len(tenants)}
}

func (a *Activities) RefreshOrganizationAIAnalyticsActivity(
	ctx context.Context,
	input *OrganizationAIAnalyticsInput,
) (*OrganizationAIAnalyticsResult, error) {
	tenant := input.TenantInfo()
	refreshed, err := a.analytics.Refresh(ctx, tenant, func(refreshedThrough int64) {
		activity.RecordHeartbeat(ctx, refreshedThrough)
	})
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
