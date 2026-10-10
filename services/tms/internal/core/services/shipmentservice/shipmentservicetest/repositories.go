package shipmentservicetest

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/stretchr/testify/mock"
)

func NewAssignmentRepository(t testing.TB) *shipmentportsmock.MockAssignmentRepository {
	t.Helper()

	repo := shipmentportsmock.NewMockAssignmentRepository(t)
	repo.EXPECT().
		FindNearestActualEventByTractorID(
			mock.Anything,
			mock.AnythingOfType("repositories.FindNearestActualTimelineEventRequest"),
			mock.AnythingOfType("pulid.ID"),
		).
		Return(nil, nil).
		Maybe()
	repo.EXPECT().
		FindNearestActualEventByPrimaryWorkerID(
			mock.Anything,
			mock.AnythingOfType("repositories.FindNearestActualTimelineEventRequest"),
			mock.AnythingOfType("pulid.ID"),
		).
		Return(nil, nil).
		Maybe()
	repo.EXPECT().
		FindOverlappingActualWindowByTractorID(
			mock.Anything,
			mock.AnythingOfType("repositories.FindOverlappingActualTimelineWindowRequest"),
			mock.AnythingOfType("pulid.ID"),
		).
		Return(nil, nil).
		Maybe()
	repo.EXPECT().
		FindOverlappingActualWindowByPrimaryWorkerID(
			mock.Anything,
			mock.AnythingOfType("repositories.FindOverlappingActualTimelineWindowRequest"),
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
