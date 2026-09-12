package invoiceservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func statementCustomer(mutate func(*customer.CustomerBillingProfile)) *customer.Customer {
	p := &customer.CustomerBillingProfile{
		InvoiceDelivery:       customer.InvoiceDeliveryConsolidated,
		BillingCycle:          customer.BillingCycleMonthly,
		BillingCycleAnchorDay: 1,
	}
	if mutate != nil {
		mutate(p)
	}

	return &customer.Customer{Name: "Acme Freight", BillingProfile: p}
}

func TestGuardStatementCadenceDemandsAReason(t *testing.T) {
	t.Parallel()

	err := guardStatementCadence(statementCustomer(nil), "")

	require.Error(t, err)

	var validationErr *errortypes.Error
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(
		t,
		"offCycleReason",
		validationErr.Field,
		"the client keys off this field to ask for a reason instead of showing a dead end",
	)
	assert.Equal(t, errortypes.ErrRequired, validationErr.Code)
	assert.Contains(t, validationErr.Error(), "Acme Freight")
	assert.Contains(t, validationErr.Error(), "monthly")
}

func TestGuardStatementCadenceAllowsADeliberateDeviation(t *testing.T) {
	t.Parallel()

	assert.NoError(
		t,
		guardStatementCadence(statementCustomer(nil), "Customer is closing their books early"),
	)
}

func TestGuardStatementCadenceIgnoresPerShipmentCustomers(t *testing.T) {
	t.Parallel()

	perShipment := statementCustomer(func(p *customer.CustomerBillingProfile) {
		p.InvoiceDelivery = customer.InvoiceDeliveryPerShipment
		p.BillingCycle = customer.BillingCycleImmediate
	})

	assert.NoError(t, guardStatementCadence(perShipment, ""))
	assert.NoError(t, guardStatementCadence(&customer.Customer{Name: "No profile"}, ""))
	assert.NoError(t, guardStatementCadence(nil, ""))
}

// Consolidated delivery on an Immediate cycle has no period to accumulate into,
// so there is no statement to take freight off. The profile's own validation
// makes it unreachable; the guard must not invent a warning for it anyway.
func TestGuardStatementCadenceIgnoresConsolidatedWithoutACycle(t *testing.T) {
	t.Parallel()

	noCycle := statementCustomer(func(p *customer.CustomerBillingProfile) {
		p.BillingCycle = customer.BillingCycleImmediate
	})

	assert.NoError(t, guardStatementCadence(noCycle, ""))
}

// A reason typed for a customer who is not on a statement is noise, and stamping
// it would put an "off cycle" marker on an ordinary invoice.
func TestOffCycleReasonOnlyStampsStatementCustomers(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Books close Friday", offCycleReasonFor(
		statementCustomer(nil),
		"Books close Friday",
	))

	perShipment := statementCustomer(func(p *customer.CustomerBillingProfile) {
		p.InvoiceDelivery = customer.InvoiceDeliveryPerShipment
		p.BillingCycle = customer.BillingCycleImmediate
	})
	assert.Empty(t, offCycleReasonFor(perShipment, "Books close Friday"))
	assert.Empty(t, offCycleReasonFor(nil, "Books close Friday"))
}
