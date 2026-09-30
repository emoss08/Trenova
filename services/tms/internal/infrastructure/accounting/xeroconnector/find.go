package xeroconnector

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/shopspring/decimal"
)

func (c *Connector) FindDocument(
	ctx context.Context,
	req *services.AccountingFindDocumentRequest,
) (*services.AccountingDocumentResult, bool, error) {
	switch {
	case req.Kind.IsSalesDocument(),
		req.Kind.IsBill(),
		req.Kind.IsBillPayment(),
		req.Kind == accountingsync.SyncObjectCustomerPayment,
		req.Kind == accountingsync.SyncObjectCreditApplication:
	default:
		return nil, false, errDocumentKind
	}
	client, err := c.client(req.Auth)
	if err != nil {
		return nil, false, err
	}

	var (
		found *services.AccountingDocumentResult
		ok    bool
	)
	switch {
	case req.Kind == accountingsync.SyncObjectCreditMemo:
		found, ok, err = findCreditNote(ctx, client, req, xero.CreditNoteTypeReceivable)
	case req.Kind.IsSalesDocument():
		found, ok, err = findInvoice(ctx, client, req, xero.InvoiceTypeReceivable)
	case req.Kind.IsBill() && req.Credit:
		found, ok, err = findCreditNote(ctx, client, req, xero.CreditNoteTypePayable)
	case req.Kind.IsBill():
		found, ok, err = findInvoice(ctx, client, req, xero.InvoiceTypePayable)
	case req.Kind == accountingsync.SyncObjectCreditApplication:
		found, ok, err = findAllocation(ctx, client, req)
	default:
		found, ok, err = findPayment(ctx, client, req)
	}
	if errors.Is(err, xero.ErrEmptyFilter) || errors.Is(err, xero.ErrInvalidFilter) {
		return nil, false, nil
	}
	if err != nil || !ok {
		return nil, false, err
	}
	if req.Kind.IsBill() {
		found = c.purchaseResult(&purchaseWritten{
			auth:       req.Auth,
			kind:       req.Kind,
			credit:     req.Credit,
			externalID: found.ExternalID,
			number:     found.DocNumber,
		})
	}
	return found, true, nil
}

func documentFilter(
	req *services.AccountingFindDocumentRequest,
	kind string,
) (xero.InvoiceFilter, error) {
	filter := xero.InvoiceFilter{Type: kind}
	if number := strings.TrimSpace(req.DocNumber); number != "" {
		filter.Numbers = []string{number}
	}
	receivable := kind == xero.InvoiceTypeReceivable || kind == xero.CreditNoteTypeReceivable
	if receivable {
		if len(filter.Numbers) == 0 {
			return filter, xero.ErrEmptyFilter
		}
		return filter, nil
	}
	if id := strings.TrimSpace(req.CounterpartyExternalID); id != "" {
		filter.ContactIDs = []string{id}
	}
	date, err := parseDate(req.TxnDate)
	if err != nil {
		return filter, err
	}
	filter.DateFrom, filter.DateTo = date, date
	if len(filter.Numbers) == 0 && len(filter.ContactIDs) == 0 {
		return filter, xero.ErrEmptyFilter
	}
	return filter, nil
}

type documentMatch struct {
	number    string
	contactID string
	status    string
	total     decimal.Decimal
}

func (m *documentMatch) matches(req *services.AccountingFindDocumentRequest, receivable bool) bool {
	if closedStatus(m.status) {
		return false
	}
	if number := strings.TrimSpace(req.DocNumber); number != "" &&
		!strings.EqualFold(strings.TrimSpace(m.number), number) {
		return false
	}
	if !req.Total.IsZero() && !sameMoney(m.total, req.Total.Abs()) {
		return false
	}
	if receivable {
		return true
	}
	return strings.TrimSpace(req.CounterpartyExternalID) == "" ||
		sameID(m.contactID, req.CounterpartyExternalID)
}

func findInvoice(
	ctx context.Context,
	client *xero.Client,
	req *services.AccountingFindDocumentRequest,
	kind string,
) (*services.AccountingDocumentResult, bool, error) {
	filter, err := documentFilter(req, kind)
	if err != nil {
		return nil, false, err
	}
	invoices, err := client.FindInvoices(ctx, filter)
	if err != nil {
		return nil, false, err
	}
	receivable := kind == xero.InvoiceTypeReceivable
	for idx := range invoices {
		invoice := &invoices[idx]
		match := documentMatch{
			number:    invoice.InvoiceNumber,
			contactID: invoice.ContactID,
			status:    invoice.Status,
			total:     invoice.Total,
		}
		if !receivable && !sameDate(invoice.Date, req.TxnDate) {
			continue
		}
		if match.matches(req, receivable) {
			return documentResult(invoice.InvoiceID, invoice.InvoiceNumber), true, nil
		}
	}
	return nil, false, nil
}

