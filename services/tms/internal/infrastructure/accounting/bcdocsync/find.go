package bcdocsync

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/bcsession"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/businesscentraldocs"
	"github.com/emoss08/trenova/shared/businesscentralerrors"
	"github.com/emoss08/trenova/shared/businesscentralpay"
)

func (c *Documents) FindDocument(
	ctx context.Context,
	req *accountingconnectorports.AccountingFindDocumentRequest,
) (*accountingconnectorports.AccountingDocumentResult, bool, error) {
	kind := req.Kind
	if !kind.IsSalesDocument() && !kind.IsBill() && !kind.IsBillPayment() &&
		kind != accountingsync.SyncObjectCustomerPayment &&
		kind != accountingsync.SyncObjectCreditApplication {
		return nil, false, bcsession.ErrDocumentKind
	}
	if kind == accountingsync.SyncObjectCreditApplication {
		return nil, false, nil
	}
	client, err := c.Client(req.Auth)
	if err != nil {
		return nil, false, err
	}

	var (
		found *accountingconnectorports.AccountingDocumentResult
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
	if errors.Is(err, businesscentralerrors.ErrInvalidFilter) ||
		errors.Is(err, businesscentralerrors.ErrInvalidID) {
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
	kind businesscentraldocs.DocumentKind,
	req *accountingconnectorports.AccountingFindDocumentRequest,
) (*businesscentraldocs.Document, bool, error) {
	listed, err := businesscentraldocs.FindDocuments(
		ctx,
		client,
		kind,
		DocReference(req.DocNumber, req.RequestID),
		strings.TrimSpace(req.CounterpartyExternalID),
	)
	if err != nil {
		return nil, false, err
	}
	for idx := range listed {
		doc := &listed[idx]
		if doc.IsDraft() || Retired(doc.Status) {
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
	req *accountingconnectorports.AccountingFindDocumentRequest,
) (*accountingconnectorports.AccountingDocumentResult, bool, error) {
	kind, err := SalesKindOf(req.Kind)
	if err != nil {
		return nil, false, err
	}
	doc, ok, err := findPosted(ctx, client, kind, req)
	if err != nil || !ok {
		return nil, false, err
	}
	return &accountingconnectorports.AccountingDocumentResult{
		ExternalID: doc.ID,
		DocNumber:  doc.Number,
		Refs:       map[string]string{accountingsync.ExternalRefDocument: doc.ID},
	}, true, nil
}

func (c *Documents) findPurchase(
	ctx context.Context,
	client *businesscentral.Client,
	req *accountingconnectorports.AccountingFindDocumentRequest,
) (*accountingconnectorports.AccountingDocumentResult, bool, error) {
	kind := PurchaseKindOf(req.Credit)
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
	req *accountingconnectorports.AccountingFindDocumentRequest,
) (*accountingconnectorports.AccountingDocumentResult, bool, error) {
	if strings.TrimSpace(req.RequestID) == "" {
		return nil, false, nil
	}
	number := RequestReference(req.RequestID)
	entries, err := businesscentralpay.GeneralLedgerEntries(ctx, client, number)
	if err != nil || len(entries) == 0 {
		return nil, false, err
	}
	docType := DocTypeCustomerPayment
	if req.Kind.IsBillPayment() {
		docType = DocTypeVendorPayment
	}
	result := newResult(nil)
	finishPayment(result, number, docType)
	return result, true, nil
}
