package bcconnector

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/businesscentral"
)

func (c *Connector) FindDocument(
	ctx context.Context,
	req *services.AccountingFindDocumentRequest,
) (*services.AccountingDocumentResult, bool, error) {
	kind := req.Kind
	if !kind.IsSalesDocument() && !kind.IsBill() && !kind.IsBillPayment() &&
		kind != accountingsync.SyncObjectCustomerPayment &&
		kind != accountingsync.SyncObjectCreditApplication {
		return nil, false, errDocumentKind
	}
	if kind == accountingsync.SyncObjectCreditApplication {
		return nil, false, nil
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
	case kind.IsSalesDocument():
		found, ok, err = findSales(ctx, client, req)
	case kind.IsBill():
		found, ok, err = c.findPurchase(ctx, client, req)
	default:
		found, ok, err = findPayment(ctx, client, req)
	}
	if errors.Is(err, businesscentral.ErrInvalidFilter) ||
		errors.Is(err, businesscentral.ErrInvalidID) {
		return nil, false, nil
	}
	if err != nil || !ok {
		return nil, false, err
	}
	return found, true, nil
}

func findPosted(
	ctx context.Context,
	client *businesscentral.Client,
	kind businesscentral.DocumentKind,
	req *services.AccountingFindDocumentRequest,
) (*businesscentral.Document, bool, error) {
	listed, err := client.FindDocuments(
		ctx,
		kind,
		docReference(req.DocNumber, req.RequestID),
		strings.TrimSpace(req.CounterpartyExternalID),
	)
	if err != nil {
		return nil, false, err
	}
	for idx := range listed {
		doc := &listed[idx]
		if doc.IsDraft() || retired(doc.Status) {
			continue
		}
		if !req.Total.IsZero() && !sameMoney(doc.TotalAmountIncludingTax, req.Total.Abs()) {
			continue
		}
		return doc, true, nil
	}
	return nil, false, nil
}

func findSales(
	ctx context.Context,
	client *businesscentral.Client,
	req *services.AccountingFindDocumentRequest,
) (*services.AccountingDocumentResult, bool, error) {
	kind, err := salesKindOf(req.Kind)
	if err != nil {
		return nil, false, err
	}
	doc, ok, err := findPosted(ctx, client, kind, req)
	if err != nil || !ok {
		return nil, false, err
	}
	return &services.AccountingDocumentResult{
		ExternalID: doc.ID,
		DocNumber:  doc.Number,
		Refs:       map[string]string{accountingsync.ExternalRefDocument: doc.ID},
	}, true, nil
}

func (c *Connector) findPurchase(
	ctx context.Context,
	client *businesscentral.Client,
	req *services.AccountingFindDocumentRequest,
) (*services.AccountingDocumentResult, bool, error) {
	kind := purchaseKindOf(req.Credit)
	doc, ok, err := findPosted(ctx, client, kind, req)
	if err != nil || !ok {
		return nil, false, err
	}
	result := newResult(nil)
	c.finishPurchase(&purchaseWritten{auth: req.Auth, kind: kind, doc: doc, result: result})
	return result, true, nil
}

func findPayment(
	ctx context.Context,
	client *businesscentral.Client,
	req *services.AccountingFindDocumentRequest,
) (*services.AccountingDocumentResult, bool, error) {
	if strings.TrimSpace(req.RequestID) == "" {
		return nil, false, nil
	}
	number := requestReference(req.RequestID)
	entries, err := client.GeneralLedgerEntries(ctx, number)
	if err != nil || len(entries) == 0 {
		return nil, false, err
	}
	docType := docTypeCustomerPayment
	if req.Kind.IsBillPayment() {
		docType = docTypeVendorPayment
	}
	result := newResult(nil)
	finishPayment(result, number, docType)
	return result, true, nil
}
