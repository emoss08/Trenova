package invoicerunservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	custA = pulid.MustNew("cus_")
	custB = pulid.MustNew("cus_")
)

func candidate(mutate func(*repositories.ConsolidationCandidate)) *repositories.ConsolidationCandidate {
	c := &repositories.ConsolidationCandidate{
		CustomerID:   custA,
		CustomerName: "Acme Foods",
	}
	if mutate != nil {
		mutate(c)
	}

	return c
}

func TestGroupKeyCustomerNeverSpansCustomers(t *testing.T) {
	t.Parallel()

	a := GroupKeyFor(customer.InvoiceSplitKeyCustomer, candidate(nil))
	b := GroupKeyFor(customer.InvoiceSplitKeyCustomer, candidate(func(c *repositories.ConsolidationCandidate) {
		c.CustomerID = custB
	}))

	assert.NotEqual(t, a.Key, b.Key)
	assert.Equal(t, "Acme Foods", a.Label)
}

func TestGroupKeyPONumber(t *testing.T) {
	t.Parallel()

	withPO := func(po string) GroupKeyResult {
		return GroupKeyFor(
			customer.InvoiceSplitKeyCustomerAndPONumber,
			candidate(func(c *repositories.ConsolidationCandidate) { c.OrderPONumber = po }),
		)
	}

	// Case and surrounding space are not a different PO.
	assert.Equal(t, withPO("po-1").Key, withPO("  PO-1 ").Key)
	assert.Equal(t, "PO PO-1", withPO("PO-1").Label)

	// A blank PO gets its own bucket rather than joining the first real one.
	blank := withPO("")
	assert.NotEqual(t, withPO("PO-1").Key, blank.Key)
	assert.Equal(t, "No PO", blank.Label)

	// ...and cannot be forged by a customer literally using that text.
	assert.NotEqual(t, blank.Key, withPO("No PO").Key)
}

func TestGroupKeySeparatorCannotBeForged(t *testing.T) {
	t.Parallel()

	// With a printable separator, "PO-1" + "|2" and "PO-1|2" would collide.
	split := GroupKeyFor(
		customer.InvoiceSplitKeyCustomerAndPONumber,
		candidate(func(c *repositories.ConsolidationCandidate) { c.OrderPONumber = "PO-1|2" }),
	)
	plain := GroupKeyFor(
		customer.InvoiceSplitKeyCustomerAndPONumber,
		candidate(func(c *repositories.ConsolidationCandidate) { c.OrderPONumber = "PO-1" }),
	)
	assert.NotEqual(t, split.Key, plain.Key)
}

func TestGroupKeyBOLAndLocations(t *testing.T) {
	t.Parallel()

	bol := GroupKeyFor(
		customer.InvoiceSplitKeyCustomerAndShipmentBOL,
		candidate(func(c *repositories.ConsolidationCandidate) { c.ShipmentBOL = "BOL-7" }),
	)
	assert.Equal(t, "BOL BOL-7", bol.Label)

	noBOL := GroupKeyFor(customer.InvoiceSplitKeyCustomerAndShipmentBOL, candidate(nil))
	assert.Equal(t, "No BOL", noBOL.Label)

	origin := GroupKeyFor(
		customer.InvoiceSplitKeyCustomerAndOrigin,
		candidate(func(c *repositories.ConsolidationCandidate) { c.OriginKey = "loc_1" }),
	)
	dest := GroupKeyFor(
		customer.InvoiceSplitKeyCustomerAndDestination,
		candidate(func(c *repositories.ConsolidationCandidate) { c.DestinationKey = "loc_1" }),
	)
	// Same location, different question — these must not share a key.
	assert.NotEqual(t, origin.Label, dest.Label)

	unknown := GroupKeyFor(customer.InvoiceSplitKeyCustomerAndDestination, candidate(nil))
	assert.Equal(t, "Unknown destination", unknown.Label)
}

