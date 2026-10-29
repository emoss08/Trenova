package xerodocs

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/xero"
	"github.com/emoss08/trenova/shared/xeroerrors"
	"github.com/shopspring/decimal"
)

const invoicesResource = "Invoices"

type InvoiceInput struct {
	Type            string
	ContactID       string
	InvoiceNumber   string
	Reference       string
	Date            time.Time
	DueDate         time.Time
	CurrencyCode    string
	CurrencyRate    decimal.Decimal
	Status          string
	LineAmountTypes string
	Lines           []LineItem
}

type Invoice struct {
	InvoiceID      string
	InvoiceNumber  string
	Type           string
	Status         string
	Reference      string
	CurrencyCode   string
	ContactID      string
	Total          decimal.Decimal
	AmountDue      decimal.Decimal
	AmountPaid     decimal.Decimal
	AmountCredited decimal.Decimal
	CurrencyRate   decimal.Decimal
	Date           time.Time
	DueDate        time.Time
	UpdatedAt      time.Time
	Payments       []DocumentPayment
	CreditNotes    []AppliedCredit
}

type InvoicePage struct {
	Invoices []Invoice
	More     bool
}

type invoiceBody struct {
	InvoiceID       string       `json:"InvoiceID,omitempty"`
	Type            string       `json:"Type,omitempty"`
	Contact         *xero.IDRef  `json:"Contact,omitempty"`
	InvoiceNumber   string       `json:"InvoiceNumber,omitempty"`
	Reference       string       `json:"Reference,omitempty"`
	Date            string       `json:"Date,omitempty"`
	DueDate         string       `json:"DueDate,omitempty"`
	CurrencyCode    string       `json:"CurrencyCode,omitempty"`
	CurrencyRate    *xero.Number `json:"CurrencyRate,omitempty"`
	Status          string       `json:"Status,omitempty"`
	LineAmountTypes string       `json:"LineAmountTypes,omitempty"`
	LineItems       []lineBody   `json:"LineItems,omitempty"`
}

type invoiceWriteEnvelope struct {
	Invoices []invoiceBody `json:"Invoices"`
}

type wireAppliedCredit struct {
	CreditNoteID  string          `json:"CreditNoteID"`
	AppliedAmount decimal.Decimal `json:"AppliedAmount"`
}

type wireInvoice struct {
	InvoiceID      string                `json:"InvoiceID"`
	InvoiceNumber  string                `json:"InvoiceNumber"`
	Type           string                `json:"Type"`
	Status         string                `json:"Status"`
	Reference      string                `json:"Reference"`
	CurrencyCode   string                `json:"CurrencyCode"`
	Contact        *xero.WireRef         `json:"Contact"`
	Total          decimal.Decimal       `json:"Total"`
	AmountDue      decimal.Decimal       `json:"AmountDue"`
	AmountPaid     decimal.Decimal       `json:"AmountPaid"`
	AmountCredited decimal.Decimal       `json:"AmountCredited"`
	CurrencyRate   decimal.Decimal       `json:"CurrencyRate"`
	Date           xero.WireTime         `json:"Date"`
	DueDate        xero.WireTime         `json:"DueDate"`
	UpdatedDateUTC xero.WireTime         `json:"UpdatedDateUTC"`
	Payments       []wireDocumentPayment `json:"Payments"`
	CreditNotes    []wireAppliedCredit   `json:"CreditNotes"`
}

func (w *wireInvoice) invoice() Invoice {
	out := Invoice{
		InvoiceID:      w.InvoiceID,
		InvoiceNumber:  w.InvoiceNumber,
		Type:           w.Type,
		Status:         w.Status,
		Reference:      w.Reference,
		CurrencyCode:   w.CurrencyCode,
		Total:          w.Total,
		AmountDue:      w.AmountDue,
		AmountPaid:     w.AmountPaid,
		AmountCredited: w.AmountCredited,
		CurrencyRate:   w.CurrencyRate,
		Date:           w.Date.Time(),
		DueDate:        w.DueDate.Time(),
		UpdatedAt:      w.UpdatedDateUTC.Time(),
		Payments:       documentPayments(w.Payments),
	}
	if w.Contact != nil {
		out.ContactID = w.Contact.ContactID
	}
	if len(w.CreditNotes) > 0 {
		out.CreditNotes = make([]AppliedCredit, 0, len(w.CreditNotes))
		for idx := range w.CreditNotes {
			out.CreditNotes = append(out.CreditNotes, AppliedCredit{
				CreditNoteID: w.CreditNotes[idx].CreditNoteID,
				Amount:       w.CreditNotes[idx].AppliedAmount,
			})
		}
	}
	return out
}

type invoicesEnvelope struct {
	Invoices   []wireInvoice        `json:"Invoices"`
	Pagination *xero.WirePagination `json:"pagination"`
}

func (e *invoicesEnvelope) invoices() []Invoice {
	out := make([]Invoice, 0, len(e.Invoices))
	for idx := range e.Invoices {
		out = append(out, e.Invoices[idx].invoice())
	}
	return out
}

//
//nolint:gocritic // value inputs are the package contract the accounting adapter is written against.
func CreateInvoice(
	ctx context.Context,
	c *xero.Client,
	key string,
	in InvoiceInput,
) (*Invoice, error) {
	body, err := in.body()
	if err != nil {
		return nil, err
	}
	return writeInvoice(ctx, c, key, "", &body)
}

