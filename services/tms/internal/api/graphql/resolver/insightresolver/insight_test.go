package insightresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInsightResolvers_CheckTheInsightPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("inst_").String()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"activeInsights", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).ActiveInsights(ctx, nil)
			return err
		}},
		{"insights", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).Insights(ctx, gqlmodel.BrowseInsightsInput{})
			return err
		}},
		{"insight", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).Insight(ctx, id)
			return err
		}},
		{"dismissInsight", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).DismissInsight(
				ctx,
				gqlmodel.DismissInsightInput{InsightID: id},
			)
			return err
		}},
		{"restoreInsight", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).RestoreInsight(ctx, id)
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(t, err, engine, permission.ResourceInsight, call.operation)
		})
	}
}

func TestActiveInsightsRequest_BoundsTheWidgetAndChecksTheReader(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	surface := insight.SurfaceDispatch
	limit := 500

	req := activeInsightsRequest(authCtx, &gqlmodel.ActiveInsightsInput{
		Categories: []insight.Category{insight.CategoryCashFlow},
		Surface:    &surface,
		Limit:      &limit,
	})

	assert.Equal(t, maxActiveLimit, req.Limit)
	assert.Equal(t, insight.SurfaceDispatch, req.Surface)
	assert.Equal(t, []insight.Category{insight.CategoryCashFlow}, req.Categories)
	assert.Equal(t, authCtx.UserID, req.UserID)
	require.NotNil(t, req.Actor)
	assert.Equal(t, authCtx.PrincipalID, req.Actor.PrincipalID)
	assert.Equal(t, authCtx.OrganizationID, req.TenantInfo.OrgID)
}

func TestActiveInsightsRequest_NoInputIsTheHomeScreensWholeView(t *testing.T) {
	t.Parallel()

	req := activeInsightsRequest(resolvertest.UserAuthContext(), nil)

	assert.Zero(t, req.Limit)
	assert.Empty(t, req.Surface)
	assert.Nil(t, req.Categories)
}

func TestBrowseInsightsRequest_BoundsThePage(t *testing.T) {
	t.Parallel()

	limit, offset := 1_000, 50_000
	negative := -5

	req := browseInsightsRequest(resolvertest.UserAuthContext(), &gqlmodel.BrowseInsightsInput{
		Statuses: []insight.Status{insight.StatusDismissed},
		Limit:    &limit,
		Offset:   &offset,
	})
	assert.Equal(t, maxBrowseLimit, req.Limit)
	assert.Equal(t, maxBrowseOffset, req.Offset)
	assert.Equal(t, []insight.Status{insight.StatusDismissed}, req.Statuses)
	assert.Nil(t, req.Categories)
	assert.Nil(t, req.Severities)

	req = browseInsightsRequest(resolvertest.UserAuthContext(), &gqlmodel.BrowseInsightsInput{
		Limit: &negative,
	})
	assert.Zero(t, req.Limit)
	assert.Zero(t, req.Offset)
}

func TestInsightMetric_ValuesAreExactDecimals(t *testing.T) {
	t.Parallel()

	resolver := &InsightMetricResolver{&Deps{}}
	baseline := decimal.RequireFromString("0.125")
	metric := &insight.Metric{Value: decimal.RequireFromString("0.1234"), Baseline: &baseline}

	value, err := resolver.Value(t.Context(), metric)
	require.NoError(t, err)
	assert.Equal(t, "0.1234", value)

	got, err := resolver.Baseline(t.Context(), metric)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "0.125", *got)

	none, err := resolver.Baseline(t.Context(), &insight.Metric{})
	require.NoError(t, err)
	assert.Nil(t, none)
}
