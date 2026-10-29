package nsdocsync

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsrefsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/netsuite"
)

func (c *Documents) UpsertVendor(
	ctx context.Context,
	doc *accountingconnectorports.AccountingVendorDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	return c.upsertParty(ctx, &partyWrite{
		auth:       doc.Auth,
		kind:       nsrefsync.PartyVendor,
		requestID:  doc.RequestID,
		externalID: doc.ExternalID,
		party:      &doc.Party,
	})
}

func purchaseRecordOf(credit bool) string {
	if credit {
		return RecordVendorCredit
	}
	return RecordVendorBill
}

func (c *Documents) CreatePurchaseDocument(
	ctx context.Context,
	doc *accountingconnectorports.AccountingPurchaseDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	return c.savePurchase(ctx, doc, false)
}

func (c *Documents) UpdatePurchaseDocument(
	ctx context.Context,
	doc *accountingconnectorports.AccountingPurchaseDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	return c.savePurchase(ctx, doc, true)
}

func (c *Documents) savePurchase(
	ctx context.Context,
	doc *accountingconnectorports.AccountingPurchaseDocument,
	update bool,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if !doc.Kind.IsBill() {
		return nil, nsbinding.ErrDocumentKind
	}
	id := strings.TrimSpace(doc.ExternalID)
	if update && id == "" {
		return nil, nsbinding.ErrExternalID
	}
	recordType := purchaseRecordOf(doc.VendorCredit)
	s, err := c.Open(doc.Auth)
	if err != nil {
		return nil, err
	}
	body, err := c.purchaseBody(ctx, s, doc)
	if err != nil {
		return nil, err
	}
	var saved *written
	if update {
		saved, err = patchRecord(ctx, s.Client, &recordUpdate{
			recordType: recordType,
			ID:         id,
			body:       body,
			replace:    []string{sublistExpense},
		})
	} else {
		saved, err = writeRecord(ctx, s.Client, &recordWrite{
			recordType: recordType,
			externalID: doc.RequestID,
			body:       body,
		})
	}
	result := newResult(nil)
	if saved != nil {
		finishPurchase(&purchaseWritten{
			auth:   doc.Auth,
			credit: doc.VendorCredit,
			saved:  saved,
			result: result,
		})
	}
	return result, err
}

func (c *Documents) purchaseBody(
	ctx context.Context,
	s *nsbinding.Session,
	doc *accountingconnectorports.AccountingPurchaseDocument,
) (*nsbinding.TransactionBody, error) {
	lines, err := purchaseLines(doc.Lines)
	if err != nil {
		return nil, err
	}
	currency, err := c.CurrencyFor(ctx, s, doc.CurrencyCode, doc.ExchangeRate)
	if err != nil {
		return nil, err
	}
	body := &nsbinding.TransactionBody{
		Entity:     nsrefsync.RefOf(doc.VendorExternalID),
		Subsidiary: s.Subsidiary(),
		TranDate:   doc.TxnDate,
		TranID:     nsrefsync.CleanText(doc.DocNumber, maxTranIDLength),
		Account:    nsrefsync.RefOf(doc.APAccountExternalID),
		Memo:       nsrefsync.CleanText(doc.PrivateNote, maxMemoLength),
		Expense:    &nsbinding.Sublist[nsbinding.ExpenseLine]{Items: lines},
	}
	body.SetCurrency(currency)
	if !doc.VendorCredit {
		body.DueDate = doc.DueDate
	}
	return body, nil
}

func purchaseLines(
	lines []accountingconnectorports.AccountingPurchaseLine,
) ([]nsbinding.ExpenseLine, error) {
	if len(lines) == 0 {
		return nil, nsbinding.ErrLinesRequired
	}
	out := make([]nsbinding.ExpenseLine, 0, len(lines))
	for idx := range lines {
		line := &lines[idx]
		purpose := linePurpose(idx, line.Description)
		account := strings.TrimSpace(line.AccountExternalID)
		if account == "" {
			return nil, unmappedAccount(purpose)
		}
		if line.Amount.IsNegative() {
			return nil, negativeLine(purpose)
		}
		out = append(out, nsbinding.ExpenseLine{
			Account: nsbinding.RecordRef{ID: account},
			Amount:  nsbinding.Amount(line.Amount),
			Memo:    nsrefsync.CleanText(lineText(line.Description), nsrefsync.MaxLineTextLength),
		})
	}
	return out, nil
}

type purchaseWritten struct {
	auth   accountingconnectorports.AccountingDocumentAuth
	credit bool
	saved  *written
	result *accountingconnectorports.AccountingDocumentResult
}

func finishPurchase(w *purchaseWritten) {
	recordType := purchaseRecordOf(w.credit)
	result := w.result
	result.ExternalID = w.saved.ID
	result.DocNumber = w.saved.number
	result.Refs[accountingsync.ExternalRefDocument] = w.saved.ID
	result.Refs[accountingsync.ExternalRefDocumentType] = recordType
	result.Refs[accountingsync.ExternalRefCreditDocument] = strconv.FormatBool(w.credit)
	if account, err := nsbinding.AccountOf(w.auth.RealmID); err == nil {
		result.Refs[accountingsync.ExternalRefURL] = netsuite.TransactionLink(account, w.saved.ID)
	}
}

