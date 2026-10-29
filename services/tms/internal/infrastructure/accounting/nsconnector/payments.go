package nsconnector

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/emoss08/trenova/shared/netsuiteerrors"
	"github.com/shopspring/decimal"
)

const (
	docTypeCustomerPayment = "customerPayment"
	docTypeVendorPayment   = "vendorPayment"
	shortPayNote           = "Short pay on "
	stepShortPayBase       = 10
)

var errApplyTarget = errors.New("netsuite: an applied amount needs the document it pays")

func (c *Connector) SavePayment(
	ctx context.Context,
	doc *accountingconnectorports.AccountingPaymentDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	cash, err := cashApplications(doc)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(doc.DepositAccountExternalID) == "" {
		return nil, unmappedAccount("The deposit account")
	}
	s, err := c.open(doc.Auth)
	if err != nil {
		return nil, err
	}
	currency, err := c.currencyFor(ctx, s, doc.CurrencyCode, doc.ExchangeRate)
	if err != nil {
		return nil, err
	}
	result := newResult(doc.Refs)
	for idx := range doc.Applications {
		if !doc.Applications[idx].ShortPayAmount.IsPositive() {
			continue
		}
		if err = shortPay(ctx, &shortPayWrite{
			session: s, doc: doc, idx: idx, currency: currency, result: result,
		}); err != nil {
			return result, err
		}
	}

	body := &transactionBody{
		Customer:   refOf(doc.CustomerExternalID),
		Subsidiary: s.subsidiary(),
		TranDate:   doc.TxnDate,
		Account:    refOf(doc.DepositAccountExternalID),
		Payment:    amountOf(doc.TotalAmount),
		CheckNum:   cleanText(doc.ReferenceNumber, maxTranIDLength),
		Memo:       cleanText(doc.PrivateNote, maxMemoLength),
		Apply:      &sublist[applyLine]{Items: cash},
	}
	body.setCurrency(currency)
	saved, err := savePaymentRecord(ctx, s, &paymentSave{
		recordType: recordCustomerPayment,
		id:         doc.ExternalID,
		requestID:  doc.RequestID,
		body:       body,
	})
	finishPayment(result, saved, docTypeCustomerPayment)
	return result, err
}

func cashApplications(
	doc *accountingconnectorports.AccountingPaymentDocument,
) ([]applyLine, error) {
	if !doc.TotalAmount.IsPositive() {
		return nil, errAmountInvalid
	}
	cash := make([]applyLine, 0, len(doc.Applications))
	for idx := range doc.Applications {
		app := &doc.Applications[idx]
		if app.AppliedAmount.IsNegative() || app.ShortPayAmount.IsNegative() {
			return nil, errAmountInvalid
		}
		invoice := strings.TrimSpace(app.InvoiceExternalID)
		if invoice == "" && (app.AppliedAmount.IsPositive() || app.ShortPayAmount.IsPositive()) {
			return nil, errApplyTarget
		}
		if app.AppliedAmount.IsPositive() {
			cash = append(cash, applyLine{
				Doc:    recordRef{ID: invoice},
				Apply:  true,
				Amount: amountOf(app.AppliedAmount),
			})
		}
	}
	return cash, nil
}

func finishPayment(
	result *accountingconnectorports.AccountingDocumentResult,
	saved *written,
	docType string,
) {
	if saved == nil {
		return
	}
	result.ExternalID = saved.id
	result.DocNumber = saved.number
	result.Refs[accountingsync.ExternalRefDocument] = saved.id
	result.Refs[accountingsync.ExternalRefDocumentType] = docType
}

type paymentSave struct {
	recordType string
	id         string
	requestID  string
	body       *transactionBody
}

func savePaymentRecord(ctx context.Context, s *session, p *paymentSave) (*written, error) {
	id := strings.TrimSpace(p.id)
	if id == "" {
		return writeRecord(ctx, s.client, &recordWrite{
			recordType: p.recordType,
			externalID: p.requestID,
			body:       p.body,
		})
	}
	var current transactionRecord
	if err := s.client.Get(ctx, netsuite.RecordRead{
		Type:               p.recordType,
		ID:                 id,
		ExpandSubResources: true,
	}, &current); err != nil {
		return nil, err
	}
	keep := make(map[string]struct{}, len(p.body.Apply.Items))
	for idx := range p.body.Apply.Items {
		keep[p.body.Apply.Items[idx].Doc.ID] = struct{}{}
	}
	for _, applied := range current.appliedDocs() {
		if _, ok := keep[applied]; !ok {
			p.body.Apply.Items = append(p.body.Apply.Items, applyLine{
				Doc:   recordRef{ID: applied},
				Apply: false,
			})
		}
	}
	return patchRecord(ctx, s.client, &recordUpdate{
		recordType: p.recordType,
		id:         id,
		body:       p.body,
	})
}

type shortPayWrite struct {
	session  *session
	doc      *accountingconnectorports.AccountingPaymentDocument
	idx      int
	currency currencyFields
	result   *accountingconnectorports.AccountingDocumentResult
}

func shortPayRefKey(invoiceID string) string {
	return accountingsync.ExternalRefShortPayPrefix + strings.TrimSpace(invoiceID)
}

