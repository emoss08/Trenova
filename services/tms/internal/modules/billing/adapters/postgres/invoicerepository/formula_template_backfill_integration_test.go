//go:build integration

package invoicerepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/schema/migrations"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const formulaTemplateBackfill = "20261231009470_invoice_line_formula_template"

type backfillTenant struct {
	ID             pulid.ID `bun:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

type backfillTemplate struct {
	ID   pulid.ID `bun:"id"`
	Name string   `bun:"name"`
}

type backfillShipment struct {
	ID             pulid.ID `bun:"id"`
	CustomerID     pulid.ID `bun:"customer_id"`
	OrganizationID pulid.ID `bun:"organization_id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

func TestFormulaTemplateBackfillNamesTheTemplateEachFreightLineWasRatedWith(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(
		db,
		seedRegistry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	)
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	var shp backfillShipment
	require.NoError(t, db.NewSelect().
		Table("shipments").
		Column("id", "customer_id", "organization_id", "business_unit_id").
		Limit(1).
		Scan(ctx, &shp))
	org := backfillTenant{ID: shp.OrganizationID, BusinessUnitID: shp.BusinessUnitID}

	rated := backfillTemplate{ID: pulid.MustNew("ft_"), Name: "Per mile – backfill rated"}
	assigned := backfillTemplate{ID: pulid.MustNew("ft_"), Name: "Per CWT – backfill assigned"}
	for _, tmpl := range []backfillTemplate{rated, assigned} {
		_, err = db.ExecContext(ctx,
			`INSERT INTO "formula_templates" ("id", "organization_id", "business_unit_id", "name", "source")
			VALUES (?, ?, ?, ?, '')`,
			tmpl.ID, org.ID, org.BusinessUnitID, tmpl.Name)
		require.NoError(t, err)
	}

	_, err = db.NewUpdate().
		Table("shipments").
		Set("formula_template_id = ?", assigned.ID).
		Set("rating_detail = jsonb_build_object('formulaTemplateId', ?::text, 'formulaTemplateName', ?::text)",
			rated.ID, " "+rated.Name+" ").
		Where("id = ?", shp.ID).
		Where("organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Exec(ctx)
	require.NoError(t, err)

	queue := &billingqueue.BillingQueueItem{
		OrganizationID:   org.ID,
		BusinessUnitID:   org.BusinessUnitID,
		ShipmentID:       shp.ID,
		BillToCustomerID: shp.CustomerID,
		Number:           "INV-BACKFILL-1",
		Status:           billingqueue.StatusApproved,
		BillType:         billingqueue.BillTypeInvoice,
	}
	_, err = db.NewInsert().Model(queue).Exec(ctx)
	require.NoError(t, err)

	inv := &invoice.Invoice{
		OrganizationID:     org.ID,
		BusinessUnitID:     org.BusinessUnitID,
		BillingQueueItemID: queue.ID,
		ShipmentID:         shp.ID,
		CustomerID:         shp.CustomerID,
		Number:             queue.Number,
		BillType:           billingqueue.BillTypeInvoice,
		Status:             invoice.StatusDraft,
		PaymentTerm:        invoice.PaymentTermNet30,
		CurrencyCode:       "USD",
		BillToName:         "Backfill Customer",
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		DisputeStatus:      invoice.DisputeStatusNone,
	}
	_, err = db.NewInsert().Model(inv).Exec(ctx)
	require.NoError(t, err)

	alreadyNamed := pulid.MustNew("ft_")
	line := func(number int, kind invoice.InvoiceLineType, name string) *invoice.InvoiceLine {
		return &invoice.InvoiceLine{
			OrganizationID:      org.ID,
			BusinessUnitID:      org.BusinessUnitID,
			InvoiceID:           inv.ID,
			ShipmentID:          shp.ID,
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
	_, err = db.NewInsert().Model(&lines).Exec(ctx)
	require.NoError(t, err)

	up, err := migrations.File(formulaTemplateBackfill)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, up)
	require.NoError(t, err)

	got := make(map[pulid.ID]pulid.ID, len(lines))
	var stored []invoice.InvoiceLine
	require.NoError(t, db.NewSelect().
		Model(&stored).
		Column("id", "formula_template_id").
		Where("invoice_id = ?", inv.ID).
		Where("organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Scan(ctx))
	for _, row := range stored {
		got[row.ID] = row.FormulaTemplateID
	}

	assert.Equal(t, rated.ID, got[byRating.ID], "the template the shipment's rating names")
	assert.Equal(t, assigned.ID, got[byAssignment.ID], "the shipment's template when it has the line's name")
	assert.True(t, got[byNeither.ID].IsNil(), "a name neither template has is not guessed")
	assert.True(t, got[accessorial.ID].IsNil(), "accessorial lines are not rated by a template")
	assert.Equal(t, alreadyNamed, got[kept.ID], "a line that already names its template keeps it")
}
