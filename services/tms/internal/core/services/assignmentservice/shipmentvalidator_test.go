package assignmentservice

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/commodityportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/hazmatsegregationruleportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
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
		CommodityRepo:             commodityportsmock.NewMockCommodityRepository(t),
		HazmatSegregationRuleRepo: hazmatsegregationruleportsmock.NewMockHazmatSegregationRuleRepository(t),
		ShipmentRepo:              shipmentportsmock.NewMockShipmentRepository(t),
	})
}
