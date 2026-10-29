package nsdocsync

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsrefsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/emoss08/trenova/shared/netsuiteerrors"
	"github.com/shopspring/decimal"
)

const (
	DocTypeCustomerPayment = "customerPayment"
	DocTypeVendorPayment   = "vendorPayment"
	shortPayNote           = "Short pay on "
	StepShortPayBase       = 10
)

func (c *Documents) SavePayment(
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
	s, err := c.Open(doc.Auth)
	if err != nil {
		return nil, err
	}
	currency, err := c.CurrencyFor(ctx, s, doc.CurrencyCode, doc.ExchangeRate)
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

	body := &nsbinding.TransactionBody{
		Customer:   nsrefsync.RefOf(doc.CustomerExternalID),
		Subsidiary: s.Subsidiary(),
		TranDate:   doc.TxnDate,
		Account:    nsrefsync.RefOf(doc.DepositAccountExternalID),
		Payment:    nsbinding.AmountOf(doc.TotalAmount),
		CheckNum:   nsrefsync.CleanText(doc.ReferenceNumber, maxTranIDLength),
		Memo:       nsrefsync.CleanText(doc.PrivateNote, maxMemoLength),
		Apply:      &nsbinding.Sublist[nsbinding.ApplyLine]{Items: cash},
	}
	body.SetCurrency(currency)
	saved, err := savePaymentRecord(ctx, s, &paymentSave{
		recordType: RecordCustomerPayment,
		ID:         doc.ExternalID,
		requestID:  doc.RequestID,
		body:       body,
	})
	finishPayment(result, saved, DocTypeCustomerPayment)
	return result, err
}

