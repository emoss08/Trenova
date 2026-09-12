package invoicerunservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurrentPeriodIsTheOpenWindow(t *testing.T) {
	t.Parallel()

	period, ok := CurrentPeriod(profile(nil), at(t, "America/Denver", "2026-03-15 12:00"))

	require.True(t, ok)
	assert.Equal(t, at(t, "America/Denver", "2026-03-01 00:00"), period.Start)
	assert.Equal(t, at(t, "America/Denver", "2026-04-01 00:00"), period.End)
}

func TestCurrentPeriodIgnoresNonStatementCustomers(t *testing.T) {
	t.Parallel()

	perShipment := profile(func(p *customer.CustomerBillingProfile) {
		p.InvoiceDelivery = customer.InvoiceDeliveryPerShipment
		p.BillingCycle = customer.BillingCycleImmediate
	})

	_, ok := CurrentPeriod(perShipment, at(t, "UTC", "2026-03-15 12:00"))
	assert.False(t, ok)

	_, ok = CurrentPeriod(nil, 0)
	assert.False(t, ok)
}

// An off-cycle bill mid-period must not move the customer's cadence. The window
// still closes on their own boundary; only its start moves up to what was
// already billed, so freight on that invoice does not show as still owing.
func TestCurrentPeriodKeepsTheBoundaryAfterAnOffCycleBill(t *testing.T) {
	t.Parallel()

	billedOn := at(t, "America/Denver", "2026-03-14 09:30")
	p := profile(func(p *customer.CustomerBillingProfile) {
		p.LastBilledPeriodEnd = &billedOn
	})

	period, ok := CurrentPeriod(p, at(t, "America/Denver", "2026-03-15 12:00"))

	require.True(t, ok)
	assert.Equal(t, billedOn, period.Start)
	assert.Equal(
		t,
		at(t, "America/Denver", "2026-04-01 00:00"),
		period.End,
		"billing early must not shift the customer's next boundary",
	)
}

// A watermark from a period that has already closed is older than the current
// boundary and must be ignored, or the open window would reach back over freight
// that was billed last month.
func TestCurrentPeriodIgnoresAStaleWatermark(t *testing.T) {
	t.Parallel()

	lastMonth := at(t, "America/Denver", "2026-03-01 00:00")
	p := profile(func(p *customer.CustomerBillingProfile) {
		p.LastBilledPeriodEnd = &lastMonth
	})

	period, ok := CurrentPeriod(p, at(t, "America/Denver", "2026-04-10 12:00"))

	require.True(t, ok)
	assert.Equal(t, at(t, "America/Denver", "2026-04-01 00:00"), period.Start)
	assert.Equal(t, at(t, "America/Denver", "2026-05-01 00:00"), period.End)
}

func statementCandidate(
	customerID pulid.ID,
	pro string,
	amount string,
	mutate func(*repositories.ConsolidationCandidate),
) *repositories.ConsolidationCandidate {
	candidate := &repositories.ConsolidationCandidate{
		BillingQueueItemID: pulid.MustNew("bqi_"),
		ShipmentID:         pulid.MustNew("shp_"),
		CustomerID:         customerID,
		CustomerName:       "Acme",
		ProNumber:          pro,
		CurrencyCode:       "USD",
		SplitBy:            customer.InvoiceSplitKeyCustomer,
		TotalChargeAmount:  decimal.NewNullDecimal(decimal.RequireFromString(amount)),
	}
	if mutate != nil {
		mutate(candidate)
	}

	return candidate
}

func TestNewStatementGroupSumsItsOwnMembers(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	groups := GroupCandidates([]*repositories.ConsolidationCandidate{
		statementCandidate(customerID, "PRO-1", "125.50", nil),
		statementCandidate(customerID, "PRO-2", "74.50", nil),
	})
	require.Len(t, groups, 1)

	group := newStatementGroup(groups[0], true)

	assert.Equal(t, 2, group.ShipmentCount)
	assert.True(t, decimal.RequireFromString("200").Equal(group.TotalAmount))
	require.Len(t, group.Shipments, 2)
	assert.Equal(t, "PRO-1", group.Shipments[0].ProNumber)
	assert.False(t, group.BelowMinimum, "no minimum configured means nothing is held")
}

func TestNewStatementGroupOmitsShipmentsForTheListView(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	groups := GroupCandidates([]*repositories.ConsolidationCandidate{
		statementCandidate(customerID, "PRO-1", "125.50", nil),
	})

	group := newStatementGroup(groups[0], false)

	assert.Equal(t, 1, group.ShipmentCount)
	assert.Nil(t, group.Shipments)
}

func TestNewStatementGroupFlagsTheCustomerMinimum(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	withMinimum := func(c *repositories.ConsolidationCandidate) {
		c.MinConsolidatedAmount = decimal.NewNullDecimal(decimal.RequireFromString("250"))
	}
	groups := GroupCandidates([]*repositories.ConsolidationCandidate{
		statementCandidate(customerID, "PRO-1", "125.50", withMinimum),
	})

	assert.True(t, newStatementGroup(groups[0], false).BelowMinimum)
}

// A statement with one billable group and one held group still bills. Reporting
// it as below minimum would tell a biller nothing is going out when something is.
func TestSyncStatementTotalsHoldsOnlyWhenEveryGroupIsHeld(t *testing.T) {
	t.Parallel()

	statement := &servicesports.OpenStatement{
		Groups: []*servicesports.StatementGroup{
			{ShipmentCount: 1, TotalAmount: decimal.RequireFromString("100"), BelowMinimum: true},
			{ShipmentCount: 3, TotalAmount: decimal.RequireFromString("900"), BelowMinimum: false},
		},
	}

	syncStatementTotals(statement)

	assert.Equal(t, 2, statement.InvoiceCount)
	assert.Equal(t, 4, statement.ShipmentCount)
	assert.True(t, decimal.RequireFromString("1000").Equal(statement.TotalAmount))
	assert.False(t, statement.BelowMinimum)
}

func TestSyncStatementTotalsHoldsWhenAllGroupsAreHeld(t *testing.T) {
	t.Parallel()

	statement := &servicesports.OpenStatement{
		Groups: []*servicesports.StatementGroup{
			{ShipmentCount: 1, TotalAmount: decimal.RequireFromString("100"), BelowMinimum: true},
		},
	}

	syncStatementTotals(statement)

	assert.True(t, statement.BelowMinimum)
}

// An empty statement is not "below minimum" — there is nothing to hold. Saying
// otherwise would put a hold badge on every customer with a quiet month.
func TestSyncStatementTotalsOnAnEmptyStatement(t *testing.T) {
	t.Parallel()

	statement := &servicesports.OpenStatement{}

	syncStatementTotals(statement)

	assert.Zero(t, statement.ShipmentCount)
	assert.Zero(t, statement.InvoiceCount)
	assert.True(t, decimal.Zero.Equal(statement.TotalAmount))
	assert.False(t, statement.BelowMinimum)
}
