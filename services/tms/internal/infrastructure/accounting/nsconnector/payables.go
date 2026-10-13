package nsconnector

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/shared/netsuite"
)

var errPurchaseDocumentType = errors.New(
	"netsuite: the purchase document type is neither a vendor bill nor a vendor credit",
)

func (c *Connector) UpsertVendor(
	ctx context.Context,
	doc *accountingsyncports.AccountingVendorDocument,
) (*accountingsyncports.AccountingDocumentResult, error) {
	return c.upsertParty(ctx, &partyWrite{
		auth:       doc.Auth,
		kind:       partyVendor,
		requestID:  doc.RequestID,
		externalID: doc.ExternalID,
		party:      &doc.Party,
	})
}

func purchaseRecordOf(credit bool) string {
	if credit {
		return recordVendorCredit
	}
	return recordVendorBill
}

func (c *Connector) CreatePurchaseDocument(
	ctx context.Context,
	doc *accountingsyncports.AccountingPurchaseDocument,
) (*accountingsyncports.AccountingDocumentResult, error) {
	return c.savePurchase(ctx, doc, false)
}

func (c *Connector) UpdatePurchaseDocument(
	ctx context.Context,
	doc *accountingsyncports.AccountingPurchaseDocument,
) (*accountingsyncports.AccountingDocumentResult, error) {
	return c.savePurchase(ctx, doc, true)
}