func cashApplications(
	doc *accountingconnectorports.AccountingPaymentDocument,
) ([]nsbinding.ApplyLine, error) {
	if !doc.TotalAmount.IsPositive() {
		return nil, nsbinding.ErrAmountInvalid
	}
	cash := make([]nsbinding.ApplyLine, 0, len(doc.Applications))
	for idx := range doc.Applications {
		app := &doc.Applications[idx]
		if app.AppliedAmount.IsNegative() || app.ShortPayAmount.IsNegative() {
			return nil, nsbinding.ErrAmountInvalid
		}
		invoice := strings.TrimSpace(app.InvoiceExternalID)
		if invoice == "" && (app.AppliedAmount.IsPositive() || app.ShortPayAmount.IsPositive()) {
			return nil, nsbinding.ErrApplyTarget
		}
		if app.AppliedAmount.IsPositive() {
			cash = append(cash, nsbinding.ApplyLine{
				Doc:    nsbinding.RecordRef{ID: invoice},
				Apply:  true,
				Amount: nsbinding.AmountOf(app.AppliedAmount),
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
	result.ExternalID = saved.ID
	result.DocNumber = saved.number
	result.Refs[accountingsync.ExternalRefDocument] = saved.ID
	result.Refs[accountingsync.ExternalRefDocumentType] = docType
}

type paymentSave struct {
	recordType string
	ID         string
	requestID  string
	body       *nsbinding.TransactionBody
}

func savePaymentRecord(
	ctx context.Context,
	s *nsbinding.Session,
	p *paymentSave,
) (*written, error) {
	id := strings.TrimSpace(p.ID)
	if id == "" {
		return writeRecord(ctx, s.Client, &recordWrite{
			recordType: p.recordType,
			externalID: p.requestID,
			body:       p.body,
		})
	}
	var current nsbinding.TransactionRecord
	if err := s.Client.Get(ctx, netsuite.RecordRead{
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
	for _, applied := range current.AppliedDocs() {
		if _, ok := keep[applied]; !ok {
			p.body.Apply.Items = append(p.body.Apply.Items, nsbinding.ApplyLine{
				Doc:   nsbinding.RecordRef{ID: applied},
				Apply: false,
			})
		}
	}
	return patchRecord(ctx, s.Client, &recordUpdate{
		recordType: p.recordType,
		ID:         id,
		body:       p.body,
	})
}

type shortPayWrite struct {
	session  *nsbinding.Session
	doc      *accountingconnectorports.AccountingPaymentDocument
	idx      int
	currency nsbinding.CurrencyFields
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
			Resolution: "Map the short pay write-off to a " + nsbinding.ProviderName + " item",
		}
	}
	text := nsrefsync.CleanText(shortPayNote+app.InvoiceNumber, nsrefsync.MaxLineTextLength)
	body := &nsbinding.TransactionBody{
		Entity:     nsrefsync.RefOf(w.doc.CustomerExternalID),
		Subsidiary: w.session.Subsidiary(),
		TranDate:   w.doc.TxnDate,
		Memo:       text,
		Item: &nsbinding.Sublist[nsbinding.ItemLine]{Items: []nsbinding.ItemLine{{
			Item:        nsbinding.RecordRef{ID: item},
			Description: text,
			Quantity:    nsbinding.Amount(decimal.NewFromInt(1)),
			Rate:        nsbinding.Amount(app.ShortPayAmount),
			Amount:      nsbinding.Amount(app.ShortPayAmount),
		}}},
		Apply: &nsbinding.Sublist[nsbinding.ApplyLine]{Items: []nsbinding.ApplyLine{{
			Doc:    nsbinding.RecordRef{ID: strings.TrimSpace(app.InvoiceExternalID)},
			Apply:  true,
			Amount: nsbinding.AmountOf(app.ShortPayAmount),
		}}},
	}
	body.SetCurrency(w.currency)
	saved, err := writeRecord(ctx, w.session.Client, &recordWrite{
		recordType: RecordCreditMemo,
		externalID: accountingsync.SyncStepRequestID(w.doc.RequestID, StepShortPayBase+w.idx),
		body:       body,
	})
	if saved != nil {
		w.result.Refs[shortPayRefKey(app.InvoiceExternalID)] = saved.ID
	}
	return err
}

func (c *Documents) VoidPayment(
	ctx context.Context,
	ref *accountingconnectorports.AccountingDocumentRef,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	s, err := c.Open(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(ref.Refs)
	result.ExternalID = ref.ExternalID
	if _, err = c.unwindRecord(ctx, s, &unwind{
		recordType: RecordCustomerPayment,
		ID:         ref.ExternalID,
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
			recordType: RecordCreditMemo,
			ID:         memo,
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

func (c *Documents) CreateCreditApplication(
	ctx context.Context,
	doc *accountingconnectorports.AccountingCreditApplicationDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if !doc.Amount.IsPositive() {
		return nil, nsbinding.ErrAmountInvalid
	}
	memo := strings.TrimSpace(doc.CreditMemoExternalID)
	invoice := strings.TrimSpace(doc.InvoiceExternalID)
	if memo == "" || invoice == "" {
		return nil, nsbinding.ErrExternalID
	}
	s, err := c.Open(doc.Auth)
	if err != nil {
		return nil, err
	}
	saved, err := patchRecord(ctx, s.Client, &recordUpdate{
		recordType: RecordCreditMemo,
		ID:         memo,
		body: &nsbinding.TransactionBody{
			Apply: &nsbinding.Sublist[nsbinding.ApplyLine]{Items: []nsbinding.ApplyLine{{
				Doc:    nsbinding.RecordRef{ID: invoice},
				Apply:  true,
				Amount: nsbinding.AmountOf(doc.Amount),
			}}},
		},
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

func (c *Documents) VoidCreditApplication(
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
	s, err := c.Open(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(ref.Refs)
	result.ExternalID = ref.ExternalID
	err = s.Client.Patch(ctx, netsuite.RecordPatch{
		Type: RecordCreditMemo,
		ID:   memo,
		Body: &nsbinding.TransactionBody{Apply: unapplyLines([]string{invoice})},
	})
	if err != nil && !netsuiteerrors.IsNotFound(err) {
		return result, err
	}
	return result, nil
}