func TestGroupKeyOrder(t *testing.T) {
	t.Parallel()

	orderID := pulid.MustNew("ord_")
	withOrder := GroupKeyFor(
		customer.InvoiceSplitKeyCustomerAndOrder,
		candidate(func(c *repositories.ConsolidationCandidate) {
			c.OrderID = orderID
			c.OrderNumber = "ORD-9"
		}),
	)
	assert.Equal(t, "ORD-9", withOrder.Label)

	// A shipment with no order is its own group, not merged with a real order.
	orphan := GroupKeyFor(
		customer.InvoiceSplitKeyCustomerAndOrder,
		candidate(func(c *repositories.ConsolidationCandidate) {
			c.ShipmentID = pulid.MustNew("shp_")
			c.ProNumber = "PRO-1"
		}),
	)
	assert.NotEqual(t, withOrder.Key, orphan.Key)
}

// A shipment booked without an order is its own commercial unit, so under a
// one-invoice-per-order split it is billed on its own.
//
// The previous fixture built a single order-less shipment, so it could never see
// that every order-less shipment shared one "No order" group — and a customer
// who asked for one invoice per order got all their standalone freight merged
// onto a single invoice. That is the refusal the grouping rules exist to avoid.
func TestOrderSplitNeverMergesStandaloneShipments(t *testing.T) {
	t.Parallel()

	first := GroupKeyFor(
		customer.InvoiceSplitKeyCustomerAndOrder,
		candidate(func(c *repositories.ConsolidationCandidate) {
			c.ShipmentID = pulid.MustNew("shp_")
			c.ProNumber = "PRO-1001"
		}),
	)
	second := GroupKeyFor(
		customer.InvoiceSplitKeyCustomerAndOrder,
		candidate(func(c *repositories.ConsolidationCandidate) {
			c.ShipmentID = pulid.MustNew("shp_")
			c.ProNumber = "PRO-1002"
		}),
	)

	assert.NotEqual(t, first.Key, second.Key, "two unrelated shipments must not share an invoice")
	assert.Equal(t, "PRO-1001", first.Label, "a standalone shipment is named by its pro number")
	assert.Equal(t, "PRO-1002", second.Label)
}

// The same shipment must land in the same group on every read, or a statement
// re-read between preview and commit would propose different invoices.
func TestStandaloneShipmentGroupIsStable(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	build := func() GroupKeyResult {
		return GroupKeyFor(
			customer.InvoiceSplitKeyCustomerAndOrder,
			candidate(func(c *repositories.ConsolidationCandidate) {
				c.ShipmentID = shipmentID
				c.ProNumber = "PRO-1001"
			}),
		)
	}

	assert.Equal(t, build(), build())
}

// A standalone shipment with no pro number still needs a readable name; an empty
// label would render a blank invoice card.
func TestStandaloneShipmentWithoutAProNumberStillHasALabel(t *testing.T) {
	t.Parallel()

	result := GroupKeyFor(
		customer.InvoiceSplitKeyCustomerAndOrder,
		candidate(func(c *repositories.ConsolidationCandidate) {
			c.ShipmentID = pulid.MustNew("shp_")
		}),
	)

	assert.NotEmpty(t, result.Label)
}

func TestSplitOversizedIsDeterministic(t *testing.T) {
	t.Parallel()

	members := []int{1, 2, 3, 4, 5, 6, 7}

	assert.Len(t, SplitOversized(members, 0), 1, "zero means unbounded")
	assert.Len(t, SplitOversized(members, 10), 1)

	parts := SplitOversized(members, 3)
	require.Len(t, parts, 3)
	assert.Equal(t, []int{1, 2, 3}, parts[0])
	assert.Equal(t, []int{4, 5, 6}, parts[1])
	assert.Equal(t, []int{7}, parts[2])

	// The same input must split the same way on a re-preview.
	assert.Equal(t, parts, SplitOversized(members, 3))
}

func TestPartLabelAndKey(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "PO PO-1", PartLabel("PO PO-1", 1, 1))
	assert.Equal(t, "PO PO-1 (2 of 3)", PartLabel("PO PO-1", 2, 3))

	assert.Equal(t, "k", PartKey("k", 1, 1))
	assert.NotEqual(t, PartKey("k", 1, 3), PartKey("k", 2, 3))
}
