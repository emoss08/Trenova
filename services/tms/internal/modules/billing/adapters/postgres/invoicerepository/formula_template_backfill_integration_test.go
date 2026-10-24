//go:build integration

package invoicerepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/schema/migrations"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const formulaTemplateBackfill = "20261231009470_invoice_line_formula_template"

type backfillTemplate struct {
	ID   pulid.ID
	Name string
}

func TestFormulaTemplateBackfillNamesTheTemplateEachFreightLineWasRatedWith(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	var shipmentID, customerID, orgID, buID pulid.ID
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT id, customer_id, organization_id, business_unit_id FROM shipments ORDER BY id LIMIT 1",
	).
		Scan(&shipmentID, &customerID, &orgID, &buID))
	ctx = dbscope.WithTenant(ctx, pagination.TenantInfo{OrgID: orgID, BuID: buID}.DBTenant())

	rated := backfillTemplate{ID: pulid.MustNew("ft_"), Name: "Per mile – backfill rated"}
	assigned := backfillTemplate{ID: pulid.MustNew("ft_"), Name: "Per CWT – backfill assigned"}
	for _, tmpl := range []backfillTemplate{rated, assigned} {
		_, err := db.Pool().Exec(ctx,
			`INSERT INTO "formula_templates" ("id", "organization_id", "business_unit_id", "name", "source")
			VALUES ($1, $2, $3, $4, '')`,
			tmpl.ID, orgID, buID, tmpl.Name)
		require.NoError(t, err)
	}

	_, err := db.Pool().Exec(ctx,
		`UPDATE shipments SET formula_template_id = $1,
			rating_detail = jsonb_build_object('formulaTemplateId', $2::text, 'formulaTemplateName', $3::text)
		WHERE id = $4 AND organization_id = $5 AND business_unit_id = $6`,
		assigned.ID, rated.ID, " "+rated.Name+" ", shipmentID, orgID, buID)
	require.NoError(t, err)

	queue := &billingqueue.BillingQueueItem{
		OrganizationID:   orgID,
		BusinessUnitID:   buID,
		ShipmentID:       shipmentID,
		BillToCustomerID: customerID,
		Number:           "INV-BACKFILL-1",
		Status:           billingqueue.StatusApproved,
		BillType:         billingqueue.BillTypeInvoice,
	}
	require.NoError(t, billingqueuedb.BillingQueueItems.Create(ctx, db, queue))

	inv := &invoice.Invoice{
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		BillingQueueItemID: queue.ID,
		ShipmentID:         shipmentID,
		CustomerID:         customerID,
		Number:             queue.Number,
		BillType:           billingqueue.BillTypeInvoice,
		Status:             invoice.StatusDraft,
		PaymentTerm:        invoice.PaymentTermNet30,
		CurrencyCode:       "USD",
		BillToName:         "Backfill Customer",
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		DisputeStatus:      invoice.DisputeStatusNone,
	}
	require.NoError(t, invoicedb.Invoices.Create(ctx, db, inv))

	alreadyNamed := pulid.MustNew("ft_")
	line := func(number int, kind invoice.InvoiceLineType, name string) *invoice.InvoiceLine {
		return &invoice.InvoiceLine{
			OrganizationID:      orgID,
			BusinessUnitID:      buID,
			InvoiceID:           inv.ID,
			ShipmentID:          shipmentID,
			LineNumber:          number,
			Type:                kind,
			Description:         "Freight charge",
			Quantity:            decimal.NewFromInt(1),
			UnitPrice:           decimal.NewFromInt(100),
			Amount:              decimal.NewFromInt(100),
			AmountMinor:         10000,
			FormulaTemplateName: name,
		}
	}
	byRating := line(1, invoice.InvoiceLineTypeFreight, rated.Name)
	byAssignment := line(2, invoice.InvoiceLineTypeFreight, assigned.Name)
	byNeither := line(3, invoice.InvoiceLineTypeFreight, "A template since deleted")
	accessorial := line(4, invoice.InvoiceLineTypeAccessorial, rated.Name)
	kept := line(5, invoice.InvoiceLineTypeFreight, rated.Name)
	kept.FormulaTemplateID = alreadyNamed
	lines := []*invoice.InvoiceLine{byRating, byAssignment, byNeither, accessorial, kept}
	require.NoError(t, invoicedb.InvoiceLines.CreateMany(ctx, db, lines))

	up, err := migrations.File(formulaTemplateBackfill)
	require.NoError(t, err)
	_, err = db.Pool().Exec(ctx, up)
	require.NoError(t, err)

	got := make(map[pulid.ID]pulid.ID, len(lines))
	il := invoicedb.InvoiceLines
	stored, err := il.Pick(il.ID, il.FormulaTemplateID).Where(il.InvoiceID.Eq(inv.ID)).All(ctx, db)
	require.NoError(t, err)
	for _, row := range stored {
		got[row.ID] = row.FormulaTemplateID
	}

	assert.Equal(t, rated.ID, got[byRating.ID], "the template the shipment's rating names")
	assert.Equal(
		t,
		assigned.ID,
		got[byAssignment.ID],
		"the shipment's template when it has the line's name",
	)
	assert.True(t, got[byNeither.ID].IsNil(), "a name neither template has is not guessed")
	assert.True(t, got[accessorial.ID].IsNil(), "accessorial lines are not rated by a template")
	assert.Equal(t, alreadyNamed, got[kept.ID], "a line that already names its template keeps it")
}
