package xero

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

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
	InvoiceID       string     `json:"InvoiceID,omitempty"`
	Type            string     `json:"Type,omitempty"`
	Contact         *idRef     `json:"Contact,omitempty"`
	InvoiceNumber   string     `json:"InvoiceNumber,omitempty"`
	Reference       string     `json:"Reference,omitempty"`
	Date            string     `json:"Date,omitempty"`
	DueDate         string     `json:"DueDate,omitempty"`
	CurrencyCode    string     `json:"CurrencyCode,omitempty"`
	CurrencyRate    *number    `json:"CurrencyRate,omitempty"`
	Status          string     `json:"Status,omitempty"`
	LineAmountTypes string     `json:"LineAmountTypes,omitempty"`
	LineItems       []lineBody `json:"LineItems,omitempty"`
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
	Contact        *wireRef              `json:"Contact"`
	Total          decimal.Decimal       `json:"Total"`
	AmountDue      decimal.Decimal       `json:"AmountDue"`
	AmountPaid     decimal.Decimal       `json:"AmountPaid"`
	AmountCredited decimal.Decimal       `json:"AmountCredited"`
	CurrencyRate   decimal.Decimal       `json:"CurrencyRate"`
	Date           wireTime              `json:"Date"`
	DueDate        wireTime              `json:"DueDate"`
	UpdatedDateUTC wireTime              `json:"UpdatedDateUTC"`
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
		Date:           w.Date.time(),
		DueDate:        w.DueDate.time(),
		UpdatedAt:      w.UpdatedDateUTC.time(),
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
	Invoices   []wireInvoice   `json:"Invoices"`
	Pagination *wirePagination `json:"pagination"`
}

func (e *invoicesEnvelope) invoices() []Invoice {
	out := make([]Invoice, 0, len(e.Invoices))
	for idx := range e.Invoices {
		out = append(out, e.Invoices[idx].invoice())
	}
	return out
}

//nolint:gocritic // value inputs are the package contract the accounting adapter is written against.
func (c *Client) CreateInvoice(ctx context.Context, key string, in InvoiceInput) (*Invoice, error) {
	body, err := in.body()
	if err != nil {
		return nil, err
	}
	return c.writeInvoice(ctx, key, "", &body)
}

//nolint:gocritic // value inputs are the package contract the accounting adapter is written against.
func (c *Client) UpdateInvoice(
	ctx context.Context,
	key, invoiceID string,
	in InvoiceInput,
) (*Invoice, error) {
	id, err := guid(invoiceID)
	if err != nil {
		return nil, err
	}
	body, err := in.body()
	if err != nil {
		return nil, err
	}
	body.InvoiceID = id
	return c.writeInvoice(ctx, key, id, &body)
}

func (c *Client) VoidInvoice(ctx context.Context, key, invoiceID string) (*Invoice, error) {
	id, err := guid(invoiceID)
	if err != nil {
		return nil, err
	}
	return c.writeInvoice(ctx, key, id, &invoiceBody{InvoiceID: id, Status: StatusVoided})
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
		DueDate:         formatDate(in.DueDate),
		CurrencyCode:    fields.currencyCode,
		CurrencyRate:    fields.currencyRate,
		Status:          fields.status,
		LineAmountTypes: fields.lineAmountTypes,
		LineItems:       fields.lines,
	}, nil
}

func (c *Client) writeInvoice(
	ctx context.Context,
	key, invoiceID string,
	body *invoiceBody,
) (*Invoice, error) {
	idem, err := idempotencyKey(key)
	if err != nil {
		return nil, err
	}
	req := &call{
		endpoint: "invoices-create",
		method:   http.MethodPut,
		path:     accountingPath(invoicesResource),
		query:    url.Values{unitDPParam: {unitDecimalPlaces}},
		key:      idem,
		body:     invoiceWriteEnvelope{Invoices: []invoiceBody{*body}},
	}
	if invoiceID != "" {
		req.endpoint = "invoices-update"
		req.method = http.MethodPost
		req.path = accountingPath(invoicesResource, invoiceID)
	}

	var out invoicesEnvelope
	req.out = &out
	if err = c.do(ctx, req); err != nil {
		return nil, err
	}
	if len(out.Invoices) == 0 {
		return nil, ErrUnexpectedPayload
	}
	written := out.Invoices[0].invoice()
	return &written, nil
}

func (c *Client) InvoicesByID(ctx context.Context, ids []string) ([]Invoice, error) {
	var out invoicesEnvelope
	requested, err := c.readByIDs(ctx, "invoices-by-id", invoicesResource, ids, &out)
	if err != nil {
		return nil, err
	}
	if !requested {
		return []Invoice{}, nil
	}
	return out.invoices(), nil
}

func (c *Client) Invoices(
	ctx context.Context,
	page int,
	modifiedSince *time.Time,
) (*InvoicePage, error) {
	query, err := pageQuery(page)
	if err != nil {
		return nil, err
	}
	var out invoicesEnvelope
	if err = c.readPage(ctx, "invoices", invoicesResource, query, modifiedSince, &out); err != nil {
		return nil, err
	}
	invoices := out.invoices()
	return &InvoicePage{
		Invoices: invoices,
		More:     hasMore(out.Pagination, page, len(invoices)),
	}, nil
}

//nolint:gocritic // value inputs are the package contract the accounting adapter is written against.
func (c *Client) FindInvoices(ctx context.Context, f InvoiceFilter) ([]Invoice, error) {
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
		base.Set(statusesParam, strings.Join(terms.statuses, ","))
	}
	from, to := terms.dateClauses()
	if where := whereJoin(
		whereAnyOf("Type", optional(terms.kind), whereString),
		from,
		to,
	); where != "" {
		base.Set(whereParam, where)
	}

	return collectPages(func(page int) ([]Invoice, bool, error) {
		query, queryErr := withPage(base, page)
		if queryErr != nil {
			return nil, false, queryErr
		}
		var out invoicesEnvelope
		if readErr := c.readPage(
			ctx,
			"invoices-search",
			invoicesResource,
			query,
			nil,
			&out,
		); readErr != nil {
			return nil, false, readErr
		}
		invoices := out.invoices()
		return invoices, hasMore(out.Pagination, page, len(invoices)), nil
	})
}

func optional(value string) []string {
	if value == "" {
		return nil
	}
	return []string{value}
}
