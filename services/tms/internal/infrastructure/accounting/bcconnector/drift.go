package bcconnector

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/bcdocsync"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/bcsession"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/businesscentraldocs"
	"github.com/google/uuid"
)

var _ accountingconnectorports.AccountingDocumentReader = (*Connector)(nil)

const (
	guidLength        = 36
	compactGUIDLength = 32
)

var readOrder = []businesscentraldocs.DocumentKind{
	businesscentraldocs.DocumentSalesInvoice,
	businesscentraldocs.DocumentSalesCreditMemo,
	businesscentraldocs.DocumentPurchaseInvoice,
	businesscentraldocs.DocumentPurchaseCreditMemo,
}

func (c *Connector) DocumentReadLimits() accountingconnectorports.AccountingDocumentReadLimits {
	return accountingconnectorports.AccountingDocumentReadLimits{
		MaxPerRead: businesscentral.MaxIDsPerRead,
	}
}

func readKindOf(
	kind accountingsync.SyncObjectType,
	target *accountingconnectorports.AccountingDocumentTarget,
) (businesscentraldocs.DocumentKind, bool, error) {
	switch {
	case kind.IsSalesDocument():
		docKind, err := bcdocsync.SalesKindOf(kind)
		return docKind, err == nil, err
	case kind.IsBill():
		credit, err := bcdocsync.PurchaseIsCredit(target.Refs)
		if err != nil {
			return 0, false, err
		}
		return bcdocsync.PurchaseKindOf(credit), true, nil
	case kind == accountingsync.SyncObjectCustomerPayment,
		kind == accountingsync.SyncObjectCreditApplication,
		kind.IsBillPayment():
		return 0, false, nil
	default:
		return 0, false, bcsession.ErrDocumentKind
	}
}

func (c *Connector) ReadDocuments(
	ctx context.Context,
	req *accountingconnectorports.ReadAccountingDocumentsRequest,
) ([]*accountingconnectorports.AccountingDocumentState, error) {
	plan := make(map[businesscentraldocs.DocumentKind][]string, len(readOrder))
	for idx := range req.Targets {
		target := &req.Targets[idx]
		kind, readable, err := readKindOf(req.Kind, target)
		if err != nil {
			return nil, err
		}
		id := strings.TrimSpace(target.ExternalID)
		if !readable || id == "" {
			continue
		}
		plan[kind] = append(plan[kind], id)
	}
	if len(plan) == 0 {
		return []*accountingconnectorports.AccountingDocumentState{}, nil
	}
	client, err := c.Client(req.Auth)
	if err != nil {
		return nil, err
	}
	out := make([]*accountingconnectorports.AccountingDocumentState, 0, len(req.Targets))
	for _, kind := range readOrder {
		states, readErr := readStates(ctx, client, kind, plan[kind])
		if readErr != nil {
			return nil, readErr
		}
		out = append(out, states...)
	}
	return out, nil
}

func readStates(
	ctx context.Context,
	client *businesscentral.Client,
	kind businesscentraldocs.DocumentKind,
	ids []string,
) ([]*accountingconnectorports.AccountingDocumentState, error) {
	valid := make([]string, 0, len(ids))
	for _, id := range ids {
		if readableID(id) {
			valid = append(valid, id)
		}
	}
	found := make(map[string]*businesscentraldocs.Document, len(valid))
	for start := 0; start < len(valid); start += businesscentral.MaxIDsPerRead {
		docs, err := businesscentraldocs.DocumentsByID(
			ctx,
			client,
			kind,
			valid[start:min(start+businesscentral.MaxIDsPerRead, len(valid))],
		)
		if err != nil {
			return nil, err
		}
		for idx := range docs {
			found[strings.ToLower(docs[idx].ID)] = &docs[idx]
		}
	}
	out := make([]*accountingconnectorports.AccountingDocumentState, 0, len(ids))
	for _, id := range ids {
		state := &accountingconnectorports.AccountingDocumentState{ExternalID: id}
		if doc, ok := found[strings.ToLower(id)]; ok {
			balance := doc.RemainingAmount
			state.Found = true
			state.Voided = bcdocsync.Retired(doc.Status)
			state.DocNumber = doc.Number
			state.Total = doc.TotalAmountIncludingTax
			state.Balance = &balance
			state.CurrencyCode = doc.CurrencyCode
			state.ModifiedAt = unixOrZero(doc.LastModified)
		}
		out = append(out, state)
	}
	return out, nil
}

func readableID(id string) bool {
	if len(id) != guidLength && len(id) != compactGUIDLength {
		return false
	}
	_, err := uuid.Parse(id)
	return err == nil
}
