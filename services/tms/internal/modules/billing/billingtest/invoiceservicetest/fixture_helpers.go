package invoiceservicetest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
)

// fakeInvoiceDB runs the transaction body directly; repositories are mocked so
// no connection is ever touched.
type FakeTx struct{}

func (FakeTx) Transact(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	return fn(ctx)
}

type Fixture struct {
	OrgID, BuID, UserID, InvoiceID, QueueItemID pulid.ID
	TenantInfo                                  pagination.TenantInfo
	Actor                                       *permissionports.RequestActor
}

func NewFixture() Fixture {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	return Fixture{
		OrgID:       orgID,
		BuID:        buID,
		UserID:      userID,
		InvoiceID:   pulid.MustNew("inv_"),
		QueueItemID: pulid.MustNew("bqi_"),
		TenantInfo:  pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		Actor:       testutil.NewSessionActor(userID, orgID, buID),
	}
}

func (f Fixture) Invoice(status invoice.Status) *invoice.Invoice {
	return &invoice.Invoice{
		ID:                 f.InvoiceID,
		OrganizationID:     f.OrgID,
		BusinessUnitID:     f.BuID,
		BillingQueueItemID: f.QueueItemID,
		ShipmentID:         pulid.MustNew("shp_"),
		CustomerID:         pulid.MustNew("cus_"),
		Scope:              invoice.ScopeShipment,
		Number:             "INV-2001",
		BillType:           billingqueue.BillTypeInvoice,
		Status:             status,
		PaymentTerm:        invoice.PaymentTermNet30,
		CurrencyCode:       "USD",
		InvoiceDate:        1_700_000_000,
		BillToName:         "Acme Logistics",
		SubtotalAmount:     decimal.NewFromInt(100),
		TotalAmount:        decimal.NewFromInt(100),
		TotalAmountMinor:   10000,
		Lines: []*invoice.InvoiceLine{{
			LineNumber:  1,
			Type:        invoice.InvoiceLineTypeFreight,
			Description: "Linehaul",
			Quantity:    decimal.NewFromInt(1),
			UnitPrice:   decimal.NewFromInt(100),
			Amount:      decimal.NewFromInt(100),
		}},
	}
}

func (f Fixture) Request(
	disposition invoice.VoidDisposition,
) *invoiceports.VoidInvoiceRequest {
	return &invoiceports.VoidInvoiceRequest{
		InvoiceID:   f.InvoiceID,
		TenantInfo:  f.TenantInfo,
		Reason:      "Billed the wrong customer",
		Disposition: disposition,
	}
}

func (f Fixture) AuditActor() kernelports.AuditActor {
	return kernelports.AuditActor{
		PrincipalType: f.Actor.PrincipalType,
		PrincipalID:   f.UserID,
		UserID:        f.UserID,
	}
}

func (f Fixture) DiscardRequest() *invoiceports.DiscardDraftForReviewRequest {
	return &invoiceports.DiscardDraftForReviewRequest{
		BillingQueueItemID: f.QueueItemID,
		TenantInfo:         f.TenantInfo,
	}
}

func (f Fixture) ExpectLookup(
	repo *invoiceportsmock.MockInvoiceRepository,
	found *invoice.Invoice,
	err error,
) {
	repo.EXPECT().
		GetByBillingQueueItemID(mock.Anything, invoiceports.GetInvoiceByBillingQueueItemIDRequest{
			BillingQueueItemID: f.QueueItemID,
			TenantInfo:         f.TenantInfo,
		}).
		Return(found, err).
		Once()
}
