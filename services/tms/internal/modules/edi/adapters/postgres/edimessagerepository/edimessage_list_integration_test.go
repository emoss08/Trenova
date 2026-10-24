//go:build integration

package edimessagerepository_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edimessagerepository"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/editest"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type messageListFixture struct {
	repo    ediports.EDIMessageRepository
	tenant  pagination.TenantInfo
	ids     map[string]pulid.ID
	now     int64
	shipID  pulid.ID
	invID   pulid.ID
	partner pulid.ID
}

type listedMessage struct {
	name       string
	ack        edi.MessageAcknowledgmentStatus
	set        edi.TransactionSet
	onShipment bool
	onInvoice  bool
	age        int64
}

func seedMessageListFixture(t *testing.T) (context.Context, *messageListFixture) {
	t.Helper()

	ctx, db := seededDevelopmentDB(t)

	var shipmentID, customerID, orgID, buID string
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT id, customer_id, organization_id, business_unit_id FROM shipments ORDER BY id LIMIT 1",
	).Scan(&shipmentID, &customerID, &orgID, &buID))
	shipment := struct{ ID, CustomerID pulid.ID }{ID: pulid.ID(shipmentID), CustomerID: pulid.ID(customerID)}
	scope := editest.TenantOf(ctx, pagination.TenantInfo{OrgID: pulid.ID(orgID), BuID: pulid.ID(buID)})
	ctx = scope.Ctx
	org := scope.Info
	documentType := findDocumentType(t, ctx, db, edi.TransactionSet204, edi.DocumentDirectionOutbound)

	fixture := &messageListFixture{
		repo: edimessagerepository.New(edimessagerepository.Params{
			DB:     postgres.NewTestConnection(db),
			Logger: zap.NewNop(),
		}),
		tenant: org,
		ids:    make(map[string]pulid.ID),
		now:    timeutils.NowUnix(),
		shipID: shipment.ID,
	}

	partner := &edi.EDIPartner{
		BusinessUnitID: org.BuID,
		OrganizationID: org.OrgID,
		Kind:           edi.PartnerKindExternal,
		Code:           "LIST-FILTERS",
		Name:           "List Filters Partner",
	}
	require.NoError(t, create(ctx, db, partner))
	fixture.partner = partner.ID

	queue := &billingqueue.BillingQueueItem{
		OrganizationID:   org.OrgID,
		BusinessUnitID:   org.BuID,
		ShipmentID:       shipment.ID,
		BillToCustomerID: shipment.CustomerID,
		Number:           "INV-EDI-LIST-1",
		Status:           billingqueue.StatusPosted,
		BillType:         billingqueue.BillTypeInvoice,
	}
	require.NoError(t, create(ctx, db, queue))
	inv := &invoice.Invoice{
		OrganizationID:     org.OrgID,
		BusinessUnitID:     org.BuID,
		BillingQueueItemID: queue.ID,
		ShipmentID:         shipment.ID,
		CustomerID:         shipment.CustomerID,
		Number:             queue.Number,
		BillType:           billingqueue.BillTypeInvoice,
		Status:             invoice.StatusPosted,
		PaymentTerm:        invoice.PaymentTermNet30,
		CurrencyCode:       "USD",
		InvoiceDate:        fixture.now,
		BillToName:         "List Filters Customer",
		SubtotalAmount:     decimal.NewFromInt(100),
		OtherAmount:        decimal.Zero,
		TotalAmount:        decimal.NewFromInt(100),
		AppliedAmount:      decimal.Zero,
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		DisputeStatus:      invoice.DisputeStatusNone,
	}
	require.NoError(t, create(ctx, db, inv))
	fixture.invID = inv.ID

	messages := []listedMessage{
		{name: "pendingNewest", ack: edi.MessageAcknowledgmentStatusPending,
			set: edi.TransactionSet210, onShipment: true, onInvoice: true, age: 60},
		{name: "pendingHour", ack: edi.MessageAcknowledgmentStatusPending,
			set: edi.TransactionSet214, onShipment: true, age: 60 * 60},
		{name: "acceptedInvoice", ack: edi.MessageAcknowledgmentStatusAccepted,
			set: edi.TransactionSet210, onInvoice: true, age: 2 * 60 * 60},
		{name: "acceptedWithErrors", ack: edi.MessageAcknowledgmentStatusAcceptedWithErrors,
			set: edi.TransactionSet214, onShipment: true, age: 3 * 60 * 60},
		{name: "rejectedOld", ack: edi.MessageAcknowledgmentStatusRejected,
			set: edi.TransactionSet214, age: 2 * 24 * 60 * 60},
	}
	for index, entry := range messages {
		message := &edi.EDIMessage{
			BusinessUnitID:           org.BuID,
			OrganizationID:           org.OrgID,
			EDIPartnerID:             partner.ID,
			DocumentTypeID:           documentType.ID,
			Direction:                edi.DocumentDirectionOutbound,
			Standard:                 edi.EDIStandardX12,
			TransactionSet:           entry.set,
			X12Version:               edi.DefaultX12204Version,
			Status:                   edi.MessageStatusGenerated,
			ValidationMode:           edi.ValidationModeDisabled,
			RawX12:                   "ISA*TEST~",
			InterchangeControlNumber: fmt.Sprintf("%09d", index+1),
			GroupControlNumber:       fmt.Sprintf("%d", index+1),
			TransactionControlNumber: fmt.Sprintf("%04d", index+1),
			DeliveryStatus:           edi.MessageDeliveryStatusSent,
			AckStatus:                entry.ack,
			GeneratedAt:              fixture.now - entry.age,
		}
		if entry.onShipment {
			message.ShipmentID = shipment.ID
		}
		if entry.onInvoice {
			message.InvoiceID = inv.ID
		}
		require.NoError(t, create(ctx, db, message), entry.name)
		editest.Exec(t, ctx, db, "UPDATE edi_messages SET created_at = $1 WHERE id = $2",
			fixture.now-entry.age, message.ID.String())
		fixture.ids[entry.name] = message.ID
	}

	return ctx, fixture
}

