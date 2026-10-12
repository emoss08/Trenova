package trailerservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/commodityportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/hazmatsegregationruleportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"

	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentservice"
	"github.com/emoss08/trenova/internal/modules/operations/operationstest"
)

func newTestShipmentValidator(t testing.TB) *shipmentservice.Validator {
	t.Helper()

	assignmentRepo := operationstest.NewAssignmentRepository(t)

	return shipmentservice.NewValidator(shipmentservice.ValidatorParams{
		AssignmentRepo:            assignmentRepo,
		ControlRepo:               operationstest.NewControlRepository(t),
		CustomerRepo:              operationstest.NewCustomerRepository(t),
		CommodityRepo:             commodityportsmock.NewMockCommodityRepository(t),
		HazmatSegregationRuleRepo: hazmatsegregationruleportsmock.NewMockHazmatSegregationRuleRepository(t),
		ShipmentRepo:              shipmentportsmock.NewMockShipmentRepository(t),
	})
}
