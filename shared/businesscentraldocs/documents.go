package businesscentraldocs

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/businesscentralerrors"
	"github.com/emoss08/trenova/shared/stringutils"
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
		return documentSpec{}, businesscentralerrors.ErrUnknownKind
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
	ID                      string                   `json:"id"`
	Number                  string                   `json:"number"`
	ExternalDocumentNumber  string                   `json:"externalDocumentNumber"`
	VendorInvoiceNumber     string                   `json:"vendorInvoiceNumber"`
	VendorCreditMemoNumber  string                   `json:"vendorCreditMemoNumber"`
	InvoiceDate             string                   `json:"invoiceDate"`
	CreditMemoDate          string                   `json:"creditMemoDate"`
	PostingDate             string                   `json:"postingDate"`
	DueDate                 string                   `json:"dueDate"`
	CustomerID              string                   `json:"customerId"`
	CustomerNumber          string                   `json:"customerNumber"`
	VendorID                string                   `json:"vendorId"`
	VendorNumber            string                   `json:"vendorNumber"`
	CurrencyCode            string                   `json:"currencyCode"`
	PaymentTermsID          string                   `json:"paymentTermsId"`
	InvoiceID               string                   `json:"invoiceId"`
	InvoiceNumber           string                   `json:"invoiceNumber"`
	Status                  string                   `json:"status"`
	TotalAmountIncludingTax decimal.Decimal          `json:"totalAmountIncludingTax"`
	RemainingAmount         decimal.Decimal          `json:"remainingAmount"`
	LastModifiedDateTime    businesscentral.WireTime `json:"lastModifiedDateTime"`
	ETag                    string                   `json:"@odata.etag"`
}

func (w *wireDocument) document(kind DocumentKind) Document {
	return Document{
		Kind:   kind,
		ID:     strings.ToLower(w.ID),
		Number: w.Number,
		ExternalDocumentNumber: stringutils.FirstNonEmptyTrimmed(
			w.ExternalDocumentNumber, w.VendorInvoiceNumber, w.VendorCreditMemoNumber),

		DocumentDate: businesscentral.OutputDate(
			stringutils.FirstNonEmptyTrimmed(w.InvoiceDate, w.CreditMemoDate),
		),
		PostingDate: businesscentral.OutputDate(w.PostingDate),
		DueDate:     businesscentral.OutputDate(w.DueDate),
		PartyID: businesscentral.EmptyGUID(
			stringutils.FirstNonEmptyTrimmed(w.CustomerID, w.VendorID),
		),
		PartyNumber:             stringutils.FirstNonEmptyTrimmed(w.CustomerNumber, w.VendorNumber),
		CurrencyCode:            strings.ToUpper(strings.TrimSpace(w.CurrencyCode)),
		PaymentTermsID:          businesscentral.EmptyGUID(w.PaymentTermsID),
		InvoiceID:               businesscentral.EmptyGUID(w.InvoiceID),
		InvoiceNumber:           w.InvoiceNumber,
		Status:                  businesscentral.EnumValue(w.Status),
		TotalAmountIncludingTax: w.TotalAmountIncludingTax,
		RemainingAmount:         w.RemainingAmount,
		LastModified:            w.LastModifiedDateTime.Time(),
		ETag:                    w.ETag,
	}
}

func documentConverter(kind DocumentKind) func(*wireDocument) Document {
	return func(w *wireDocument) Document { return w.document(kind) }
}

