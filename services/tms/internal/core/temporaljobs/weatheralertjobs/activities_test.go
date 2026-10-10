package weatheralertjobs

import (
	"context"
	"errors"
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPollNWSAlertsActivity_ReportsEachTenant(t *testing.T) {
	t.Parallel()

	healthy := pagination.TenantInfo{OrgID: pulid.ID("org_a"), BuID: pulid.ID("bu_a")}
	broken := pagination.TenantInfo{OrgID: pulid.ID("org_b"), BuID: pulid.ID("bu_b")}
	quiet := pagination.TenantInfo{OrgID: pulid.ID("org_c"), BuID: pulid.ID("bu_c")}

	service := mocks.NewMockWeatherAlertService(t)
	service.EXPECT().
		PollNWSAlerts(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			heartbeat serviceports.WeatherAlertPollHeartbeat,
		) (*serviceports.PollNWSAlertsResult, error) {
			require.NotNil(t, heartbeat)
			heartbeat("outside an activity a heartbeat is a no-op")

			return &serviceports.PollNWSAlertsResult{
				AlertsInFeed:   10,
				TenantsScanned: 3,
				Tenants: []serviceports.WeatherAlertTenantSync{
					{TenantInfo: healthy, Written: 3, Unchanged: 7},
					{TenantInfo: broken, Written: 1, Err: errors.New("database unavailable")},
					{TenantInfo: quiet, Unchanged: 10},
				},
			}, nil
		}).
		Once()

	activities := NewActivities(ActivitiesParams{Service: service, Logger: zap.NewNop()})
	result, err := activities.PollNWSAlertsActivity(t.Context())
	require.NoError(t, err)

	assert.False(t, result.FeedUnchanged)
	assert.Equal(t, 10, result.AlertsInFeed)
	assert.Equal(t, 3, result.TenantsScanned)
	assert.Equal(t, 2, result.TenantsProcessed)
	assert.Equal(t, 3, result.RecordsProcessed)
	assert.Equal(t, 17, result.SkippedCount)
	assert.Equal(t, 1, result.FailureCount)
	require.Len(t, result.PartialFailures, 1)
	assert.Equal(t, broken.OrgID, result.PartialFailures[0].OrganizationID)
	assert.Equal(t, broken.BuID, result.PartialFailures[0].BusinessUnitID)
	assert.Equal(t, "database unavailable", result.PartialFailures[0].Error)
}

func TestPollNWSAlertsActivity_ReportsAnUnchangedFeed(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockWeatherAlertService(t)
	service.EXPECT().
		PollNWSAlerts(mock.Anything, mock.Anything).
		Return(&serviceports.PollNWSAlertsResult{FeedUnchanged: true, TenantsScanned: 4}, nil).
		Once()

	activities := NewActivities(ActivitiesParams{Service: service, Logger: zap.NewNop()})
	result, err := activities.PollNWSAlertsActivity(t.Context())
	require.NoError(t, err)

	assert.True(t, result.FeedUnchanged)
	assert.Equal(t, 4, result.TenantsScanned)
	assert.Zero(t, result.TenantsProcessed)
	assert.Zero(t, result.FailureCount)
}

func TestPollNWSAlertsActivity_ReturnsAPollFailure(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockWeatherAlertService(t)
	service.EXPECT().
		PollNWSAlerts(mock.Anything, mock.Anything).
		Return(nil, errors.New("feed down")).
		Once()

	activities := NewActivities(ActivitiesParams{Service: service, Logger: zap.NewNop()})
	_, err := activities.PollNWSAlertsActivity(t.Context())
	require.Error(t, err)
}
