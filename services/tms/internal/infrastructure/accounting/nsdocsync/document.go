package nsdocsync

import (
	"context"
	"net/url"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsrefsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/shopspring/decimal"
)

type Documents struct {
	*nsbinding.Binding
}

const (
	RecordInvoice         = "invoice"
	RecordCreditMemo      = "creditMemo"
	RecordCustomerPayment = "customerPayment"
	RecordVendorBill      = "vendorBill"
	RecordVendorCredit    = "vendorCredit"
	RecordVendorPayment   = "vendorPayment"
	RefCorrective         = "corrective"
	FieldTranID           = "tranId"
	sublistItem           = "item"
	sublistExpense        = "expense"
	customerPagePath      = "/app/common/entity/custjob.nl"
	vendorPagePath        = "/app/common/entity/vendor.nl"
)

func (c *Documents) DocumentLimits() accountingconnectorports.AccountingDocumentLimits {
	return accountingconnectorports.AccountingDocumentLimits{
		MaxDocNumberLength:      maxTranIDLength,
		SupportsDebitMemo:       false,
		CanVoidCreditMemo:       true,
		CanVoidPurchaseDocument: true,
		IdempotencyWindow:       0,
	}
}

func (c *Documents) DocumentURL(
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
	if _, ok := nsbinding.ParseInternalID(id); err != nil || !ok {
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
	ID     string
	number string
}

func writeRecord(
	ctx context.Context,
	client *netsuite.Client,
	w *recordWrite,
) (*written, error) {
	externalID := strings.TrimSpace(w.externalID)
	if externalID == "" {
		return nil, nsbinding.ErrRequestID
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
	return &written{ID: id, number: number}, err
}

type recordUpdate struct {
	recordType string
	ID         string
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
		ID:      u.ID,
		Body:    u.body,
		Replace: u.replace,
	}); err != nil {
		return nil, err
	}
	number, err := tranIDOf(ctx, client, u.recordType, u.ID)
	return &written{ID: u.ID, number: number}, err
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
		Fields: []string{FieldTranID},
	}, &out); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.TranID), nil
}

func salesRecordOf(kind accountingsync.SyncObjectType) (string, error) {
	if kind == accountingsync.SyncObjectInvoice || kind == accountingsync.SyncObjectDebitMemo {
		return RecordInvoice, nil
	}
	if kind == accountingsync.SyncObjectCreditMemo {
		return RecordCreditMemo, nil
	}
	return "", nsbinding.ErrDocumentKind
}

func (c *Documents) UpsertCustomer(
	ctx context.Context,
	doc *accountingconnectorports.AccountingCustomerDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	return c.upsertParty(ctx, &partyWrite{
		auth:       doc.Auth,
		kind:       nsrefsync.PartyCustomer,
		requestID:  doc.RequestID,
		externalID: doc.ExternalID,
		party:      &doc.Party,
	})
}

func (c *Documents) CreateSalesDocument(
	ctx context.Context,
	doc *accountingconnectorports.AccountingSalesDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	recordType, err := salesRecordOf(doc.Kind)
	if err != nil {
		return nil, err
	}
	s, err := c.Open(doc.Auth)
	if err != nil {
		return nil, err
	}
	body, err := c.salesBody(ctx, s, recordType, doc)
	if err != nil {
		return nil, err
	}
	result := newResult(doc.Refs)
	saved, err := writeRecord(ctx, s.Client, &recordWrite{
		recordType: recordType,
		externalID: doc.RequestID,
		body:       body,
	})
	finishSales(result, saved, body)
	return result, err
}

func (c *Documents) UpdateSalesDocument(
	ctx context.Context,
	doc *accountingconnectorports.AccountingSalesDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	recordType, err := salesRecordOf(doc.Kind)
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(doc.ExternalID)
	if id == "" {
		return nil, nsbinding.ErrExternalID
	}
	s, err := c.Open(doc.Auth)
	if err != nil {
		return nil, err
	}
	body, err := c.salesBody(ctx, s, recordType, doc)
	if err != nil {
		return nil, err
	}
	result := newResult(doc.Refs)
	saved, err := patchRecord(ctx, s.Client, &recordUpdate{
		recordType: recordType,
		ID:         id,
		body:       body,
		replace:    []string{sublistItem},
	})
	finishSales(result, saved, body)
	return result, err
}

