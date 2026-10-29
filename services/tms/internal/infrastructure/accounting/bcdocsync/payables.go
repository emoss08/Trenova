package bcdocsync

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/bcsession"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/businesscentraldocs"
	"github.com/emoss08/trenova/shared/businesscentralerrors"
	"github.com/emoss08/trenova/shared/businesscentralref"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/shopspring/decimal"
)

const (
	DocTypePurchaseInvoice    = "PurchaseInvoice"
	DocTypePurchaseCreditMemo = "PurchaseCreditMemo"
	revisionSeparator         = "-R"
	revisionHashLength        = 4
)

func (c *Documents) UpsertVendor(
	ctx context.Context,
	doc *accountingconnectorports.AccountingVendorDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	return c.upsertParty(ctx, &partyWrite{
		auth:       doc.Auth,
		kind:       businesscentralref.PartyVendor,
		externalID: doc.ExternalID,
		party:      &doc.Party,
	})
}

func PurchaseKindOf(credit bool) businesscentraldocs.DocumentKind {
	if credit {
		return businesscentraldocs.DocumentPurchaseCreditMemo
	}
	return businesscentraldocs.DocumentPurchaseInvoice
}

func (c *Documents) CreatePurchaseDocument(
	ctx context.Context,
	doc *accountingconnectorports.AccountingPurchaseDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if !doc.Kind.IsBill() {
		return nil, bcsession.ErrDocumentKind
	}
	client, err := c.Client(doc.Auth)
	if err != nil {
		return nil, err
	}
	return c.writePurchase(ctx, client, doc, newResult(nil))
}

