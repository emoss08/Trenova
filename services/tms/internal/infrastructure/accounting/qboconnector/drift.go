package qboconnector

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/quickbooks"
)

var _ services.AccountingDocumentReader = (*Connector)(nil)

var errExternalIDRequired = errors.New(
	"quickbooks: updating a document needs the id QuickBooks gave it",
)

func (c *Connector) DocumentReadLimits() services.AccountingDocumentReadLimits {
	return services.AccountingDocumentReadLimits{MaxPerRead: quickbooks.MaxQueryIDs}
}

func readKindOf(
	kind accountingsync.SyncObjectType,
	target *services.AccountingDocumentTarget,
) (quickbooks.TxnKind, error) {
	switch {
	case kind == accountingsync.SyncObjectInvoice || kind == accountingsync.SyncObjectDebitMemo:
		return quickbooks.TxnInvoice, nil
	case kind == accountingsync.SyncObjectCreditMemo:
		return quickbooks.TxnCreditMemo, nil
	case kind == accountingsync.SyncObjectCustomerPayment ||
		kind == accountingsync.SyncObjectCreditApplication:
		return quickbooks.TxnPayment, nil
	case kind.IsBill():
		return purchaseKindOf(target.Refs)
	case kind.IsBillPayment():
		return quickbooks.TxnBillPayment, nil
	default:
		return "", errDocumentKind
	}
}

func (c *Connector) ReadDocuments(
	ctx context.Context,
	req *services.ReadAccountingDocumentsRequest,
) ([]*services.AccountingDocumentState, error) {
	groups := make(map[quickbooks.TxnKind][]string, 2)
	order := make([]quickbooks.TxnKind, 0, 2)
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
	found := make(map[quickbooks.TxnKind]map[string]*quickbooks.DocumentState, len(order))
	for _, kind := range order {
		byID := make(map[string]*quickbooks.DocumentState, len(groups[kind]))
		ids := groups[kind]
		for start := 0; start < len(ids); start += quickbooks.MaxQueryIDs {
			states, readErr := client.QueryByIDs(
				ctx,
				kind,
				ids[start:min(start+quickbooks.MaxQueryIDs, len(ids))],
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

	out := make([]*services.AccountingDocumentState, 0, len(req.Targets))
	for idx := range req.Targets {
		target := &req.Targets[idx]
		kind, _ := readKindOf(req.Kind, target)
		id := strings.TrimSpace(target.ExternalID)
		state := &services.AccountingDocumentState{ExternalID: id}
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
	doc *services.AccountingSalesDocument,
) (*services.AccountingDocumentResult, error) {
	id := strings.TrimSpace(doc.ExternalID)
	if id == "" {
		return nil, errExternalIDRequired
	}
	var update func(context.Context, string, string, *quickbooks.SalesTxn) (*quickbooks.TxnResult, error)
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	switch doc.Kind {
	case accountingsync.SyncObjectInvoice, accountingsync.SyncObjectDebitMemo:
		update = client.UpdateInvoice
	case accountingsync.SyncObjectCreditMemo:
		update = client.UpdateCreditMemo
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
	doc *services.AccountingPurchaseDocument,
) (*services.AccountingDocumentResult, error) {
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
	kind := quickbooks.TxnBill
	var saved *quickbooks.TxnResult
	if doc.VendorCredit {
		kind = quickbooks.TxnVendorCredit
		saved, err = client.UpdateVendorCredit(ctx, doc.RequestID, id, txn)
	} else {
		saved, err = client.UpdateBill(ctx, doc.RequestID, id, txn)
	}
	if err != nil {
		return nil, err
	}
	return &services.AccountingDocumentResult{
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
	doc *services.AccountingBillPaymentDocument,
) (*services.AccountingDocumentResult, error) {
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
	paid, err := client.UpdateBillPayment(ctx, doc.RequestID, id, billPaymentTxnOf(doc))
	if err != nil {
		return nil, err
	}
	return &services.AccountingDocumentResult{
		ExternalID: paid.ID,
		DocNumber:  paid.DocNumber,
		Refs:       map[string]string{accountingsync.ExternalRefDocument: paid.ID},
	}, nil
}

func changedDocumentTypes(kind quickbooks.TxnKind) []accountingsync.SyncObjectType {
	switch kind {
	case quickbooks.TxnInvoice:
		return []accountingsync.SyncObjectType{
			accountingsync.SyncObjectInvoice,
			accountingsync.SyncObjectDebitMemo,
		}
	case quickbooks.TxnCreditMemo:
		return []accountingsync.SyncObjectType{accountingsync.SyncObjectCreditMemo}
	case quickbooks.TxnBill, quickbooks.TxnVendorCredit:
		return []accountingsync.SyncObjectType{
			accountingsync.SyncObjectCarrierBill,
			accountingsync.SyncObjectDriverBill,
		}
	case quickbooks.TxnPayment, quickbooks.TxnBillPayment:
		return nil
	default:
		return nil
	}
}

func changedDocumentOf(doc *quickbooks.ChangedDocument) services.AccountingChangedDocument {
	return services.AccountingChangedDocument{
		ObjectTypes: changedDocumentTypes(doc.Kind),
		ExternalID:  doc.ID,
		Operation:   changeOperation(&doc.ChangeMeta),
		ModifiedAt:  doc.LastUpdatedAt,
		ModifiedBy:  doc.LastModifiedBy,
	}
}

func documentChangeEntities() []quickbooks.ChangeEntity {
	return []quickbooks.ChangeEntity{
		quickbooks.DocumentChangeEntity(quickbooks.TxnInvoice),
		quickbooks.DocumentChangeEntity(quickbooks.TxnCreditMemo),
		quickbooks.DocumentChangeEntity(quickbooks.TxnBill),
		quickbooks.DocumentChangeEntity(quickbooks.TxnVendorCredit),
	}
}