func shortPay(ctx context.Context, w *shortPayWrite) error {
	app := &w.doc.Applications[w.idx]
	if strings.TrimSpace(app.ShortPayCreditExternalID) != "" {
		return nil
	}
	item := strings.TrimSpace(w.doc.ShortPayItemExternalID)
	if item == "" {
		return &accountingsync.SyncError{
			Category:   accountingsync.SyncErrorMapping,
			Code:       accountingsync.ItemRoleShortPayWriteOff,
			Message:    "The short pay write-off item is not mapped",
			Resolution: "Map the short pay write-off to a " + providerName + " item",
		}
	}
	text := cleanText(shortPayNote+app.InvoiceNumber, maxLineTextLength)
	body := &transactionBody{
		Entity:     refOf(w.doc.CustomerExternalID),
		Subsidiary: w.session.subsidiary(),
		TranDate:   w.doc.TxnDate,
		Memo:       text,
		Item: &sublist[itemLine]{Items: []itemLine{{
			Item:        recordRef{ID: item},
			Description: text,
			Quantity:    amount(decimal.NewFromInt(1)),
			Rate:        amount(app.ShortPayAmount),
			Amount:      amount(app.ShortPayAmount),
		}}},
		Apply: &sublist[applyLine]{Items: []applyLine{{
			Doc:    recordRef{ID: strings.TrimSpace(app.InvoiceExternalID)},
			Apply:  true,
			Amount: amountOf(app.ShortPayAmount),
		}}},
	}
	body.setCurrency(w.currency)
	saved, err := writeRecord(ctx, w.session.client, &recordWrite{
		recordType: recordCreditMemo,
		externalID: accountingsync.SyncStepRequestID(w.doc.RequestID, stepShortPayBase+w.idx),
		body:       body,
	})
	if saved != nil {
		w.result.Refs[shortPayRefKey(app.InvoiceExternalID)] = saved.id
	}
	return err
}

func (c *Connector) VoidPayment(
	ctx context.Context,
	ref *accountingconnectorports.AccountingDocumentRef,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	s, err := c.open(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(ref.Refs)
	result.ExternalID = ref.ExternalID
	if _, err = c.unwindRecord(ctx, s, &unwind{
		recordType: recordCustomerPayment,
		id:         ref.ExternalID,
		locked:     salesLocked,
		refused:    func() error { return reversalRefused() },
	}); err != nil {
		return result, err
	}
	for _, key := range slices.Sorted(maps.Keys(result.Refs)) {
		if !strings.HasPrefix(key, accountingsync.ExternalRefShortPayPrefix) {
			continue
		}
		memo := strings.TrimSpace(result.Refs[key])
		if memo == "" {
			continue
		}
		corrective, unwindErr := c.unwindRecord(ctx, s, &unwind{
			recordType: recordCreditMemo,
			id:         memo,
			locked:     salesLocked,
			refused:    func() error { return creditReversalRefused("short pay credit memo") },
			offset:     invoiceOffset(shortPayOffsetID(ref.RequestID, memo)),
		})
		if corrective != "" {
			result.Refs[shortPayCorrective(key)] = corrective
		}
		if unwindErr != nil {
			return result, unwindErr
		}
	}
	return result, nil
}

func (c *Connector) CreateCreditApplication(
	ctx context.Context,
	doc *accountingconnectorports.AccountingCreditApplicationDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if !doc.Amount.IsPositive() {
		return nil, errAmountInvalid
	}
	memo := strings.TrimSpace(doc.CreditMemoExternalID)
	invoice := strings.TrimSpace(doc.InvoiceExternalID)
	if memo == "" || invoice == "" {
		return nil, errExternalID
	}
	s, err := c.open(doc.Auth)
	if err != nil {
		return nil, err
	}
	saved, err := patchRecord(ctx, s.client, &recordUpdate{
		recordType: recordCreditMemo,
		id:         memo,
		body: &transactionBody{Apply: &sublist[applyLine]{Items: []applyLine{{
			Doc:    recordRef{ID: invoice},
			Apply:  true,
			Amount: amountOf(doc.Amount),
		}}}},
	})
	if saved == nil {
		return nil, err
	}
	return &accountingconnectorports.AccountingDocumentResult{
		ExternalID: memo,
		DocNumber:  saved.number,
		Refs: map[string]string{
			accountingsync.ExternalRefDocument:    memo,
			accountingsync.ExternalRefApplication: invoice,
		},
	}, err
}

func (c *Connector) VoidCreditApplication(
	ctx context.Context,
	ref *accountingconnectorports.AccountingDocumentRef,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	memo := strings.TrimSpace(ref.Refs[accountingsync.ExternalRefDocument])
	if memo == "" {
		memo = strings.TrimSpace(ref.ExternalID)
	}
	invoice := strings.TrimSpace(ref.Refs[accountingsync.ExternalRefApplication])
	if memo == "" || invoice == "" {
		return nil, creditApplicationUnknown()
	}
	s, err := c.open(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(ref.Refs)
	result.ExternalID = ref.ExternalID
	err = s.client.Patch(ctx, netsuite.RecordPatch{
		Type: recordCreditMemo,
		ID:   memo,
		Body: &transactionBody{Apply: unapplyLines([]string{invoice})},
	})
	if err != nil && !netsuiteerrors.IsNotFound(err) {
		return result, err
	}
	return result, nil
}
