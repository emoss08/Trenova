package shipmentservice

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/commodityportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/hazmatsegregationruleportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"testing"

	"github.com/emoss08/trenova/internal/core/services/shipmentservice/shipmentservicetest"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
)

func newTestValidator(t testing.TB) *Validator {
	t.Helper()

	return newTestValidatorWithAssignmentRepo(t, shipmentservicetest.NewAssignmentRepository(t))
}

func newTestValidatorWithAssignmentRepo(
	t testing.TB,
	assignmentRepo shipmentports.AssignmentRepository,
) *Validator {
	t.Helper()

	return NewValidator(ValidatorParams{
		AssignmentRepo:            assignmentRepo,
		ControlRepo:               shipmentservicetest.NewControlRepository(t),
		CustomerRepo:              shipmentservicetest.NewCustomerRepository(t),
		CommodityRepo:             commodityportsmock.NewMockCommodityRepository(t),
		HazmatSegregationRuleRepo: hazmatsegregationruleportsmock.NewMockHazmatSegregationRuleRepository(t),
		ShipmentRepo:              shipmentportsmock.NewMockShipmentRepository(t),
	})
}
