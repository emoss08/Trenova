package xeroconnector

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/shopspring/decimal"
)

var _ services.AccountingDocumentReader = (*Connector)(nil)

type readKind int

const (
	readInvoice readKind = iota + 1
	readCreditNote
	readPayment
	readAllocation
)

func (c *Connector) DocumentReadLimits() services.AccountingDocumentReadLimits {
	return services.AccountingDocumentReadLimits{MaxPerRead: xero.MaxIDsPerRead}
}

func readKindOf(
	kind accountingsync.SyncObjectType,
	target *services.AccountingDocumentTarget,
) (readKind, error) {
	switch {
	case kind == accountingsync.SyncObjectInvoice || kind == accountingsync.SyncObjectDebitMemo:
		return readInvoice, nil
	case kind == accountingsync.SyncObjectCreditMemo:
		return readCreditNote, nil
	case kind == accountingsync.SyncObjectCustomerPayment || kind.IsBillPayment():
		return readPayment, nil
	case kind == accountingsync.SyncObjectCreditApplication:
		return readAllocation, nil
	case kind.IsBill():
		credit, err := purchaseIsCredit(target.Refs)
		if err != nil {
			return 0, err
		}
		if credit {
			return readCreditNote, nil
		}
		return readInvoice, nil
	default:
		return 0, errDocumentKind
	}
}

type driftStep func(
	ctx context.Context,
	client *xero.Client,
	plan *driftRead,
) ([]*services.AccountingDocumentState, error)

type driftRead struct {
	invoices    []string
	creditNotes []string
	payments    []*services.AccountingDocumentTarget
	allocations []*services.AccountingDocumentTarget
}

func (c *Connector) ReadDocuments(
	ctx context.Context,
	req *services.ReadAccountingDocumentsRequest,
) ([]*services.AccountingDocumentState, error) {
	var plan driftRead
	for idx := range req.Targets {
		target := &req.Targets[idx]
		kind, err := readKindOf(req.Kind, target)
		if err != nil {
			return nil, err
		}
		id := strings.TrimSpace(target.ExternalID)
		if id == "" {
			continue
		}
		switch kind {
		case readInvoice:
			plan.invoices = append(plan.invoices, id)
		case readCreditNote:
			plan.creditNotes = append(plan.creditNotes, id)
		case readPayment:
			plan.payments = append(plan.payments, target)
		case readAllocation:
			plan.allocations = append(plan.allocations, target)
		}
	}

	client, err := c.client(req.Auth)
	if err != nil {
		return nil, err
	}
	out := make([]*services.AccountingDocumentState, 0, len(req.Targets))
	for _, step := range []driftStep{
		readInvoices,
		readCreditNotes,
		readPayments,
		readAllocations,
	} {
		states, stepErr := step(ctx, client, &plan)
		if stepErr != nil {
			return nil, stepErr
		}
		out = append(out, states...)
	}
	return out, nil
}

func chunks(ids []string) [][]string {
	out := make([][]string, 0, len(ids)/xero.MaxIDsPerRead+1)
	for start := 0; start < len(ids); start += xero.MaxIDsPerRead {
		out = append(out, ids[start:min(start+xero.MaxIDsPerRead, len(ids))])
	}
	return out
}

func readInvoices(
	ctx context.Context,
	client *xero.Client,
	plan *driftRead,
) ([]*services.AccountingDocumentState, error) {
	found := make(map[string]*xero.Invoice, len(plan.invoices))
	for _, chunk := range chunks(plan.invoices) {
		invoices, err := client.InvoicesByID(ctx, chunk)
		if err != nil {
			return nil, err
		}
		for idx := range invoices {
			found[strings.ToLower(invoices[idx].InvoiceID)] = &invoices[idx]
		}
	}
	out := make([]*services.AccountingDocumentState, 0, len(plan.invoices))
	for _, id := range plan.invoices {
		state := &services.AccountingDocumentState{ExternalID: id}
		if invoice, ok := found[strings.ToLower(id)]; ok {
			balance := invoice.AmountDue
			state.Found = true
			state.Voided = closedStatus(invoice.Status)
			state.DocNumber = invoice.InvoiceNumber
			state.Total = invoice.Total
			state.Balance = &balance
			state.CurrencyCode = invoice.CurrencyCode
			state.ModifiedAt = unixOrZero(invoice.UpdatedAt)
		}
		out = append(out, state)
	}
	return out, nil
}

