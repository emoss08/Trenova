package demandforecastjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/demandforecastjobs/demandforecastcontract"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

var (
	tenantA = pagination.TenantInfo{OrgID: pulid.ID("org_a"), BuID: pulid.ID("bu_a")}
	tenantB = pagination.TenantInfo{OrgID: pulid.ID("org_b"), BuID: pulid.ID("bu_b")}
)

type fakeForecasts struct {
	demandforecastports.Service
	quality *demandforecast.Quality
	err     error
	fitted  []pagination.TenantInfo
}

func (f *fakeForecasts) Fit(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
) (*demandforecast.Quality, error) {
	f.fitted = append(f.fitted, tenantInfo)
	if f.err != nil {
		return nil, f.err
	}

	return f.quality, nil
}

type fakeTenants struct {
	tenants  []pagination.TenantInfo
	err      error
	requests []*demandforecastports.ListTenantsRequest
}

func (f *fakeTenants) ListTenants(
	_ context.Context,
	req *demandforecastports.ListTenantsRequest,
) ([]pagination.TenantInfo, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, f.err
	}

	return f.tenants, nil
}

func newActivities(forecasts *fakeForecasts, tenants *fakeTenants) *Activities {
	return NewActivities(ActivitiesParams{
		Forecasts: forecasts,
		Tenants:   tenants,
		Logger:    zap.NewNop(),
	})
}

func TestFitDemandForecastActivity_FitsThePayloadsTenantAndReportsTheGrade(t *testing.T) {
	t.Parallel()

	forecasts := &fakeForecasts{quality: &demandforecast.Quality{
		Status:       demandforecast.StatusReady,
		HistoryWeeks: 26,
		LoadsTested:  640,
		Skill:        0.31,
	}}

	result, err := newActivities(forecasts, &fakeTenants{}).FitDemandForecastActivity(
		t.Context(),
		&TenantPayload{TenantWorkItem: temporaljobs.NewTenantWorkItem(tenantA, 1)},
	)
	require.NoError(t, err)

	assert.Equal(t, []pagination.TenantInfo{tenantA}, forecasts.fitted)
	assert.Equal(t, &FitResult{Status: "ready", LoadsTested: 640, Skill: 0.31}, result)
}

func TestFitDemandForecastActivity_ReturnsTheFitError(t *testing.T) {
	t.Parallel()

	historyDown := errors.New("history down")

	result, err := newActivities(&fakeForecasts{err: historyDown}, &fakeTenants{}).
		FitDemandForecastActivity(
			t.Context(),
			&TenantPayload{TenantWorkItem: temporaljobs.NewTenantWorkItem(tenantA, 1)},
		)

	require.ErrorIs(t, err, historyDown)
	assert.Nil(t, result)
}

func TestListForecastTenantsActivity_PagesThroughTenants(t *testing.T) {
	t.Parallel()

	after := temporaljobs.NewTenantWorkItem(tenantA, 1)
	tests := []struct {
		name    string
		payload *ListTenantsPayload
		tenants []pagination.TenantInfo
		want    *demandforecastports.ListTenantsRequest
		hasMore bool
	}{
		{
			name:    "a full page has more after it",
			payload: &ListTenantsPayload{After: &after, Limit: 2},
			tenants: []pagination.TenantInfo{tenantB, tenantA},
			want:    &demandforecastports.ListTenantsRequest{After: tenantA, Limit: 2},
			hasMore: true,
		},
		{
			name:    "a short page is the last",
			payload: &ListTenantsPayload{After: &after, Limit: 2},
			tenants: []pagination.TenantInfo{tenantB},
			want:    &demandforecastports.ListTenantsRequest{After: tenantA, Limit: 2},
		},
		{
			name:    "the first page starts at the beginning with the default size",
			payload: &ListTenantsPayload{},
			tenants: []pagination.TenantInfo{tenantA},
			want:    &demandforecastports.ListTenantsRequest{Limit: tenantPageSize},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tenants := &fakeTenants{tenants: tt.tenants}

			page, err := newActivities(&fakeForecasts{}, tenants).
				ListForecastTenantsActivity(t.Context(), tt.payload)
			require.NoError(t, err)

			assert.Equal(t, []*demandforecastports.ListTenantsRequest{tt.want}, tenants.requests)
			assert.Equal(t, &temporaljobs.TenantPage{
				Tenants: temporaljobs.BuildTenantWorkItems(tt.tenants, 1),
				HasMore: tt.hasMore,
			}, page)
		})
	}
}

