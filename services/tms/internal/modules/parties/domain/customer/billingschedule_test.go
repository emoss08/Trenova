package customer

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scheduleErrors(t *testing.T, mutate func(*CustomerBillingProfile)) map[string]bool {
	t.Helper()

	profile := NewDefaultBillingProfile("org_1", "bu_1", "cus_1")
	profile.BillingCurrency = "USD"
	profile.InvoiceCopies = 1
	mutate(profile)

	multiErr := errortypes.NewMultiError()
	profile.Validate(multiErr)

	fields := make(map[string]bool)
	for _, e := range multiErr.Errors {
		fields[e.Field] = true
	}

	return fields
}

func TestBillingScheduleDefaultsAreValid(t *testing.T) {
	t.Parallel()

	assert.Empty(t, scheduleErrors(t, func(*CustomerBillingProfile) {}))
}

func TestBillingScheduleRejectsContradictoryCadence(t *testing.T) {
	t.Parallel()

	// The old model allowed both of these at once, which is two different answers
	// to the same question.
	consolidatedImmediate := scheduleErrors(t, func(p *CustomerBillingProfile) {
		p.InvoiceDelivery = InvoiceDeliveryConsolidated
		p.BillingCycle = BillingCycleImmediate
	})
	assert.True(t, consolidatedImmediate["billingCycle"])

	perShipmentMonthly := scheduleErrors(t, func(p *CustomerBillingProfile) {
		p.InvoiceDelivery = InvoiceDeliveryPerShipment
		p.BillingCycle = BillingCycleMonthly
	})
	assert.True(t, perShipmentMonthly["invoiceDelivery"])

	consolidatedMonthly := scheduleErrors(t, func(p *CustomerBillingProfile) {
		p.InvoiceDelivery = InvoiceDeliveryConsolidated
		p.BillingCycle = BillingCycleMonthly
	})
	assert.Empty(t, consolidatedMonthly)
}

func TestBillingScheduleAnchorDayBounds(t *testing.T) {
	t.Parallel()

	// The anchor means a weekday on the weekly cycles.
	weeklyOutOfRange := scheduleErrors(t, func(p *CustomerBillingProfile) {
		p.InvoiceDelivery = InvoiceDeliveryConsolidated
		p.BillingCycle = BillingCycleWeekly
		p.BillingCycleAnchorDay = 7
	})
	assert.True(t, weeklyOutOfRange["billingCycleAnchorDay"])

	weeklyValid := scheduleErrors(t, func(p *CustomerBillingProfile) {
		p.InvoiceDelivery = InvoiceDeliveryConsolidated
		p.BillingCycle = BillingCycleWeekly
		p.BillingCycleAnchorDay = 0
	})
	assert.Empty(t, weeklyValid)

	// ...and a day of the month on the rest, capped at 28 so February never
	// silently shifts the boundary.
	monthlyTooLate := scheduleErrors(t, func(p *CustomerBillingProfile) {
		p.InvoiceDelivery = InvoiceDeliveryConsolidated
		p.BillingCycle = BillingCycleMonthly
		p.BillingCycleAnchorDay = 29
	})
	assert.True(t, monthlyTooLate["billingCycleAnchorDay"])

	monthlyZero := scheduleErrors(t, func(p *CustomerBillingProfile) {
		p.InvoiceDelivery = InvoiceDeliveryConsolidated
		p.BillingCycle = BillingCycleMonthly
		p.BillingCycleAnchorDay = 0
	})
	assert.True(t, monthlyZero["billingCycleAnchorDay"])
}

func TestBillingScheduleRejectsUnknownTimezone(t *testing.T) {
	t.Parallel()

	bad := scheduleErrors(t, func(p *CustomerBillingProfile) {
		p.BillingCycleTimezone = "Mars/Olympus_Mons"
	})
	assert.True(t, bad["billingCycleTimezone"])

	good := scheduleErrors(t, func(p *CustomerBillingProfile) {
		p.BillingCycleTimezone = "America/Denver"
	})
	assert.Empty(t, good)
}

func TestBillingScheduleShipmentCap(t *testing.T) {
	t.Parallel()

	// Zero means unbounded rather than "never bill anything".
	unbounded := scheduleErrors(t, func(p *CustomerBillingProfile) {
		p.MaxShipmentsPerInvoice = 0
	})
	assert.Empty(t, unbounded)

	tooMany := scheduleErrors(t, func(p *CustomerBillingProfile) {
		p.MaxShipmentsPerInvoice = 501
	})
	assert.True(t, tooMany["maxShipmentsPerInvoice"])
}

func TestBillingCycleIsPeriodic(t *testing.T) {
	t.Parallel()

	assert.False(t, BillingCycle("").IsPeriodic())
	assert.False(t, BillingCycleImmediate.IsPeriodic())
	for _, cycle := range []BillingCycle{
		BillingCycleDaily,
		BillingCycleWeekly,
		BillingCycleBiWeekly,
		BillingCycleSemiMonthly,
		BillingCycleMonthly,
		BillingCycleQuarterly,
	} {
		require.True(t, cycle.IsPeriodic(), string(cycle))
	}
}

func TestInvoiceSplitKeyHasNoDivision(t *testing.T) {
	t.Parallel()

	// Division named an entity that does not exist anywhere in the domain, so it
	// must not come back through the new enum.
	assert.False(t, InvoiceSplitKey("Division").IsValid())
	assert.False(t, InvoiceSplitKey("CustomerAndDivision").IsValid())
	assert.True(t, InvoiceSplitKeyCustomer.IsValid())
}
