package xerodocsync

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/xerosession"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/emoss08/trenova/shared/xerodocs"
	"github.com/emoss08/trenova/shared/xeroerrors"
	"github.com/emoss08/trenova/shared/xeropay"
	"github.com/emoss08/trenova/shared/xeroref"
	"github.com/shopspring/decimal"
)

type Documents struct {
	*xerosession.Session
}

const (
	appBaseURL          = "https://go.xero.com/app/"
	salesViewPath       = "/invoicing/view/"
	billsViewPath       = "/bills/view/"
	debitMemoNotePrefix = "Debit memo"
	defaultLineText     = "Charges"
	idempotencyWindow   = 6 * time.Minute
	moneyPlaces         = 2
	RefCreditNote       = "creditNote"
)

func (c *Documents) DocumentLimits() accountingconnectorports.AccountingDocumentLimits {
	return accountingconnectorports.AccountingDocumentLimits{
		MaxDocNumberLength:      xerodocs.MaxDocumentNumberLength,
		SupportsDebitMemo:       false,
		CanVoidCreditMemo:       true,
		CanVoidPurchaseDocument: true,
		IdempotencyWindow:       idempotencyWindow,
	}
}

func (c *Documents) DocumentURL(
	auth accountingconnectorports.AccountingDocumentAuth,
	link accountingconnectorports.AccountingDocumentLink,
) string {
	kind := link.Kind
	code := strings.TrimSpace(auth.CompanyCode)
	id := strings.TrimSpace(link.ExternalID)
	if code == "" || id == "" {
		return ""
	}
	var path string
	switch {
	case kind.IsSalesDocument():
		path = salesViewPath
	case kind.IsBill():
		path = billsViewPath
	default:
		return ""
	}
	return appBaseURL + url.PathEscape(code) + path + url.PathEscape(id)
}

func (c *Documents) UpsertCustomer(
	ctx context.Context,
	doc *accountingconnectorports.AccountingCustomerDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	return c.upsertContact(ctx, &contactWrite{
		auth:       doc.Auth,
		requestID:  doc.RequestID,
		externalID: doc.ExternalID,
		party:      &doc.Party,
	})
}

type contactWrite struct {
	auth       accountingconnectorports.AccountingDocumentAuth
	requestID  string
	externalID string
	party      *accountingconnectorports.AccountingPartyDraft
}

func (c *Documents) upsertContact(
	ctx context.Context,
	write *contactWrite,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	client, err := c.Client(write.auth)
	if err != nil {
		return nil, err
	}
	input := xerosession.ContactInputOf(write.party)

	var saved *xeroref.Contact
	if id := strings.TrimSpace(write.externalID); id != "" {
		saved, err = xeroref.UpdateContact(ctx, client, write.requestID, id, input)
	} else {
		saved, err = xeroref.CreateContact(ctx, client, write.requestID, input)
	}
	if err != nil {
		return nil, err
	}
	return &accountingconnectorports.AccountingDocumentResult{
		ExternalID: saved.ContactID,
		DocNumber:  saved.Name,
		Refs:       map[string]string{},
	}, nil
}

type salesKind struct {
	credit bool
}

func salesKindOf(kind accountingsync.SyncObjectType) (salesKind, error) {
	switch kind {
	case accountingsync.SyncObjectInvoice, accountingsync.SyncObjectDebitMemo:
		return salesKind{}, nil
	case accountingsync.SyncObjectCreditMemo:
		return salesKind{credit: true}, nil
	case accountingsync.SyncObjectCustomer,
		accountingsync.SyncObjectCustomerPayment,
		accountingsync.SyncObjectCreditApplication,
		accountingsync.SyncObjectCarrierVendor,
		accountingsync.SyncObjectDriverVendor,
		accountingsync.SyncObjectCarrierBill,
		accountingsync.SyncObjectCarrierBillPay,
		accountingsync.SyncObjectDriverBill,
		accountingsync.SyncObjectDriverBillPay,
		accountingsync.SyncObjectJournalEntry,
		accountingsync.SyncObjectJournalSummary,
		accountingsync.DriftObjectGLAccount:
		return salesKind{}, xerosession.ErrDocumentKind
	default:
		return salesKind{}, xerosession.ErrDocumentKind
	}
}

type documentBody struct {
	contactID    string
	number       string
	reference    string
	date         time.Time
	dueDate      time.Time
	currencyCode string
	currencyRate decimal.Decimal
	lines        []xerodocs.LineItem
}

func (b *documentBody) invoice(kind string) xerodocs.InvoiceInput {
	return xerodocs.InvoiceInput{
		Type:            kind,
		ContactID:       b.contactID,
		InvoiceNumber:   b.number,
		Reference:       b.reference,
		Date:            b.date,
		DueDate:         b.dueDate,
		CurrencyCode:    b.currencyCode,
		CurrencyRate:    b.currencyRate,
		Status:          xerodocs.StatusAuthorised,
		LineAmountTypes: xerodocs.LineAmountsNoTax,
		Lines:           b.lines,
	}
}

