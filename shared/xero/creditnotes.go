package xero

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/shopspring/decimal"
)

const (
	creditNotesResource = "CreditNotes"
	allocationsResource = "Allocations"
)

type CreditNoteInput struct {
	Type             string
	ContactID        string
	CreditNoteNumber string
	Reference        string
	Date             time.Time
	CurrencyCode     string
	CurrencyRate     decimal.Decimal
	Status           string
	LineAmountTypes  string
	Lines            []LineItem
}

type CreditNote struct {
	CreditNoteID     string
	CreditNoteNumber string
	Type             string
	Status           string
	Reference        string
	CurrencyCode     string
	ContactID        string
	Total            decimal.Decimal
	RemainingCredit  decimal.Decimal
	CurrencyRate     decimal.Decimal
	Date             time.Time
	UpdatedAt        time.Time
	Allocations      []Allocation
	Payments         []DocumentPayment
}

type CreditNotePage struct {
	CreditNotes []CreditNote
	More        bool
}

type AllocationInput struct {
	InvoiceID string
	Amount    decimal.Decimal
	Date      time.Time
}

type Allocation struct {
	AllocationID string
	InvoiceID    string
	CreditNoteID string
	Amount       decimal.Decimal
	Date         time.Time
	IsDeleted    bool
}

type creditNoteBody struct {
	CreditNoteID     string     `json:"CreditNoteID,omitempty"`
	Type             string     `json:"Type,omitempty"`
	Contact          *idRef     `json:"Contact,omitempty"`
	CreditNoteNumber string     `json:"CreditNoteNumber,omitempty"`
	Reference        string     `json:"Reference,omitempty"`
	Date             string     `json:"Date,omitempty"`
	CurrencyCode     string     `json:"CurrencyCode,omitempty"`
	CurrencyRate     *number    `json:"CurrencyRate,omitempty"`
	Status           string     `json:"Status,omitempty"`
	LineAmountTypes  string     `json:"LineAmountTypes,omitempty"`
	LineItems        []lineBody `json:"LineItems,omitempty"`
}

type creditNoteWriteEnvelope struct {
	CreditNotes []creditNoteBody `json:"CreditNotes"`
}

type wireAllocation struct {
	AllocationID string          `json:"AllocationID"`
	Amount       decimal.Decimal `json:"Amount"`
	Date         wireTime        `json:"Date"`
	IsDeleted    bool            `json:"IsDeleted"`
	Invoice      *wireRef        `json:"Invoice"`
	CreditNote   *wireRef        `json:"CreditNote"`
}

func (w *wireAllocation) allocation(creditNoteID string) Allocation {
	out := Allocation{
		AllocationID: w.AllocationID,
		CreditNoteID: creditNoteID,
		Amount:       w.Amount,
		Date:         w.Date.time(),
		IsDeleted:    w.IsDeleted,
	}
	if w.Invoice != nil {
		out.InvoiceID = w.Invoice.InvoiceID
	}
	if w.CreditNote != nil && w.CreditNote.CreditNoteID != "" {
		out.CreditNoteID = w.CreditNote.CreditNoteID
	}
	return out
}

type wireCreditNote struct {
	CreditNoteID     string                `json:"CreditNoteID"`
	CreditNoteNumber string                `json:"CreditNoteNumber"`
	Type             string                `json:"Type"`
	Status           string                `json:"Status"`
	Reference        string                `json:"Reference"`
	CurrencyCode     string                `json:"CurrencyCode"`
	Contact          *wireRef              `json:"Contact"`
	Total            decimal.Decimal       `json:"Total"`
	RemainingCredit  decimal.Decimal       `json:"RemainingCredit"`
	CurrencyRate     decimal.Decimal       `json:"CurrencyRate"`
	Date             wireTime              `json:"Date"`
	UpdatedDateUTC   wireTime              `json:"UpdatedDateUTC"`
	Allocations      []wireAllocation      `json:"Allocations"`
	Payments         []wireDocumentPayment `json:"Payments"`
}

