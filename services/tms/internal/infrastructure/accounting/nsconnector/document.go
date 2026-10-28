package nsconnector

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/shopspring/decimal"
)

var _ accountingconnectorports.AccountingDocumentWriter = (*Connector)(nil)

const (
	recordInvoice         = "invoice"
	recordCreditMemo      = "creditMemo"
	recordCustomerPayment = "customerPayment"
	recordVendorBill      = "vendorBill"
	recordVendorCredit    = "vendorCredit"
	recordVendorPayment   = "vendorPayment"
	recordCustomer        = "customer"
	recordVendor          = "vendor"
	refCorrective         = "corrective"
	fieldTranID           = "tranId"
	sublistItem           = "item"
	sublistExpense        = "expense"
	customerPagePath      = "/app/common/entity/custjob.nl"
	vendorPagePath        = "/app/common/entity/vendor.nl"
)

var (
	errDocumentKind = errors.New(
		"netsuite: this record type is not a document NetSuite holds",
	)
	errExternalID = errors.New(
		"netsuite: changing a document needs the id NetSuite gave it",
	)
	errRequestID     = errors.New("netsuite: a write needs the request id to key its external id")
	errLinesRequired = errors.New("netsuite: a document needs at least one line")
	errAmountInvalid = errors.New("netsuite: the amount must be positive")
)

func (c *Connector) DocumentLimits() accountingconnectorports.AccountingDocumentLimits {
	return accountingconnectorports.AccountingDocumentLimits{
		MaxDocNumberLength:      maxTranIDLength,
		SupportsDebitMemo:       false,
		CanVoidCreditMemo:       true,
		CanVoidPurchaseDocument: true,
		IdempotencyWindow:       0,
	}
}

func (c *Connector) DocumentURL(
	auth accountingconnectorports.AccountingDocumentAuth,
	link accountingconnectorports.AccountingDocumentLink,
) string {
	ref, err := netsuite.ParseCompanyRef(auth.RealmID)
	if err != nil {
		return ""
	}
	id := strings.TrimSpace(link.ExternalID)
	switch {
	case link.Kind == accountingsync.SyncObjectCustomer:
		return entityLink(ref.AccountID, customerPagePath, id)
	case link.Kind.IsVendor():
		return entityLink(ref.AccountID, vendorPagePath, id)
	case link.Kind.IsLedger():
		return ""
	default:
		return netsuite.TransactionLink(ref.AccountID, id)
	}
}

func entityLink(accountID, path, id string) string {
	host, err := netsuite.AppHost(accountID)
	if _, ok := parseInternalID(id); err != nil || !ok {
		return ""
	}
	return "https://" + host + path + "?id=" + url.QueryEscape(id)
}

type recordWrite struct {
	recordType string
	externalID string
	body       any
}

type written struct {
	id     string
	number string
}

func writeRecord(
	ctx context.Context,
	client *netsuite.Client,
	w *recordWrite,
) (*written, error) {
	externalID := strings.TrimSpace(w.externalID)
	if externalID == "" {
		return nil, errRequestID
	}
	id, err := client.Upsert(ctx, netsuite.RecordWrite{
		Type:       w.recordType,
		ExternalID: externalID,
		Body:       w.body,
	})
	if err != nil {
		return nil, err
	}
	number, err := tranIDOf(ctx, client, w.recordType, id)
	return &written{id: id, number: number}, err
}

type recordUpdate struct {
	recordType string
	id         string
	body       any
	replace    []string
}

func patchRecord(
	ctx context.Context,
	client *netsuite.Client,
	u *recordUpdate,
) (*written, error) {
	if err := client.Patch(ctx, netsuite.RecordPatch{
		Type:    u.recordType,
		ID:      u.id,
		Body:    u.body,
		Replace: u.replace,
	}); err != nil {
		return nil, err
	}
	number, err := tranIDOf(ctx, client, u.recordType, u.id)
	return &written{id: u.id, number: number}, err
}

func tranIDOf(
	ctx context.Context,
	client *netsuite.Client,
	recordType, id string,
) (string, error) {
	var out struct {
		TranID string `json:"tranId"`
	}
	if err := client.Get(ctx, netsuite.RecordRead{
		Type:   recordType,
		ID:     id,
		Fields: []string{fieldTranID},
	}, &out); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.TranID), nil
}

func salesRecordOf(kind accountingsync.SyncObjectType) (string, error) {
	if kind == accountingsync.SyncObjectInvoice || kind == accountingsync.SyncObjectDebitMemo {
		return recordInvoice, nil
	}
	if kind == accountingsync.SyncObjectCreditMemo {
		return recordCreditMemo, nil
	}
	return "", errDocumentKind
}

func (c *Connector) UpsertCustomer(
	ctx context.Context,
	doc *accountingconnectorports.AccountingCustomerDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	return c.upsertParty(ctx, &partyWrite{
		auth:       doc.Auth,
		kind:       partyCustomer,
		requestID:  doc.RequestID,
		externalID: doc.ExternalID,
		party:      &doc.Party,
	})
}

func (c *Connector) CreateSalesDocument(
	ctx context.Context,
	doc *accountingconnectorports.AccountingSalesDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	recordType, err := salesRecordOf(doc.Kind)
	if err != nil {
		return nil, err
	}
	s, err := c.open(doc.Auth)
	if err != nil {
		return nil, err
	}
	body, err := c.salesBody(ctx, s, recordType, doc)
	if err != nil {
		return nil, err
	}
	result := newResult(doc.Refs)
	saved, err := writeRecord(ctx, s.client, &recordWrite{
		recordType: recordType,
		externalID: doc.RequestID,
		body:       body,
	})
	finishSales(result, saved, body)
	return result, err
}