func (b *documentBody) creditNote(kind string) xerodocs.CreditNoteInput {
	return xerodocs.CreditNoteInput{
		Type:             kind,
		ContactID:        b.contactID,
		CreditNoteNumber: b.number,
		Reference:        b.reference,
		Date:             b.date,
		CurrencyCode:     b.currencyCode,
		CurrencyRate:     b.currencyRate,
		Status:           xerodocs.StatusAuthorised,
		LineAmountTypes:  xerodocs.LineAmountsNoTax,
		Lines:            b.lines,
	}
}

func (c *Documents) salesBody(
	ctx context.Context,
	client *xero.Client,
	doc *accountingconnectorports.AccountingSalesDocument,
) (*documentBody, error) {
	date, err := ParseDate(doc.TxnDate)
	if err != nil {
		return nil, err
	}
	due, err := ParseDate(doc.DueDate)
	if err != nil {
		return nil, err
	}
	note := doc.PrivateNote
	if doc.Kind == accountingsync.SyncObjectDebitMemo {
		note = strings.TrimSpace(debitMemoNotePrefix + ". " + note)
	}
	body := &documentBody{
		contactID:    doc.CustomerExternalID,
		number:       doc.DocNumber,
		reference:    Reference(note),
		date:         date,
		dueDate:      due,
		currencyCode: doc.CurrencyCode,
		currencyRate: doc.ExchangeRate,
		lines:        make([]xerodocs.LineItem, 0, len(doc.Lines)),
	}
	if doc.Kind == accountingsync.SyncObjectCreditMemo {
		body.dueDate = time.Time{}
	}
	for idx := range doc.Lines {
		line := &doc.Lines[idx]
		code, codeErr := c.AccountCode(
			ctx,
			client,
			line.AccountExternalID,
			"Line "+strconv.Itoa(idx+1)+" ("+lineText(line.Description)+")",
		)
		if codeErr != nil {
			return nil, codeErr
		}
		body.lines = append(body.lines, LineItem(&LineSpec{
			Description: line.Description,
			Quantity:    line.Quantity,
			UnitPrice:   line.UnitPrice,
			Amount:      line.Amount,
			AccountCode: code,
		}))
	}
	return body, nil
}

type LineSpec struct {
	Description string
	Quantity    decimal.Decimal
	UnitPrice   decimal.Decimal
	Amount      decimal.Decimal
	AccountCode string
}

func LineItem(spec *LineSpec) xerodocs.LineItem {
	item := xerodocs.LineItem{
		Description: stringutils.TruncateRunes(
			lineText(spec.Description),
			xeroref.MaxDescriptionLength,
		),
		Quantity:    decimal.NewFromInt(1),
		UnitAmount:  spec.Amount,
		LineAmount:  spec.Amount,
		AccountCode: spec.AccountCode,
	}
	if spec.Quantity.IsPositive() && !spec.UnitPrice.IsZero() &&
		spec.Quantity.Mul(spec.UnitPrice).Round(moneyPlaces).Equal(spec.Amount.Round(moneyPlaces)) {
		item.Quantity = spec.Quantity
		item.UnitAmount = spec.UnitPrice
	}
	return item
}

func lineText(description string) string {
	if text := strings.TrimSpace(description); text != "" {
		return text
	}
	return defaultLineText
}

func Reference(note string) string {
	return strings.TrimSpace(stringutils.OneLine(note, xerodocs.MaxReferenceLength))
}

func (c *Documents) writeSales(
	ctx context.Context,
	client *xero.Client,
	doc *accountingconnectorports.AccountingSalesDocument,
	externalID string,
) (id, number string, err error) {
	kind, err := salesKindOf(doc.Kind)
	if err != nil {
		return "", "", err
	}
	body, err := c.salesBody(ctx, client, doc)
	if err != nil {
		return "", "", err
	}

	if kind.credit {
		input := body.creditNote(xerodocs.CreditNoteTypeReceivable)
		var note *xerodocs.CreditNote
		if externalID == "" {
			note, err = xerodocs.CreateCreditNote(ctx, client, doc.RequestID, input)
		} else {
			note, err = xerodocs.UpdateCreditNote(ctx, client, doc.RequestID, externalID, input)
		}
		if err != nil {
			return "", "", err
		}
		return note.CreditNoteID, note.CreditNoteNumber, nil
	}

	input := body.invoice(xerodocs.InvoiceTypeReceivable)
	var invoice *xerodocs.Invoice
	if externalID == "" {
		invoice, err = xerodocs.CreateInvoice(ctx, client, doc.RequestID, input)
	} else {
		invoice, err = xerodocs.UpdateInvoice(ctx, client, doc.RequestID, externalID, input)
	}
	if err != nil {
		return "", "", err
	}
	return invoice.InvoiceID, invoice.InvoiceNumber, nil
}

