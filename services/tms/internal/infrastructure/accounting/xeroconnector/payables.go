package xeroconnector

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/xero"
)

var errPurchaseDocumentType = errors.New(
	"xero: the purchase document type is neither a bill nor a supplier credit",
)

func (c *Connector) UpsertVendor(
	ctx context.Context,
	doc *services.AccountingVendorDocument,
) (*services.AccountingDocumentResult, error) {
	return c.upsertContact(ctx, &contactWrite{
		auth:       doc.Auth,
		requestID:  doc.RequestID,
		externalID: doc.ExternalID,
		party:      &doc.Party,
	})
}

func (c *Connector) purchaseBody(
	ctx context.Context,
	client *xero.Client,
	doc *services.AccountingPurchaseDocument,
) (*documentBody, error) {
	date, err := parseDate(doc.TxnDate)
	if err != nil {
		return nil, err
	}
	due, err := parseDate(doc.DueDate)
	if err != nil {
		return nil, err
	}
	body := &documentBody{
		contactID:    doc.VendorExternalID,
		number:       doc.DocNumber,
		reference:    reference(doc.PrivateNote),
		date:         date,
		dueDate:      due,
		currencyCode: doc.CurrencyCode,
		currencyRate: doc.ExchangeRate,
		lines:        make([]xero.LineItem, 0, len(doc.Lines)),
	}
	for idx := range doc.Lines {
		line := &doc.Lines[idx]
		code, codeErr := c.accountCode(
			ctx,
			client,
			line.AccountExternalID,
			"Line "+strconv.Itoa(idx+1)+" ("+lineText(line.Description)+")",
		)
		if codeErr != nil {
			return nil, codeErr
		}
		body.lines = append(body.lines, lineItem(&lineSpec{
			description: line.Description,
			amount:      line.Amount,
			accountCode: code,
		}))
	}
	return body, nil
}

type purchaseWritten struct {
	auth       services.AccountingDocumentAuth
	kind       accountingsync.SyncObjectType
	credit     bool
	externalID string
	number     string
}

func (c *Connector) purchaseResult(written *purchaseWritten) *services.AccountingDocumentResult {
	docType := xero.InvoiceTypePayable
	if written.credit {
		docType = xero.CreditNoteTypePayable
	}
	refs := map[string]string{
		accountingsync.ExternalRefDocument:       written.externalID,
		accountingsync.ExternalRefDocumentType:   docType,
		accountingsync.ExternalRefCreditDocument: strconv.FormatBool(written.credit),
	}
	if !written.credit {
		if link := c.DocumentURL(written.auth, services.AccountingDocumentLink{
			Kind:       written.kind,
			ExternalID: written.externalID,
		}); link != "" {
			refs[accountingsync.ExternalRefURL] = link
		}
	}
	return &services.AccountingDocumentResult{
		ExternalID: written.externalID,
		DocNumber:  written.number,
		Refs:       refs,
	}
}

func (c *Connector) CreatePurchaseDocument(
	ctx context.Context,
	doc *services.AccountingPurchaseDocument,
) (*services.AccountingDocumentResult, error) {
	return c.writePurchase(ctx, doc, "")
}

func (c *Connector) UpdatePurchaseDocument(
	ctx context.Context,
	doc *services.AccountingPurchaseDocument,
) (*services.AccountingDocumentResult, error) {
	id := strings.TrimSpace(doc.ExternalID)
	if id == "" {
		return nil, errExternalID
	}
	return c.writePurchase(ctx, doc, id)
}

