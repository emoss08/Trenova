package invoice

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scopeErrors(t *testing.T, entity *Invoice) map[string]bool {
	t.Helper()

	multiErr := errortypes.NewMultiError()
	entity.validateScope(multiErr)

	fields := make(map[string]bool)
	for _, e := range multiErr.Errors {
		fields[e.Field] = true
	}

	return fields
}

func TestValidateScopeShipment(t *testing.T) {
	t.Parallel()

	entity := &Invoice{Scope: ScopeShipment, ShipmentID: pulid.MustNew("shp_")}
	assert.Empty(t, scopeErrors(t, entity))

	missing := &Invoice{Scope: ScopeShipment}
	assert.True(t, scopeErrors(t, missing)["shipmentId"])
}

func TestValidateScopeOrder(t *testing.T) {
	t.Parallel()

	entity := &Invoice{Scope: ScopeOrder, OrderID: pulid.MustNew("ord_")}
	assert.Empty(t, scopeErrors(t, entity))

	// An order invoice carries no header shipment, so the shipment rule must not
	// fire for it.
	missing := &Invoice{Scope: ScopeOrder}
	fields := scopeErrors(t, missing)
	assert.True(t, fields["orderId"])
	assert.False(t, fields["shipmentId"])
}

func TestValidateScopeConsolidated(t *testing.T) {
	t.Parallel()

	start := int64(1_700_000_000)
	end := start + 30*86_400

	// A consolidated invoice has neither a header shipment nor a header order;
	// its attributed lines and its period are what make it valid.
	valid := &Invoice{
		Scope:       ScopeConsolidated,
		PeriodStart: &start,
		PeriodEnd:   &end,
		Lines: []*InvoiceLine{
			{ShipmentID: pulid.MustNew("shp_")},
		},
	}
	assert.Empty(t, scopeErrors(t, valid))

	noLines := &Invoice{Scope: ScopeConsolidated, PeriodStart: &start, PeriodEnd: &end}
	assert.True(t, scopeErrors(t, noLines)["lines"])

	noPeriod := &Invoice{
		Scope: ScopeConsolidated,
		Lines: []*InvoiceLine{{ShipmentID: pulid.MustNew("shp_")}},
	}
	assert.True(t, scopeErrors(t, noPeriod)["periodStart"])

	// Lines with no shipment attribution do not count as billed shipments.
	unattributed := &Invoice{
		Scope:       ScopeConsolidated,
		PeriodStart: &start,
		PeriodEnd:   &end,
		Lines:       []*InvoiceLine{{Description: "Customs brokerage"}},
	}
	assert.True(t, scopeErrors(t, unattributed)["lines"])
}

func TestValidateScopeAdjustment(t *testing.T) {
	t.Parallel()

	// An adjustment of a consolidated invoice has no shipment and no order, so
	// only its place in a correction chain identifies it.
	viaGroup := &Invoice{Scope: ScopeAdjustment, CorrectionGroupID: pulid.MustNew("icg_")}
	assert.Empty(t, scopeErrors(t, viaGroup))

	viaSupersedes := &Invoice{Scope: ScopeAdjustment, SupersedesInvoiceID: pulid.MustNew("inv_")}
	assert.Empty(t, scopeErrors(t, viaSupersedes))

	orphan := &Invoice{Scope: ScopeAdjustment}
	assert.True(t, scopeErrors(t, orphan)["correctionGroupId"])
}

func TestScopeIsValid(t *testing.T) {
	t.Parallel()

	for _, scope := range []Scope{
		ScopeShipment,
		ScopeOrder,
		ScopeConsolidated,
		ScopeAdjustment,
	} {
		require.True(t, scope.IsValid(), string(scope))
	}

	assert.False(t, Scope("").IsValid())
	assert.False(t, Scope("Statement").IsValid())
}
