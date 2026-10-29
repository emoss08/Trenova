package xerodocsync

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/xerosession"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/emoss08/trenova/shared/xerodocs"
	"github.com/emoss08/trenova/shared/xeroerrors"
	"github.com/emoss08/trenova/shared/xeropay"
)

var ErrPurchaseDocumentType = errors.New(
	"xero: the purchase document type is neither a bill nor a supplier credit",
)

func (c *Documents) UpsertVendor(
	ctx context.Context,
	doc *accountingconnectorports.AccountingVendorDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	return c.upsertContact(ctx, &contactWrite{
		auth:       doc.Auth,
		requestID:  doc.RequestID,
		externalID: doc.ExternalID,
		party:      &doc.Party,
	})
}

func (c *Documents) purchaseBody(
	ctx context.Context,
	client *xero.Client,
	doc *accountingconnectorports.AccountingPurchaseDocument,
) (*documentBody, error) {
	date, err := ParseDate(doc.TxnDate)
	if err != nil {
		return nil, err
	}
	due, err := ParseDate(doc.DueDate)
	if err != nil {
		return nil, err
	}
	body := &documentBody{
		contactID:    doc.VendorExternalID,
		number:       doc.DocNumber,
		reference:    Reference(doc.PrivateNote),
		date:         date,
		dueDate:      due,
		currencyCode: doc.CurrencyCode,
		currencyRate: doc.ExchangeRate,
		lines:        make([]xerodocs.LineItem, 0, len(doc.Lines)),
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
			Amount:      line.Amount,
			AccountCode: code,
		}))
	}
	return body, nil
}

type purchaseWritten struct {
	auth       accountingconnectorports.AccountingDocumentAuth
	kind       accountingsync.SyncObjectType
	credit     bool
	externalID string
	number     string
}

func (c *Documents) purchaseResult(
	written *purchaseWritten,
) *accountingconnectorports.AccountingDocumentResult {
	docType := xerodocs.InvoiceTypePayable
	if written.credit {
		docType = xerodocs.CreditNoteTypePayable
	}
	refs := map[string]string{
		accountingsync.ExternalRefDocument:       written.externalID,
		accountingsync.ExternalRefDocumentType:   docType,
		accountingsync.ExternalRefCreditDocument: strconv.FormatBool(written.credit),
	}
	if !written.credit {
		if link := c.DocumentURL(written.auth, accountingconnectorports.AccountingDocumentLink{
			Kind:       written.kind,
			ExternalID: written.externalID,
		}); link != "" {
			refs[accountingsync.ExternalRefURL] = link
		}
	}
	return &accountingconnectorports.AccountingDocumentResult{
		ExternalID: written.externalID,
		DocNumber:  written.number,
		Refs:       refs,
	}
}

func (c *Documents) CreatePurchaseDocument(
	ctx context.Context,
	doc *accountingconnectorports.AccountingPurchaseDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	return c.writePurchase(ctx, doc, "")
}

func (c *Documents) UpdatePurchaseDocument(
	ctx context.Context,
	doc *accountingconnectorports.AccountingPurchaseDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	id := strings.TrimSpace(doc.ExternalID)
	if id == "" {
		return nil, xerosession.ErrExternalID
	}
	return c.writePurchase(ctx, doc, id)
}

