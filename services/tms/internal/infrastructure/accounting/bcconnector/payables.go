package bcconnector

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/shopspring/decimal"
)

const (
	docTypePurchaseInvoice    = "PurchaseInvoice"
	docTypePurchaseCreditMemo = "PurchaseCreditMemo"
	revisionSeparator         = "-R"
	revisionHashLength        = 4
)

var errPurchaseDocumentType = errors.New(
	"businesscentral: the purchase document type is neither a bill nor a vendor credit",
)

func (c *Connector) UpsertVendor(
	ctx context.Context,
	doc *services.AccountingVendorDocument,
) (*services.AccountingDocumentResult, error) {
	return c.upsertParty(ctx, &partyWrite{
		auth:       doc.Auth,
		kind:       businesscentral.PartyVendor,
		externalID: doc.ExternalID,
		party:      &doc.Party,
	})
}

func purchaseKindOf(credit bool) businesscentral.DocumentKind {
	if credit {
		return businesscentral.DocumentPurchaseCreditMemo
	}
	return businesscentral.DocumentPurchaseInvoice
}

func (c *Connector) CreatePurchaseDocument(
	ctx context.Context,
	doc *services.AccountingPurchaseDocument,
) (*services.AccountingDocumentResult, error) {
	if !doc.Kind.IsBill() {
		return nil, errDocumentKind
	}
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	return c.writePurchase(ctx, client, doc, newResult(nil))
}

func (c *Connector) writePurchase(
	ctx context.Context,
	client *businesscentral.Client,
	doc *services.AccountingPurchaseDocument,
	result *services.AccountingDocumentResult,
) (*services.AccountingDocumentResult, error) {
	kind := purchaseKindOf(doc.VendorCredit)
	input, err := c.purchaseInput(ctx, client, doc)
	if err != nil {
		return result, err
	}
	posted, err := postDocument(ctx, &documentWrite{
		client:  client,
		kind:    kind,
		input:   input,
		refs:    result.Refs,
		refKey:  accountingsync.ExternalRefDocument,
		exclude: result.Refs[refReplaced],
	})
	if err != nil {
		return result, err
	}
	c.finishPurchase(&purchaseWritten{
		auth:   doc.Auth,
		kind:   kind,
		doc:    posted,
		result: result,
	})
	for key, amount := range accountTotals(doc.Lines) {
		result.Refs[key] = amount.String()
	}
	return result, nil
}

func accountTotals(lines []services.AccountingPurchaseLine) map[string]decimal.Decimal {
	totals := make(map[string]decimal.Decimal, len(lines))
	for idx := range lines {
		key := refAccountPrefix + strings.ToLower(strings.TrimSpace(lines[idx].AccountExternalID))
		totals[key] = totals[key].Add(lines[idx].Amount)
	}
	return totals
}

func (c *Connector) purchaseInput(
	ctx context.Context,
	client *businesscentral.Client,
	doc *services.AccountingPurchaseDocument,
) (*businesscentral.DocumentInput, error) {
	lines, err := purchaseLines(doc.Lines)
	if err != nil {
		return nil, err
	}
	posting, err := c.postingFor(ctx, client, doc.TxnDate)
	if err != nil {
		return nil, err
	}
	input := &businesscentral.DocumentInput{
		PartyID:                doc.VendorExternalID,
		ExternalDocumentNumber: docReference(doc.DocNumber, doc.RequestID),
		DocumentDate:           doc.TxnDate,
		PostingDate:            doc.TxnDate,
		CurrencyCode:           posting.currency(doc.CurrencyCode),
		Lines:                  lines,
	}
	if !doc.VendorCredit {
		input.DueDate = doc.DueDate
	}
	return input, nil
}

func purchaseLines(lines []services.AccountingPurchaseLine) ([]businesscentral.LineInput, error) {
	if len(lines) == 0 {
		return nil, errLinesRequired
	}
	out := make([]businesscentral.LineInput, 0, len(lines))
	for idx := range lines {
		line := &lines[idx]
		purpose := linePurpose(idx, line.Description)
		if strings.TrimSpace(line.AccountExternalID) == "" {
			return nil, unmappedAccount(purpose)
		}
		if line.Amount.IsNegative() {
			return nil, negativeLine(purpose)
		}
		out = append(out, businesscentral.LineInput{
			AccountID:   strings.TrimSpace(line.AccountExternalID),
			Description: lineDescription(line.Description),
			Quantity:    decimal.NewFromInt(1),
			UnitPrice:   line.Amount,
		})
	}
	return out, nil
}

type purchaseWritten struct {
	auth   services.AccountingDocumentAuth
	kind   businesscentral.DocumentKind
	doc    *businesscentral.Document
	result *services.AccountingDocumentResult
}