func findCreditNote(
	ctx context.Context,
	client *xero.Client,
	req *services.AccountingFindDocumentRequest,
	kind string,
) (*services.AccountingDocumentResult, bool, error) {
	filter, err := documentFilter(req, kind)
	if err != nil {
		return nil, false, err
	}
	notes, err := client.FindCreditNotes(ctx, filter)
	if err != nil {
		return nil, false, err
	}
	receivable := kind == xero.CreditNoteTypeReceivable
	for idx := range notes {
		note := &notes[idx]
		match := documentMatch{
			number:    note.CreditNoteNumber,
			contactID: note.ContactID,
			status:    note.Status,
			total:     note.Total,
		}
		if !receivable && !sameDate(note.Date, req.TxnDate) {
			continue
		}
		if match.matches(req, receivable) {
			return documentResult(note.CreditNoteID, note.CreditNoteNumber), true, nil
		}
	}
	return nil, false, nil
}

func documentResult(externalID, number string) *services.AccountingDocumentResult {
	return &services.AccountingDocumentResult{
		ExternalID: externalID,
		DocNumber:  number,
		Refs:       map[string]string{accountingsync.ExternalRefDocument: externalID},
	}
}

type paymentGroup struct {
	batchID  string
	payments []*xero.Payment
	total    decimal.Decimal
}

func findPayment(
	ctx context.Context,
	client *xero.Client,
	req *services.AccountingFindDocumentRequest,
) (*services.AccountingDocumentResult, bool, error) {
	requested := make(map[string]struct{}, len(req.AppliesToExternalIDs))
	for _, id := range req.AppliesToExternalIDs {
		if trimmed := strings.ToLower(strings.TrimSpace(id)); trimmed != "" {
			requested[trimmed] = struct{}{}
		}
	}
	if len(requested) == 0 || len(requested) > xero.MaxIDsPerRead {
		return nil, false, nil
	}

	payments, err := client.FindPayments(
		ctx,
		xero.PaymentFilter{InvoiceIDs: req.AppliesToExternalIDs},
	)
	if err != nil {
		return nil, false, err
	}
	expected := paymentReference(req.DocNumber, req.RequestID)

	groups := make([]*paymentGroup, 0, len(payments))
	byBatch := make(map[string]*paymentGroup, len(payments))
	for idx := range payments {
		payment := &payments[idx]
		if strings.EqualFold(payment.Status, xero.StatusDeleted) ||
			!sameDate(payment.Date, req.TxnDate) {
			continue
		}
		if payment.BatchPaymentID == "" {
			groups = append(groups, &paymentGroup{
				payments: []*xero.Payment{payment},
				total:    payment.Amount,
			})
			continue
		}
		group, ok := byBatch[payment.BatchPaymentID]
		if !ok {
			group = &paymentGroup{batchID: payment.BatchPaymentID}
			byBatch[payment.BatchPaymentID] = group
			groups = append(groups, group)
		}
		group.payments = append(group.payments, payment)
		group.total = group.total.Add(payment.Amount)
	}

	for _, group := range groups {
		if group.fits(req, requested, expected) {
			return group.result(), true, nil
		}
	}
	return nil, false, nil
}

func (g *paymentGroup) fits(
	req *services.AccountingFindDocumentRequest,
	requested map[string]struct{},
	expected string,
) bool {
	if !req.Total.IsZero() && !sameMoney(g.total, req.Total) {
		return false
	}
	for _, payment := range g.payments {
		if _, ok := requested[strings.ToLower(payment.InvoiceID)]; !ok {
			return false
		}
		switch {
		case payment.Reference == expected:
		case g.batchID != "" && payment.Reference == "":
		default:
			return false
		}
	}
	return true
}

func (g *paymentGroup) result() *services.AccountingDocumentResult {
	result := newResult(nil)
	if g.batchID == "" {
		finishPayment(result, g.payments[0])
		delete(result.Refs, refReplaced)
		return result
	}
	result.ExternalID = g.batchID
	result.DocNumber = g.payments[0].Reference
	result.Refs[accountingsync.ExternalRefDocument] = g.batchID
	result.Refs[accountingsync.ExternalRefDocumentType] = docTypeBatchPayment
	for _, payment := range g.payments {
		result.Refs[refPaymentPrefix+strings.ToLower(payment.InvoiceID)] = payment.PaymentID
	}
	return result
}

func findAllocation(
	ctx context.Context,
	client *xero.Client,
	req *services.AccountingFindDocumentRequest,
) (*services.AccountingDocumentResult, bool, error) {
	noteID := strings.TrimSpace(req.CreditExternalID)
	if noteID == "" || len(req.AppliesToExternalIDs) == 0 {
		return nil, false, nil
	}
	invoiceID := req.AppliesToExternalIDs[0]
	notes, err := client.CreditNotesByID(ctx, []string{noteID})
	if err != nil {
		if xero.IsNotFound(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	for idx := range notes {
		for aIdx := range notes[idx].Allocations {
			allocation := &notes[idx].Allocations[aIdx]
			if allocation.IsDeleted || !sameID(allocation.InvoiceID, invoiceID) ||
				!sameDate(allocation.Date, req.TxnDate) {
				continue
			}
			if !req.Total.IsZero() && !sameMoney(allocation.Amount, req.Total) {
				continue
			}
			if allocation.CreditNoteID == "" {
				allocation.CreditNoteID = notes[idx].CreditNoteID
			}
			return allocationResult(allocation), true, nil
		}
	}
	return nil, false, nil
}
