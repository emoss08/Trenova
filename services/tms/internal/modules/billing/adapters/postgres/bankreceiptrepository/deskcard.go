package bankreceiptrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt/bankreceiptdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem/bankreceiptworkitemdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/shared/pulid"
)

func NewDeskCardReader(p Params) bankreceiptports.DeskCardReader {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.bank-receipt-desk-card")}
}

type cardPaymentRow struct {
	ID                   pulid.ID
	CustomerID           pulid.ID
	CustomerName         string
	PaymentMethod        string
	ReferenceNumber      string
	PaymentDate          int64
	Status               string
	CurrencyCode         string
	AmountMinor          int64
	AppliedAmountMinor   int64
	UnappliedAmountMinor int64
}

type cardApplicationRow struct {
	InvoiceID           pulid.ID
	InvoiceNumber       string
	AppliedAmountMinor  int64
	ShortPayAmountMinor int64
}

type cardWorkItemRow struct {
	ID             pulid.ID
	Status         string
	AssignedTo     string
	AssignedAt     *int64
	ResolutionType string
	ResolutionNote string
}

func orEmpty(e keel.Expr) keel.StringExpr {
	return keel.CastText(e).Coalesce("")
}

func (r *repository) GetDeskCard(
	ctx context.Context,
	req *bankreceiptports.GetDeskCardRequest,
) (*bankreceiptports.DeskCard, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*bankreceiptports.DeskCard, error) {
		receipt, err := bankreceiptdb.BankReceipts.Get(ctx, r.db.Keel(ctx), req.ReceiptID)
		if err != nil {
			return nil, bankReceiptErrors.Map(err, req.ReceiptID.String())
		}
		card := &bankreceiptports.DeskCard{Receipt: receipt, WorkItemShown: req.IncludeWorkItem}
		if receipt.MatchedCustomerPaymentID.IsNotNil() {
			if !req.IncludePayment {
				card.PaymentHidden = true
			} else if card.Payment, err = r.cardPayment(ctx, receipt.MatchedCustomerPaymentID); err != nil {
				return nil, err
			}
		}
		if req.IncludeWorkItem {
			if card.WorkItem, err = r.cardWorkItem(ctx, receipt); err != nil {
				return nil, err
			}
		}

		return card, nil
	})
}

func (r *repository) cardPayment(
	ctx context.Context,
	paymentID pulid.ID,
) (*bankreceiptports.DeskCardPayment, error) {
	cp := customerpaymentdb.Payments
	cus := customerdb.Customers
	payments, err := keel.Into[cardPaymentRow](ctx, r.db.Keel(ctx), cp.
		LeftJoin(cus, cus.ID.EqCol(cp.CustomerID)).
		Pick(
			cp.ID, cp.CustomerID, orEmpty(cus.Name), orEmpty(cp.PaymentMethod),
			orEmpty(cp.ReferenceNumber), cp.PaymentDate, orEmpty(cp.Status),
			orEmpty(cp.CurrencyCode), cp.AmountMinor, cp.AppliedAmountMinor,
			cp.UnappliedAmountMinor,
		).
		Where(cp.ID.Eq(paymentID)))
	if err != nil || len(payments) == 0 {
		return nil, err
	}

	cpa := customerpaymentdb.Applications
	inv := invoicedb.Invoices
	applications, err := keel.Into[cardApplicationRow](ctx, r.db.Keel(ctx), cpa.
		LeftJoin(inv, inv.ID.EqCol(cpa.InvoiceID)).
		Pick(cpa.InvoiceID, orEmpty(inv.Number), cpa.AppliedAmountMinor, cpa.ShortPayAmountMinor).
		Where(cpa.CustomerPaymentID.Eq(paymentID)).
		Order(cpa.LineNumber.Asc()))
	if err != nil {
		return nil, err
	}

	row := payments[0]
	payment := &bankreceiptports.DeskCardPayment{
		ID:                   row.ID,
		CustomerID:           row.CustomerID,
		CustomerName:         row.CustomerName,
		PaymentMethod:        row.PaymentMethod,
		ReferenceNumber:      row.ReferenceNumber,
		PaymentDate:          row.PaymentDate,
		Status:               row.Status,
		CurrencyCode:         row.CurrencyCode,
		AmountMinor:          row.AmountMinor,
		AppliedAmountMinor:   row.AppliedAmountMinor,
		UnappliedAmountMinor: row.UnappliedAmountMinor,
		Applications:         make([]bankreceiptports.DeskCardApplication, 0, len(applications)),
	}
	for _, application := range applications {
		payment.Applications = append(
			payment.Applications,
			bankreceiptports.DeskCardApplication(application),
		)
	}

	return payment, nil
}

func (r *repository) cardWorkItem(
	ctx context.Context,
	receipt *bankreceipt.BankReceipt,
) (*bankreceiptports.DeskCardWorkItem, error) {
	wi := bankreceiptworkitemdb.WorkItems
	usr := tenantdb.Users
	items, err := keel.Into[cardWorkItemRow](ctx, r.db.Keel(ctx), wi.
		LeftJoin(usr, usr.ID.EqCol(wi.AssignedToUserID)).
		Pick(
			wi.ID, orEmpty(wi.Status), orEmpty(usr.Name), wi.AssignedAt,
			orEmpty(wi.ResolutionType), orEmpty(wi.ResolutionNote),
		).
		Where(wi.BankReceiptID.Eq(receipt.ID)).
		Order(wi.CreatedAt.Desc()).
		Limit(1))
	if err != nil || len(items) == 0 {
		return nil, err
	}
	item := bankreceiptports.DeskCardWorkItem(items[0])

	return &item, nil
}
