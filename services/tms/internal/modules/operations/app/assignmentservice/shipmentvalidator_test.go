package assignmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentvalidator"
	"github.com/emoss08/trenova/internal/modules/operations/operationstest"
	"github.com/emoss08/trenova/internal/testutil/mocks/commodityportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/hazmatsegregationruleportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
)

func newTestShipmentValidator(t testing.TB) *shipmentvalidator.Validator {
	t.Helper()

	assignmentRepo := operationstest.NewAssignmentRepository(t)

	return shipmentvalidator.New(shipmentvalidator.Params{
		AssignmentRepo: assignmentRepo,
		ControlRepo:    operationstest.NewControlRepository(t),
		CustomerRepo:   operationstest.NewCustomerRepository(t),
		CommodityRepo:  commodityportsmock.NewMockCommodityRepository(t),
		HazmatSegregationRuleRepo: hazmatsegregationruleportsmock.NewMockHazmatSegregationRuleRepository(
			t,
		),
		ShipmentRepo: shipmentportsmock.NewMockShipmentRepository(t),
	})
}
