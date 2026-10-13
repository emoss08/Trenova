package businesscentral

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	StatusDraft      = "Draft"
	StatusInReview   = "In Review"
	StatusOpen       = "Open"
	StatusPaid       = "Paid"
	StatusCanceled   = "Canceled"
	StatusCorrective = "Corrective"
	LineTypeItem     = "Item"
	LineTypeAccount  = "Account"
	actionPost       = "post"
	actionCancel     = "cancel"
	idField          = "id"
)

type DocumentKind int

const (
	DocumentSalesInvoice DocumentKind = iota + 1
	DocumentSalesCreditMemo
	DocumentPurchaseInvoice
	DocumentPurchaseCreditMemo
)

type documentSpec struct {
	entity         string
	linesEntity    string
	referenceField string
	partyField     string
	sales          bool
	creditMemo     bool
	cancellable    bool
}

func (k DocumentKind) spec() (documentSpec, error) {
	switch k {
	case DocumentSalesInvoice:
		return documentSpec{
			entity: "salesInvoices", linesEntity: "salesInvoiceLines",
			referenceField: "externalDocumentNumber", partyField: "customerId",
			sales: true, cancellable: true,
		}, nil
	case DocumentSalesCreditMemo:
		return documentSpec{
			entity: "salesCreditMemos", linesEntity: "salesCreditMemoLines",
			referenceField: "externalDocumentNumber", partyField: "customerId",
			sales: true, creditMemo: true, cancellable: true,
		}, nil
	case DocumentPurchaseInvoice:
		return documentSpec{
			entity: "purchaseInvoices", linesEntity: "purchaseInvoiceLines",
			referenceField: "vendorInvoiceNumber", partyField: "vendorId",
		}, nil
	case DocumentPurchaseCreditMemo:
		return documentSpec{
			entity: "purchaseCreditMemos", linesEntity: "purchaseCreditMemoLines",
			referenceField: "vendorCreditMemoNumber", partyField: "vendorId",
			creditMemo: true, cancellable: true,
		}, nil
	default:
		return documentSpec{}, ErrUnknownKind
	}
}

type Document struct {
	Kind                    DocumentKind
	ID                      string
	Number                  string
	ExternalDocumentNumber  string
	DocumentDate            string
	PostingDate             string
	DueDate                 string
	PartyID                 string
	PartyNumber             string
	CurrencyCode            string
	PaymentTermsID          string
	InvoiceID               string
	InvoiceNumber           string
	Status                  string
	TotalAmountIncludingTax decimal.Decimal
	RemainingAmount         decimal.Decimal
	LastModified            time.Time
	ETag                    string
}

func (d *Document) IsDraft() bool {
	return d.Status == StatusDraft
}

type LineInput struct {
	ItemID      string
	AccountID   string
	Description string
	Quantity    decimal.Decimal
	UnitPrice   decimal.Decimal
}

type DocumentInput struct {
	PartyID                string
	ExternalDocumentNumber string
	DocumentDate           string
	PostingDate            string
	DueDate                string
	CurrencyCode           string
	PaymentTermsID         string
	InvoiceID              string
	Lines                  []LineInput
}

type Line struct {
	ID                 string
	DocumentID         string
	Sequence           int
	LineType           string
	ItemID             string
	AccountID          string
	Description        string
	Quantity           decimal.Decimal
	UnitPrice          decimal.Decimal
	AmountIncludingTax decimal.Decimal
}

type wireDocument struct {
	ID                      string          `json:"id"`
	Number                  string          `json:"number"`
	ExternalDocumentNumber  string          `json:"externalDocumentNumber"`
	VendorInvoiceNumber     string          `json:"vendorInvoiceNumber"`
	VendorCreditMemoNumber  string          `json:"vendorCreditMemoNumber"`
	InvoiceDate             string          `json:"invoiceDate"`
	CreditMemoDate          string          `json:"creditMemoDate"`
	PostingDate             string          `json:"postingDate"`
	DueDate                 string          `json:"dueDate"`
	CustomerID              string          `json:"customerId"`
	CustomerNumber          string          `json:"customerNumber"`
	VendorID                string          `json:"vendorId"`
	VendorNumber            string          `json:"vendorNumber"`
	CurrencyCode            string          `json:"currencyCode"`
	PaymentTermsID          string          `json:"paymentTermsId"`
	InvoiceID               string          `json:"invoiceId"`
	InvoiceNumber           string          `json:"invoiceNumber"`
	Status                  string          `json:"status"`
	TotalAmountIncludingTax decimal.Decimal `json:"totalAmountIncludingTax"`
	RemainingAmount         decimal.Decimal `json:"remainingAmount"`
	LastModifiedDateTime    wireTime        `json:"lastModifiedDateTime"`
	ETag                    string          `json:"@odata.etag"`
}