func finishSales(
	result *accountingconnectorports.AccountingDocumentResult,
	saved *written,
	body *nsbinding.TransactionBody,
) {
	if saved == nil {
		return
	}
	result.ExternalID = saved.ID
	result.DocNumber = saved.number
	result.Refs[accountingsync.ExternalRefDocument] = saved.ID
	if body.Apply != nil && len(body.Apply.Items) > 0 {
		result.Refs[accountingsync.ExternalRefApplication] = body.Apply.Items[0].Doc.ID
	}
}

func (c *Documents) salesBody(
	ctx context.Context,
	s *nsbinding.Session,
	recordType string,
	doc *accountingconnectorports.AccountingSalesDocument,
) (*nsbinding.TransactionBody, error) {
	lines, err := salesLines(doc.Lines)
	if err != nil {
		return nil, err
	}
	currency, err := c.CurrencyFor(ctx, s, doc.CurrencyCode, doc.ExchangeRate)
	if err != nil {
		return nil, err
	}
	body := &nsbinding.TransactionBody{
		Entity:      nsrefsync.RefOf(doc.CustomerExternalID),
		Subsidiary:  s.Subsidiary(),
		TranDate:    doc.TxnDate,
		OtherRefNum: nsrefsync.CleanText(doc.DocNumber, maxTranIDLength),
		Memo:        nsrefsync.CleanText(doc.PrivateNote, maxMemoLength),
		Message:     nsrefsync.CleanText(doc.CustomerMemo, maxMemoLength),
		Item:        &nsbinding.Sublist[nsbinding.ItemLine]{Items: lines},
	}
	body.SetCurrency(currency)
	if recordType == RecordInvoice {
		body.DueDate = doc.DueDate
		body.Terms = nsrefsync.RefOf(doc.TermExternalID)
	}
	invoiceID := strings.TrimSpace(doc.ApplyToExternalID)
	if recordType == RecordCreditMemo && invoiceID != "" && doc.ApplyAmount.IsPositive() {
		body.Apply = &nsbinding.Sublist[nsbinding.ApplyLine]{Items: []nsbinding.ApplyLine{{
			Doc:    nsbinding.RecordRef{ID: invoiceID},
			Apply:  true,
			Amount: nsbinding.AmountOf(doc.ApplyAmount),
		}}}
	}
	return body, nil
}

func salesLines(
	lines []accountingconnectorports.AccountingDocumentLine,
) ([]nsbinding.ItemLine, error) {
	if len(lines) == 0 {
		return nil, nsbinding.ErrLinesRequired
	}
	out := make([]nsbinding.ItemLine, 0, len(lines))
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
		out = append(out, nsbinding.ItemLine{
			Item: nsbinding.RecordRef{ID: item},
			Description: nsrefsync.CleanText(
				lineText(line.Description),
				nsrefsync.MaxLineTextLength,
			),
			Quantity: nsbinding.Amount(quantity),
			Rate:     nsbinding.Amount(rate),
			Amount:   nsbinding.Amount(line.Amount),
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

func (c *Documents) VoidSalesDocument(
	ctx context.Context,
	ref *accountingconnectorports.AccountingDocumentRef,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	recordType, err := salesRecordOf(ref.Kind)
	if err != nil {
		return nil, err
	}
	s, err := c.Open(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(ref.Refs)
	result.ExternalID = ref.ExternalID
	var corrective string
	if recordType == RecordCreditMemo {
		corrective, err = c.unwindRecord(ctx, s, &unwind{
			recordType: RecordCreditMemo,
			ID:         ref.ExternalID,
			locked:     salesLocked,
			refused:    func() error { return creditReversalRefused("credit memo") },
			offset:     invoiceOffset(ref.RequestID),
		})
	} else {
		corrective, err = c.retireDocument(ctx, s, &Reversal{
			recordType:  RecordInvoice,
			reverseType: RecordCreditMemo,
			reverseCode: TypeCreditMemo,
			ID:          ref.ExternalID,
			requestID:   ref.RequestID,
			locked:      salesLocked,
		})
	}
	if corrective != "" {
		result.Refs[RefCorrective] = corrective
	}
	return result, err
}