//nolint:gocritic // value inputs are the package contract the accounting adapter is written against.
func UpdateInvoice(
	ctx context.Context,
	c *xero.Client,
	key, invoiceID string,
	in InvoiceInput,
) (*Invoice, error) {
	id, err := xero.GUID(invoiceID)
	if err != nil {
		return nil, err
	}
	body, err := in.body()
	if err != nil {
		return nil, err
	}
	body.InvoiceID = id
	return writeInvoice(ctx, c, key, id, &body)
}

func VoidInvoice(ctx context.Context, c *xero.Client, key, invoiceID string) (*Invoice, error) {
	id, err := xero.GUID(invoiceID)
	if err != nil {
		return nil, err
	}
	return writeInvoice(ctx, c, key, id, &invoiceBody{InvoiceID: id, Status: StatusVoided})
}

func (in *InvoiceInput) body() (invoiceBody, error) {
	spec := documentSpec{
		kind:            in.Type,
		contactID:       in.ContactID,
		number:          in.InvoiceNumber,
		reference:       in.Reference,
		date:            in.Date,
		currencyCode:    in.CurrencyCode,
		currencyRate:    in.CurrencyRate,
		status:          in.Status,
		lineAmountTypes: in.LineAmountTypes,
		lines:           in.Lines,
	}
	fields, err := spec.fields(InvoiceTypeReceivable, InvoiceTypePayable)
	if err != nil {
		return invoiceBody{}, err
	}
	return invoiceBody{
		Type:            fields.kind,
		Contact:         fields.contact,
		InvoiceNumber:   fields.number,
		Reference:       fields.reference,
		Date:            fields.date,
		DueDate:         xero.FormatDate(in.DueDate),
		CurrencyCode:    fields.currencyCode,
		CurrencyRate:    fields.currencyRate,
		Status:          fields.status,
		LineAmountTypes: fields.lineAmountTypes,
		LineItems:       fields.lines,
	}, nil
}

func writeInvoice(
	ctx context.Context,
	c *xero.Client,
	key, invoiceID string,
	body *invoiceBody,
) (*Invoice, error) {
	idem, err := xero.IdempotencyKey(key)
	if err != nil {
		return nil, err
	}
	req := &xero.Call{
		Endpoint: "invoices-create",
		Method:   http.MethodPut,
		Path:     xero.AccountingPath(invoicesResource),
		Query:    url.Values{unitDPParam: {unitDecimalPlaces}},
		Key:      idem,
		Body:     invoiceWriteEnvelope{Invoices: []invoiceBody{*body}},
	}
	if invoiceID != "" {
		req.Endpoint = "invoices-update"
		req.Method = http.MethodPost
		req.Path = xero.AccountingPath(invoicesResource, invoiceID)
	}

	var out invoicesEnvelope
	req.Out = &out
	if err = c.Do(ctx, req); err != nil {
		return nil, err
	}
	if len(out.Invoices) == 0 {
		return nil, xeroerrors.ErrUnexpectedPayload
	}
	written := out.Invoices[0].invoice()
	return &written, nil
}

func InvoicesByID(ctx context.Context, c *xero.Client, ids []string) ([]Invoice, error) {
	var out invoicesEnvelope
	requested, err := readByIDs(ctx, c, "invoices-by-id", invoicesResource, ids, &out)
	if err != nil {
		return nil, err
	}
	if !requested {
		return []Invoice{}, nil
	}
	return out.invoices(), nil
}

func Invoices(
	ctx context.Context,
	c *xero.Client,
	page int,
	modifiedSince *time.Time,
) (*InvoicePage, error) {
	query, err := xero.PageQuery(page)
	if err != nil {
		return nil, err
	}
	var out invoicesEnvelope
	if err = xero.ReadPage(ctx, c, "invoices", invoicesResource, query, modifiedSince, &out); err != nil {
		return nil, err
	}
	invoices := out.invoices()
	return &InvoicePage{
		Invoices: invoices,
		More:     xero.HasMore(out.Pagination, page, len(invoices)),
	}, nil
}

//nolint:gocritic // value inputs are the package contract the accounting adapter is written against.
func FindInvoices(ctx context.Context, c *xero.Client, f InvoiceFilter) ([]Invoice, error) {
	terms, err := f.terms(InvoiceTypeReceivable, InvoiceTypePayable)
	if err != nil {
		return nil, err
	}
	base := url.Values{}
	if len(terms.numbers) > 0 {
		base.Set("InvoiceNumbers", strings.Join(terms.numbers, ","))
	}
	if len(terms.contactIDs) > 0 {
		base.Set("ContactIDs", strings.Join(terms.contactIDs, ","))
	}
	if len(terms.statuses) > 0 {
		base.Set(xero.StatusesParam, strings.Join(terms.statuses, ","))
	}
	from, to := terms.dateClauses()
	if where := xero.WhereJoin(
		xero.WhereAnyOf("Type", xero.Optional(terms.kind), xero.WhereString),
		from,
		to,
	); where != "" {
		base.Set(xero.WhereParam, where)
	}

	return xero.CollectPages(func(page int) ([]Invoice, bool, error) {
		query, queryErr := xero.WithPage(base, page)
		if queryErr != nil {
			return nil, false, queryErr
		}
		var out invoicesEnvelope
		if readErr := xero.ReadPage(
			ctx,
			c,
			"invoices-search",
			invoicesResource,
			query,
			nil,
			&out,
		); readErr != nil {
			return nil, false, readErr
		}
		invoices := out.invoices()
		return invoices, xero.HasMore(out.Pagination, page, len(invoices)), nil
	})
}