func (w *wireCreditNote) creditNote() CreditNote {
	out := CreditNote{
		CreditNoteID:     w.CreditNoteID,
		CreditNoteNumber: w.CreditNoteNumber,
		Type:             w.Type,
		Status:           w.Status,
		Reference:        w.Reference,
		CurrencyCode:     w.CurrencyCode,
		Total:            w.Total,
		RemainingCredit:  w.RemainingCredit,
		CurrencyRate:     w.CurrencyRate,
		Date:             w.Date.time(),
		UpdatedAt:        w.UpdatedDateUTC.time(),
		Payments:         documentPayments(w.Payments),
	}
	if w.Contact != nil {
		out.ContactID = w.Contact.ContactID
	}
	if len(w.Allocations) > 0 {
		out.Allocations = make([]Allocation, 0, len(w.Allocations))
		for idx := range w.Allocations {
			out.Allocations = append(out.Allocations, w.Allocations[idx].allocation(w.CreditNoteID))
		}
	}
	return out
}

type creditNotesEnvelope struct {
	CreditNotes []wireCreditNote `json:"CreditNotes"`
	Pagination  *wirePagination  `json:"pagination"`
}

func (e *creditNotesEnvelope) creditNotes() []CreditNote {
	out := make([]CreditNote, 0, len(e.CreditNotes))
	for idx := range e.CreditNotes {
		out = append(out, e.CreditNotes[idx].creditNote())
	}
	return out
}

type allocationBody struct {
	Invoice idRef  `json:"Invoice"`
	Amount  amount `json:"Amount"`
	Date    string `json:"Date,omitempty"`
}

type allocationWriteEnvelope struct {
	Allocations []allocationBody `json:"Allocations"`
}

type allocationsEnvelope struct {
	Allocations []wireAllocation `json:"Allocations"`
}

//nolint:gocritic // value inputs are the package contract the accounting adapter is written against.
func (c *Client) CreateCreditNote(
	ctx context.Context,
	key string,
	in CreditNoteInput,
) (*CreditNote, error) {
	body, err := in.body()
	if err != nil {
		return nil, err
	}
	return c.writeCreditNote(ctx, key, "", &body)
}

//nolint:gocritic // value inputs are the package contract the accounting adapter is written against.
func (c *Client) UpdateCreditNote(
	ctx context.Context,
	key, id string,
	in CreditNoteInput,
) (*CreditNote, error) {
	creditNoteID, err := guid(id)
	if err != nil {
		return nil, err
	}
	body, err := in.body()
	if err != nil {
		return nil, err
	}
	body.CreditNoteID = creditNoteID
	return c.writeCreditNote(ctx, key, creditNoteID, &body)
}

func (c *Client) VoidCreditNote(ctx context.Context, key, id string) (*CreditNote, error) {
	creditNoteID, err := guid(id)
	if err != nil {
		return nil, err
	}
	return c.writeCreditNote(ctx, key, creditNoteID, &creditNoteBody{
		CreditNoteID: creditNoteID,
		Status:       StatusVoided,
	})
}

func (in *CreditNoteInput) body() (creditNoteBody, error) {
	spec := documentSpec{
		kind:            in.Type,
		contactID:       in.ContactID,
		number:          in.CreditNoteNumber,
		reference:       in.Reference,
		date:            in.Date,
		currencyCode:    in.CurrencyCode,
		currencyRate:    in.CurrencyRate,
		status:          in.Status,
		lineAmountTypes: in.LineAmountTypes,
		lines:           in.Lines,
	}
	fields, err := spec.fields(CreditNoteTypeReceivable, CreditNoteTypePayable)
	if err != nil {
		return creditNoteBody{}, err
	}
	return creditNoteBody{
		Type:             fields.kind,
		Contact:          fields.contact,
		CreditNoteNumber: fields.number,
		Reference:        fields.reference,
		Date:             fields.date,
		CurrencyCode:     fields.currencyCode,
		CurrencyRate:     fields.currencyRate,
		Status:           fields.status,
		LineAmountTypes:  fields.lineAmountTypes,
		LineItems:        fields.lines,
	}, nil
}

func (c *Client) writeCreditNote(
	ctx context.Context,
	key, creditNoteID string,
	body *creditNoteBody,
) (*CreditNote, error) {
	idem, err := idempotencyKey(key)
	if err != nil {
		return nil, err
	}
	req := &call{
		endpoint: "credit-notes-create",
		method:   http.MethodPut,
		path:     accountingPath(creditNotesResource),
		query:    url.Values{unitDPParam: {unitDecimalPlaces}},
		key:      idem,
		body:     creditNoteWriteEnvelope{CreditNotes: []creditNoteBody{*body}},
	}
	if creditNoteID != "" {
		req.endpoint = "credit-notes-update"
		req.method = http.MethodPost
		req.path = accountingPath(creditNotesResource, creditNoteID)
	}

	var out creditNotesEnvelope
	req.out = &out
	if err = c.do(ctx, req); err != nil {
		return nil, err
	}
	if len(out.CreditNotes) == 0 {
		return nil, ErrUnexpectedPayload
	}
	written := out.CreditNotes[0].creditNote()
	return &written, nil
}