func (w *wireDocument) document(kind DocumentKind) Document {
	return Document{
		Kind:   kind,
		ID:     strings.ToLower(w.ID),
		Number: w.Number,
		ExternalDocumentNumber: firstNonBlank(
			w.ExternalDocumentNumber, w.VendorInvoiceNumber, w.VendorCreditMemoNumber,
		),
		DocumentDate:            outputDate(firstNonBlank(w.InvoiceDate, w.CreditMemoDate)),
		PostingDate:             outputDate(w.PostingDate),
		DueDate:                 outputDate(w.DueDate),
		PartyID:                 emptyGUID(firstNonBlank(w.CustomerID, w.VendorID)),
		PartyNumber:             firstNonBlank(w.CustomerNumber, w.VendorNumber),
		CurrencyCode:            strings.ToUpper(strings.TrimSpace(w.CurrencyCode)),
		PaymentTermsID:          emptyGUID(w.PaymentTermsID),
		InvoiceID:               emptyGUID(w.InvoiceID),
		InvoiceNumber:           w.InvoiceNumber,
		Status:                  enumValue(w.Status),
		TotalAmountIncludingTax: w.TotalAmountIncludingTax,
		RemainingAmount:         w.RemainingAmount,
		LastModified:            w.LastModifiedDateTime.time(),
		ETag:                    w.ETag,
	}
}

func documentConverter(kind DocumentKind) func(*wireDocument) Document {
	return func(w *wireDocument) Document { return w.document(kind) }
}

type lineBody struct {
	LineType       string  `json:"lineType"`
	ItemID         string  `json:"itemId,omitempty"`
	AccountID      string  `json:"accountId,omitempty"`
	Description    string  `json:"description,omitempty"`
	Quantity       *number `json:"quantity"`
	UnitPrice      *number `json:"unitPrice,omitempty"`
	DirectUnitCost *number `json:"directUnitCost,omitempty"`
}

type wireLine struct {
	ID                 string          `json:"id"`
	DocumentID         string          `json:"documentId"`
	Sequence           int             `json:"sequence"`
	LineType           string          `json:"lineType"`
	ItemID             string          `json:"itemId"`
	AccountID          string          `json:"accountId"`
	Description        string          `json:"description"`
	Quantity           decimal.Decimal `json:"quantity"`
	UnitPrice          decimal.Decimal `json:"unitPrice"`
	DirectUnitCost     decimal.Decimal `json:"directUnitCost"`
	AmountIncludingTax decimal.Decimal `json:"amountIncludingTax"`
}

func lineConverter(sales bool) func(*wireLine) Line {
	return func(w *wireLine) Line {
		price := w.DirectUnitCost
		if sales {
			price = w.UnitPrice
		}
		return Line{
			ID:                 strings.ToLower(w.ID),
			DocumentID:         strings.ToLower(w.DocumentID),
			Sequence:           w.Sequence,
			LineType:           enumValue(w.LineType),
			ItemID:             emptyGUID(w.ItemID),
			AccountID:          emptyGUID(w.AccountID),
			Description:        w.Description,
			Quantity:           w.Quantity,
			UnitPrice:          price,
			AmountIncludingTax: w.AmountIncludingTax,
		}
	}
}

type documentBody struct {
	ExternalDocumentNumber  string     `json:"externalDocumentNumber,omitempty"`
	VendorInvoiceNumber     string     `json:"vendorInvoiceNumber,omitempty"`
	VendorCreditMemoNumber  string     `json:"vendorCreditMemoNumber,omitempty"`
	InvoiceDate             string     `json:"invoiceDate,omitempty"`
	CreditMemoDate          string     `json:"creditMemoDate,omitempty"`
	PostingDate             string     `json:"postingDate,omitempty"`
	DueDate                 string     `json:"dueDate,omitempty"`
	CustomerID              string     `json:"customerId,omitempty"`
	VendorID                string     `json:"vendorId,omitempty"`
	CurrencyCode            string     `json:"currencyCode,omitempty"`
	PaymentTermsID          string     `json:"paymentTermsId,omitempty"`
	InvoiceID               string     `json:"invoiceId,omitempty"`
	SalesInvoiceLines       []lineBody `json:"salesInvoiceLines,omitempty"`
	SalesCreditMemoLines    []lineBody `json:"salesCreditMemoLines,omitempty"`
	PurchaseInvoiceLines    []lineBody `json:"purchaseInvoiceLines,omitempty"`
	PurchaseCreditMemoLines []lineBody `json:"purchaseCreditMemoLines,omitempty"`
}

type documentFields struct {
	reference string
	date      string
	party     string
	lines     []lineBody
}

