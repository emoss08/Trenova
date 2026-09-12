package invoicerunservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func ruleItem(amount string) *invoicerun.InvoiceRunGroupItem {
	return &invoicerun.InvoiceRunGroupItem{
		BillingQueueItemID: pulid.MustNew("bqi_"),
		ShipmentID:         pulid.MustNew("shp_"),
		Amount:             decimal.RequireFromString(amount),
	}
}

func TestBelowMinimumDefersATrivialStatement(t *testing.T) {
	t.Parallel()

	group := &invoicerun.InvoiceRunGroup{
		MinimumAmount: decimal.NewNullDecimal(decimal.RequireFromString("250.00")),
	}
	items := []*invoicerun.InvoiceRunGroupItem{ruleItem("40.00"), ruleItem("60.00")}

	reason := belowMinimum(group, items)
	assert.NotEmpty(t, reason)
	assert.Contains(t, reason, "250.00")
	// The wording has to tell a biller the freight is not lost.
	assert.Contains(t, reason, "next period")
}

func TestBelowMinimumBillsAtTheFloor(t *testing.T) {
	t.Parallel()

	group := &invoicerun.InvoiceRunGroup{
		MinimumAmount: decimal.NewNullDecimal(decimal.RequireFromString("100.00")),
	}

	// Exactly at the floor bills; a minimum is a floor, not a threshold to clear.
	assert.Empty(t, belowMinimum(group, []*invoicerun.InvoiceRunGroupItem{ruleItem("100.00")}))
	assert.Empty(t, belowMinimum(group, []*invoicerun.InvoiceRunGroupItem{ruleItem("100.01")}))
	assert.NotEmpty(t, belowMinimum(group, []*invoicerun.InvoiceRunGroupItem{ruleItem("99.99")}))
}

func TestBelowMinimumIgnoredWhenUnset(t *testing.T) {
	t.Parallel()

	// No floor configured means every amount bills, including a very small one.
	group := &invoicerun.InvoiceRunGroup{}
	assert.Empty(t, belowMinimum(group, []*invoicerun.InvoiceRunGroupItem{ruleItem("0.01")}))
}

func TestRunIsAutoBillableOnlyWhenEveryGroupOptedIn(t *testing.T) {
	t.Parallel()

	pending := func(autoBill bool) *invoicerun.InvoiceRunGroup {
		return &invoicerun.InvoiceRunGroup{
			Status:   invoicerun.GroupStatusPending,
			AutoBill: autoBill,
		}
	}

	assert.True(t, runIsAutoBillable(&invoicerun.InvoiceRun{
		Groups: []*invoicerun.InvoiceRunGroup{pending(true), pending(true)},
	}))

	// One customer who wants review holds the whole run back, rather than the run
	// half-committing and leaving a biller looking at numbers that no longer match
	// what it billed.
	assert.False(t, runIsAutoBillable(&invoicerun.InvoiceRun{
		Groups: []*invoicerun.InvoiceRunGroup{pending(true), pending(false)},
	}))

	// A group already settled does not veto the rest.
	assert.True(t, runIsAutoBillable(&invoicerun.InvoiceRun{
		Groups: []*invoicerun.InvoiceRunGroup{
			pending(true),
			{Status: invoicerun.GroupStatusCommitted},
		},
	}))

	assert.False(t, runIsAutoBillable(&invoicerun.InvoiceRun{}))
}
