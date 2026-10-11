package invoicerun

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func item(amount string, excluded bool) *InvoiceRunGroupItem {
	it := &InvoiceRunGroupItem{
		BillingQueueItemID: pulid.MustNew("bqi_"),
		ShipmentID:         pulid.MustNew("shp_"),
		Amount:             decimal.RequireFromString(amount),
		Excluded:           excluded,
	}
	if excluded {
		it.ExclusionReason = "Operator removed it"
	}

	return it
}

func TestCanTransition(t *testing.T) {
	t.Parallel()

	assert.True(t, CanTransition(StatusBuilding, StatusReady))
	assert.True(t, CanTransition(StatusReady, StatusCommitting))
	assert.True(t, CanTransition(StatusCommitting, StatusCommitted))
	assert.True(t, CanTransition(StatusCommitting, StatusFailed))
	assert.True(t, CanTransition(StatusReady, StatusCanceled))

	// A retry re-applying the status it already holds is not an error; that is
	// what makes the commit idempotent.
	assert.True(t, CanTransition(StatusCommitting, StatusCommitting))

	// Nothing leaves a terminal status. A failed run may hold invoices already,
	// so re-committing it would bill them twice.
	assert.False(t, CanTransition(StatusFailed, StatusCommitting))
	assert.False(t, CanTransition(StatusCommitted, StatusReady))
	assert.False(t, CanTransition(StatusCanceled, StatusReady))

	// Building cannot skip review.
	assert.False(t, CanTransition(StatusBuilding, StatusCommitting))
}

func TestGroupSyncTotalsIgnoresExcludedItems(t *testing.T) {
	t.Parallel()

	group := &InvoiceRunGroup{
		Items: []*InvoiceRunGroupItem{
			item("100.00", false),
			item("250.50", false),
			item("999.99", true),
		},
	}
	group.SyncTotals()

	assert.Equal(t, 2, group.ItemCount)
	assert.True(t, group.TotalAmount.Equal(decimal.RequireFromString("350.50")))
	assert.Equal(t, int64(35050), group.TotalAmountMinor)
	assert.Equal(t, 1, group.ExcludedCount())
}

func TestRunSyncTotalsSkipsSkippedGroups(t *testing.T) {
	t.Parallel()

	run := &InvoiceRun{
		Groups: []*InvoiceRunGroup{
			{Items: []*InvoiceRunGroupItem{item("100.00", false)}},
			{
				Status:     GroupStatusSkipped,
				SkipReason: "Below the customer minimum",
				Items:      []*InvoiceRunGroupItem{item("5.00", false)},
			},
			{
				Items: []*InvoiceRunGroupItem{item("200.00", false), item("1.00", true)},
			},
		},
	}
	run.SyncTotals()

	// A skipped group bills nothing, so it contributes neither a group nor money.
	assert.Equal(t, 2, run.GroupCount)
	assert.Equal(t, 2, run.ItemCount)
	assert.True(t, run.TotalAmount.Equal(decimal.RequireFromString("300.00")))
	assert.Equal(t, int64(30000), run.TotalAmountMinor)
	// Excluded items are still counted, across every group, so the operator can
	// see what was pulled.
	assert.Equal(t, 1, run.ExcludedCount)
}

func runErrors(t *testing.T, mutate func(*InvoiceRun)) map[string]bool {
	t.Helper()

	run := &InvoiceRun{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Number:         "IRUN-2601-00001",
		Status:         StatusBuilding,
		Source:         SourceManual,
		PeriodStart:    1_700_000_000,
		PeriodEnd:      1_700_086_400,
		InvoiceDate:    1_700_086_400,
		CurrencyCode:   "USD",
	}
	mutate(run)

	multiErr := errortypes.NewMultiError()
	run.Validate(multiErr)

	fields := make(map[string]bool)
	for _, e := range multiErr.Errors {
		fields[e.Field] = true
	}

	return fields
}

func TestInvoiceRunValidate(t *testing.T) {
	t.Parallel()

	assert.Empty(t, runErrors(t, func(*InvoiceRun) {}))

	inverted := runErrors(t, func(r *InvoiceRun) {
		r.PeriodEnd = r.PeriodStart - 1
	})
	assert.True(t, inverted["periodEnd"])

	// A scheduled run is keyed on its cycle, so two workers racing the same cron
	// tick collide on the unique index instead of both building the period.
	scheduledNoCycle := runErrors(t, func(r *InvoiceRun) {
		r.Source = SourceScheduled
	})
	assert.True(t, scheduledNoCycle["cycle"])

	scheduledWithCycle := runErrors(t, func(r *InvoiceRun) {
		r.Source = SourceScheduled
		r.Cycle = "Monthly"
	})
	assert.Empty(t, scheduledWithCycle)
}

func TestGroupValidateRequiresReasons(t *testing.T) {
	t.Parallel()

	validate := func(g *InvoiceRunGroup) map[string]bool {
		multiErr := errortypes.NewMultiError()
		g.Validate(multiErr)
		fields := make(map[string]bool)
		for _, e := range multiErr.Errors {
			fields[e.Field] = true
		}

		return fields
	}

	base := func() *InvoiceRunGroup {
		return &InvoiceRunGroup{
			CustomerID: pulid.MustNew("cus_"),
			GroupKey:   "cus_1",
			GroupLabel: "Acme Foods",
			Status:     GroupStatusPending,
		}
	}

	require.Empty(t, validate(base()))

	// A skipped group is a decision somebody has to be able to explain later.
	skipped := base()
	skipped.Status = GroupStatusSkipped
	assert.True(t, validate(skipped)["skipReason"])

	committed := base()
	committed.Status = GroupStatusCommitted
	assert.True(t, validate(committed)["invoiceId"])
}

func TestGroupItemExclusionNeedsReason(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	it := item("10.00", true)
	it.ExclusionReason = ""
	it.Validate(multiErr)

	found := false
	for _, e := range multiErr.Errors {
		if e.Field == "exclusionReason" {
			found = true
		}
	}
	assert.True(t, found)
}