func (c *Connector) savePurchase(
	ctx context.Context,
	doc *accountingsyncports.AccountingPurchaseDocument,
	update bool,
) (*accountingsyncports.AccountingDocumentResult, error) {
	if !doc.Kind.IsBill() {
		return nil, errDocumentKind
	}
	id := strings.TrimSpace(doc.ExternalID)
	if update && id == "" {
		return nil, errExternalID
	}
	recordType := purchaseRecordOf(doc.VendorCredit)
	s, err := c.open(doc.Auth)
	if err != nil {
		return nil, err
	}
	body, err := c.purchaseBody(ctx, s, doc)
	if err != nil {
		return nil, err
	}
	var saved *written
	if update {
		saved, err = patchRecord(ctx, s.client, &recordUpdate{
			recordType: recordType,
			id:         id,
			body:       body,
			replace:    []string{sublistExpense},
		})
	} else {
		saved, err = writeRecord(ctx, s.client, &recordWrite{
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

func (c *Connector) purchaseBody(
	ctx context.Context,
	s *session,
	doc *accountingsyncports.AccountingPurchaseDocument,
) (*transactionBody, error) {
	lines, err := purchaseLines(doc.Lines)
	if err != nil {
		return nil, err
	}
	currency, err := c.currencyFor(ctx, s, doc.CurrencyCode, doc.ExchangeRate)
	if err != nil {
		return nil, err
	}
	body := &transactionBody{
		Entity:     refOf(doc.VendorExternalID),
		Subsidiary: s.subsidiary(),
		TranDate:   doc.TxnDate,
		TranID:     cleanText(doc.DocNumber, maxTranIDLength),
		Account:    refOf(doc.APAccountExternalID),
		Memo:       cleanText(doc.PrivateNote, maxMemoLength),
		Expense:    &sublist[expenseLine]{Items: lines},
	}
	body.setCurrency(currency)
	if !doc.VendorCredit {
		body.DueDate = doc.DueDate
	}
	return body, nil
}

func purchaseLines(lines []accountingsyncports.AccountingPurchaseLine) ([]expenseLine, error) {
	if len(lines) == 0 {
		return nil, errLinesRequired
	}
	out := make([]expenseLine, 0, len(lines))
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
		out = append(out, expenseLine{
			Account: recordRef{ID: account},
			Amount:  amount(line.Amount),
			Memo:    cleanText(lineText(line.Description), maxLineTextLength),
		})
	}
	return out, nil
}

type purchaseWritten struct {
	auth   accountingsyncports.AccountingDocumentAuth
	credit bool
	saved  *written
	result *accountingsyncports.AccountingDocumentResult
}

func finishPurchase(w *purchaseWritten) {
	recordType := purchaseRecordOf(w.credit)
	result := w.result
	result.ExternalID = w.saved.id
	result.DocNumber = w.saved.number
	result.Refs[accountingsync.ExternalRefDocument] = w.saved.id
	result.Refs[accountingsync.ExternalRefDocumentType] = recordType
	result.Refs[accountingsync.ExternalRefCreditDocument] = strconv.FormatBool(w.credit)
	if account, err := accountOf(w.auth.RealmID); err == nil {
		result.Refs[accountingsync.ExternalRefURL] = netsuite.TransactionLink(account, w.saved.id)
	}
}

func purchaseIsCredit(refs map[string]string) (bool, error) {
	if value := strings.TrimSpace(refs[accountingsync.ExternalRefCreditDocument]); value != "" {
		credit, err := strconv.ParseBool(value)
		if err != nil {
			return false, errPurchaseDocumentType
		}
		return credit, nil
	}
	switch strings.TrimSpace(refs[accountingsync.ExternalRefDocumentType]) {
	case "", recordVendorBill:
		return false, nil
	case recordVendorCredit:
		return true, nil
	default:
		return false, errPurchaseDocumentType
	}
}

func (c *Connector) VoidPurchaseDocument(
	ctx context.Context,
	ref *accountingsyncports.AccountingDocumentRef,
) (*accountingsyncports.AccountingDocumentResult, error) {
	if !ref.Kind.IsBill() && !ref.Kind.IsBillPayment() {
		return nil, errDocumentKind
	}
	credit := false
	if ref.Kind.IsBill() {
		var err error
		if credit, err = purchaseIsCredit(ref.Refs); err != nil {
			return nil, err
		}
	}
	s, err := c.open(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(ref.Refs)
	result.ExternalID = ref.ExternalID
	var corrective string
	switch {
	case ref.Kind.IsBillPayment():
		_, err = c.unwindRecord(ctx, s, &unwind{
			recordType: recordVendorPayment,
			id:         ref.ExternalID,
			locked:     purchaseLocked,
			refused:    func() error { return reversalRefused() },
		})
	case credit:
		corrective, err = c.unwindRecord(ctx, s, &unwind{
			recordType: recordVendorCredit,
			id:         ref.ExternalID,
			locked:     purchaseLocked,
			refused:    func() error { return creditReversalRefused("vendor credit") },
			offset: &creditOffset{
				recordType: recordVendorBill,
				typeCode:   typeVendorBill,
				requestID:  ref.RequestID,
			},
		})
	default:
		corrective, err = c.retireDocument(ctx, s, &reversal{
			recordType:  recordVendorBill,
			reverseType: recordVendorCredit,
			reverseCode: typeVendorCredit,
			id:          ref.ExternalID,
			requestID:   ref.RequestID,
			locked:      purchaseLocked,
		})
	}
	if corrective != "" {
		result.Refs[refCorrective] = corrective
	}
	return result, err
}

func (c *Connector) CreateBillPayment(
	ctx context.Context,
	doc *accountingsyncports.AccountingBillPaymentDocument,
) (*accountingsyncports.AccountingDocumentResult, error) {
	return c.saveBillPayment(ctx, doc, false)
}

func (c *Connector) UpdateBillPayment(
	ctx context.Context,
	doc *accountingsyncports.AccountingBillPaymentDocument,
) (*accountingsyncports.AccountingDocumentResult, error) {
	return c.saveBillPayment(ctx, doc, true)
}

func (c *Connector) saveBillPayment(
	ctx context.Context,
	doc *accountingsyncports.AccountingBillPaymentDocument,
	update bool,
) (*accountingsyncports.AccountingDocumentResult, error) {
	if !doc.Kind.IsBillPayment() {
		return nil, errDocumentKind
	}
	id := strings.TrimSpace(doc.ExternalID)
	if update && id == "" {
		return nil, errExternalID
	}
	if !doc.Amount.IsPositive() {
		return nil, errAmountInvalid
	}
	bill := strings.TrimSpace(doc.BillExternalID)
	if bill == "" {
		return nil, errApplyTarget
	}
	if strings.TrimSpace(doc.BankAccountExternalID) == "" {
		return nil, unmappedAccount("The bank account")
	}
	s, err := c.open(doc.Auth)
	if err != nil {
		return nil, err
	}
	currency, err := c.currencyFor(ctx, s, doc.CurrencyCode, doc.ExchangeRate)
	if err != nil {
		return nil, err
	}
	body := &transactionBody{
		Entity:     refOf(doc.VendorExternalID),
		Subsidiary: s.subsidiary(),
		TranDate:   doc.TxnDate,
		TranID:     cleanText(doc.DocNumber, maxTranIDLength),
		Account:    refOf(doc.BankAccountExternalID),
		Memo:       cleanText(doc.PrivateNote, maxMemoLength),
		Apply: &sublist[applyLine]{Items: []applyLine{{
			Doc:    recordRef{ID: bill},
			Apply:  true,
			Amount: amountOf(doc.Amount),
		}}},
	}
	body.setCurrency(currency)
	if !update {
		id = ""
	}
	saved, err := savePaymentRecord(ctx, s, &paymentSave{
		recordType: recordVendorPayment,
		id:         id,
		requestID:  doc.RequestID,
		body:       body,
	})
	result := newResult(nil)
	finishPayment(result, saved, docTypeVendorPayment)
	return result, err
}
