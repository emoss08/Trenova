package bcconnector

import (
	"context"
	"errors"
	"maps"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

var _ services.AccountingDocumentWriter = (*Connector)(nil)

const (
	providerName       = "Business Central"
	defaultLineText    = "Charges"
	idempotencyWindow  = 30 * 24 * time.Hour
	moneyPlaces        = 2
	referencePrefix    = "T"
	correctiveLookback = 5 * time.Minute
	refReplaced        = "replaced"
	refCorrective      = "corrective"
	refJournal         = "journal"
	refVoidCredit      = "voidCredit"
	refAccountPrefix   = "account:"
)

var (
	errDocumentKind = errors.New(
		"businesscentral: this record type is not a document Business Central holds",
	)
	errExternalID = errors.New(
		"businesscentral: changing a document needs the id Business Central gave it",
	)
	errLinesRequired = errors.New("businesscentral: a document needs at least one line")
)

func (c *Connector) DocumentLimits() services.AccountingDocumentLimits {
	return services.AccountingDocumentLimits{
		MaxDocNumberLength:      businesscentral.MaxExternalDocumentNumberLength,
		SupportsDebitMemo:       false,
		CanVoidCreditMemo:       true,
		CanVoidPurchaseDocument: true,
		IdempotencyWindow:       idempotencyWindow,
	}
}

func (c *Connector) DocumentURL(
	auth services.AccountingDocumentAuth,
	link services.AccountingDocumentLink,
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
	return businesscentral.NewLinker(c.apiOpts...).DocumentLink(ref, name, page, number)
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

func purchasePage(kind businesscentral.DocumentKind) int {
	if kind == businesscentral.DocumentPurchaseCreditMemo {
		return businesscentral.PagePostedPurchaseCreditMemo
	}
	return businesscentral.PagePostedPurchaseInvoice
}

func salesKindOf(kind accountingsync.SyncObjectType) (businesscentral.DocumentKind, error) {
	if kind == accountingsync.SyncObjectInvoice || kind == accountingsync.SyncObjectDebitMemo {
		return businesscentral.DocumentSalesInvoice, nil
	}
	if kind == accountingsync.SyncObjectCreditMemo {
		return businesscentral.DocumentSalesCreditMemo, nil
	}
	return 0, errDocumentKind
}

type documentWrite struct {
	client *businesscentral.Client
	kind   businesscentral.DocumentKind
	input  *businesscentral.DocumentInput
	refs    map[string]string
	refKey  string
	exclude string
}

func postDocument(ctx context.Context, w *documentWrite) (*businesscentral.Document, error) {
	current, found, err := resumeDocument(ctx, w)
	if err != nil {
		return nil, err
	}
	if !found {
		if current, err = w.client.CreateDocument(ctx, w.kind, w.input); err != nil {
			return nil, err
		}
	}
	w.refs[w.refKey] = current.ID
	if !current.IsDraft() {
		return current, nil
	}
	if err = w.client.PostDocument(ctx, w.kind, current.ID); err != nil {
		return nil, err
	}
	return w.client.Document(ctx, w.kind, current.ID)
}

func resumeDocument(
	ctx context.Context,
	w *documentWrite,
) (*businesscentral.Document, bool, error) {
	if id := strings.TrimSpace(w.refs[w.refKey]); id != "" {
		held, err := w.client.Document(ctx, w.kind, id)
		switch {
		case err == nil && !retired(held.Status) && !sameID(held.ID, w.exclude):
			return held, true, nil
		case err != nil && !businesscentral.IsNotFound(err):
			return nil, false, err
		}
	}
	if w.input.ExternalDocumentNumber == "" {
		return nil, false, nil
	}
	listed, err := w.client.FindDocuments(ctx, w.kind, w.input.ExternalDocumentNumber, w.input.PartyID)
	if err != nil {
		return nil, false, err
	}
	var draft *businesscentral.Document
	for idx := range listed {
		doc := &listed[idx]
		if retired(doc.Status) || sameID(doc.ID, w.exclude) {
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

func retired(status string) bool {
	return status == businesscentral.StatusCanceled || status == businesscentral.StatusCorrective
}

func (c *Connector) UpsertCustomer(
	ctx context.Context,
	doc *services.AccountingCustomerDocument,
) (*services.AccountingDocumentResult, error) {
	return c.upsertParty(ctx, &partyWrite{
		auth:       doc.Auth,
		kind:       businesscentral.PartyCustomer,
		externalID: doc.ExternalID,
		party:      &doc.Party,
	})
}

func (c *Connector) CreateSalesDocument(
	ctx context.Context,
	doc *services.AccountingSalesDocument,
) (*services.AccountingDocumentResult, error) {
	kind, err := salesKindOf(doc.Kind)
	if err != nil {
		return nil, err
	}
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(doc.Refs)
	if err = c.writeSales(ctx, client, kind, doc, result); err != nil {
		return result, err
	}
	return result, nil
}

func (c *Connector) writeSales(
	ctx context.Context,
	client *businesscentral.Client,
	kind businesscentral.DocumentKind,
	doc *services.AccountingSalesDocument,
	result *services.AccountingDocumentResult,
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
		exclude: result.Refs[refReplaced],
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

func (c *Connector) salesInput(
	ctx context.Context,
	client *businesscentral.Client,
	kind businesscentral.DocumentKind,
	doc *services.AccountingSalesDocument,
) (*businesscentral.DocumentInput, error) {
	lines, err := salesLines(doc.Lines)
	if err != nil {
		return nil, err
	}
	posting, err := c.postingFor(ctx, client, doc.TxnDate)
	if err != nil {
		return nil, err
	}
	input := &businesscentral.DocumentInput{
		PartyID:                doc.CustomerExternalID,
		ExternalDocumentNumber: docReference(doc.DocNumber, doc.RequestID),
		DocumentDate:           doc.TxnDate,
		PostingDate:            doc.TxnDate,
		CurrencyCode:           posting.currency(doc.CurrencyCode),
		Lines:                  lines,
	}
	if kind == businesscentral.DocumentSalesInvoice {
		input.DueDate = doc.DueDate
		input.PaymentTermsID = doc.TermExternalID
	}
	if kind == businesscentral.DocumentSalesCreditMemo &&
		strings.TrimSpace(doc.ApplyToExternalID) != "" && doc.ApplyAmount.IsPositive() {
		input.InvoiceID = strings.TrimSpace(doc.ApplyToExternalID)
	}
	return input, nil
}

func salesLines(lines []services.AccountingDocumentLine) ([]businesscentral.LineInput, error) {
	if len(lines) == 0 {
		return nil, errLinesRequired
	}
	out := make([]businesscentral.LineInput, 0, len(lines))
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
		out = append(out, businesscentral.LineInput{
			ItemID:      strings.TrimSpace(line.ItemExternalID),
			Description: lineDescription(line.Description),
			Quantity:    quantity,
			UnitPrice:   price,
		})
	}
	return out, nil
}

func lineQuantity(line *services.AccountingDocumentLine) (quantity, price decimal.Decimal) {
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
	return cleanText(lineText(description), businesscentral.MaxDescriptionLength)
}

func (c *Connector) UpdateSalesDocument(
	ctx context.Context,
	doc *services.AccountingSalesDocument,
) (*services.AccountingDocumentResult, error) {
	kind, err := salesKindOf(doc.Kind)
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(doc.ExternalID)
	if id == "" {
		return nil, errExternalID
	}
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(doc.Refs)
	corrective, err := c.retireDocument(ctx, client, kind, id)
	if err != nil {
		return result, err
	}
	result.Refs[refReplaced] = id
	result.Refs[refCorrective] = corrective
	result.Refs[accountingsync.ExternalRefDocument] = ""
	delete(result.Refs, accountingsync.ExternalRefApplication)
	if err = c.writeSales(ctx, client, kind, doc, result); err != nil {
		return result, err
	}
	return result, nil
}

func (c *Connector) VoidSalesDocument(
	ctx context.Context,
	ref *services.AccountingDocumentRef,
) (*services.AccountingDocumentResult, error) {
	kind, err := salesKindOf(ref.Kind)
	if err != nil {
		return nil, err
	}
	client, err := c.client(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(ref.Refs)
	result.ExternalID = ref.ExternalID
	corrective, err := c.retireDocument(ctx, client, kind, ref.ExternalID)
	if err != nil {
		return result, err
	}
	if corrective != "" {
		result.Refs[refCorrective] = corrective
	}
	return result, nil
}

func (c *Connector) retireDocument(
	ctx context.Context,
	client *businesscentral.Client,
	kind businesscentral.DocumentKind,
	id string,
) (string, error) {
	current, err := client.Document(ctx, kind, id)
	if err != nil {
		if businesscentral.IsNotFound(err) {
			return "", nil
		}
		return "", err
	}
	switch {
	case retired(current.Status):
		return correctiveOf(ctx, client, kind, current)
	case current.IsDraft():
		if err = client.DeleteDocument(ctx, kind, current.ID, current.ETag); err != nil &&
			!businesscentral.IsNotFound(err) {
			return "", err
		}
		return "", nil
	case kind == businesscentral.DocumentSalesInvoice && settled(current):
		return "", paidConflict()
	}
	if err = client.CancelDocument(ctx, kind, current.ID); err != nil {
		return "", err
	}
	return correctiveOf(ctx, client, kind, current)
}

func settled(doc *businesscentral.Document) bool {
	if doc.Status == businesscentral.StatusPaid {
		return true
	}
	return doc.TotalAmountIncludingTax.IsPositive() &&
		doc.RemainingAmount.LessThan(doc.TotalAmountIncludingTax)
}

func correctiveOf(
	ctx context.Context,
	client *businesscentral.Client,
	kind businesscentral.DocumentKind,
	invoice *businesscentral.Document,
) (string, error) {
	if kind != businesscentral.DocumentSalesInvoice || invoice.LastModified.IsZero() {
		return "", nil
	}
	since := invoice.LastModified.Add(-correctiveLookback)
	memos, err := client.Documents(ctx, businesscentral.DocumentSalesCreditMemo, &since)
	if err != nil {
		return "", err
	}
	var found *businesscentral.Document
	for idx := range memos {
		memo := &memos[idx]
		if !sameID(memo.InvoiceID, invoice.ID) {
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

func docReference(number, requestID string) string {
	value := cleanText(number, businesscentral.MaxExternalDocumentNumberLength+1)
	if value != "" && len([]rune(value)) <= businesscentral.MaxExternalDocumentNumberLength {
		return value
	}
	return requestReference(requestID)
}

func requestReference(requestID string) string {
	digest := strings.ToUpper(hashutils.SHA256Hex(strings.TrimSpace(requestID)))
	return referencePrefix + digest[:businesscentral.MaxDocumentNumberLength-len(referencePrefix)]
}

func cleanText(value string, limit int) string {
	stripped := strings.Map(func(r rune) rune {
		if r < ' ' || r == 0x7f || (r >= 0x80 && r < 0xa0) {
			return ' '
		}
		return r
	}, value)
	return strings.TrimSpace(stringutils.OneLine(stripped, limit))
}

func sameMoney(a, b decimal.Decimal) bool {
	return a.Round(moneyPlaces).Equal(b.Round(moneyPlaces))
}

func sameID(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func newResult(refs map[string]string) *services.AccountingDocumentResult {
	copied := make(map[string]string, len(refs)+4)
	maps.Copy(copied, refs)
	return &services.AccountingDocumentResult{Refs: copied}
}

func unixOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func unixOf(t time.Time) *int64 {
	if t.IsZero() {
		return nil
	}
	value := t.Unix()
	return &value
}
