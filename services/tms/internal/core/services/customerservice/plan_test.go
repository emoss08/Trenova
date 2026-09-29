package customerservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

func TestPlanBulkUpdateStatus_RefusesAStatusAndARecordItDoesNotKnow(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	entity := newTestEntity()
	deps.repo.EXPECT().GetByIDs(mock.Anything, mock.Anything).Return(
		[]*customer.Customer{entity}, nil,
	)

	_, err := deps.svc.PlanBulkUpdateStatus(t.Context(), &repositories.BulkUpdateCustomerStatusRequest{
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