func (c *Connector) finishPurchase(written *purchaseWritten) {
	credit := written.kind == businesscentral.DocumentPurchaseCreditMemo
	docType := docTypePurchaseInvoice
	if credit {
		docType = docTypePurchaseCreditMemo
	}
	result := written.result
	result.ExternalID = written.doc.ID
	result.DocNumber = written.doc.Number
	result.Refs[accountingsync.ExternalRefDocument] = written.doc.ID
	result.Refs[accountingsync.ExternalRefDocumentType] = docType
	result.Refs[accountingsync.ExternalRefCreditDocument] = strconv.FormatBool(credit)
	result.Refs[accountingsync.ExternalRefURL] = c.purchaseURL(written)
}

func (c *Connector) purchaseURL(written *purchaseWritten) string {
	name := strings.TrimSpace(written.auth.CompanyName)
	ref, err := businesscentral.ParseCompanyRef(written.auth.RealmID)
	if err != nil || name == "" {
		return ""
	}
	return businesscentral.NewLinker(c.apiOpts...).DocumentLink(
		ref,
		name,
		purchasePage(written.kind),
		written.doc.Number,
	)
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
	case "", docTypePurchaseInvoice:
		return false, nil
	case docTypePurchaseCreditMemo:
		return true, nil
	default:
		return false, errPurchaseDocumentType
	}
}

func (c *Connector) UpdatePurchaseDocument(
	ctx context.Context,
	doc *services.AccountingPurchaseDocument,
) (*services.AccountingDocumentResult, error) {
	if !doc.Kind.IsBill() {
		return nil, errDocumentKind
	}
	id := strings.TrimSpace(doc.ExternalID)
	if id == "" {
		return nil, errExternalID
	}
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(nil)
	retire := &billRetirement{
		client:    client,
		requestID: doc.RequestID,
		id:        id,
		credit:    doc.VendorCredit,
		accounts:  firstAccount(doc.Lines),
		refs:      result.Refs,
	}
	if err = c.retirePurchase(ctx, retire); err != nil {
		return result, err
	}
	result.Refs[refReplaced] = id
	result.Refs[refCorrective] = result.Refs[voidCreditKey(id)]
	revised := *doc
	revised.DocNumber = revisedNumber(doc.DocNumber, doc.RequestID)
	return c.writePurchase(ctx, client, &revised, result)
}

func revisedNumber(number, requestID string) string {
	number = strings.TrimSpace(number)
	if number == "" {
		return ""
	}
	suffix := revisionSeparator + strings.ToUpper(hashutils.SHA256Hex(requestID)[:revisionHashLength])
	limit := businesscentral.MaxExternalDocumentNumberLength - len(suffix)
	if len(number) > limit {
		number = number[:limit]
	}
	return number + suffix
}

func firstAccount(lines []services.AccountingPurchaseLine) map[string]string {
	for idx := range lines {
		if id := strings.TrimSpace(lines[idx].AccountExternalID); id != "" {
			return map[string]string{refAccountPrefix + strings.ToLower(id): ""}
		}
	}
	return nil
}

