package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/shipmentservice/shipmentservicetest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
)

func newTestValidator(t testing.TB) *Validator {
	t.Helper()

	return newTestValidatorWithAssignmentRepo(t, shipmentservicetest.NewAssignmentRepository(t))
}

func newTestValidatorWithAssignmentRepo(
	t testing.TB,
	assignmentRepo repositories.AssignmentRepository,
) *Validator {
	t.Helper()

	return NewValidator(ValidatorParams{
		AssignmentRepo:            assignmentRepo,
		ControlRepo:               shipmentservicetest.NewControlRepository(t),
		CustomerRepo:              shipmentservicetest.NewCustomerRepository(t),
		CommodityRepo:             mocks.NewMockCommodityRepository(t),
		HazmatSegregationRuleRepo: mocks.NewMockHazmatSegregationRuleRepository(t),
		ShipmentRepo:              mocks.NewMockShipmentRepository(t),
	})
}