type lineBody struct {
	LineType       string                  `json:"lineType"`
	ItemID         string                  `json:"itemId,omitempty"`
	AccountID      string                  `json:"accountId,omitempty"`
	Description    string                  `json:"description,omitempty"`
	Quantity       *businesscentral.Number `json:"quantity"`
	UnitPrice      *businesscentral.Number `json:"unitPrice,omitempty"`
	DirectUnitCost *businesscentral.Number `json:"directUnitCost,omitempty"`
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
			LineType:           businesscentral.EnumValue(w.LineType),
			ItemID:             businesscentral.EmptyGUID(w.ItemID),
			AccountID:          businesscentral.EmptyGUID(w.AccountID),
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

func Documents(
	ctx context.Context,
	c *businesscentral.Client,
	kind DocumentKind,
	modifiedSince *time.Time,
) ([]Document, error) {
	spec, err := kind.spec()
	if err != nil {
		return nil, err
	}
	return businesscentral.Collect(ctx, c.Core(), &businesscentral.ListCall{
		Endpoint: spec.entity,
		Path:     c.CollectionPath(spec.entity),
		Filter:   businesscentral.FilterModifiedSince(modifiedSince),
	}, documentConverter(kind))
}

func DocumentsByID(
	ctx context.Context,
	c *businesscentral.Client,
	kind DocumentKind,
	ids []string,
) ([]Document, error) {
	spec, err := kind.spec()
	if err != nil {
		return nil, err
	}
	unique, err := businesscentral.GUIDs(ids, businesscentral.MaxIDsPerRead)
	if err != nil {
		return nil, err
	}
	if len(unique) == 0 {
		return []Document{}, nil
	}
	return businesscentral.Collect(ctx, c.Core(), &businesscentral.ListCall{
		Endpoint: spec.entity + "-by-id",
		Path:     c.CollectionPath(spec.entity),
		Filter:   businesscentral.FilterIn(idField, unique),
	}, documentConverter(kind))
}

func FindDocuments(
	ctx context.Context,
	c *businesscentral.Client,
	kind DocumentKind,
	externalDocumentNumber, partyID string,
) ([]Document, error) {
	spec, err := kind.spec()
	if err != nil {
		return nil, err
	}
	reference := strings.TrimSpace(externalDocumentNumber)
	if !businesscentral.TextWithin(reference, businesscentral.MaxExternalDocumentNumberLength) {
		return nil, businesscentralerrors.ErrInvalidFilter
	}
	literal, err := businesscentral.ODataString(reference)
	if err != nil {
		return nil, err
	}
	party, err := businesscentral.OptionalGUID(partyID)
	if err != nil {
		return nil, err
	}
	partyClause := ""
	if party != "" {
		partyClause = businesscentral.FilterEquals(spec.partyField, party)
	}
	return businesscentral.Collect(ctx, c.Core(), &businesscentral.ListCall{
		Endpoint: spec.entity + businesscentral.SuffixSearch,
		Path:     c.CollectionPath(spec.entity),
		Filter: businesscentral.FilterJoin(
			businesscentral.FilterEquals(spec.referenceField, literal),
			partyClause,
		),
	}, documentConverter(kind))
}

func ReadDocument(
	ctx context.Context,
	c *businesscentral.Client,
	kind DocumentKind,
	documentID string,
) (*Document, error) {
	spec, err := kind.spec()
	if err != nil {
		return nil, err
	}
	id, err := businesscentral.GUID(documentID)
	if err != nil {
		return nil, err
	}
	return businesscentral.FetchOne(ctx, c.Core(), &businesscentral.Call{
		Endpoint: spec.entity + businesscentral.SuffixGet,
		Method:   http.MethodGet,
		Path:     c.EntityPath(spec.entity, id),
	}, documentConverter(kind))
}

func CreateDocument(
	ctx context.Context,
	c *businesscentral.Client,
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
	return businesscentral.FetchOne(ctx, c.Core(), &businesscentral.Call{
		Endpoint: spec.entity + businesscentral.SuffixCreate,
		Method:   http.MethodPost,
		Path:     c.CollectionPath(spec.entity),
		Body:     body,
		Expected: []int{http.StatusCreated},
	}, documentConverter(kind))
}

func CreateDocumentLine(
	ctx context.Context,
	c *businesscentral.Client,
	kind DocumentKind,
	documentID string,
	in *LineInput,
) (*Line, error) {
	spec, err := kind.spec()
	if err != nil {
		return nil, err
	}
	id, err := businesscentral.GUID(documentID)
	if err != nil {
		return nil, err
	}
	body, err := in.body(spec.sales)
	if err != nil {
		return nil, err
	}
	return businesscentral.FetchOne(ctx, c.Core(), &businesscentral.Call{
		Endpoint: spec.linesEntity + businesscentral.SuffixCreate,
		Method:   http.MethodPost,
		Path:     c.EntityPath(spec.entity, id) + "/" + spec.linesEntity,
		Body:     body,
		Expected: []int{http.StatusCreated},
	}, lineConverter(spec.sales))
}

func DeleteDocument(
	ctx context.Context,
	c *businesscentral.Client,
	kind DocumentKind,
	documentID, etag string,
) error {
	spec, err := kind.spec()
	if err != nil {
		return err
	}
	id, err := businesscentral.GUID(documentID)
	if err != nil {
		return err
	}
	return c.Remove(
		ctx,
		spec.entity+businesscentral.SuffixDelete,
		c.EntityPath(spec.entity, id),
		etag,
	)
}

func PostDocument(
	ctx context.Context,
	c *businesscentral.Client,
	kind DocumentKind,
	documentID string,
) error {
	return documentAction(ctx, c, kind, documentID, businesscentral.ActionPost)
}

func CancelDocument(
	ctx context.Context,
	c *businesscentral.Client,
	kind DocumentKind,
	documentID string,
) error {
	return documentAction(ctx, c, kind, documentID, actionCancel)
}

func documentAction(
	ctx context.Context,
	c *businesscentral.Client,
	kind DocumentKind,
	documentID, action string,
) error {
	spec, err := kind.spec()
	if err != nil {
		return err
	}
	if action == actionCancel && !spec.cancellable {
		return businesscentralerrors.ErrUnsupportedAction
	}
	id, err := businesscentral.GUID(documentID)
	if err != nil {
		return err
	}
	return c.Invoke(ctx, spec.entity+"-"+action, c.ActionPath(spec.entity, id, action))
}

func (in *DocumentInput) body(kind DocumentKind, spec *documentSpec) (documentBody, error) {
	fields, err := in.fields(spec)
	if err != nil {
		return documentBody{}, err
	}
	var body documentBody
	body.assign(kind, &fields)
	if body.PostingDate, err = businesscentral.InputDate(in.PostingDate, false); err != nil {
		return documentBody{}, err
	}
	if body.DueDate, err = businesscentral.InputDate(in.DueDate, false); err != nil {
		return documentBody{}, err
	}
	if body.CurrencyCode, err = businesscentral.CurrencyCode(in.CurrencyCode); err != nil {
		return documentBody{}, err
	}
	if body.PaymentTermsID, err = businesscentral.OptionalGUID(in.PaymentTermsID); err != nil {
		return documentBody{}, err
	}
	if body.PaymentTermsID != "" && !spec.sales {
		return documentBody{}, businesscentralerrors.ErrFieldNotSupported
	}
	if body.InvoiceID, err = businesscentral.OptionalGUID(in.InvoiceID); err != nil {
		return documentBody{}, err
	}
	if body.InvoiceID != "" && !spec.creditMemo {
		return documentBody{}, businesscentralerrors.ErrFieldNotSupported
	}
	return body, nil
}

func (in *DocumentInput) fields(spec *documentSpec) (documentFields, error) {
	party, err := businesscentral.GUID(in.PartyID)
	if err != nil {
		if strings.TrimSpace(in.PartyID) == "" {
			return documentFields{}, businesscentralerrors.ErrPartyRequired
		}
		return documentFields{}, err
	}
	reference, err := businesscentral.CleanText(
		in.ExternalDocumentNumber,
		businesscentral.MaxExternalDocumentNumberLength,
	)
	if err != nil {
		return documentFields{}, err
	}
	date, err := businesscentral.InputDate(in.DocumentDate, false)
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
	itemID, err := businesscentral.OptionalGUID(in.ItemID)
	if err != nil {
		return lineBody{}, err
	}
	accountID, err := businesscentral.OptionalGUID(in.AccountID)
	if err != nil {
		return lineBody{}, err
	}
	if (itemID == "") == (accountID == "") {
		return lineBody{}, businesscentralerrors.ErrLinesRequired
	}
	description, err := businesscentral.CleanText(
		in.Description,
		businesscentral.MaxDescriptionLength,
	)
	if err != nil {
		return lineBody{}, err
	}
	if !in.Quantity.IsPositive() {
		return lineBody{}, businesscentralerrors.ErrQuantityInvalid
	}
	if in.UnitPrice.IsNegative() {
		return lineBody{}, businesscentralerrors.ErrPriceInvalid
	}
	body := lineBody{
		LineType:    LineTypeAccount,
		ItemID:      itemID,
		AccountID:   accountID,
		Description: description,
		Quantity:    businesscentral.NumberOf(in.Quantity),
	}
	if itemID != "" {
		body.LineType = LineTypeItem
	}
	if sales {
		body.UnitPrice = businesscentral.NumberOf(in.UnitPrice)
	} else {
		body.DirectUnitCost = businesscentral.NumberOf(in.UnitPrice)
	}
	return body, nil
}
