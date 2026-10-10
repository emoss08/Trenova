package trailerservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/shipmentservice"
	"github.com/emoss08/trenova/internal/core/services/shipmentservice/shipmentservicetest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
)

func newTestShipmentValidator(t testing.TB) *shipmentservice.Validator {
	t.Helper()

	assignmentRepo := shipmentservicetest.NewAssignmentRepository(t)

	return shipmentservice.NewValidator(shipmentservice.ValidatorParams{
		AssignmentRepo:            assignmentRepo,
		ControlRepo:               shipmentservicetest.NewControlRepository(t),
		CustomerRepo:              shipmentservicetest.NewCustomerRepository(t),
		CommodityRepo:             mocks.NewMockCommodityRepository(t),
		HazmatSegregationRuleRepo: mocks.NewMockHazmatSegregationRuleRepository(t),
		ShipmentRepo:              mocks.NewMockShipmentRepository(t),
	})
}