func PurchaseIsCredit(refs map[string]string) (bool, error) {
	if value := strings.TrimSpace(refs[accountingsync.ExternalRefCreditDocument]); value != "" {
		credit, err := strconv.ParseBool(value)
		if err != nil {
			return false, nsbinding.ErrPurchaseDocumentType
		}
		return credit, nil
	}
	switch strings.TrimSpace(refs[accountingsync.ExternalRefDocumentType]) {
	case "", RecordVendorBill:
		return false, nil
	case RecordVendorCredit:
		return true, nil
	default:
		return false, nsbinding.ErrPurchaseDocumentType
	}
}

func (c *Documents) VoidPurchaseDocument(
	ctx context.Context,
	ref *accountingconnectorports.AccountingDocumentRef,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if !ref.Kind.IsBill() && !ref.Kind.IsBillPayment() {
		return nil, nsbinding.ErrDocumentKind
	}
	credit := false
	if ref.Kind.IsBill() {
		var err error
		if credit, err = PurchaseIsCredit(ref.Refs); err != nil {
			return nil, err
		}
	}
	s, err := c.Open(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(ref.Refs)
	result.ExternalID = ref.ExternalID
	var corrective string
	switch {
	case ref.Kind.IsBillPayment():
		_, err = c.unwindRecord(ctx, s, &unwind{
			recordType: RecordVendorPayment,
			ID:         ref.ExternalID,
			locked:     purchaseLocked,
			refused:    func() error { return reversalRefused() },
		})
	case credit:
		corrective, err = c.unwindRecord(ctx, s, &unwind{
			recordType: RecordVendorCredit,
			ID:         ref.ExternalID,
			locked:     purchaseLocked,
			refused:    func() error { return creditReversalRefused("vendor credit") },
			offset: &creditOffset{
				recordType: RecordVendorBill,
				typeCode:   TypeVendorBill,
				requestID:  ref.RequestID,
			},
		})
	default:
		corrective, err = c.retireDocument(ctx, s, &Reversal{
			recordType:  RecordVendorBill,
			reverseType: RecordVendorCredit,
			reverseCode: TypeVendorCredit,
			ID:          ref.ExternalID,
			requestID:   ref.RequestID,
			locked:      purchaseLocked,
		})
	}
	if corrective != "" {
		result.Refs[RefCorrective] = corrective
	}
	return result, err
}

func (c *Documents) CreateBillPayment(
	ctx context.Context,
	doc *accountingconnectorports.AccountingBillPaymentDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	return c.saveBillPayment(ctx, doc, false)
}

func (c *Documents) UpdateBillPayment(
	ctx context.Context,
	doc *accountingconnectorports.AccountingBillPaymentDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	return c.saveBillPayment(ctx, doc, true)
}

func (c *Documents) saveBillPayment(
	ctx context.Context,
	doc *accountingconnectorports.AccountingBillPaymentDocument,
	update bool,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if !doc.Kind.IsBillPayment() {
		return nil, nsbinding.ErrDocumentKind
	}
	id := strings.TrimSpace(doc.ExternalID)
	if update && id == "" {
		return nil, nsbinding.ErrExternalID
	}
	if !doc.Amount.IsPositive() {
		return nil, nsbinding.ErrAmountInvalid
	}
	bill := strings.TrimSpace(doc.BillExternalID)
	if bill == "" {
		return nil, nsbinding.ErrApplyTarget
	}
	if strings.TrimSpace(doc.BankAccountExternalID) == "" {
		return nil, unmappedAccount("The bank account")
	}
	s, err := c.Open(doc.Auth)
	if err != nil {
		return nil, err
	}
	currency, err := c.CurrencyFor(ctx, s, doc.CurrencyCode, doc.ExchangeRate)
	if err != nil {
		return nil, err
	}
	body := &nsbinding.TransactionBody{
		Entity:     nsrefsync.RefOf(doc.VendorExternalID),
		Subsidiary: s.Subsidiary(),
		TranDate:   doc.TxnDate,
		TranID:     nsrefsync.CleanText(doc.DocNumber, maxTranIDLength),
		Account:    nsrefsync.RefOf(doc.BankAccountExternalID),
		Memo:       nsrefsync.CleanText(doc.PrivateNote, maxMemoLength),
		Apply: &nsbinding.Sublist[nsbinding.ApplyLine]{Items: []nsbinding.ApplyLine{{
			Doc:    nsbinding.RecordRef{ID: bill},
			Apply:  true,
			Amount: nsbinding.AmountOf(doc.Amount),
		}}},
	}
	body.SetCurrency(currency)
	if !update {
		id = ""
	}
	saved, err := savePaymentRecord(ctx, s, &paymentSave{
		recordType: RecordVendorPayment,
		ID:         id,
		requestID:  doc.RequestID,
		body:       body,
	})
	result := newResult(nil)
	finishPayment(result, saved, DocTypeVendorPayment)
	return result, err
}