func (b *documentBody) assign(kind DocumentKind, f *documentFields) {
	switch kind {
	case DocumentSalesInvoice:
		b.ExternalDocumentNumber, b.InvoiceDate = f.reference, f.date
		b.CustomerID, b.SalesInvoiceLines = f.party, f.lines
	case DocumentSalesCreditMemo:
		b.ExternalDocumentNumber, b.CreditMemoDate = f.reference, f.date
		b.CustomerID, b.SalesCreditMemoLines = f.party, f.lines
	case DocumentPurchaseInvoice:
		b.VendorInvoiceNumber, b.InvoiceDate = f.reference, f.date
		b.VendorID, b.PurchaseInvoiceLines = f.party, f.lines
	case DocumentPurchaseCreditMemo:
		b.VendorCreditMemoNumber, b.CreditMemoDate = f.reference, f.date
		b.VendorID, b.PurchaseCreditMemoLines = f.party, f.lines
	}
}

func (c *Client) Documents(
	ctx context.Context,
	kind DocumentKind,
	modifiedSince *time.Time,
) ([]Document, error) {
	spec, err := kind.spec()
	if err != nil {
		return nil, err
	}
	return collect(ctx, c.core, &listCall{
		endpoint: spec.entity,
		path:     c.collectionPath(spec.entity),
		filter:   filterModifiedSince(modifiedSince),
	}, documentConverter(kind))
}

func (c *Client) DocumentsByID(
	ctx context.Context,
	kind DocumentKind,
	ids []string,
) ([]Document, error) {
	spec, err := kind.spec()
	if err != nil {
		return nil, err
	}
	unique, err := guids(ids, MaxIDsPerRead)
	if err != nil {
		return nil, err
	}
	if len(unique) == 0 {
		return []Document{}, nil
	}
	return collect(ctx, c.core, &listCall{
		endpoint: spec.entity + "-by-id",
		path:     c.collectionPath(spec.entity),
		filter:   filterIn(idField, unique),
	}, documentConverter(kind))
}

func (c *Client) FindDocuments(
	ctx context.Context,
	kind DocumentKind,
	externalDocumentNumber, partyID string,
) ([]Document, error) {
	spec, err := kind.spec()
	if err != nil {
		return nil, err
	}
	reference := strings.TrimSpace(externalDocumentNumber)
	if !textWithin(reference, MaxExternalDocumentNumberLength) {
		return nil, ErrInvalidFilter
	}
	literal, err := odataString(reference)
	if err != nil {
		return nil, err
	}
	party, err := optionalGUID(partyID)
	if err != nil {
		return nil, err
	}
	partyClause := ""
	if party != "" {
		partyClause = filterEquals(spec.partyField, party)
	}
	return collect(ctx, c.core, &listCall{
		endpoint: spec.entity + suffixSearch,
		path:     c.collectionPath(spec.entity),
		filter:   filterJoin(filterEquals(spec.referenceField, literal), partyClause),
	}, documentConverter(kind))
}

func (c *Client) Document(
	ctx context.Context,
	kind DocumentKind,
	documentID string,
) (*Document, error) {
	spec, err := kind.spec()
	if err != nil {
		return nil, err
	}
	id, err := guid(documentID)
	if err != nil {
		return nil, err
	}
	return fetchOne(ctx, c.core, &call{
		endpoint: spec.entity + suffixGet,
		method:   http.MethodGet,
		path:     c.entityPath(spec.entity, id),
	}, documentConverter(kind))
}

func (c *Client) CreateDocument(
	ctx context.Context,
	kind DocumentKind,
	in *DocumentInput,
) (*Document, error) {
	spec, err := kind.spec()
	if err != nil {
		return nil, err
	}
	body, err := in.body(kind, &spec)
	if err != nil {
		return nil, err
	}
	return fetchOne(ctx, c.core, &call{
		endpoint: spec.entity + suffixCreate,
		method:   http.MethodPost,
		path:     c.collectionPath(spec.entity),
		body:     body,
		expected: []int{http.StatusCreated},
	}, documentConverter(kind))
}

func (c *Client) CreateDocumentLine(
	ctx context.Context,
	kind DocumentKind,
	documentID string,
	in *LineInput,
) (*Line, error) {
	spec, err := kind.spec()
	if err != nil {
		return nil, err
	}
	id, err := guid(documentID)
	if err != nil {
		return nil, err
	}
	body, err := in.body(spec.sales)
	if err != nil {
		return nil, err
	}
	return fetchOne(ctx, c.core, &call{
		endpoint: spec.linesEntity + suffixCreate,
		method:   http.MethodPost,
		path:     c.entityPath(spec.entity, id) + "/" + spec.linesEntity,
		body:     body,
		expected: []int{http.StatusCreated},
	}, lineConverter(spec.sales))
}

