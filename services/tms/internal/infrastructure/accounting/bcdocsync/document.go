package bcdocsync

import (
	"context"
	"maps"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/bcrefsync"
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

type Documents struct {
	*bcsession.Session
}

const (
	defaultLineText    = "Charges"
	idempotencyWindow  = 30 * 24 * time.Hour
	moneyPlaces        = 2
	referencePrefix    = "T"
	correctiveLookback = 5 * time.Minute
	RefReplaced        = "replaced"
	RefCorrective      = "corrective"
	refVoidCredit      = "voidCredit"
	RefAccountPrefix   = "account:"
)

func (c *Documents) DocumentLimits() accountingconnectorports.AccountingDocumentLimits {
	return accountingconnectorports.AccountingDocumentLimits{
		MaxDocNumberLength:      businesscentral.MaxExternalDocumentNumberLength,
		SupportsDebitMemo:       false,
		CanVoidCreditMemo:       true,
		CanVoidPurchaseDocument: true,
		IdempotencyWindow:       idempotencyWindow,
	}
}

func (c *Documents) DocumentURL(
	auth accountingconnectorports.AccountingDocumentAuth,
	link accountingconnectorports.AccountingDocumentLink,
) string {
	page := documentPage(link.Kind)
	number := strings.TrimSpace(link.DocNumber)
	name := strings.TrimSpace(auth.CompanyName)
	if page == 0 || number == "" || name == "" {
		return ""
	}
	ref, err := businesscentral.ParseCompanyRef(auth.RealmID)
	if err != nil {
		return ""
	}
	return c.Linker().DocumentLink(ref, name, page, number)
}

func documentPage(kind accountingsync.SyncObjectType) int {
	switch {
	case kind == accountingsync.SyncObjectCreditMemo:
		return businesscentral.PagePostedSalesCreditMemo
	case kind.IsSalesDocument():
		return businesscentral.PagePostedSalesInvoice
	case kind.IsBill():
		return businesscentral.PagePostedPurchaseInvoice
	default:
		return 0
	}
}

func purchasePage(kind businesscentraldocs.DocumentKind) int {
	if kind == businesscentraldocs.DocumentPurchaseCreditMemo {
		return businesscentral.PagePostedPurchaseCreditMemo
	}
	return businesscentral.PagePostedPurchaseInvoice
}

func SalesKindOf(kind accountingsync.SyncObjectType) (businesscentraldocs.DocumentKind, error) {
	if kind == accountingsync.SyncObjectInvoice || kind == accountingsync.SyncObjectDebitMemo {
		return businesscentraldocs.DocumentSalesInvoice, nil
	}
	if kind == accountingsync.SyncObjectCreditMemo {
		return businesscentraldocs.DocumentSalesCreditMemo, nil
	}
	return 0, bcsession.ErrDocumentKind
}

type documentWrite struct {
	client  *businesscentral.Client
	kind    businesscentraldocs.DocumentKind
	input   *businesscentraldocs.DocumentInput
	refs    map[string]string
	refKey  string
	exclude string
}

func postDocument(ctx context.Context, w *documentWrite) (*businesscentraldocs.Document, error) {
	current, found, err := resumeDocument(ctx, w)
	if err != nil {
		return nil, err
	}
	if !found {
		if current, err = businesscentraldocs.CreateDocument(ctx, w.client, w.kind, w.input); err != nil {
			return nil, err
		}
	}
	w.refs[w.refKey] = current.ID
	if !current.IsDraft() {
		return current, nil
	}
	if err = businesscentraldocs.PostDocument(ctx, w.client, w.kind, current.ID); err != nil {
		return nil, err
	}
	return businesscentraldocs.ReadDocument(ctx, w.client, w.kind, current.ID)
}

func resumeDocument(
	ctx context.Context,
	w *documentWrite,
) (*businesscentraldocs.Document, bool, error) {
	if id := strings.TrimSpace(w.refs[w.refKey]); id != "" {
		held, err := businesscentraldocs.ReadDocument(ctx, w.client, w.kind, id)
		switch {
		case err == nil && !Retired(held.Status) && !SameID(held.ID, w.exclude):
			return held, true, nil
		case err != nil && !businesscentralerrors.IsNotFound(err):
			return nil, false, err
		}
	}
	if w.input.ExternalDocumentNumber == "" {
		return nil, false, nil
	}
	listed, err := businesscentraldocs.FindDocuments(
		ctx,
		w.client,
		w.kind,
		w.input.ExternalDocumentNumber,
		w.input.PartyID,
	)
	if err != nil {
		return nil, false, err
	}
	var draft *businesscentraldocs.Document
	for idx := range listed {
		doc := &listed[idx]
		if Retired(doc.Status) || SameID(doc.ID, w.exclude) {
			continue
		}
		if !doc.IsDraft() {
			return doc, true, nil
		}
		if draft == nil {
			draft = doc
		}
	}
	return draft, draft != nil, nil
}

func Retired(status string) bool {
	return status == businesscentraldocs.StatusCanceled ||
		status == businesscentraldocs.StatusCorrective
}

func (c *Documents) UpsertCustomer(
	ctx context.Context,
	doc *accountingconnectorports.AccountingCustomerDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	return c.upsertParty(ctx, &partyWrite{
		auth:       doc.Auth,
		kind:       businesscentralref.PartyCustomer,
		externalID: doc.ExternalID,
		party:      &doc.Party,
	})
}

func (c *Documents) CreateSalesDocument(
	ctx context.Context,
	doc *accountingconnectorports.AccountingSalesDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	kind, err := SalesKindOf(doc.Kind)
	if err != nil {
		return nil, err
	}
	client, err := c.Client(doc.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(doc.Refs)
	if err = c.writeSales(ctx, client, kind, doc, result); err != nil {
		return result, err
	}
	return result, nil
}

func (c *Documents) writeSales(
	ctx context.Context,
	client *businesscentral.Client,
	kind businesscentraldocs.DocumentKind,
	doc *accountingconnectorports.AccountingSalesDocument,
	result *accountingconnectorports.AccountingDocumentResult,
) error {
	input, err := c.salesInput(ctx, client, kind, doc)
	if err != nil {
		return err
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
		return err
	}
	result.ExternalID = posted.ID
	result.DocNumber = posted.Number
	result.Refs[accountingsync.ExternalRefDocument] = posted.ID
	if input.InvoiceID != "" {
		result.Refs[accountingsync.ExternalRefApplication] = input.InvoiceID
	}
	return nil
}

func (c *Documents) salesInput(
	ctx context.Context,
	client *businesscentral.Client,
	kind businesscentraldocs.DocumentKind,
	doc *accountingconnectorports.AccountingSalesDocument,
) (*businesscentraldocs.DocumentInput, error) {
	lines, err := salesLines(doc.Lines)
	if err != nil {
		return nil, err
	}
	posting, err := c.PostingFor(ctx, client, doc.TxnDate)
	if err != nil {
		return nil, err
	}
	input := &businesscentraldocs.DocumentInput{
		PartyID:                doc.CustomerExternalID,
		ExternalDocumentNumber: DocReference(doc.DocNumber, doc.RequestID),
		DocumentDate:           doc.TxnDate,
		PostingDate:            doc.TxnDate,
		CurrencyCode:           posting.Currency(doc.CurrencyCode),
		Lines:                  lines,
	}
	if kind == businesscentraldocs.DocumentSalesInvoice {
		input.DueDate = doc.DueDate
		input.PaymentTermsID = doc.TermExternalID
	}
	if kind == businesscentraldocs.DocumentSalesCreditMemo &&
		strings.TrimSpace(doc.ApplyToExternalID) != "" && doc.ApplyAmount.IsPositive() {
		input.InvoiceID = strings.TrimSpace(doc.ApplyToExternalID)
	}
	return input, nil
}

func salesLines(
	lines []accountingconnectorports.AccountingDocumentLine,
) ([]businesscentraldocs.LineInput, error) {
	if len(lines) == 0 {
		return nil, bcsession.ErrLinesRequired
	}
	out := make([]businesscentraldocs.LineInput, 0, len(lines))
	for idx := range lines {
		line := &lines[idx]
		purpose := linePurpose(idx, line.Description)
		if strings.TrimSpace(line.ItemExternalID) == "" {
			return nil, unmappedItem(purpose)
		}
		if line.Amount.IsNegative() {
			return nil, negativeLine(purpose)
		}
		quantity, price := lineQuantity(line)
		out = append(out, businesscentraldocs.LineInput{
			ItemID:      strings.TrimSpace(line.ItemExternalID),
			Description: lineDescription(line.Description),
			Quantity:    quantity,
			UnitPrice:   price,
		})
	}
	return out, nil
}

func lineQuantity(
	line *accountingconnectorports.AccountingDocumentLine,
) (quantity, price decimal.Decimal) {
	if line.Quantity.IsPositive() && !line.UnitPrice.IsNegative() &&
		sameMoney(line.Quantity.Mul(line.UnitPrice), line.Amount) {
		return line.Quantity, line.UnitPrice
	}
	return decimal.NewFromInt(1), line.Amount
}

func linePurpose(idx int, description string) string {
	return "Line " + strconv.Itoa(idx+1) + " (" + lineText(description) + ")"
}

func lineText(description string) string {
	if text := strings.TrimSpace(description); text != "" {
		return text
	}
	return defaultLineText
}

func lineDescription(description string) string {
	return bcrefsync.CleanText(lineText(description), businesscentral.MaxDescriptionLength)
}

func (c *Documents) UpdateSalesDocument(
	ctx context.Context,
	doc *accountingconnectorports.AccountingSalesDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	kind, err := SalesKindOf(doc.Kind)
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(doc.ExternalID)
	if id == "" {
		return nil, bcsession.ErrExternalID
	}
	client, err := c.Client(doc.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(doc.Refs)
	corrective, err := retireDocument(ctx, client, kind, id)
	if err != nil {
		return result, err
	}
	result.Refs[RefReplaced] = id
	result.Refs[RefCorrective] = corrective
	result.Refs[accountingsync.ExternalRefDocument] = ""
	delete(result.Refs, accountingsync.ExternalRefApplication)
	if err = c.writeSales(ctx, client, kind, doc, result); err != nil {
		return result, err
	}
	return result, nil
}

func (c *Documents) VoidSalesDocument(
	ctx context.Context,
	ref *accountingconnectorports.AccountingDocumentRef,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	kind, err := SalesKindOf(ref.Kind)
	if err != nil {
		return nil, err
	}
	client, err := c.Client(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(ref.Refs)
	result.ExternalID = ref.ExternalID
	corrective, err := retireDocument(ctx, client, kind, ref.ExternalID)
	if err != nil {
		return result, err
	}
	if corrective != "" {
		result.Refs[RefCorrective] = corrective
	}
	return result, nil
}

func retireDocument(
	ctx context.Context,
	client *businesscentral.Client,
	kind businesscentraldocs.DocumentKind,
	id string,
) (string, error) {
	current, err := businesscentraldocs.ReadDocument(ctx, client, kind, id)
	if err != nil {
		if businesscentralerrors.IsNotFound(err) {
			return "", nil
		}
		return "", err
	}
	switch {
	case Retired(current.Status):
		return correctiveOf(ctx, client, kind, current)
	case current.IsDraft():
		if err = businesscentraldocs.DeleteDocument(ctx, client, kind, current.ID, current.ETag); err != nil &&
			!businesscentralerrors.IsNotFound(err) {
			return "", err
		}
		return "", nil
	case kind == businesscentraldocs.DocumentSalesInvoice && settled(current):
		return "", paidConflict()
	}
	if err = businesscentraldocs.CancelDocument(ctx, client, kind, current.ID); err != nil {
		return "", err
	}
	return correctiveOf(ctx, client, kind, current)
}

func settled(doc *businesscentraldocs.Document) bool {
	if doc.Status == businesscentraldocs.StatusPaid {
		return true
	}
	return doc.TotalAmountIncludingTax.IsPositive() &&
		doc.RemainingAmount.LessThan(doc.TotalAmountIncludingTax)
}

func correctiveOf(
	ctx context.Context,
	client *businesscentral.Client,
	kind businesscentraldocs.DocumentKind,
	invoice *businesscentraldocs.Document,
) (string, error) {
	if kind != businesscentraldocs.DocumentSalesInvoice || invoice.LastModified.IsZero() {
		return "", nil
	}
	since := invoice.LastModified.Add(-correctiveLookback)
	memos, err := businesscentraldocs.Documents(
		ctx,
		client,
		businesscentraldocs.DocumentSalesCreditMemo,
		&since,
	)
	if err != nil {
		return "", err
	}
	var found *businesscentraldocs.Document
	for idx := range memos {
		memo := &memos[idx]
		if !SameID(memo.InvoiceID, invoice.ID) {
			continue
		}
		if found == nil || memo.LastModified.After(found.LastModified) {
			found = memo
		}
	}
	if found == nil {
		return "", nil
	}
	return found.ID, nil
}

func DocReference(number, requestID string) string {
	value := bcrefsync.CleanText(number, businesscentral.MaxExternalDocumentNumberLength+1)
	if value != "" && len([]rune(value)) <= businesscentral.MaxExternalDocumentNumberLength {
		return value
	}
	return RequestReference(requestID)
}

func RequestReference(requestID string) string {
	digest := strings.ToUpper(hashutils.SHA256Hex(strings.TrimSpace(requestID)))
	return referencePrefix + digest[:businesscentral.MaxDocumentNumberLength-len(referencePrefix)]
}

func sameMoney(a, b decimal.Decimal) bool {
	return a.Round(moneyPlaces).Equal(b.Round(moneyPlaces))
}

func SameID(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func newResult(refs map[string]string) *accountingconnectorports.AccountingDocumentResult {
	copied := make(map[string]string, len(refs)+4)
	maps.Copy(copied, refs)
	return &accountingconnectorports.AccountingDocumentResult{Refs: copied}
}
