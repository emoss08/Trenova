package qboconnector

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/quickbookssync"
	"github.com/emoss08/trenova/shared/quickbookstxn"
)

var _ accountingconnectorports.AccountingDocumentReader = (*Connector)(nil)

var errExternalIDRequired = errors.New(
	"quickbooks: updating a document needs the id QuickBooks gave it",
)

func (c *Connector) DocumentReadLimits() accountingconnectorports.AccountingDocumentReadLimits {
	return accountingconnectorports.AccountingDocumentReadLimits{
		MaxPerRead: quickbookssync.MaxQueryIDs,
	}
}

func readKindOf(
	kind accountingsync.SyncObjectType,
	target *accountingconnectorports.AccountingDocumentTarget,
) (quickbookstxn.TxnKind, error) {
	switch {
	case kind == accountingsync.SyncObjectInvoice || kind == accountingsync.SyncObjectDebitMemo:
		return quickbookstxn.TxnInvoice, nil
	case kind == accountingsync.SyncObjectCreditMemo:
		return quickbookstxn.TxnCreditMemo, nil
	case kind == accountingsync.SyncObjectCustomerPayment ||
		kind == accountingsync.SyncObjectCreditApplication:
		return quickbookstxn.TxnPayment, nil
	case kind.IsBill():
		return purchaseKindOf(target.Refs)
	case kind.IsBillPayment():
		return quickbookstxn.TxnBillPayment, nil
	case kind.IsLedger():
		return quickbookstxn.TxnJournalEntry, nil
	default:
		return "", errDocumentKind
	}
}

func (c *Connector) ReadDocuments(
	ctx context.Context,
	req *accountingconnectorports.ReadAccountingDocumentsRequest,
) ([]*accountingconnectorports.AccountingDocumentState, error) {
	groups := make(map[quickbookstxn.TxnKind][]string, 2)
	order := make([]quickbookstxn.TxnKind, 0, 2)
	for idx := range req.Targets {
		kind, err := readKindOf(req.Kind, &req.Targets[idx])
		if err != nil {
			return nil, err
		}
		if _, seen := groups[kind]; !seen {
			order = append(order, kind)
		}
		groups[kind] = append(groups[kind], strings.TrimSpace(req.Targets[idx].ExternalID))
	}

	client, err := c.client(req.Auth)
	if err != nil {
		return nil, err
	}
	found := make(map[quickbookstxn.TxnKind]map[string]*quickbookssync.DocumentState, len(order))
	for _, kind := range order {
		byID := make(map[string]*quickbookssync.DocumentState, len(groups[kind]))
		ids := groups[kind]
		for start := 0; start < len(ids); start += quickbookssync.MaxQueryIDs {
			states, readErr := quickbookssync.QueryByIDs(
				ctx,
				client,
				kind,
				ids[start:min(start+quickbookssync.MaxQueryIDs, len(ids))],
			)
			if readErr != nil {
				return nil, readErr
			}
			for idx := range states {
				byID[states[idx].ID] = &states[idx]
			}
		}
		found[kind] = byID
	}

	out := make([]*accountingconnectorports.AccountingDocumentState, 0, len(req.Targets))
	for idx := range req.Targets {
		target := &req.Targets[idx]
		kind, _ := readKindOf(req.Kind, target)
		id := strings.TrimSpace(target.ExternalID)
		state := &accountingconnectorports.AccountingDocumentState{ExternalID: id}
		if provider, ok := found[kind][id]; ok {
			state.Found = true
			state.Voided = provider.Voided
			state.DocNumber = provider.DocNumber
			state.Total = provider.TotalAmount
			state.Balance = provider.Balance
			state.CurrencyCode = provider.CurrencyCode
			state.ModifiedAt = provider.LastUpdatedAt
			state.ModifiedBy = provider.LastModifiedBy
		}
		out = append(out, state)
	}
	return out, nil
}

func (c *Connector) UpdateSalesDocument(
	ctx context.Context,
	doc *accountingconnectorports.AccountingSalesDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	id := strings.TrimSpace(doc.ExternalID)
	if id == "" {
		return nil, errExternalIDRequired
	}
	var update func(context.Context, string, string, *quickbookstxn.SalesTxn) (*quickbookstxn.TxnResult, error)
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	switch doc.Kind {
	case accountingsync.SyncObjectInvoice, accountingsync.SyncObjectDebitMemo:
		update = func(ctx context.Context, requestID, id string, txn *quickbookstxn.SalesTxn) (*quickbookstxn.TxnResult, error) {
			return quickbookstxn.UpdateInvoice(ctx, client, requestID, id, txn)
		}
	case accountingsync.SyncObjectCreditMemo:
		update = func(ctx context.Context, requestID, id string, txn *quickbookstxn.SalesTxn) (*quickbookstxn.TxnResult, error) {
			return quickbookstxn.UpdateCreditMemo(ctx, client, requestID, id, txn)
		}
	case accountingsync.SyncObjectCustomer,
		accountingsync.SyncObjectCustomerPayment,
		accountingsync.SyncObjectCreditApplication,
		accountingsync.SyncObjectCarrierVendor,
		accountingsync.SyncObjectDriverVendor,
		accountingsync.SyncObjectCarrierBill,
		accountingsync.SyncObjectCarrierBillPay,
		accountingsync.SyncObjectDriverBill,
		accountingsync.SyncObjectDriverBillPay:
		return nil, errDocumentKind
	default:
		return nil, errDocumentKind
	}
	saved, err := update(ctx, doc.RequestID, id, salesTxnOf(doc))
	if err != nil {
		return nil, err
	}
	result := newResult(doc.Refs)
	result.ExternalID = saved.ID
	result.DocNumber = saved.DocNumber
	result.Refs[accountingsync.ExternalRefDocument] = saved.ID
	return result, nil
}

