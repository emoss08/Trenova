package invoiceledger

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerledger"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/services/journalposting"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

var ErrNoLedgerEntry = errors.New("no ledger entry for this invoice")

type ControlReader interface {
	GetByOrgID(ctx context.Context, orgID pulid.ID) (*tenant.AccountingControl, error)
}

type Policy interface {
	CanCreateInvoiceLedgerEntry(
		control *tenant.AccountingControl,
		event tenant.JournalSourceEventType,
	) bool
}

type LedgerAppender interface {
	AppendEntries(ctx context.Context, entries []*customerledger.CustomerLedgerEntry) error
}

type Poster struct {
	Controls ControlReader
	Policy   Policy
	Writer   journalposting.Writer
	Ledger   LedgerAppender
}

type Request struct {
	Invoice          *invoice.Invoice
	ActorID          pulid.ID
	AccountingDate   int64
	Now              int64
	RelatedInvoiceID pulid.ID
	LedgerOnly       bool
}

type Plan struct {
	Event        tenant.JournalSourceEventType
	Journal      *journalposting.Plan
	LedgerAmount int64
	LedgerDate   int64
	write        *journalposting.WriteRequest
}

type Result struct {
	Event          tenant.JournalSourceEventType
	Journal        *journalposting.WriteResult
	LedgerAmount   int64
	LedgerRecorded bool
}

func SourceEvent(billType billingqueue.BillType) tenant.JournalSourceEventType {
	switch billType { //nolint:exhaustive // every other bill type posts as an invoice
	case billingqueue.BillTypeCreditMemo:
		return tenant.JournalSourceEventCreditMemoPosted
	case billingqueue.BillTypeDebitMemo:
		return tenant.JournalSourceEventDebitMemoPosted
	default:
		return tenant.JournalSourceEventInvoicePosted
	}
}

func HasRequiredAccounts(control *tenant.AccountingControl) bool {
	return control != nil && control.DefaultARAccountID.IsNotNil() &&
		control.DefaultRevenueAccountID.IsNotNil()
}

func CreditMemoRequest(
	kind invoiceadjustment.Kind,
	creditMemo *invoice.Invoice,
	sourceInvoiceID pulid.ID,
	actorID pulid.ID,
) *Request {
	creditMemo.SyncMinorAmounts()
	return &Request{
		Invoice:          creditMemo,
		ActorID:          actorID,
		AccountingDate:   creditMemo.InvoiceDate,
		RelatedInvoiceID: sourceInvoiceID,
		LedgerOnly:       kind == invoiceadjustment.KindWriteOff,
	}
}

func IdempotencyKey(invoiceID pulid.ID) string {
	return "invoice-posted:" + invoiceID.String()
}

func (p *Poster) Plan(ctx context.Context, req *Request) (*Plan, error) {
	entity := req.Invoice
	if entity == nil || req.ActorID.IsNil() || p.Controls == nil || p.Policy == nil {
		return nil, ErrNoLedgerEntry
	}

	control, err := p.Controls.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, ErrNoLedgerEntry
		}
		return nil, err
	}
	event := SourceEvent(entity.BillType)
	if !p.Policy.CanCreateInvoiceLedgerEntry(control, event) {
		return nil, ErrNoLedgerEntry
	}

	amount := entity.TotalAmountMinor
	if amount < 0 {
		amount = -amount
	}
	if amount == 0 {
		return nil, ErrNoLedgerEntry
	}

	plan := &Plan{
		Event:        event,
		LedgerAmount: ledgerAmount(entity.BillType, entity.TotalAmountMinor, amount),
		LedgerDate:   req.accountingDate(),
	}
	if req.LedgerOnly {
		return plan, nil
	}

	if !HasRequiredAccounts(control) {
		return nil, errortypes.NewValidationError(
			"accountingControl",
			errortypes.ErrRequired,
			"Invoice posting requires default Accounts Receivable and revenue accounts",
		)
	}

	plan.write = journalRequest(req, control, event, amount)
	if plan.Journal, err = p.Writer.Plan(ctx, plan.write); err != nil {
		return nil, err
	}
	plan.LedgerDate = plan.Journal.AccountingDate

	return plan, nil
}

func (p *Poster) Post(ctx context.Context, req *Request) (*Result, error) {
	plan, err := p.Plan(ctx, req)
	if err != nil {
		return nil, err
	}

	result := &Result{Event: plan.Event, LedgerAmount: plan.LedgerAmount}
	if plan.write != nil {
		if result.Journal, err = p.Writer.WritePlan(ctx, plan.write, plan.Journal); err != nil {
			return nil, err
		}
		plan.LedgerDate = result.Journal.AccountingDate
	}
	if p.Ledger == nil {
		return result, nil
	}

	entity := req.Invoice
	if err = p.Ledger.AppendEntries(ctx, []*customerledger.CustomerLedgerEntry{{
		ID:               pulid.MustNew("cledg_"),
		OrganizationID:   entity.OrganizationID,
		BusinessUnitID:   entity.BusinessUnitID,
		CustomerID:       entity.CustomerID,
		SourceObjectType: "Invoice",
		SourceObjectID:   entity.ID.String(),
		SourceEventType:  plan.Event.String(),
		RelatedInvoiceID: req.RelatedInvoiceID,
		DocumentNumber:   entity.Number,
		TransactionDate:  plan.LedgerDate,
		LineNumber:       1,
		AmountMinor:      plan.LedgerAmount,
		CreatedByID:      req.ActorID,
	}}); err != nil {
		return nil, err
	}
	result.LedgerRecorded = true

	return result, nil
}

func journalRequest(
	req *Request,
	control *tenant.AccountingControl,
	event tenant.JournalSourceEventType,
	amount int64,
) *journalposting.WriteRequest {
	entity := req.Invoice
	debitAccountID := control.DefaultARAccountID
	creditAccountID := control.DefaultRevenueAccountID
	if entity.BillType == billingqueue.BillTypeCreditMemo {
		debitAccountID, creditAccountID = creditAccountID, debitAccountID
	}
	description := fmt.Sprintf("Invoice posted for %s", entity.Number)

	return &journalposting.WriteRequest{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		Control:        control,
		ActorID:        req.ActorID,
		AccountingDate: req.accountingDate(),
		Now:            req.now(),
		Subject:        "invoice",
		DateField:      "postedAt",
		Description:    description,
		AutoGenerated:  true,
		Source: journalposting.Source{
			ObjectType:     "Invoice",
			ObjectID:       entity.ID,
			DocumentNumber: entity.Number,
			Event:          event,
			IdempotencyKey: IdempotencyKey(entity.ID),
		},
		Lines: []journalposting.Line{
			{AccountID: debitAccountID, Debit: amount, CustomerID: entity.CustomerID},
			{AccountID: creditAccountID, Credit: amount, CustomerID: entity.CustomerID},
		},
	}
}

func ledgerAmount(billType billingqueue.BillType, total, amount int64) int64 {
	switch billType { //nolint:exhaustive // other bill types keep the document's own sign
	case billingqueue.BillTypeCreditMemo:
		return -amount
	case billingqueue.BillTypeDebitMemo:
		return amount
	default:
		return total
	}
}

func (req *Request) accountingDate() int64 {
	if req.AccountingDate != 0 {
		return req.AccountingDate
	}
	if req.Invoice.PostedAt != nil {
		return *req.Invoice.PostedAt
	}
	return req.now()
}

func (req *Request) now() int64 {
	if req.Now != 0 {
		return req.Now
	}
	if req.Invoice.PostedAt != nil {
		return *req.Invoice.PostedAt
	}
	return 0
}
