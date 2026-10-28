package nsconnector

import (
	"context"
	"strings"

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
		return typeInvoice, true, nil
	case kind == accountingsync.SyncObjectCreditMemo:
		return typeCreditMemo, true, nil
	case kind == accountingsync.SyncObjectCustomerPayment:
		return typeCustomerPayment, true, nil
	case kind.IsBill() && req.Credit:
		return typeVendorCredit, true, nil
	case kind.IsBill():
		return typeVendorBill, true, nil
	case kind.IsBillPayment():
		return typeVendorPayment, true, nil
	default:
		return "", false, errDocumentKind
	}
}

func (c *Connector) FindDocument(
	ctx context.Context,
	req *accountingconnectorports.AccountingFindDocumentRequest,
) (*accountingconnectorports.AccountingDocumentResult, bool, error) {
	typeCode, searchable, err := findTypeOf(req)
	if err != nil || !searchable {
		return nil, false, err
	}
	s, err := c.open(req.Auth)
	if err != nil {
		return nil, false, err
	}
	row, ok, err := findByExternalID(ctx, s.client, typeCode, strings.TrimSpace(req.RequestID))
	if err != nil {
		return nil, false, err
	}
	if !ok {
		row, ok, err = findByNumber(ctx, s.client, typeCode, req)
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
) (*transactionRow, bool, error) {
	number := cleanText(req.DocNumber, maxTranIDLength)
	if number == "" {
		return nil, false, nil
	}
	b := (&sqlBuilder{}).
		where("t.type = ?", typeCode).
		where("(t.tranid = ? OR t.otherrefnum = ?)", number, number).
		where("t.voided = 'F'")
	if party := strings.TrimSpace(req.CounterpartyExternalID); party != "" {
		partyID, ok := parseInternalID(party)
		if !ok {
			return nil, false, nil
		}
		b.where("t.entity = ?", partyID)
	}
	var rows []transactionRow
	if _, err := client.Query(ctx, netsuite.Query{
		SQL:    b.sql(transactionSQL, "t.id"),
		Params: b.params,
		Limit:  findMatchLimit,
	}, &rows); err != nil {
		return nil, false, err
	}
	for idx := range rows {
		row := &rows[idx]
		if !req.Total.IsZero() && !sameMoney(row.total(), req.Total.Abs()) {
			continue
		}
		return row, true, nil
	}
	return nil, false, nil
}

func foundResult(
	req *accountingconnectorports.AccountingFindDocumentRequest,
	row *transactionRow,
) *accountingconnectorports.AccountingDocumentResult {
	saved := &written{id: row.id(), number: row.TranID.Value()}
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
		finishPayment(result, saved, docTypeCustomerPayment)
	case req.Kind.IsBillPayment():
		finishPayment(result, saved, docTypeVendorPayment)
	default:
		result.ExternalID = saved.id
		result.DocNumber = saved.number
		result.Refs[accountingsync.ExternalRefDocument] = saved.id
	}
	return result
}
