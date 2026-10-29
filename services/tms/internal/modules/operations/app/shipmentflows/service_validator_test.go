package shipmentflows

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentvalidator"
	"github.com/emoss08/trenova/internal/modules/operations/operationstest"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/internal/testutil/mocks/commodityportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/hazmatsegregationruleportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
)

func newServiceValidator(t testing.TB) *shipmentvalidator.Validator {
	t.Helper()

	return newServiceValidatorWithAssignmentRepo(t, operationstest.NewAssignmentRepository(t))
}

func newServiceValidatorWithAssignmentRepo(
	t testing.TB,
	assignmentRepo assignmentports.AssignmentRepository,
) *shipmentvalidator.Validator {
	t.Helper()

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

// coverageValidator is the validator with the rate coverage rule reading the
// given billing control.
func coverageValidator(
	t *testing.T,
	billing tenantports.BillingControlRepository,
) *shipmentvalidator.Validator {
	t.Helper()

	return shipmentvalidator.New(shipmentvalidator.Params{
		AssignmentRepo: operationstest.NewAssignmentRepository(t),
		ControlRepo:    operationstest.NewControlRepository(t),
		CustomerRepo:   operationstest.NewCustomerRepository(t),
		CommodityRepo:  commodityportsmock.NewMockCommodityRepository(t),
		HazmatSegregationRuleRepo: hazmatsegregationruleportsmock.NewMockHazmatSegregationRuleRepository(
			t,
		),
		ShipmentRepo: shipmentportsmock.NewMockShipmentRepository(t),
		BillingRepo:  billing,
	})
}