func (c *Documents) writePurchase(
	ctx context.Context,
	client *businesscentral.Client,
	doc *accountingconnectorports.AccountingPurchaseDocument,
	result *accountingconnectorports.AccountingDocumentResult,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	kind := PurchaseKindOf(doc.VendorCredit)
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
		exclude: result.Refs[RefReplaced],
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

func accountTotals(
	lines []accountingconnectorports.AccountingPurchaseLine,
) map[string]decimal.Decimal {
	totals := make(map[string]decimal.Decimal, len(lines))
	for idx := range lines {
		key := RefAccountPrefix + strings.ToLower(strings.TrimSpace(lines[idx].AccountExternalID))
		totals[key] = totals[key].Add(lines[idx].Amount)
	}
	return totals
}

func (c *Documents) purchaseInput(
	ctx context.Context,
	client *businesscentral.Client,
	doc *accountingconnectorports.AccountingPurchaseDocument,
) (*businesscentraldocs.DocumentInput, error) {
	lines, err := purchaseLines(doc.Lines)
	if err != nil {
		return nil, err
	}
	posting, err := c.PostingFor(ctx, client, doc.TxnDate)
	if err != nil {
		return nil, err
	}
	input := &businesscentraldocs.DocumentInput{
		PartyID:                doc.VendorExternalID,
		ExternalDocumentNumber: DocReference(doc.DocNumber, doc.RequestID),
		DocumentDate:           doc.TxnDate,
		PostingDate:            doc.TxnDate,
		CurrencyCode:           posting.Currency(doc.CurrencyCode),
		Lines:                  lines,
	}
	if !doc.VendorCredit {
		input.DueDate = doc.DueDate
	}
	return input, nil
}

func purchaseLines(
	lines []accountingconnectorports.AccountingPurchaseLine,
) ([]businesscentraldocs.LineInput, error) {
	if len(lines) == 0 {
		return nil, bcsession.ErrLinesRequired
	}
	out := make([]businesscentraldocs.LineInput, 0, len(lines))
	for idx := range lines {
		line := &lines[idx]
		purpose := linePurpose(idx, line.Description)
		if strings.TrimSpace(line.AccountExternalID) == "" {
			return nil, unmappedAccount(purpose)
		}
		if line.Amount.IsNegative() {
			return nil, negativeLine(purpose)
		}
		out = append(out, businesscentraldocs.LineInput{
			AccountID:   strings.TrimSpace(line.AccountExternalID),
			Description: lineDescription(line.Description),
			Quantity:    decimal.NewFromInt(1),
			UnitPrice:   line.Amount,
		})
	}
	return out, nil
}

type purchaseWritten struct {
	auth   accountingconnectorports.AccountingDocumentAuth
	kind   businesscentraldocs.DocumentKind
	doc    *businesscentraldocs.Document
	result *accountingconnectorports.AccountingDocumentResult
}

func (c *Documents) finishPurchase(written *purchaseWritten) {
	credit := written.kind == businesscentraldocs.DocumentPurchaseCreditMemo
	docType := DocTypePurchaseInvoice
	if credit {
		docType = DocTypePurchaseCreditMemo
	}
	result := written.result
	result.ExternalID = written.doc.ID
	result.DocNumber = written.doc.Number
	result.Refs[accountingsync.ExternalRefDocument] = written.doc.ID
	result.Refs[accountingsync.ExternalRefDocumentType] = docType
	result.Refs[accountingsync.ExternalRefCreditDocument] = strconv.FormatBool(credit)
	result.Refs[accountingsync.ExternalRefURL] = c.purchaseURL(written)
}

func (c *Documents) purchaseURL(written *purchaseWritten) string {
	name := strings.TrimSpace(written.auth.CompanyName)
	ref, err := businesscentral.ParseCompanyRef(written.auth.RealmID)
	if err != nil || name == "" {
		return ""
	}
	return c.Linker().DocumentLink(
		ref,
		name,
		purchasePage(written.kind),
		written.doc.Number,
	)
}

func PurchaseIsCredit(refs map[string]string) (bool, error) {
	if value := strings.TrimSpace(refs[accountingsync.ExternalRefCreditDocument]); value != "" {
		credit, err := strconv.ParseBool(value)
		if err != nil {
			return false, bcsession.ErrPurchaseDocumentType
		}
		return credit, nil
	}
	switch strings.TrimSpace(refs[accountingsync.ExternalRefDocumentType]) {
	case "", DocTypePurchaseInvoice:
		return false, nil
	case DocTypePurchaseCreditMemo:
		return true, nil
	default:
		return false, bcsession.ErrPurchaseDocumentType
	}
}

func (c *Documents) UpdatePurchaseDocument(
	ctx context.Context,
	doc *accountingconnectorports.AccountingPurchaseDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if !doc.Kind.IsBill() {
		return nil, bcsession.ErrDocumentKind
	}
	id := strings.TrimSpace(doc.ExternalID)
	if id == "" {
		return nil, bcsession.ErrExternalID
	}
	client, err := c.Client(doc.Auth)
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
	result.Refs[RefReplaced] = id
	result.Refs[RefCorrective] = result.Refs[VoidCreditKey(id)]
	revised := *doc
	revised.DocNumber = RevisedNumber(doc.DocNumber, doc.RequestID)
	return c.writePurchase(ctx, client, &revised, result)
}

func RevisedNumber(number, requestID string) string {
	number = strings.TrimSpace(number)
	if number == "" {
		return ""
	}
	suffix := revisionSeparator + strings.ToUpper(
		hashutils.SHA256Hex(requestID)[:revisionHashLength],
	)
	limit := businesscentral.MaxExternalDocumentNumberLength - len(suffix)
	if len(number) > limit {
		number = number[:limit]
	}
	return number + suffix
}

func firstAccount(lines []accountingconnectorports.AccountingPurchaseLine) map[string]string {
	for idx := range lines {
		if id := strings.TrimSpace(lines[idx].AccountExternalID); id != "" {
			return map[string]string{RefAccountPrefix + strings.ToLower(id): ""}
		}
	}
	return nil
}

func (c *Documents) VoidPurchaseDocument(
	ctx context.Context,
	ref *accountingconnectorports.AccountingDocumentRef,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if ref.Kind.IsBillPayment() {
		return nil, reversalRefused()
	}
	if !ref.Kind.IsBill() {
		return nil, bcsession.ErrDocumentKind
	}
	credit, err := PurchaseIsCredit(ref.Refs)
	if err != nil {
		return nil, err
	}
	client, err := c.Client(ref.Auth)
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

func (c *Documents) retirePurchase(ctx context.Context, r *billRetirement) error {
	if r.credit {
		_, err := retireDocument(
			ctx,
			r.client,
			businesscentraldocs.DocumentPurchaseCreditMemo,
			r.id,
		)
		return err
	}
	bill, err := businesscentraldocs.ReadDocument(
		ctx,
		r.client,
		businesscentraldocs.DocumentPurchaseInvoice,
		r.id,
	)
	if err != nil {
		if businesscentralerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if bill.IsDraft() {
		err = businesscentraldocs.DeleteDocument(
			ctx,
			r.client,
			businesscentraldocs.DocumentPurchaseInvoice,
			bill.ID,
			bill.ETag,
		)
		if err != nil && !businesscentralerrors.IsNotFound(err) {
			return err
		}
		return nil
	}
	return c.creditBill(ctx, r, bill)
}

func (c *Documents) creditBill(
	ctx context.Context,
	r *billRetirement,
	bill *businesscentraldocs.Document,
) error {
	input := &businesscentraldocs.DocumentInput{
		PartyID:                bill.PartyID,
		ExternalDocumentNumber: RequestReference(r.requestID),
		InvoiceID:              bill.ID,
	}
	key := VoidCreditKey(bill.ID)
	held, found, err := resumeDocument(ctx, &documentWrite{
		client: r.client,
		kind:   businesscentraldocs.DocumentPurchaseCreditMemo,
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
	setup, err := c.PostingSetup(ctx, r.client)
	if err != nil {
		return err
	}
	input.CurrencyCode = bcsession.DocumentCurrency(setup, bill.CurrencyCode)
	memo, err := postDocument(ctx, &documentWrite{
		client: r.client,
		kind:   businesscentraldocs.DocumentPurchaseCreditMemo,
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

func VoidCreditKey(billID string) string {
	return refVoidCredit + ":" + strings.ToLower(strings.TrimSpace(billID))
}

func creditLines(
	accounts map[string]string,
	total decimal.Decimal,
) ([]businesscentraldocs.LineInput, error) {
	keys := make([]string, 0, len(accounts))
	for key := range accounts {
		if strings.HasPrefix(key, RefAccountPrefix) && len(key) > len(RefAccountPrefix) {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 || !total.IsPositive() {
		return nil, billLinesUnknown()
	}
	slices.Sort(keys)
	if len(keys) == 1 {
		return []businesscentraldocs.LineInput{creditLine(keys[0], total)}, nil
	}
	lines := make([]businesscentraldocs.LineInput, 0, len(keys))
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

func creditLine(key string, amount decimal.Decimal) businesscentraldocs.LineInput {
	return businesscentraldocs.LineInput{
		AccountID:   strings.TrimPrefix(key, RefAccountPrefix),
		Description: "Void",
		Quantity:    decimal.NewFromInt(1),
		UnitPrice:   amount,
	}
}