func TestListForecastTenantsActivity_ReturnsTheListError(t *testing.T) {
	t.Parallel()

	listDown := errors.New("list down")

	page, err := newActivities(&fakeForecasts{}, &fakeTenants{err: listDown}).
		ListForecastTenantsActivity(t.Context(), &ListTenantsPayload{Limit: 2})

	require.ErrorIs(t, err, listDown)
	assert.Nil(t, page)
}

func newEnv() *testsuite.TestWorkflowEnvironment {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	return env
}

func forTenant(tenantInfo pagination.TenantInfo) any {
	return mock.MatchedBy(func(payload *TenantPayload) bool {
		return payload.TenantInfo() == tenantInfo
	})
}

func TestFitDemandForecastWorkflow_FitsTheOneTenantNow(t *testing.T) {
	t.Parallel()

	env := newEnv()
	var a *Activities
	env.OnActivity(a.FitDemandForecastActivity, mock.Anything, forTenant(tenantA)).
		Return(&FitResult{Status: "ready", LoadsTested: 640, Skill: 0.31}, nil).
		Once()

	env.ExecuteWorkflow(FitDemandForecastWorkflow, &demandforecastcontract.FitPayload{
		OrganizationID: tenantA.OrgID,
		BusinessUnitID: tenantA.BuID,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var result FitResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, FitResult{Status: "ready", LoadsTested: 640, Skill: 0.31}, result)
	env.AssertExpectations(t)
}

// One tenant whose fit fails must not stop the nightly run from fitting
// every other tenant.
func TestFitDemandForecastsWorkflow_FitsEveryTenantAndRecordsTheOneThatFailed(t *testing.T) {
	t.Parallel()

	env := newEnv()
	var a *Activities
	env.OnActivity(a.ListForecastTenantsActivity, mock.Anything, mock.MatchedBy(
		func(payload *ListTenantsPayload) bool {
			return payload.After == nil && payload.Limit == tenantPageSize
		},
	)).Return(&temporaljobs.TenantPage{
		Tenants: temporaljobs.BuildTenantWorkItems(
			[]pagination.TenantInfo{tenantA, tenantB},
			1,
		),
	}, nil).Once()
	env.OnActivity(a.FitDemandForecastActivity, mock.Anything, forTenant(tenantA)).
		Return(&FitResult{Status: "ready", LoadsTested: 300}, nil).
		Once()
	env.OnActivity(a.FitDemandForecastActivity, mock.Anything, forTenant(tenantB)).
		Return(nil, temporal.NewNonRetryableApplicationError("no history", "test", nil)).
		Once()

	env.ExecuteWorkflow(FitDemandForecastsWorkflow, &FanOutInput{})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var result temporaljobs.TenantRunResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 2, result.TenantsScanned)
	assert.Equal(t, 1, result.TenantsProcessed)
	assert.Equal(t, 300, result.RecordsProcessed)
	assert.Equal(t, 1, result.FailureCount)
	require.Len(t, result.PartialFailures, 1)
	assert.Equal(t, tenantB.OrgID, result.PartialFailures[0].OrganizationID)
	assert.Equal(t, tenantB.BuID, result.PartialFailures[0].BusinessUnitID)
	assert.NotEmpty(t, result.PartialFailures[0].Error)
	env.AssertExpectations(t)
}