func (c *Connector) UpdateSalesDocument(
	ctx context.Context,
	doc *accountingconnectorports.AccountingSalesDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	recordType, err := salesRecordOf(doc.Kind)
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(doc.ExternalID)
	if id == "" {
		return nil, errExternalID
	}
	s, err := c.open(doc.Auth)
	if err != nil {
		return nil, err
	}
	body, err := c.salesBody(ctx, s, recordType, doc)
	if err != nil {
		return nil, err
	}
	result := newResult(doc.Refs)
	saved, err := patchRecord(ctx, s.client, &recordUpdate{
		recordType: recordType,
		id:         id,
		body:       body,
		replace:    []string{sublistItem},
	})
	finishSales(result, saved, body)
	return result, err
}

func finishSales(
	result *accountingconnectorports.AccountingDocumentResult,
	saved *written,
	body *transactionBody,
) {
	if saved == nil {
		return
	}
	result.ExternalID = saved.id
	result.DocNumber = saved.number
	result.Refs[accountingsync.ExternalRefDocument] = saved.id
	if body.Apply != nil && len(body.Apply.Items) > 0 {
		result.Refs[accountingsync.ExternalRefApplication] = body.Apply.Items[0].Doc.ID
	}
}

func (c *Connector) salesBody(
	ctx context.Context,
	s *session,
	recordType string,
	doc *accountingconnectorports.AccountingSalesDocument,
) (*transactionBody, error) {
	lines, err := salesLines(doc.Lines)
	if err != nil {
		return nil, err
	}
	currency, err := c.currencyFor(ctx, s, doc.CurrencyCode, doc.ExchangeRate)
	if err != nil {
		return nil, err
	}
	body := &transactionBody{
		Entity:      refOf(doc.CustomerExternalID),
		Subsidiary:  s.subsidiary(),
		TranDate:    doc.TxnDate,
		OtherRefNum: cleanText(doc.DocNumber, maxTranIDLength),
		Memo:        cleanText(doc.PrivateNote, maxMemoLength),
		Message:     cleanText(doc.CustomerMemo, maxMemoLength),
		Item:        &sublist[itemLine]{Items: lines},
	}
	body.setCurrency(currency)
	if recordType == recordInvoice {
		body.DueDate = doc.DueDate
		body.Terms = refOf(doc.TermExternalID)
	}
	invoiceID := strings.TrimSpace(doc.ApplyToExternalID)
	if recordType == recordCreditMemo && invoiceID != "" && doc.ApplyAmount.IsPositive() {
		body.Apply = &sublist[applyLine]{Items: []applyLine{{
			Doc:    recordRef{ID: invoiceID},
			Apply:  true,
			Amount: amountOf(doc.ApplyAmount),
		}}}
	}
	return body, nil
}

func salesLines(lines []accountingconnectorports.AccountingDocumentLine) ([]itemLine, error) {
	if len(lines) == 0 {
		return nil, errLinesRequired
	}
	out := make([]itemLine, 0, len(lines))
	for idx := range lines {
		line := &lines[idx]
		purpose := linePurpose(idx, line.Description)
		item := strings.TrimSpace(line.ItemExternalID)
		if item == "" {
			return nil, unmappedItem(purpose)
		}
		if line.Amount.IsNegative() {
			return nil, negativeLine(purpose)
		}
		quantity, rate := lineQuantity(line)
		out = append(out, itemLine{
			Item:        recordRef{ID: item},
			Description: cleanText(lineText(line.Description), maxLineTextLength),
			Quantity:    amount(quantity),
			Rate:        amount(rate),
			Amount:      amount(line.Amount),
		})
	}
	return out, nil
}

func lineQuantity(
	line *accountingconnectorports.AccountingDocumentLine,
) (quantity, rate decimal.Decimal) {
	if line.Quantity.IsPositive() && !line.UnitPrice.IsNegative() &&
		sameMoney(line.Quantity.Mul(line.UnitPrice), line.Amount) {
		return line.Quantity, line.UnitPrice
	}
	return decimal.NewFromInt(1), line.Amount
}

func (c *Connector) VoidSalesDocument(
	ctx context.Context,
	ref *accountingconnectorports.AccountingDocumentRef,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	recordType, err := salesRecordOf(ref.Kind)
	if err != nil {
		return nil, err
	}
	s, err := c.open(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(ref.Refs)
	result.ExternalID = ref.ExternalID
	var corrective string
	if recordType == recordCreditMemo {
		corrective, err = c.unwindRecord(ctx, s, &unwind{
			recordType: recordCreditMemo,
			id:         ref.ExternalID,
			locked:     salesLocked,
			refused:    func() error { return creditReversalRefused("credit memo") },
			offset:     invoiceOffset(ref.RequestID),
		})
	} else {
		corrective, err = c.retireDocument(ctx, s, &reversal{
			recordType:  recordInvoice,
			reverseType: recordCreditMemo,
			reverseCode: typeCreditMemo,
			id:          ref.ExternalID,
			requestID:   ref.RequestID,
			locked:      salesLocked,
		})
	}
	if corrective != "" {
		result.Refs[refCorrective] = corrective
	}
	return result, err
}