func (c *Connector) UpdatePurchaseDocument(
	ctx context.Context,
	doc *accountingconnectorports.AccountingPurchaseDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if !doc.Kind.IsBill() {
		return nil, errDocumentKind
	}
	id := strings.TrimSpace(doc.ExternalID)
	if id == "" {
		return nil, errExternalIDRequired
	}
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}

	txn := purchaseTxnOf(doc)
	kind := quickbookstxn.TxnBill
	var saved *quickbookstxn.TxnResult
	if doc.VendorCredit {
		kind = quickbookstxn.TxnVendorCredit
		saved, err = quickbookstxn.UpdateVendorCredit(ctx, client, doc.RequestID, id, txn)
	} else {
		saved, err = quickbookstxn.UpdateBill(ctx, client, doc.RequestID, id, txn)
	}
	if err != nil {
		return nil, err
	}
	return &accountingconnectorports.AccountingDocumentResult{
		ExternalID: saved.ID,
		DocNumber:  saved.DocNumber,
		Refs: map[string]string{
			accountingsync.ExternalRefDocument:     saved.ID,
			accountingsync.ExternalRefDocumentType: string(kind),
			accountingsync.ExternalRefURL:          c.appURL(kind.AppPath(), saved.ID),
		},
	}, nil
}

func (c *Connector) UpdateBillPayment(
	ctx context.Context,
	doc *accountingconnectorports.AccountingBillPaymentDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if !doc.Kind.IsBillPayment() {
		return nil, errDocumentKind
	}
	id := strings.TrimSpace(doc.ExternalID)
	if id == "" {
		return nil, errExternalIDRequired
	}
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	paid, err := quickbookstxn.UpdateBillPayment(
		ctx,
		client,
		doc.RequestID,
		id,
		billPaymentTxnOf(doc),
	)
	if err != nil {
		return nil, err
	}
	return &accountingconnectorports.AccountingDocumentResult{
		ExternalID: paid.ID,
		DocNumber:  paid.DocNumber,
		Refs:       map[string]string{accountingsync.ExternalRefDocument: paid.ID},
	}, nil
}

func changedDocumentTypes(kind quickbookstxn.TxnKind) []accountingsync.SyncObjectType {
	switch kind {
	case quickbookstxn.TxnInvoice:
		return []accountingsync.SyncObjectType{
			accountingsync.SyncObjectInvoice,
			accountingsync.SyncObjectDebitMemo,
		}
	case quickbookstxn.TxnCreditMemo:
		return []accountingsync.SyncObjectType{accountingsync.SyncObjectCreditMemo}
	case quickbookstxn.TxnBill, quickbookstxn.TxnVendorCredit:
		return []accountingsync.SyncObjectType{
			accountingsync.SyncObjectCarrierBill,
			accountingsync.SyncObjectDriverBill,
		}
	case quickbookstxn.TxnPayment, quickbookstxn.TxnBillPayment:
		return nil
	default:
		return nil
	}
}

func changedDocumentOf(
	doc *quickbookssync.ChangedDocument,
) accountingconnectorports.AccountingChangedDocument {
	return accountingconnectorports.AccountingChangedDocument{
		ObjectTypes: changedDocumentTypes(doc.Kind),
		ExternalID:  doc.ID,
		Operation:   changeOperation(&doc.ChangeMeta),
		ModifiedAt:  doc.LastUpdatedAt,
		ModifiedBy:  doc.LastModifiedBy,
	}
}

func documentChangeEntities() []quickbookssync.ChangeEntity {
	return []quickbookssync.ChangeEntity{
		quickbookssync.DocumentChangeEntity(quickbookstxn.TxnInvoice),
		quickbookssync.DocumentChangeEntity(quickbookstxn.TxnCreditMemo),
		quickbookssync.DocumentChangeEntity(quickbookstxn.TxnBill),
		quickbookssync.DocumentChangeEntity(quickbookstxn.TxnVendorCredit),
	}
}
