package carrierservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPlanCreate_RefusesWhatCreateRefusesAndWritesNothing(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	entity := newCreateEntity()
	entity.Name = ""

	_, err := deps.svc.PlanCreate(t.Context(), entity)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	deps.repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestPlanUpdate_ReturnsTheStoredCarrierBesideTheChange(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	entity := newTestEntity()
	stored := *entity
	stored.Name = "Old Name"
	deps.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&stored, nil)

	change, err := deps.svc.PlanUpdate(t.Context(), entity)
	require.NoError(t, err)
	assert.Equal(t, "Old Name", change.Before.Name)
	assert.Equal(t, entity.Name, change.After.Name)
	deps.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestPlanBulkUpdateStatus_RefusesACarrierThatIsNotHere(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	entity := newTestEntity()
	deps.repo.EXPECT().GetByIDs(mock.Anything, mock.Anything).Return(
		[]*carrier.Carrier{entity}, nil,
	)

	_, err := deps.svc.PlanBulkUpdateStatus(t.Context(), &repositories.BulkUpdateCarrierStatusRequest{
		TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		CarrierIDs: []pulid.ID{entity.ID, pulid.MustNew("car_")},
		Status:     carrier.StatusInactive,
	})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "carrierIds", multiErr.Errors[0].Field)
	deps.repo.AssertNotCalled(t, "BulkUpdateStatus", mock.Anything, mock.Anything)
}
