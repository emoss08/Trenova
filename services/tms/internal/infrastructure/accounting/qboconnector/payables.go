package qboconnector

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/quickbooksref"
	"github.com/emoss08/trenova/shared/quickbookstxn"
)

var errPurchaseDocumentType = errors.New(
	"quickbooks: the purchase document type is neither a bill nor a vendor credit",
)

func (c *Connector) UpsertVendor(
	ctx context.Context,
	doc *accountingconnectorports.AccountingVendorDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	draft := partyDraftOf(&doc.Party)
	draft.DisplayName = quickbooksref.SanitizeName(
		draft.DisplayName,
		quickbooksref.MaxPartyNameLength,
	)

	var saved *quickbooksref.ReferenceObject
	if id := strings.TrimSpace(doc.ExternalID); id != "" {
		saved, err = quickbookstxn.UpdateVendor(ctx, client, doc.RequestID, id, draft)
	} else {
		saved, err = quickbooksref.CreateVendor(ctx, client, doc.RequestID, draft)
	}
	if err != nil {
		return nil, err
	}
	return &accountingconnectorports.AccountingDocumentResult{
		ExternalID: saved.ID,
		DocNumber:  saved.Name,
	}, nil
}

func (c *Connector) CreatePurchaseDocument(
	ctx context.Context,
	doc *accountingconnectorports.AccountingPurchaseDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if !doc.Kind.IsBill() {
		return nil, errDocumentKind
	}
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}

	txn := purchaseTxnOf(doc)
	kind := quickbookstxn.TxnBill
	var created *quickbookstxn.TxnResult
	if doc.VendorCredit {
		kind = quickbookstxn.TxnVendorCredit
		created, err = quickbookstxn.CreateVendorCredit(ctx, client, doc.RequestID, txn)
	} else {
		created, err = quickbookstxn.CreateBill(ctx, client, doc.RequestID, txn)
	}
	if err != nil {
		return nil, err
	}

	return &accountingconnectorports.AccountingDocumentResult{
		ExternalID: created.ID,
		DocNumber:  created.DocNumber,
		Refs: map[string]string{
			accountingsync.ExternalRefDocument:     created.ID,
			accountingsync.ExternalRefDocumentType: string(kind),
			accountingsync.ExternalRefCreditDocument: strconv.FormatBool(
				kind == quickbookstxn.TxnVendorCredit,
			),
			accountingsync.ExternalRefURL: c.appURL(kind.AppPath(), created.ID),
		},
	}, nil
}

func purchaseTxnOf(
	doc *accountingconnectorports.AccountingPurchaseDocument,
) *quickbookstxn.PurchaseTxn {
	txn := &quickbookstxn.PurchaseTxn{
		VendorID:     doc.VendorExternalID,
		APAccountID:  doc.APAccountExternalID,
		DocNumber:    doc.DocNumber,
		TxnDate:      doc.TxnDate,
		DueDate:      doc.DueDate,
		CurrencyCode: doc.CurrencyCode,
		ExchangeRate: doc.ExchangeRate,
		PrivateNote:  doc.PrivateNote,
		Lines:        make([]quickbookstxn.PurchaseLine, 0, len(doc.Lines)),
	}
	if doc.VendorCredit {
		txn.DueDate = ""
	}
	for idx := range doc.Lines {
		line := &doc.Lines[idx]
		txn.Lines = append(txn.Lines, quickbookstxn.PurchaseLine{
			Description: line.Description,
			AccountID:   line.AccountExternalID,
			Amount:      line.Amount,
		})
	}
	return txn
}

func (c *Connector) VoidPurchaseDocument(
	ctx context.Context,
	ref *accountingconnectorports.AccountingDocumentRef,
) (*accountingconnectorports.AccountingDocumentResult, error) {
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

	if kind == quickbookstxn.TxnVendorCredit {
		_, err = quickbookstxn.DeleteVendorCredit(ctx, client, ref.RequestID, ref.ExternalID)
	} else {
		_, err = quickbookstxn.DeleteBill(ctx, client, ref.RequestID, ref.ExternalID)
	}
	if err != nil && !quickbookstxn.IsObjectNotFound(err) {
		return result, err
	}
	return result, nil
}

func purchaseKindOf(refs map[string]string) (quickbookstxn.TxnKind, error) {
	switch kind := quickbookstxn.TxnKind(strings.TrimSpace(refs[accountingsync.ExternalRefDocumentType])); kind {
	case "", quickbookstxn.TxnBill:
		return quickbookstxn.TxnBill, nil
	case quickbookstxn.TxnVendorCredit:
		return quickbookstxn.TxnVendorCredit, nil
	case quickbookstxn.TxnInvoice,
		quickbookstxn.TxnCreditMemo,
		quickbookstxn.TxnPayment,
		quickbookstxn.TxnBillPayment,
		quickbookstxn.TxnJournalEntry:
		return "", errPurchaseDocumentType
	default:
		return "", errPurchaseDocumentType
	}
}

func (c *Connector) CreateBillPayment(
	ctx context.Context,
	doc *accountingconnectorports.AccountingBillPaymentDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if !doc.Kind.IsBillPayment() {
		return nil, errDocumentKind
	}
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}

	paid, err := quickbookstxn.CreateBillPayment(ctx, client, doc.RequestID, billPaymentTxnOf(doc))
	if err != nil {
		return nil, err
	}
	return &accountingconnectorports.AccountingDocumentResult{
		ExternalID: paid.ID,
		DocNumber:  paid.DocNumber,
		Refs:       map[string]string{accountingsync.ExternalRefDocument: paid.ID},
	}, nil
}

func billPaymentTxnOf(
	doc *accountingconnectorports.AccountingBillPaymentDocument,
) *quickbookstxn.BillPaymentTxn {
	return &quickbookstxn.BillPaymentTxn{
		VendorID:      doc.VendorExternalID,
		BankAccountID: doc.BankAccountExternalID,
		BillID:        doc.BillExternalID,
		DocNumber:     doc.DocNumber,
		TxnDate:       doc.TxnDate,
		CurrencyCode:  doc.CurrencyCode,
		ExchangeRate:  doc.ExchangeRate,
		PrivateNote:   doc.PrivateNote,
		Amount:        doc.Amount,
	}
}
