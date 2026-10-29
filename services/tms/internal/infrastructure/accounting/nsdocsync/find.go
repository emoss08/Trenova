package nsdocsync

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsrefsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/netsuite"
)

func findTypeOf(
	req *accountingconnectorports.AccountingFindDocumentRequest,
) (typeCode string, searchable bool, err error) {
	kind := req.Kind
	switch {
	case kind == accountingsync.SyncObjectCreditApplication:
		return "", false, nil
	case kind == accountingsync.SyncObjectInvoice || kind == accountingsync.SyncObjectDebitMemo:
		return TypeInvoice, true, nil
	case kind == accountingsync.SyncObjectCreditMemo:
		return TypeCreditMemo, true, nil
	case kind == accountingsync.SyncObjectCustomerPayment:
		return TypeCustomerPayment, true, nil
	case kind.IsBill() && req.Credit:
		return TypeVendorCredit, true, nil
	case kind.IsBill():
		return TypeVendorBill, true, nil
	case kind.IsBillPayment():
		return TypeVendorPayment, true, nil
	default:
		return "", false, nsbinding.ErrDocumentKind
	}
}

func (c *Documents) FindDocument(
	ctx context.Context,
	req *accountingconnectorports.AccountingFindDocumentRequest,
) (*accountingconnectorports.AccountingDocumentResult, bool, error) {
	typeCode, searchable, err := findTypeOf(req)
	if err != nil || !searchable {
		return nil, false, err
	}
	s, err := c.Open(req.Auth)
	if err != nil {
		return nil, false, err
	}
	row, ok, err := findByExternalID(ctx, s.Client, typeCode, strings.TrimSpace(req.RequestID))
	if err != nil {
		return nil, false, err
	}
	if !ok {
		row, ok, err = findByNumber(ctx, s.Client, typeCode, req)
		if err != nil || !ok {
			return nil, false, err
		}
	}
	return foundResult(req, row), true, nil
}

func findByNumber(
	ctx context.Context,
	client *netsuite.Client,
	typeCode string,
	req *accountingconnectorports.AccountingFindDocumentRequest,
) (*nsbinding.TransactionRow, bool, error) {
	number := nsrefsync.CleanText(req.DocNumber, maxTranIDLength)
	if number == "" {
		return nil, false, nil
	}
	b := (&nsbinding.SQLBuilder{}).
		Where("t.type = ?", typeCode).
		Where("(t.tranid = ? OR t.otherrefnum = ?)", number, number).
		Where("t.voided = 'F'")
	if party := strings.TrimSpace(req.CounterpartyExternalID); party != "" {
		partyID, ok := nsbinding.ParseInternalID(party)
		if !ok {
			return nil, false, nil
		}
		b.Where("t.entity = ?", partyID)
	}
	var rows []nsbinding.TransactionRow
	if _, err := client.Query(ctx, netsuite.Query{
		SQL:    b.SQL(TransactionSQL, "t.id"),
		Params: b.Params,
		Limit:  findMatchLimit,
	}, &rows); err != nil {
		return nil, false, err
	}
	for idx := range rows {
		row := &rows[idx]
		if !req.Total.IsZero() && !sameMoney(row.AbsTotal(), req.Total.Abs()) {
			continue
		}
		return row, true, nil
	}
	return nil, false, nil
}

func foundResult(
	req *accountingconnectorports.AccountingFindDocumentRequest,
	row *nsbinding.TransactionRow,
) *accountingconnectorports.AccountingDocumentResult {
	saved := &written{ID: row.IDValue(), number: row.TranID.Value()}
	result := newResult(nil)
	switch {
	case req.Kind.IsBill():
		finishPurchase(&purchaseWritten{
			auth:   req.Auth,
			credit: req.Credit,
			saved:  saved,
			result: result,
		})
	case req.Kind == accountingsync.SyncObjectCustomerPayment:
		finishPayment(result, saved, DocTypeCustomerPayment)
	case req.Kind.IsBillPayment():
		finishPayment(result, saved, DocTypeVendorPayment)
	default:
		result.ExternalID = saved.ID
		result.DocNumber = saved.number
		result.Refs[accountingsync.ExternalRefDocument] = saved.ID
	}
	return result
}