func (c *Client) DeleteDocument(
	ctx context.Context,
	kind DocumentKind,
	documentID, etag string,
) error {
	spec, err := kind.spec()
	if err != nil {
		return err
	}
	id, err := guid(documentID)
	if err != nil {
		return err
	}
	return c.remove(ctx, spec.entity+suffixDelete, c.entityPath(spec.entity, id), etag)
}

func (c *Client) PostDocument(ctx context.Context, kind DocumentKind, documentID string) error {
	return c.documentAction(ctx, kind, documentID, actionPost)
}

func (c *Client) CancelDocument(ctx context.Context, kind DocumentKind, documentID string) error {
	return c.documentAction(ctx, kind, documentID, actionCancel)
}

func (c *Client) documentAction(
	ctx context.Context,
	kind DocumentKind,
	documentID, action string,
) error {
	spec, err := kind.spec()
	if err != nil {
		return err
	}
	if action == actionCancel && !spec.cancellable {
		return ErrUnsupportedAction
	}
	id, err := guid(documentID)
	if err != nil {
		return err
	}
	return c.invoke(ctx, spec.entity+"-"+action, c.actionPath(spec.entity, id, action))
}

func (in *DocumentInput) body(kind DocumentKind, spec *documentSpec) (documentBody, error) {
	fields, err := in.fields(spec)
	if err != nil {
		return documentBody{}, err
	}
	var body documentBody
	body.assign(kind, &fields)
	if body.PostingDate, err = inputDate(in.PostingDate, false); err != nil {
		return documentBody{}, err
	}
	if body.DueDate, err = inputDate(in.DueDate, false); err != nil {
		return documentBody{}, err
	}
	if body.CurrencyCode, err = currencyCode(in.CurrencyCode); err != nil {
		return documentBody{}, err
	}
	if body.PaymentTermsID, err = optionalGUID(in.PaymentTermsID); err != nil {
		return documentBody{}, err
	}
	if body.PaymentTermsID != "" && !spec.sales {
		return documentBody{}, ErrFieldNotSupported
	}
	if body.InvoiceID, err = optionalGUID(in.InvoiceID); err != nil {
		return documentBody{}, err
	}
	if body.InvoiceID != "" && !spec.creditMemo {
		return documentBody{}, ErrFieldNotSupported
	}
	return body, nil
}

func (in *DocumentInput) fields(spec *documentSpec) (documentFields, error) {
	party, err := guid(in.PartyID)
	if err != nil {
		if strings.TrimSpace(in.PartyID) == "" {
			return documentFields{}, ErrPartyRequired
		}
		return documentFields{}, err
	}
	reference, err := cleanText(in.ExternalDocumentNumber, MaxExternalDocumentNumberLength)
	if err != nil {
		return documentFields{}, err
	}
	date, err := inputDate(in.DocumentDate, false)
	if err != nil {
		return documentFields{}, err
	}
	lines, err := lineBodies(in.Lines, spec.sales)
	if err != nil {
		return documentFields{}, err
	}
	return documentFields{reference: reference, date: date, party: party, lines: lines}, nil
}

func lineBodies(lines []LineInput, sales bool) ([]lineBody, error) {
	if len(lines) == 0 {
		return nil, nil
	}
	out := make([]lineBody, 0, len(lines))
	for idx := range lines {
		body, err := lines[idx].body(sales)
		if err != nil {
			return nil, err
		}
		out = append(out, body)
	}
	return out, nil
}

func (in *LineInput) body(sales bool) (lineBody, error) {
	itemID, err := optionalGUID(in.ItemID)
	if err != nil {
		return lineBody{}, err
	}
	accountID, err := optionalGUID(in.AccountID)
	if err != nil {
		return lineBody{}, err
	}
	if (itemID == "") == (accountID == "") {
		return lineBody{}, ErrLinesRequired
	}
	description, err := cleanText(in.Description, MaxDescriptionLength)
	if err != nil {
		return lineBody{}, err
	}
	if !in.Quantity.IsPositive() {
		return lineBody{}, ErrQuantityInvalid
	}
	if in.UnitPrice.IsNegative() {
		return lineBody{}, ErrPriceInvalid
	}
	body := lineBody{
		LineType:    LineTypeAccount,
		ItemID:      itemID,
		AccountID:   accountID,
		Description: description,
		Quantity:    numberOf(in.Quantity),
	}
	if itemID != "" {
		body.LineType = LineTypeItem
	}
	if sales {
		body.UnitPrice = numberOf(in.UnitPrice)
	} else {
		body.DirectUnitCost = numberOf(in.UnitPrice)
	}
	return body, nil
}