func (c *Documents) writePurchase(
	ctx context.Context,
	doc *accountingconnectorports.AccountingPurchaseDocument,
	externalID string,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if !doc.Kind.IsBill() {
		return nil, xerosession.ErrDocumentKind
	}
	client, err := c.Client(doc.Auth)
	if err != nil {
		return nil, err
	}
	body, err := c.purchaseBody(ctx, client, doc)
	if err != nil {
		return nil, err
	}

	if doc.VendorCredit {
		input := body.creditNote(xerodocs.CreditNoteTypePayable)
		var note *xerodocs.CreditNote
		if externalID == "" {
			note, err = xerodocs.CreateCreditNote(ctx, client, doc.RequestID, input)
		} else {
			note, err = xerodocs.UpdateCreditNote(ctx, client, doc.RequestID, externalID, input)
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

	input := body.invoice(xerodocs.InvoiceTypePayable)
	var bill *xerodocs.Invoice
	if externalID == "" {
		bill, err = xerodocs.CreateInvoice(ctx, client, doc.RequestID, input)
	} else {
		bill, err = xerodocs.UpdateInvoice(ctx, client, doc.RequestID, externalID, input)
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

func PurchaseIsCredit(refs map[string]string) (bool, error) {
	if value := strings.TrimSpace(refs[accountingsync.ExternalRefCreditDocument]); value != "" {
		credit, err := strconv.ParseBool(value)
		if err != nil {
			return false, ErrPurchaseDocumentType
		}
		return credit, nil
	}
	switch strings.TrimSpace(refs[accountingsync.ExternalRefDocumentType]) {
	case "", xerodocs.InvoiceTypePayable:
		return false, nil
	case xerodocs.CreditNoteTypePayable:
		return true, nil
	default:
		return false, ErrPurchaseDocumentType
	}
}

func (c *Documents) VoidPurchaseDocument(
	ctx context.Context,
	ref *accountingconnectorports.AccountingDocumentRef,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if !ref.Kind.IsBill() {
		return nil, xerosession.ErrDocumentKind
	}
	credit, err := PurchaseIsCredit(ref.Refs)
	if err != nil {
		return nil, err
	}
	client, err := c.Client(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := NewResult(ref.Refs)
	result.ExternalID = ref.ExternalID

	if credit {
		err = VoidCreditNote(ctx, client, ref.RequestID, ref.ExternalID)
	} else {
		err = voidInvoice(ctx, client, ref.RequestID, ref.ExternalID)
	}
	if err != nil {
		return result, err
	}
	return result, nil
}

func (c *Documents) CreateBillPayment(
	ctx context.Context,
	doc *accountingconnectorports.AccountingBillPaymentDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if !doc.Kind.IsBillPayment() {
		return nil, xerosession.ErrDocumentKind
	}
	return c.writeBillPayment(ctx, doc, doc.RequestID)
}

func (c *Documents) UpdateBillPayment(
	ctx context.Context,
	doc *accountingconnectorports.AccountingBillPaymentDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if !doc.Kind.IsBillPayment() {
		return nil, xerosession.ErrDocumentKind
	}
	id := strings.TrimSpace(doc.ExternalID)
	if id == "" {
		return nil, xerosession.ErrExternalID
	}
	client, err := c.Client(doc.Auth)
	if err != nil {
		return nil, err
	}
	if err = xeropay.DeletePayment(
		ctx,
		client,
		accountingsync.SyncStepRequestID(doc.RequestID, StepReplaceDelete),
		id,
	); err != nil && !xeroerrors.IsNotFound(err) {
		return nil, err
	}
	return c.writeBillPayment(
		ctx,
		doc,
		accountingsync.SyncStepRequestID(doc.RequestID, StepReplaceCreate),
	)
}

func (c *Documents) writeBillPayment(
	ctx context.Context,
	doc *accountingconnectorports.AccountingBillPaymentDocument,
	key string,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	date, err := ParseDate(doc.TxnDate)
	if err != nil {
		return nil, err
	}
	client, err := c.Client(doc.Auth)
	if err != nil {
		return nil, err
	}
	payment, err := xeropay.CreatePayment(ctx, client, key, xeropay.PaymentInput{
		InvoiceID:    doc.BillExternalID,
		AccountID:    doc.BankAccountExternalID,
		Date:         date,
		Amount:       doc.Amount,
		CurrencyRate: doc.ExchangeRate,
		Reference:    PaymentReference(doc.DocNumber, doc.RequestID),
	})
	if err != nil {
		return nil, err
	}
	result := NewResult(nil)
	FinishPayment(result, payment)
	delete(result.Refs, RefReplaced)
	return result, nil
}