func (f *messageListFixture) list(
	ctx context.Context,
	t *testing.T,
	configure func(req *ediports.ListEDIMessagesRequest),
) []pulid.ID {
	t.Helper()

	cursor, err := pagination.NewCursorInfo(50, "")
	require.NoError(t, err)
	req := &ediports.ListEDIMessagesRequest{
		Filter:    &pagination.QueryOptions{TenantInfo: f.tenant},
		Cursor:    cursor,
		PartnerID: f.partner,
	}
	configure(req)

	result, err := f.repo.ListMessagesCursor(ctx, req)
	require.NoError(t, err)

	ids := make([]pulid.ID, 0, len(result.Items))
	for _, item := range result.Items {
		ids = append(ids, item.ID)
	}

	return ids
}

func (f *messageListFixture) named(names ...string) []pulid.ID {
	ids := make([]pulid.ID, 0, len(names))
	for _, name := range names {
		ids = append(ids, f.ids[name])
	}

	return ids
}

func TestListMessagesCursor_FiltersOnAcknowledgmentRecordsAndCreationTime(t *testing.T) {
	ctx, f := seedMessageListFixture(t)

	cases := []struct {
		name      string
		configure func(req *ediports.ListEDIMessagesRequest)
		want      []string
	}{
		{
			name:      "no new filter keeps every message newest first",
			configure: func(*ediports.ListEDIMessagesRequest) {},
			want: []string{
				"pendingNewest", "pendingHour", "acceptedInvoice",
				"acceptedWithErrors", "rejectedOld",
			},
		},
		{
			name: "ack status pending",
			configure: func(req *ediports.ListEDIMessagesRequest) {
				req.AckStatus = edi.MessageAcknowledgmentStatusPending
			},
			want: []string{"pendingNewest", "pendingHour"},
		},
		{
			name: "ack status accepted leaves out accepted with errors",
			configure: func(req *ediports.ListEDIMessagesRequest) {
				req.AckStatus = edi.MessageAcknowledgmentStatusAccepted
			},
			want: []string{"acceptedInvoice"},
		},
		{
			name: "ack status accepted with errors",
			configure: func(req *ediports.ListEDIMessagesRequest) {
				req.AckStatus = edi.MessageAcknowledgmentStatusAcceptedWithErrors
			},
			want: []string{"acceptedWithErrors"},
		},
		{
			name: "shipment",
			configure: func(req *ediports.ListEDIMessagesRequest) {
				req.ShipmentID = f.shipID
			},
			want: []string{"pendingNewest", "pendingHour", "acceptedWithErrors"},
		},
		{
			name: "invoice",
			configure: func(req *ediports.ListEDIMessagesRequest) {
				req.InvoiceID = f.invID
			},
			want: []string{"pendingNewest", "acceptedInvoice"},
		},
		{
			name: "created after is inclusive",
			configure: func(req *ediports.ListEDIMessagesRequest) {
				req.CreatedAfter = f.now - 2*60*60
			},
			want: []string{"pendingNewest", "pendingHour", "acceptedInvoice"},
		},
		{
			name: "created before is exclusive",
			configure: func(req *ediports.ListEDIMessagesRequest) {
				req.CreatedBefore = f.now - 2*60*60
			},
			want: []string{"acceptedWithErrors", "rejectedOld"},
		},
		{
			name: "a window",
			configure: func(req *ediports.ListEDIMessagesRequest) {
				req.CreatedAfter = f.now - 4*60*60
				req.CreatedBefore = f.now - 30*60
			},
			want: []string{"pendingHour", "acceptedInvoice", "acceptedWithErrors"},
		},
		{
			name: "shipment, ack status and a window together",
			configure: func(req *ediports.ListEDIMessagesRequest) {
				req.ShipmentID = f.shipID
				req.AckStatus = edi.MessageAcknowledgmentStatusPending
				req.CreatedAfter = f.now - 30*60
			},
			want: []string{"pendingNewest"},
		},
		{
			name: "invoice with the transaction set and delivery filters it already had",
			configure: func(req *ediports.ListEDIMessagesRequest) {
				req.InvoiceID = f.invID
				req.TransactionSet = edi.TransactionSet210
				req.AckStatus = edi.MessageAcknowledgmentStatusAccepted
			},
			want: []string{"acceptedInvoice"},
		},
		{
			name: "nothing matches",
			configure: func(req *ediports.ListEDIMessagesRequest) {
				req.ShipmentID = f.shipID
				req.AckStatus = edi.MessageAcknowledgmentStatusRejected
			},
			want: []string{},
		},
		{
			name: "a window with nothing in it",
			configure: func(req *ediports.ListEDIMessagesRequest) {
				req.CreatedAfter = f.now + 60
			},
			want: []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, f.named(tc.want...), f.list(ctx, t, tc.configure))
		})
	}
}

