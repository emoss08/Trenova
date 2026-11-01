package customerservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/trackingprofile"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPlanUpdate_TransformsAndReturnsTheStoredRecordBesideTheChange(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	entity := newTestEntity()
	stored := *entity
	stored.Name = "Old Name"
	deps.transformer.On("TransformCustomer", mock.Anything, entity).Return(nil)
	deps.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&stored, nil)

	change, err := deps.svc.PlanUpdate(t.Context(), entity)
	require.NoError(t, err)
	assert.Equal(t, "Old Name", change.Before.Name)
	assert.Equal(t, entity.Name, change.After.Name)
	deps.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestPlanUpdate_ReadsTheStoredTrackingSettingsAndKeepsTheirStartTime(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	entity := newTestEntity()
	entity.TrackingProfile = trackingprofile.NewDefault(
		entity.OrganizationID,
		entity.BusinessUnitID,
		entity.ID,
	)
	entity.TrackingProfile.NotifyDelivered = true
	enabledAt := int64(1_700_000_000)
	stored := *entity
	stored.TrackingProfile = trackingprofile.NewDefault(
		entity.OrganizationID,
		entity.BusinessUnitID,
		entity.ID,
	)
	stored.TrackingProfile.NotifyDelivered = true
	stored.TrackingProfile.NotificationsEnabledAt = &enabledAt
	deps.transformer.On("TransformCustomer", mock.Anything, entity).Return(nil)
	deps.repo.EXPECT().GetByID(
		mock.Anything,
		mock.MatchedBy(func(req customerports.GetCustomerByIDRequest) bool {
			return req.IncludeTrackingProfile
		}),
	).Return(&stored, nil)

	change, err := deps.svc.PlanUpdate(t.Context(), entity)
	require.NoError(t, err)
	require.NotNil(t, change.After.TrackingProfile.NotificationsEnabledAt)
	assert.Equal(t, enabledAt, *change.After.TrackingProfile.NotificationsEnabledAt)
}

func TestPlanCreate_StampsWhenStatusEmailsStart(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	entity := newCreateEntity()
	entity.TrackingProfile = trackingprofile.NewDefault(
		entity.OrganizationID,
		entity.BusinessUnitID,
		entity.ID,
	)
	entity.TrackingProfile.NotifyPickedUp = true
	deps.transformer.On("TransformCustomer", mock.Anything, entity).Return(nil)

	before := timeutils.NowUnix()
	planned, err := deps.svc.PlanCreate(t.Context(), entity)
	require.NoError(t, err)
	require.NotNil(t, planned.TrackingProfile.NotificationsEnabledAt)
	assert.GreaterOrEqual(t, *planned.TrackingProfile.NotificationsEnabledAt, before)
}

func TestPlanBulkUpdateStatus_RefusesAStatusAndARecordItDoesNotKnow(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	entity := newTestEntity()
	deps.repo.EXPECT().GetByIDs(mock.Anything, mock.Anything).Return(
		[]*customer.Customer{entity}, nil,
	)

	_, err := deps.svc.PlanBulkUpdateStatus(t.Context(), &customerports.BulkUpdateCustomerStatusRequest{
		TenantInfo:  pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		CustomerIDs: []pulid.ID{entity.ID, pulid.MustNew("cus_")},
		Status:      domaintypes.Status("Retired"),
	})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}
	assert.ElementsMatch(t, []string{"status", "customerIds"}, fields)
	deps.repo.AssertNotCalled(t, "BulkUpdateStatus", mock.Anything, mock.Anything)
}
