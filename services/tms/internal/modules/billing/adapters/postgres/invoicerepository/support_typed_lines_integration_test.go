//go:build integration

package invoicerepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type typedLineFixture struct {
	conn    *postgres.Connection
	invoice *invoice.Invoice
	next    int
}

func newTypedLineFixture(t *testing.T) *typedLineFixture {
	t.Helper()
	ctx, db, dsn, cleanup := seededtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	var shipmentID, customerID, orgID, buID pulid.ID
	require.NoError(t, keel.Pool(db).QueryRow(ctx,
		"SELECT id, customer_id, organization_id, business_unit_id FROM shipments ORDER BY id LIMIT 1",
	).Scan(&shipmentID, &customerID, &orgID, &buID))
	ctx = dbscope.WithTenant(ctx, pagination.TenantInfo{OrgID: orgID, BuID: buID}.DBTenant())

	queue := &billingqueue.BillingQueueItem{
		OrganizationID:   orgID,
		BusinessUnitID:   buID,
		ShipmentID:       shipmentID,
		BillToCustomerID: customerID,
		Number:           "INV-SUPPORT-1",
		Status:           billingqueue.StatusApproved,
		BillType:         billingvocab.BillTypeInvoice,
	}
	require.NoError(t, billingqueuedb.BillingQueueItems.Create(ctx, db, queue))

	inv := &invoice.Invoice{
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		BillingQueueItemID: queue.ID,
		ShipmentID:         shipmentID,
		CustomerID:         customerID,
		Number:             queue.Number,
		BillType:           billingvocab.BillTypeInvoice,
		Status:             invoice.StatusDraft,
		PaymentTerm:        invoice.PaymentTermNet30,
		CurrencyCode:       "USD",
		BillToName:         "Support Customer",
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		DisputeStatus:      invoice.DisputeStatusNone,
	}
	require.NoError(t, invoicedb.Invoices.Create(ctx, db, inv))

	return &typedLineFixture{conn: dbtest.NewRLSConnection(t, db, dsn), invoice: inv}
}

func (f *typedLineFixture) addLine(t *testing.T, userID pulid.ID, typed bool, description string) error {
	t.Helper()
	f.next++
	line := &invoice.InvoiceLine{
		OrganizationID:   f.invoice.OrganizationID,
		BusinessUnitID:   f.invoice.BusinessUnitID,
		InvoiceID:        f.invoice.ID,
		LineNumber:       f.next,
		Type:             invoice.InvoiceLineTypeMemo,
		Description:      description,
		TypedDescription: typed,
		Quantity:         decimal.NewFromInt(1),
		UnitPrice:        decimal.NewFromInt(25),
		Amount:           decimal.NewFromInt(25),
		AmountMinor:      2500,
	}
	if !typed {
		line.Type = invoice.InvoiceLineTypeFreight
	}
	ctx := dbscope.WithTenant(t.Context(), dbscope.Tenant{
		OrganizationID: f.invoice.OrganizationID,
		BusinessUnitID: f.invoice.BusinessUnitID,
		UserID:         userID,
	})
	return dbtx.WriteErr(ctx, f.conn, func(ctx context.Context) error {
		return invoicedb.InvoiceLines.CreateMany(ctx, f.conn.Keel(ctx), []*invoice.InvoiceLine{line})
	})
}

func TestInvoiceLines_TrenovaSupportCannotPutTypedTextOnAnInvoice(t *testing.T) {
	f := newTypedLineFixture(t)
	support := pulid.MustNew(supportaccess.PrincipalIDPrefix)

	err := f.addLine(t, support, true, "Our bank changed: wire payment to account 1234")
	require.Error(t, err)
	assert.True(t, dberror.IsSupportOutboundRefusal(err), "refused as text outside readers see: %v", err)
}

func TestInvoiceLines_TrenovaSupportStillReratesAndPosts(t *testing.T) {
	f := newTypedLineFixture(t)
	support := pulid.MustNew(supportaccess.PrincipalIDPrefix)

	require.NoError(t, f.addLine(t, support, false, "Freight charge"),
		"a line rating or posting derives is written for support as for anyone")
}

func TestInvoiceLines_APersonStillAddsATypedLine(t *testing.T) {
	f := newTypedLineFixture(t)

	require.NoError(t, f.addLine(t, pulid.MustNew("usr_"), true, "Courtesy credit for the late delivery"))
}