func (c *Documents) CreateSalesDocument(
	ctx context.Context,
	doc *accountingconnectorports.AccountingSalesDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	kind, err := salesKindOf(doc.Kind)
	if err != nil {
		return nil, err
	}
	client, err := c.Client(doc.Auth)
	if err != nil {
		return nil, err
	}
	result := NewResult(doc.Refs)

	if existing := result.Refs[accountingsync.ExternalRefDocument]; existing != "" {
		result.ExternalID = existing
	} else {
		result.ExternalID, result.DocNumber, err = c.writeSales(ctx, client, doc, "")
		if err != nil {
			return result, err
		}
		result.Refs[accountingsync.ExternalRefDocument] = result.ExternalID
	}

	if !kind.credit ||
		strings.TrimSpace(doc.ApplyToExternalID) == "" ||
		!doc.ApplyAmount.IsPositive() ||
		result.Refs[accountingsync.ExternalRefApplication] != "" {
		return result, nil
	}

	date, err := ParseDate(doc.TxnDate)
	if err != nil {
		return result, err
	}
	allocation, err := xerodocs.Allocate(
		ctx,
		client,
		accountingsync.SyncStepRequestID(doc.RequestID, 1),
		result.ExternalID,
		xerodocs.AllocationInput{
			InvoiceID: doc.ApplyToExternalID,
			Amount:    doc.ApplyAmount,
			Date:      date,
		},
	)
	if err != nil {
		return result, err
	}
	result.Refs[accountingsync.ExternalRefApplication] = allocation.AllocationID
	return result, nil
}

func (c *Documents) UpdateSalesDocument(
	ctx context.Context,
	doc *accountingconnectorports.AccountingSalesDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if _, err := salesKindOf(doc.Kind); err != nil {
		return nil, err
	}
	id := strings.TrimSpace(doc.ExternalID)
	if id == "" {
		return nil, xerosession.ErrExternalID
	}
	client, err := c.Client(doc.Auth)
	if err != nil {
		return nil, err
	}

	result := NewResult(doc.Refs)
	result.ExternalID, result.DocNumber, err = c.writeSales(ctx, client, doc, id)
	if err != nil {
		return nil, err
	}
	result.Refs[accountingsync.ExternalRefDocument] = result.ExternalID
	return result, nil
}

