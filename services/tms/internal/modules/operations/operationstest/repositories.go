package operationstest

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/testutil/mocks/assignmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/stretchr/testify/mock"
)

func NewAssignmentRepository(t testing.TB) *assignmentportsmock.MockAssignmentRepository {
	t.Helper()

	repo := assignmentportsmock.NewMockAssignmentRepository(t)
	repo.EXPECT().
		FindNearestActualEventByTractorID(
			mock.Anything,
			mock.AnythingOfType("assignmentports.FindNearestActualTimelineEventRequest"),
			mock.AnythingOfType("pulid.ID"),
		).
		Return(nil, nil).
		Maybe()
	repo.EXPECT().
		FindNearestActualEventByPrimaryWorkerID(
			mock.Anything,
			mock.AnythingOfType("assignmentports.FindNearestActualTimelineEventRequest"),
			mock.AnythingOfType("pulid.ID"),
		).
		Return(nil, nil).
		Maybe()
	repo.EXPECT().
		FindOverlappingActualWindowByTractorID(
			mock.Anything,
			mock.AnythingOfType("assignmentports.FindOverlappingActualTimelineWindowRequest"),
			mock.AnythingOfType("pulid.ID"),
		).
		Return(nil, nil).
		Maybe()
	repo.EXPECT().
		FindOverlappingActualWindowByPrimaryWorkerID(
			mock.Anything,
			mock.AnythingOfType("assignmentports.FindOverlappingActualTimelineWindowRequest"),
			mock.AnythingOfType("pulid.ID"),
		).
		Return(nil, nil).
		Maybe()

	return repo
}

func NewControlRepository(t testing.TB) *tenantportsmock.MockShipmentControlRepository {
	t.Helper()

	repo := tenantportsmock.NewMockShipmentControlRepository(t)
	repo.EXPECT().
		Get(mock.Anything, mock.Anything).
		Return(&tenant.ShipmentControl{
			AllowMoveRemovals:      true,
			MaxShipmentWeightLimit: 1000000,
		}, nil).
		Maybe()

	return repo
}

func NewCustomerRepository(t testing.TB) *customerportsmock.MockCustomerRepository {
	t.Helper()

	repo := customerportsmock.NewMockCustomerRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&customer.Customer{
			BillingProfile: &customer.CustomerBillingProfile{},
		}, nil).
		Maybe()

	return repo
}
