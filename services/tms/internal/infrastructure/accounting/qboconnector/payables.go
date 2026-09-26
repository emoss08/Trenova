package qboconnector

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/quickbooks"
)

var errPurchaseDocumentType = errors.New(
	"quickbooks: the purchase document type is neither a bill nor a vendor credit",
)

func (c *Connector) UpsertVendor(
	ctx context.Context,
	doc *services.AccountingVendorDocument,
) (*services.AccountingDocumentResult, error) {
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	draft := partyDraftOf(&doc.Party)
	draft.DisplayName = quickbooks.SanitizeName(draft.DisplayName, quickbooks.MaxPartyNameLength)

	var saved *quickbooks.ReferenceObject
	if id := strings.TrimSpace(doc.ExternalID); id != "" {
		saved, err = client.UpdateVendor(ctx, doc.RequestID, id, draft)
	} else {
		saved, err = client.CreateVendor(ctx, doc.RequestID, draft)
	}
	if err != nil {
		return nil, err
	}
	return &services.AccountingDocumentResult{ExternalID: saved.ID, DocNumber: saved.Name}, nil
}

func (c *Connector) CreatePurchaseDocument(
	ctx context.Context,
	doc *services.AccountingPurchaseDocument,
) (*services.AccountingDocumentResult, error) {
	if !doc.Kind.IsBill() {
		return nil, errDocumentKind
	}
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}

	txn := purchaseTxnOf(doc)
	kind := quickbooks.TxnBill
	var created *quickbooks.TxnResult
	if doc.VendorCredit {
		kind = quickbooks.TxnVendorCredit
		created, err = client.CreateVendorCredit(ctx, doc.RequestID, txn)
	} else {
		created, err = client.CreateBill(ctx, doc.RequestID, txn)
	}
	if err != nil {
		return nil, err
	}

	return &services.AccountingDocumentResult{
		ExternalID: created.ID,
		DocNumber:  created.DocNumber,
		Refs: map[string]string{
			accountingsync.ExternalRefDocument:     created.ID,
			accountingsync.ExternalRefDocumentType: string(kind),
			accountingsync.ExternalRefURL:          c.appURL(kind.AppPath(), created.ID),
		},
	}, nil
}

func purchaseTxnOf(doc *services.AccountingPurchaseDocument) *quickbooks.PurchaseTxn {
	txn := &quickbooks.PurchaseTxn{
		VendorID:     doc.VendorExternalID,
		APAccountID:  doc.APAccountExternalID,
		DocNumber:    doc.DocNumber,
		TxnDate:      doc.TxnDate,
		DueDate:      doc.DueDate,
		CurrencyCode: doc.CurrencyCode,
		PrivateNote:  doc.PrivateNote,
		Lines:        make([]quickbooks.PurchaseLine, 0, len(doc.Lines)),
	}
	if doc.VendorCredit {
		txn.DueDate = ""
	}
	for idx := range doc.Lines {
		line := &doc.Lines[idx]
		txn.Lines = append(txn.Lines, quickbooks.PurchaseLine{
			Description: line.Description,
			AccountID:   line.AccountExternalID,
			Amount:      line.Amount,
		})
	}
	return txn
}

func (c *Connector) VoidPurchaseDocument(
	ctx context.Context,
	ref *services.AccountingDocumentRef,
) (*services.AccountingDocumentResult, error) {
	if !ref.Kind.IsBill() {
		return nil, errDocumentKind
	}
	kind, err := purchaseKindOf(ref.Refs)
	if err != nil {
		return nil, err
	}
	client, err := c.client(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(ref.Refs)
	result.ExternalID = ref.ExternalID

	if kind == quickbooks.TxnVendorCredit {
		_, err = client.DeleteVendorCredit(ctx, ref.RequestID, ref.ExternalID)
	} else {
		_, err = client.DeleteBill(ctx, ref.RequestID, ref.ExternalID)
	}
	if err != nil && !quickbooks.IsObjectNotFound(err) {
		return result, err
	}
	return result, nil
}

func purchaseKindOf(refs map[string]string) (quickbooks.TxnKind, error) {
	switch kind := quickbooks.TxnKind(strings.TrimSpace(refs[accountingsync.ExternalRefDocumentType])); kind {
	case "", quickbooks.TxnBill:
		return quickbooks.TxnBill, nil
	case quickbooks.TxnVendorCredit:
		return quickbooks.TxnVendorCredit, nil
	case quickbooks.TxnInvoice,
		quickbooks.TxnCreditMemo,
		quickbooks.TxnPayment,
		quickbooks.TxnBillPayment:
		return "", errPurchaseDocumentType
	default:
		return "", errPurchaseDocumentType
	}
}

func (c *Connector) CreateBillPayment(
	ctx context.Context,
	doc *services.AccountingBillPaymentDocument,
) (*services.AccountingDocumentResult, error) {
	if !doc.Kind.IsBillPayment() {
		return nil, errDocumentKind
	}
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}

	paid, err := client.CreateBillPayment(ctx, doc.RequestID, billPaymentTxnOf(doc))
	if err != nil {
		return nil, err
	}
	return &services.AccountingDocumentResult{
		ExternalID: paid.ID,
		DocNumber:  paid.DocNumber,
		Refs:       map[string]string{accountingsync.ExternalRefDocument: paid.ID},
	}, nil
}

func billPaymentTxnOf(doc *services.AccountingBillPaymentDocument) *quickbooks.BillPaymentTxn {
	return &quickbooks.BillPaymentTxn{
		VendorID:      doc.VendorExternalID,
		BankAccountID: doc.BankAccountExternalID,
		BillID:        doc.BillExternalID,
		DocNumber:     doc.DocNumber,
		TxnDate:       doc.TxnDate,
		CurrencyCode:  doc.CurrencyCode,
		PrivateNote:   doc.PrivateNote,
		Amount:        doc.Amount,
	}
}