func (c *Documents) VoidSalesDocument(
	ctx context.Context,
	ref *accountingconnectorports.AccountingDocumentRef,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	kind, err := salesKindOf(ref.Kind)
	if err != nil {
		return nil, err
	}
	client, err := c.Client(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := NewResult(ref.Refs)
	result.ExternalID = ref.ExternalID

	if kind.credit {
		err = VoidCreditNote(ctx, client, ref.RequestID, ref.ExternalID)
	} else {
		err = voidInvoice(ctx, client, ref.RequestID, ref.ExternalID)
	}
	if err != nil {
		return result, err
	}
	return result, nil
}

func voidInvoice(ctx context.Context, client *xero.Client, requestID, invoiceID string) error {
	found, err := xerodocs.InvoicesByID(ctx, client, []string{invoiceID})
	if err != nil {
		if xeroerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if len(found) == 0 || ClosedStatus(found[0].Status) {
		return nil
	}
	invoice := &found[0]

	for idx := range invoice.Payments {
		if delErr := xeropay.DeletePayment(
			ctx,
			client,
			accountingsync.SyncStepRequestID(requestID, idx+1),
			invoice.Payments[idx].PaymentID,
		); delErr != nil && !xeroerrors.IsNotFound(delErr) {
			return delErr
		}
	}
	if err = removeAllocationsTo(ctx, client, invoice); err != nil {
		return err
	}
	if _, err = xerodocs.VoidInvoice(ctx, client, requestID, invoice.InvoiceID); err != nil &&
		!xeroerrors.IsNotFound(err) {
		return err
	}
	return nil
}

func removeAllocationsTo(
	ctx context.Context,
	client *xero.Client,
	invoice *xerodocs.Invoice,
) error {
	if len(invoice.CreditNotes) == 0 {
		return nil
	}
	noteIDs := make([]string, 0, len(invoice.CreditNotes))
	for idx := range invoice.CreditNotes {
		noteIDs = append(noteIDs, invoice.CreditNotes[idx].CreditNoteID)
	}
	for start := 0; start < len(noteIDs); start += xero.MaxIDsPerRead {
		notes, err := xerodocs.CreditNotesByID(
			ctx,
			client,
			noteIDs[start:min(start+xero.MaxIDsPerRead, len(noteIDs))],
		)
		if err != nil {
			return err
		}
		for idx := range notes {
			for _, allocation := range notes[idx].Allocations {
				if allocation.IsDeleted || !sameID(allocation.InvoiceID, invoice.InvoiceID) {
					continue
				}
				if delErr := xerodocs.DeleteAllocation(
					ctx,
					client,
					notes[idx].CreditNoteID,
					allocation.AllocationID,
				); delErr != nil && !xeroerrors.IsNotFound(delErr) {
					return delErr
				}
			}
		}
	}
	return nil
}

func VoidCreditNote(ctx context.Context, client *xero.Client, requestID, noteID string) error {
	found, err := xerodocs.CreditNotesByID(ctx, client, []string{noteID})
	if err != nil {
		if xeroerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if len(found) == 0 || ClosedStatus(found[0].Status) {
		return nil
	}
	note := &found[0]
	for _, allocation := range note.Allocations {
		if allocation.IsDeleted {
			continue
		}
		if delErr := xerodocs.DeleteAllocation(
			ctx,
			client,
			note.CreditNoteID,
			allocation.AllocationID,
		); delErr != nil && !xeroerrors.IsNotFound(delErr) {
			return delErr
		}
	}
	for idx := range note.Payments {
		if delErr := xeropay.DeletePayment(
			ctx,
			client,
			accountingsync.SyncStepRequestID(requestID, idx+1),
			note.Payments[idx].PaymentID,
		); delErr != nil && !xeroerrors.IsNotFound(delErr) {
			return delErr
		}
	}
	if _, err = xerodocs.VoidCreditNote(ctx, client, requestID, note.CreditNoteID); err != nil &&
		!xeroerrors.IsNotFound(err) {
		return err
	}
	return nil
}

func ClosedStatus(status string) bool {
	return strings.EqualFold(status, xerodocs.StatusVoided) ||
		strings.EqualFold(status, xerodocs.StatusDeleted)
}

func (c *Documents) CreateCreditApplication(
	ctx context.Context,
	doc *accountingconnectorports.AccountingCreditApplicationDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if !doc.Amount.IsPositive() {
		return nil, xeroerrors.ErrAmountNotPositive
	}
	date, err := ParseDate(doc.TxnDate)
	if err != nil {
		return nil, err
	}
	client, err := c.Client(doc.Auth)
	if err != nil {
		return nil, err
	}
	allocation, err := xerodocs.Allocate(
		ctx,
		client,
		doc.RequestID,
		doc.CreditMemoExternalID,
		xerodocs.AllocationInput{
			InvoiceID: doc.InvoiceExternalID,
			Amount:    doc.Amount,
			Date:      date,
		},
	)
	if err != nil {
		return nil, err
	}
	return allocationResult(allocation), nil
}

func allocationResult(
	allocation *xerodocs.Allocation,
) *accountingconnectorports.AccountingDocumentResult {
	return &accountingconnectorports.AccountingDocumentResult{
		ExternalID: allocation.AllocationID,
		Refs: map[string]string{
			accountingsync.ExternalRefDocument: allocation.AllocationID,
			RefCreditNote:                      allocation.CreditNoteID,
		},
	}
}

func (c *Documents) VoidCreditApplication(
	ctx context.Context,
	ref *accountingconnectorports.AccountingDocumentRef,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	noteID := strings.TrimSpace(ref.Refs[RefCreditNote])
	if noteID == "" {
		return nil, xerosession.ErrCreditNoteID
	}
	client, err := c.Client(ref.Auth)
	if err != nil {
		return nil, err
	}
	if err = xerodocs.DeleteAllocation(ctx, client, noteID, ref.ExternalID); err != nil &&
		!xeroerrors.IsNotFound(err) {
		return nil, err
	}
	return &accountingconnectorports.AccountingDocumentResult{
		ExternalID: ref.ExternalID,
		Refs:       map[string]string{},
	}, nil
}

func ParseDate(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %q", xerosession.ErrInvalidDate, value)
	}
	return parsed, nil
}

func sameDate(t time.Time, raw string) bool {
	value := strings.TrimSpace(raw)
	return value == "" || (!t.IsZero() && t.UTC().Format(time.DateOnly) == value)
}

func SameMoney(a, b decimal.Decimal) bool {
	return a.Round(moneyPlaces).Equal(b.Round(moneyPlaces))
}

func sameID(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func NewResult(refs map[string]string) *accountingconnectorports.AccountingDocumentResult {
	copied := make(map[string]string, len(refs)+2)
	maps.Copy(copied, refs)
	return &accountingconnectorports.AccountingDocumentResult{Refs: copied}
}