func (c *Client) CreditNotesByID(ctx context.Context, ids []string) ([]CreditNote, error) {
	var out creditNotesEnvelope
	requested, err := c.readByIDs(ctx, "credit-notes-by-id", creditNotesResource, ids, &out)
	if err != nil {
		return nil, err
	}
	if !requested {
		return []CreditNote{}, nil
	}
	return out.creditNotes(), nil
}

func (c *Client) CreditNotes(
	ctx context.Context,
	page int,
	modifiedSince *time.Time,
) (*CreditNotePage, error) {
	query, err := pageQuery(page)
	if err != nil {
		return nil, err
	}
	var out creditNotesEnvelope
	if err = c.readPage(
		ctx, "credit-notes", creditNotesResource, query, modifiedSince, &out,
	); err != nil {
		return nil, err
	}
	notes := out.creditNotes()
	return &CreditNotePage{
		CreditNotes: notes,
		More:        hasMore(out.Pagination, page, len(notes)),
	}, nil
}

//nolint:gocritic // value inputs are the package contract the accounting adapter is written against.
func (c *Client) FindCreditNotes(ctx context.Context, f InvoiceFilter) ([]CreditNote, error) {
	terms, err := f.terms(CreditNoteTypeReceivable, CreditNoteTypePayable)
	if err != nil {
		return nil, err
	}
	from, to := terms.dateClauses()
	base := url.Values{whereParam: {whereJoin(
		whereAnyOf("Type", optional(terms.kind), whereString),
		whereAnyOf("CreditNoteNumber", terms.numbers, whereString),
		whereAnyOf("Contact.ContactID", terms.contactIDs, whereGUID),
		whereAnyOf("Status", terms.statuses, whereString),
		from,
		to,
	)}}

	return collectPages(func(page int) ([]CreditNote, bool, error) {
		query, queryErr := withPage(base, page)
		if queryErr != nil {
			return nil, false, queryErr
		}
		var out creditNotesEnvelope
		if readErr := c.readPage(
			ctx, "credit-notes-search", creditNotesResource, query, nil, &out,
		); readErr != nil {
			return nil, false, readErr
		}
		notes := out.creditNotes()
		return notes, hasMore(out.Pagination, page, len(notes)), nil
	})
}

func (c *Client) Allocate(
	ctx context.Context,
	key, creditNoteID string,
	a AllocationInput,
) (*Allocation, error) {
	idem, err := idempotencyKey(key)
	if err != nil {
		return nil, err
	}
	noteID, err := guid(creditNoteID)
	if err != nil {
		return nil, err
	}
	invoiceID, err := guid(a.InvoiceID)
	if err != nil {
		return nil, err
	}
	if !a.Amount.IsPositive() {
		return nil, ErrAmountNotPositive
	}

	var out allocationsEnvelope
	if err = c.do(ctx, &call{
		endpoint: "credit-note-allocate",
		method:   http.MethodPut,
		path:     accountingPath(creditNotesResource, noteID, allocationsResource),
		key:      idem,
		body: allocationWriteEnvelope{Allocations: []allocationBody{{
			Invoice: idRef{InvoiceID: invoiceID},
			Amount:  amount(a.Amount),
			Date:    formatDate(a.Date),
		}}},
		out: &out,
	}); err != nil {
		return nil, err
	}
	if len(out.Allocations) == 0 {
		return nil, ErrUnexpectedPayload
	}
	allocation := out.Allocations[0].allocation(noteID)
	return &allocation, nil
}

func (c *Client) DeleteAllocation(ctx context.Context, creditNoteID, allocationID string) error {
	noteID, err := guid(creditNoteID)
	if err != nil {
		return err
	}
	allocID, err := guid(allocationID)
	if err != nil {
		return err
	}
	return c.do(ctx, &call{
		endpoint: "credit-note-allocation-delete",
		method:   http.MethodDelete,
		path:     accountingPath(creditNotesResource, noteID, allocationsResource, allocID),
		expected: []int{http.StatusOK, http.StatusNoContent},
	})
}
