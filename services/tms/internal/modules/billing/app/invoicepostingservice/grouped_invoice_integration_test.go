//go:build integration

package invoicepostingservice

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicecreationservice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type partialScenarioParams struct {
	tenantInfo pagination.TenantInfo
	actor      *permissionports.RequestActor
	org        seededInvoiceOrg
	customerID pulid.ID
	readyLeg   seededInvoiceShipment
	lateLeg    seededInvoiceShipment
	mockLeg    func(pulid.ID, string, pulid.ID, decimal.Decimal)
}

// runPartialOrderScenario covers partial-order billing: only ready legs are invoiced,
// order-level charges are billed exactly once (first pass), and the follow-up
// single-leg invoice still carries the order correlation.
func runPartialOrderScenario(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	svc *invoicecreationservice.Service,
	orderRepo orderports.OrderRepository,
	p partialScenarioParams,
) {
	t.Helper()

	ord, err := orderRepo.Create(ctx, &order.Order{
		OrganizationID: p.org.ID,
		BusinessUnitID: p.org.BusinessUnitID,
		CustomerID:     p.customerID,
		Status:         order.StatusInProgress,
		OrderNumber:    "O-TEST-0002",
		CurrencyCode:   "USD",
	})
	require.NoError(t, err)

	_, err = orderRepo.AddCharge(ctx, &order.OrderCharge{
		OrganizationID: p.org.ID,
		BusinessUnitID: p.org.BusinessUnitID,
		OrderID:        ord.ID,
		Description:    "Border handling",
		Amount:         decimal.NewFromInt(50),
	})
	require.NoError(t, err)

	legStates := map[pulid.ID]shipmentvocab.Status{
		p.readyLeg.ID: shipmentvocab.StatusReadyToInvoice,
		p.lateLeg.ID:  shipmentvocab.StatusInTransit,
	}
	legFreights := map[pulid.ID]decimal.Decimal{
		p.readyLeg.ID: decimal.NewFromInt(100),
		p.lateLeg.ID:  decimal.NewFromInt(200),
	}
	for legID, status := range legStates {
		setLeg(t, ctx, db, legID, ord.ID, p.customerID, status, legFreights[legID])
	}
	p.mockLeg(p.readyLeg.ID, p.readyLeg.ProNumber, ord.ID, legFreights[p.readyLeg.ID])
	p.mockLeg(p.lateLeg.ID, p.lateLeg.ProNumber, ord.ID, legFreights[p.lateLeg.ID])

	// First pass: only the ready leg is billable, and the order charge rides along.
	first, err := svc.CreateFromOrder(ctx, &invoiceports.CreateInvoiceFromOrderRequest{
		OrderID:    ord.ID,
		TenantInfo: p.tenantInfo,
	}, p.actor)
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(150).Equal(first.TotalAmount),
		"first pass should bill the ready leg plus the order charge, got %s", first.TotalAmount)

	// The charge was stamped to the first invoice inside the same transaction.
	var stampedInvoiceID pulid.ID
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT invoice_id FROM order_charges WHERE order_id = $1", ord.ID,
	).Scan(&stampedInvoiceID))
	assert.Equal(
		t,
		first.ID,
		stampedInvoiceID,
		"order charge should be stamped to the first invoice",
	)

	// The late leg delivers; the ready leg was settled by billing (simulated here since
	// the first invoice is left in Draft).
	_, err = db.Pool().Exec(ctx, "UPDATE shipments SET status = $1 WHERE id = $2",
		string(shipmentvocab.StatusInvoiced), p.readyLeg.ID)
	require.NoError(t, err)
	_, err = db.Pool().Exec(ctx, "UPDATE shipments SET status = $1 WHERE id = $2",
		string(shipmentvocab.StatusReadyToInvoice), p.lateLeg.ID)
	require.NoError(t, err)

	// Second pass through the single-shipment entry point: the invoice carries the
	// order correlation (G9) and must NOT re-bill the order charge.
	secondResult, err := svc.CreateInvoicesFromShipments(
		ctx,
		&invoiceports.CreateInvoiceFromShipmentsRequest{
			ShipmentIDs: []pulid.ID{p.lateLeg.ID},
			TenantInfo:  p.tenantInfo,
		},
		p.actor,
	)
	require.NoError(t, err)
	second := secondResult.Primary
	assert.Equal(t, ord.ID, second.OrderID, "single-leg invoice should keep the order correlation")
	assert.Equal(t, "O-TEST-0002", second.OrderNumber)
	assert.True(
		t,
		decimal.NewFromInt(200).Equal(second.TotalAmount),
		"second pass must bill only the leg — no duplicated order charge, got %s",
		second.TotalAmount,
	)
	for _, line := range second.Lines {
		assert.False(t, line.ShipmentID.IsNil(),
			"second invoice must not carry order-charge lines")
	}
}

func setLeg(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	legID, orderID, customerID pulid.ID,
	status shipmentvocab.Status,
	freight decimal.Decimal,
) {
	t.Helper()

	_, err := db.Pool().Exec(ctx,
		`UPDATE shipments SET order_id = $1, customer_id = $2, status = $3,
			freight_charge_amount = $4, total_charge_amount = $4, other_charge_amount = 0
		WHERE id = $5`,
		orderID, customerID, string(status), freight.String(), legID,
	)
	require.NoError(t, err)
}
