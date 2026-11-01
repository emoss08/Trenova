package apikeyanalyticsresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyAnalytics_ChecksTheAPIKeyPermission(t *testing.T) {
	t.Parallel()

	engine := resolvertest.RefusingEngine()
	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}

	_, err := (&QueryResolver{deps}).APIKeyAnalytics(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		nil,
	)

	resolvertest.RequireRefused(t, err, engine, permission.ResourceAPIKey, permission.OpRead)
}

func TestAPIKeyAnalytics_ReadsTheAPIKeyPageInTheGivenTimezone(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	timezone := "America/Chicago"
	service := kernelportsmock.NewMockAnalyticsService(t)
	service.EXPECT().
		GetAnalytics(mock.Anything, mock.MatchedBy(func(opts *kernelports.AnalyticsRequestOptions) bool {
			return opts.Page == kernelports.APIKeyAnalyticsPage &&
				opts.Timezone == timezone &&
				opts.OrgID == authCtx.OrganizationID &&
				opts.BuID == authCtx.BusinessUnitID &&
				opts.Principal.UserID == authCtx.UserID
		})).
		Return(kernelports.AnalyticsData{
			"totalKeys":   map[string]any{"count": 4, "newThisMonth": 1},
			"activeKeys":  map[string]any{"count": 3, "percentOfTotal": 75.0},
			"revokedKeys": map[string]any{"count": 1, "percentOfTotal": 25.0},
			"requests30d": map[string]any{
				"total":     int64(42),
				"sparkline": []map[string]any{{"day": "Jan 2", "value": int64(42)}},
			},
		}, nil)
	deps := &Deps{
		Core:             &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}},
		AnalyticsService: service,
	}

	got, err := (&QueryResolver{deps}).APIKeyAnalytics(
		resolvertest.AuthedContext(t, authCtx),
		&timezone,
	)
	require.NoError(t, err)

	assert.Equal(t, 4, got.TotalKeys.Count)
	assert.Equal(t, 1, got.TotalKeys.NewThisMonth)
	assert.Equal(t, 3, got.ActiveKeys.Count)
	assert.InDelta(t, 75.0, got.ActiveKeys.PercentOfTotal, 0.001)
	assert.InDelta(t, 25.0, got.RevokedKeys.PercentOfTotal, 0.001)
	assert.Equal(t, 42, got.Requests30d.Total)
	require.Len(t, got.Requests30d.Sparkline, 1)
	assert.Equal(t, "Jan 2", got.Requests30d.Sparkline[0].Day)
	assert.Equal(t, 42, got.Requests30d.Sparkline[0].Value)
}
