//go:build integration

package invoicerepository_test

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoicerepository"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestReplacingAnInvoicesAttachmentsKeepsTheTurnsOfTheDocumentsThatStay(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	var shipmentID, customerID, orgID, buID, userID pulid.ID
	require.NoError(t, keel.Pool(db).QueryRow(ctx,
		`SELECT s.id, s.customer_id, s.organization_id, s.business_unit_id, u.id
		FROM shipments s JOIN users u ON u.current_organization_id = s.organization_id
		ORDER BY s.id LIMIT 1`,
	).Scan(&shipmentID, &customerID, &orgID, &buID, &userID))
	info := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}
	ctx = dbscope.WithTenant(ctx, info.DBTenant())

	queue := &billingqueue.BillingQueueItem{
		OrganizationID:   orgID,
		BusinessUnitID:   buID,
		ShipmentID:       shipmentID,
		BillToCustomerID: customerID,
		Number:           "INV-TURNS-1",
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
		BillToName:         "Turns Customer",
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		DisputeStatus:      invoice.DisputeStatusNone,
	}
	require.NoError(t, invoicedb.Invoices.Create(ctx, db, inv))

	stored := func(name string) *document.Document {
		doc := &document.Document{
			ID:               pulid.MustNew("doc_"),
			OrganizationID:   orgID,
			BusinessUnitID:   buID,
			LineageID:        pulid.MustNew("dln_"),
			IsCurrentVersion: true,
			FileName:         name,
			OriginalName:     name,
			FileSize:         128,
			FileType:         "application/pdf",
			StoragePath:      "test/" + name,
			Status:           document.StatusActive,
			ResourceID:       shipmentID.String(),
			ResourceType:     "shipment",
			UploadedByID:     userID,
		}
		require.NoError(t, documentdb.Documents.Create(ctx, db, doc))
		return doc
	}
	pod, bol, receipt := stored("pod.pdf"), stored("bol.pdf"), stored("receipt.pdf")
	turns := []billingvocab.PageRotation{{Page: 2, Rotation: 90}}
	require.NoError(t, invoicedb.Attachments.CreateMany(ctx, db, []*invoice.Attachment{
		{
			OrganizationID: orgID, BusinessUnitID: buID, InvoiceID: inv.ID,
			DocumentID: pod.ID, Selected: true, SortOrder: 1, PageRotations: turns,
		},
		{
			OrganizationID: orgID, BusinessUnitID: buID, InvoiceID: inv.ID,
			DocumentID: bol.ID, Selected: true, SortOrder: 2,
			PageRotations: []billingvocab.PageRotation{{Page: 1, Rotation: 180}},
		},
	}))

	repo := invoicerepository.New(invoicerepository.Params{
		DB:     postgres.WrapDB(db),
		Logger: zap.NewNop(),
	})
	got, err := repo.UpsertAttachments(ctx, invoiceports.UpsertInvoiceAttachmentsRequest{
		InvoiceID:      inv.ID,
		DocumentIDs:    []pulid.ID{receipt.ID, pod.ID},
		OrganizationID: orgID,
		BusinessUnitID: buID,
		TenantInfo:     info,
	})

	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, receipt.ID, got[0].DocumentID)
	assert.Empty(t, got[0].PageRotations)
	assert.Equal(t, pod.ID, got[1].DocumentID)
	assert.Equal(t, turns, got[1].PageRotations)
}
