package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/stretchr/testify/mock"
)

func NewTestValidator(t testing.TB) *Validator {
	return NewTestValidatorWithAssignmentRepo(t, nil)
}

func NewTestValidatorWithAssignmentRepo(
	t testing.TB,
	assignmentRepo repositories.AssignmentRepository,
) *Validator {
	t.Helper()

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, mock.Anything).
		Return(&tenant.ShipmentControl{
			AllowMoveRemovals:      true,
			MaxShipmentWeightLimit: 1000000,
		}, nil).
		Maybe()
	customerRepo := NewTestCustomerRepository(t)
	if assignmentRepo == nil {
		mockAssignmentRepo := mocks.NewMockAssignmentRepository(t)
		mockAssignmentRepo.EXPECT().
			FindNearestActualEventByTractorID(
				mock.Anything,
				mock.AnythingOfType("repositories.FindNearestActualTimelineEventRequest"),
				mock.AnythingOfType("pulid.ID"),
			).
			Return(nil, nil).
			Maybe()
		mockAssignmentRepo.EXPECT().
			FindNearestActualEventByPrimaryWorkerID(
				mock.Anything,
				mock.AnythingOfType("repositories.FindNearestActualTimelineEventRequest"),
				mock.AnythingOfType("pulid.ID"),
			).
			Return(nil, nil).
			Maybe()
		mockAssignmentRepo.EXPECT().
			FindOverlappingActualWindowByTractorID(
				mock.Anything,
				mock.AnythingOfType("repositories.FindOverlappingActualTimelineWindowRequest"),
				mock.AnythingOfType("pulid.ID"),
			).
			Return(nil, nil).
			Maybe()
		mockAssignmentRepo.EXPECT().
			FindOverlappingActualWindowByPrimaryWorkerID(
				mock.Anything,
				mock.AnythingOfType("repositories.FindOverlappingActualTimelineWindowRequest"),
				mock.AnythingOfType("pulid.ID"),
			).
			Return(nil, nil).
			Maybe()
		assignmentRepo = mockAssignmentRepo
	}

	return &Validator{
		validator: newValidatorBuilder(
			nil,
			controlRepo,
			customerRepo,
			mocks.NewMockCommodityRepository(t),
			mocks.NewMockHazmatSegregationRuleRepository(t),
			mocks.NewMockShipmentRepository(t),
		).Build(),
		assignmentRepo: assignmentRepo,
	}
}

func NewTestCustomerRepository(t testing.TB) *mocks.MockCustomerRepository {
	t.Helper()

	customerRepo := mocks.NewMockCustomerRepository(t)
	customerRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&customer.Customer{
			BillingProfile: &customer.CustomerBillingProfile{},
		}, nil).
		Maybe()

	return customerRepo
}