func (c *Connector) writePurchase(
	ctx context.Context,
	doc *services.AccountingPurchaseDocument,
	externalID string,
) (*services.AccountingDocumentResult, error) {
	if !doc.Kind.IsBill() {
		return nil, errDocumentKind
	}
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	body, err := c.purchaseBody(ctx, client, doc)
	if err != nil {
		return nil, err
	}

	if doc.VendorCredit {
		input := body.creditNote(xero.CreditNoteTypePayable)
		var note *xero.CreditNote
		if externalID == "" {
			note, err = client.CreateCreditNote(ctx, doc.RequestID, input)
		} else {
			note, err = client.UpdateCreditNote(ctx, doc.RequestID, externalID, input)
		}
		if err != nil {
			return nil, err
		}
		return c.purchaseResult(&purchaseWritten{
			auth:       doc.Auth,
			kind:       doc.Kind,
			credit:     true,
			externalID: note.CreditNoteID,
			number:     note.CreditNoteNumber,
		}), nil
	}

	input := body.invoice(xero.InvoiceTypePayable)
	var bill *xero.Invoice
	if externalID == "" {
		bill, err = client.CreateInvoice(ctx, doc.RequestID, input)
	} else {
		bill, err = client.UpdateInvoice(ctx, doc.RequestID, externalID, input)
	}
	if err != nil {
		return nil, err
	}
	return c.purchaseResult(&purchaseWritten{
		auth:       doc.Auth,
		kind:       doc.Kind,
		externalID: bill.InvoiceID,
		number:     bill.InvoiceNumber,
	}), nil
}

func purchaseIsCredit(refs map[string]string) (bool, error) {
	if value := strings.TrimSpace(refs[accountingsync.ExternalRefCreditDocument]); value != "" {
		credit, err := strconv.ParseBool(value)
		if err != nil {
			return false, errPurchaseDocumentType
		}
		return credit, nil
	}
	switch strings.TrimSpace(refs[accountingsync.ExternalRefDocumentType]) {
	case "", xero.InvoiceTypePayable:
		return false, nil
	case xero.CreditNoteTypePayable:
		return true, nil
	default:
		return false, errPurchaseDocumentType
	}
}

func (c *Connector) VoidPurchaseDocument(
	ctx context.Context,
	ref *services.AccountingDocumentRef,
) (*services.AccountingDocumentResult, error) {
	if !ref.Kind.IsBill() {
		return nil, errDocumentKind
	}
	credit, err := purchaseIsCredit(ref.Refs)
	if err != nil {
		return nil, err
	}
	client, err := c.client(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(ref.Refs)
	result.ExternalID = ref.ExternalID

	if credit {
		err = voidCreditNote(ctx, client, ref.RequestID, ref.ExternalID)
	} else {
		err = voidInvoice(ctx, client, ref.RequestID, ref.ExternalID)
	}
	if err != nil {
		return result, err
	}
	return result, nil
}

func (c *Connector) CreateBillPayment(
	ctx context.Context,
	doc *services.AccountingBillPaymentDocument,
) (*services.AccountingDocumentResult, error) {
	if !doc.Kind.IsBillPayment() {
		return nil, errDocumentKind
	}
	return c.writeBillPayment(ctx, doc, doc.RequestID)
}

func (c *Connector) UpdateBillPayment(
	ctx context.Context,
	doc *services.AccountingBillPaymentDocument,
) (*services.AccountingDocumentResult, error) {
	if !doc.Kind.IsBillPayment() {
		return nil, errDocumentKind
	}
	id := strings.TrimSpace(doc.ExternalID)
	if id == "" {
		return nil, errExternalID
	}
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	if err = client.DeletePayment(
		ctx,
		accountingsync.SyncStepRequestID(doc.RequestID, stepReplaceDelete),
		id,
	); err != nil && !xero.IsNotFound(err) {
		return nil, err
	}
	return c.writeBillPayment(
		ctx,
		doc,
		accountingsync.SyncStepRequestID(doc.RequestID, stepReplaceCreate),
	)
}

func (c *Connector) writeBillPayment(
	ctx context.Context,
	doc *services.AccountingBillPaymentDocument,
	key string,
) (*services.AccountingDocumentResult, error) {
	date, err := parseDate(doc.TxnDate)
	if err != nil {
		return nil, err
	}
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	payment, err := client.CreatePayment(ctx, key, xero.PaymentInput{
		InvoiceID:    doc.BillExternalID,
		AccountID:    doc.BankAccountExternalID,
		Date:         date,
		Amount:       doc.Amount,
		CurrencyRate: doc.ExchangeRate,
		Reference:    paymentReference(doc.DocNumber, doc.RequestID),
	})
	if err != nil {
		return nil, err
	}
	result := newResult(nil)
	finishPayment(result, payment)
	delete(result.Refs, refReplaced)
	return result, nil
}