func TestListMessagesCursor_PagesThroughAFilteredListInOrder(t *testing.T) {
	ctx, f := seedMessageListFixture(t)

	first, err := pagination.NewCursorInfo(2, "")
	require.NoError(t, err)
	req := &ediports.ListEDIMessagesRequest{
		Filter:     &pagination.QueryOptions{TenantInfo: f.tenant},
		Cursor:     first,
		PartnerID:  f.partner,
		ShipmentID: f.shipID,
	}
	page, err := f.repo.ListMessagesCursor(ctx, req)
	require.NoError(t, err)
	require.True(t, page.HasNextPage)
	require.Len(t, page.Items, 2)
	require.Equal(t, f.ids["pendingNewest"], page.Items[0].ID)
	require.Equal(t, f.ids["pendingHour"], page.Items[1].ID)

	next, err := page.NextCursor()
	require.NoError(t, err)
	second, err := pagination.NewCursorInfo(2, next)
	require.NoError(t, err)
	req.Cursor = second
	page, err = f.repo.ListMessagesCursor(ctx, req)
	require.NoError(t, err)
	require.False(t, page.HasNextPage)
	require.Len(t, page.Items, 1)
	require.Equal(t, f.ids["acceptedWithErrors"], page.Items[0].ID)
}

func TestListMessages_AppliesTheSameFilters(t *testing.T) {
	ctx, f := seedMessageListFixture(t)

	result, err := f.repo.ListMessages(ctx, &ediports.ListEDIMessagesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: f.tenant,
			Pagination: pagination.Info{Limit: 20},
		},
		PartnerID:    f.partner,
		ShipmentID:   f.shipID,
		AckStatus:    edi.MessageAcknowledgmentStatusAcceptedWithErrors,
		CreatedAfter: f.now - 4*60*60,
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Equal(t, f.ids["acceptedWithErrors"], result.Items[0].ID)
}