func readCreditNotes(
	ctx context.Context,
	client *xero.Client,
	plan *driftRead,
) ([]*services.AccountingDocumentState, error) {
	found := make(map[string]*xero.CreditNote, len(plan.creditNotes))
	for _, chunk := range chunks(plan.creditNotes) {
		notes, err := client.CreditNotesByID(ctx, chunk)
		if err != nil {
			return nil, err
		}
		for idx := range notes {
			found[strings.ToLower(notes[idx].CreditNoteID)] = &notes[idx]
		}
	}
	out := make([]*services.AccountingDocumentState, 0, len(plan.creditNotes))
	for _, id := range plan.creditNotes {
		state := &services.AccountingDocumentState{ExternalID: id}
		if note, ok := found[strings.ToLower(id)]; ok {
			balance := note.RemainingCredit
			state.Found = true
			state.Voided = closedStatus(note.Status)
			state.DocNumber = note.CreditNoteNumber
			state.Total = note.Total
			state.Balance = &balance
			state.CurrencyCode = note.CurrencyCode
			state.ModifiedAt = unixOrZero(note.UpdatedAt)
		}
		out = append(out, state)
	}
	return out, nil
}

func paymentIDsOf(refs map[string]string) map[string]string {
	out := make(map[string]string, len(refs))
	for key, value := range refs {
		if value == "" || !strings.HasPrefix(key, refPaymentPrefix) {
			continue
		}
		out[strings.ToLower(strings.TrimPrefix(key, refPaymentPrefix))] = strings.ToLower(value)
	}
	return out
}

func readPayments(
	ctx context.Context,
	client *xero.Client,
	plan *driftRead,
) ([]*services.AccountingDocumentState, error) {
	invoiceIDs := make([]string, 0, len(plan.payments))
	seen := make(map[string]struct{}, len(plan.payments))
	for _, target := range plan.payments {
		for invoiceID := range paymentIDsOf(target.Refs) {
			if _, dup := seen[invoiceID]; dup {
				continue
			}
			seen[invoiceID] = struct{}{}
			invoiceIDs = append(invoiceIDs, invoiceID)
		}
	}
	byID := make(map[string]*xero.Payment, len(invoiceIDs))
	for _, chunk := range chunks(invoiceIDs) {
		payments, err := client.FindPayments(ctx, xero.PaymentFilter{InvoiceIDs: chunk})
		if err != nil {
			return nil, err
		}
		for idx := range payments {
			byID[strings.ToLower(payments[idx].PaymentID)] = &payments[idx]
		}
	}

	out := make([]*services.AccountingDocumentState, 0, len(plan.payments))
	for _, target := range plan.payments {
		wanted := paymentIDsOf(target.Refs)
		if len(wanted) == 0 {
			continue
		}
		state := &services.AccountingDocumentState{ExternalID: strings.TrimSpace(target.ExternalID)}
		total := decimal.Zero
		deleted, read := 0, 0
		for _, paymentID := range wanted {
			payment, ok := byID[paymentID]
			if !ok {
				continue
			}
			read++
			state.DocNumber = payment.Reference
			if strings.EqualFold(payment.Status, xero.StatusDeleted) {
				deleted++
				continue
			}
			total = total.Add(payment.Amount)
			state.ModifiedAt = max(state.ModifiedAt, unixOrZero(payment.UpdatedAt))
		}
		if read > 0 {
			state.Found = true
			state.Voided = deleted == read
			state.Total = total
		}
		out = append(out, state)
	}
	return out, nil
}

func readAllocations(
	ctx context.Context,
	client *xero.Client,
	plan *driftRead,
) ([]*services.AccountingDocumentState, error) {
	noteIDs := make([]string, 0, len(plan.allocations))
	seen := make(map[string]struct{}, len(plan.allocations))
	for _, target := range plan.allocations {
		noteID := strings.ToLower(strings.TrimSpace(target.Refs[refCreditNote]))
		if noteID == "" {
			continue
		}
		if _, dup := seen[noteID]; !dup {
			seen[noteID] = struct{}{}
			noteIDs = append(noteIDs, noteID)
		}
	}
	byID := make(map[string]*xero.Allocation, len(noteIDs))
	for _, chunk := range chunks(noteIDs) {
		notes, err := client.CreditNotesByID(ctx, chunk)
		if err != nil {
			return nil, err
		}
		for idx := range notes {
			for aIdx := range notes[idx].Allocations {
				allocation := &notes[idx].Allocations[aIdx]
				byID[strings.ToLower(allocation.AllocationID)] = allocation
			}
		}
	}

	out := make([]*services.AccountingDocumentState, 0, len(plan.allocations))
	for _, target := range plan.allocations {
		if strings.TrimSpace(target.Refs[refCreditNote]) == "" {
			continue
		}
		id := strings.TrimSpace(target.ExternalID)
		state := &services.AccountingDocumentState{ExternalID: id}
		if allocation, ok := byID[strings.ToLower(id)]; ok {
			state.Found = true
			state.Voided = allocation.IsDeleted
			state.Total = allocation.Amount
		}
		out = append(out, state)
	}
	return out, nil
}

func unixOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}