func (c *Connector) VoidPurchaseDocument(
	ctx context.Context,
	ref *services.AccountingDocumentRef,
) (*services.AccountingDocumentResult, error) {
	if ref.Kind.IsBillPayment() {
		return nil, reversalRefused()
	}
	if !ref.Kind.IsBill() {
		return nil, errDocumentKind
	}
	credit, err := purchaseIsCredit(ref.Refs)
	if err != nil {
		return nil, err
	}
	client, err := c.client(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(ref.Refs)
	result.ExternalID = ref.ExternalID
	if err = c.retirePurchase(ctx, &billRetirement{
		client:    client,
		requestID: ref.RequestID,
		id:        ref.ExternalID,
		credit:    credit,
		accounts:  ref.Refs,
		refs:      result.Refs,
	}); err != nil {
		return result, err
	}
	return result, nil
}

type billRetirement struct {
	client    *businesscentral.Client
	requestID string
	id        string
	credit    bool
	accounts  map[string]string
	refs      map[string]string
}

func (c *Connector) retirePurchase(ctx context.Context, r *billRetirement) error {
	if r.credit {
		_, err := c.retireDocument(ctx, r.client, businesscentral.DocumentPurchaseCreditMemo, r.id)
		return err
	}
	bill, err := r.client.Document(ctx, businesscentral.DocumentPurchaseInvoice, r.id)
	if err != nil {
		if businesscentral.IsNotFound(err) {
			return nil
		}
		return err
	}
	if bill.IsDraft() {
		err = r.client.DeleteDocument(ctx, businesscentral.DocumentPurchaseInvoice, bill.ID, bill.ETag)
		if err != nil && !businesscentral.IsNotFound(err) {
			return err
		}
		return nil
	}
	return c.creditBill(ctx, r, bill)
}

func (c *Connector) creditBill(
	ctx context.Context,
	r *billRetirement,
	bill *businesscentral.Document,
) error {
	input := &businesscentral.DocumentInput{
		PartyID:                bill.PartyID,
		ExternalDocumentNumber: requestReference(r.requestID),
		InvoiceID:              bill.ID,
	}
	key := voidCreditKey(bill.ID)
	held, found, err := resumeDocument(ctx, &documentWrite{
		client: r.client,
		kind:   businesscentral.DocumentPurchaseCreditMemo,
		input:  input,
		refs:   r.refs,
		refKey: key,
	})
	if err != nil {
		return err
	}
	if found && !held.IsDraft() {
		r.refs[key] = held.ID
		return nil
	}
	if !found && settled(bill) {
		return paidConflict()
	}
	if input.Lines, err = creditLines(r.accounts, bill.TotalAmountIncludingTax); err != nil {
		return err
	}
	setup, err := c.setups.get(ctx, r.client)
	if err != nil {
		return err
	}
	input.CurrencyCode = documentCurrency(setup, bill.CurrencyCode)
	memo, err := postDocument(ctx, &documentWrite{
		client: r.client,
		kind:   businesscentral.DocumentPurchaseCreditMemo,
		input:  input,
		refs:   r.refs,
		refKey: key,
	})
	if err != nil {
		return err
	}
	r.refs[key] = memo.ID
	return nil
}

func voidCreditKey(billID string) string {
	return refVoidCredit + ":" + strings.ToLower(strings.TrimSpace(billID))
}

func creditLines(
	accounts map[string]string,
	total decimal.Decimal,
) ([]businesscentral.LineInput, error) {
	keys := make([]string, 0, len(accounts))
	for key := range accounts {
		if strings.HasPrefix(key, refAccountPrefix) && len(key) > len(refAccountPrefix) {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 || !total.IsPositive() {
		return nil, billLinesUnknown()
	}
	slices.Sort(keys)
	if len(keys) == 1 {
		return []businesscentral.LineInput{creditLine(keys[0], total)}, nil
	}
	lines := make([]businesscentral.LineInput, 0, len(keys))
	for _, key := range keys {
		amount, err := decimal.NewFromString(accounts[key])
		if err != nil || amount.IsNegative() {
			return nil, billLinesUnknown()
		}
		if amount.IsPositive() {
			lines = append(lines, creditLine(key, amount))
		}
	}
	if len(lines) == 0 {
		return nil, billLinesUnknown()
	}
	return lines, nil
}

func creditLine(key string, amount decimal.Decimal) businesscentral.LineInput {
	return businesscentral.LineInput{
		AccountID:   strings.TrimPrefix(key, refAccountPrefix),
		Description: "Void",
		Quantity:    decimal.NewFromInt(1),
		UnitPrice:   amount,
	}
}

func (c *Connector) CreateBillPayment(
	ctx context.Context,
	doc *services.AccountingBillPaymentDocument,
) (*services.AccountingDocumentResult, error) {
	if !doc.Kind.IsBillPayment() {
		return nil, errDocumentKind
	}
	if !doc.Amount.IsPositive() {
		return nil, businesscentral.ErrAmountInvalid
	}
	if strings.TrimSpace(doc.BankAccountExternalID) == "" {
		return nil, unmappedAccount("The bank account")
	}
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	if _, err = c.postingFor(ctx, client, doc.TxnDate); err != nil {
		return nil, err
	}
	result := newResult(nil)
	number := requestReference(doc.RequestID)
	if err = c.postPayment(ctx, &journalWrite{
		client:         client,
		party:          businesscentral.PartyVendor,
		accountID:      doc.BankAccountExternalID,
		documentNumber: number,
		lines: []businesscentral.PaymentInput{{
			PartyID:        doc.VendorExternalID,
			PostingDate:    doc.TxnDate,
			DocumentNumber: number,
			ExternalDocumentNumber: cleanText(
				doc.DocNumber,
				businesscentral.MaxExternalDocumentNumberLength,
			),
			Amount:             doc.Amount,
			AppliesToInvoiceID: strings.TrimSpace(doc.BillExternalID),
			Description:        cleanText(doc.PrivateNote, businesscentral.MaxDescriptionLength),
		}},
		refs: result.Refs,
	}); err != nil {
		return result, err
	}
	finishPayment(result, number, docTypeVendorPayment)
	return result, nil
}

func (c *Connector) UpdateBillPayment(
	_ context.Context,
	_ *services.AccountingBillPaymentDocument,
) (*services.AccountingDocumentResult, error) {
	return nil, reversalRefused()
}
