package xerodocs

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/emoss08/trenova/shared/xero"
	"github.com/emoss08/trenova/shared/xeroerrors"
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
	CreditNoteID     string       `json:"CreditNoteID,omitempty"`
	Type             string       `json:"Type,omitempty"`
	Contact          *xero.IDRef  `json:"Contact,omitempty"`
	CreditNoteNumber string       `json:"CreditNoteNumber,omitempty"`
	Reference        string       `json:"Reference,omitempty"`
	Date             string       `json:"Date,omitempty"`
	CurrencyCode     string       `json:"CurrencyCode,omitempty"`
	CurrencyRate     *xero.Number `json:"CurrencyRate,omitempty"`
	Status           string       `json:"Status,omitempty"`
	LineAmountTypes  string       `json:"LineAmountTypes,omitempty"`
	LineItems        []lineBody   `json:"LineItems,omitempty"`
}

type creditNoteWriteEnvelope struct {
	CreditNotes []creditNoteBody `json:"CreditNotes"`
}

type wireAllocation struct {
	AllocationID string          `json:"AllocationID"`
	Amount       decimal.Decimal `json:"Amount"`
	Date         xero.WireTime   `json:"Date"`
	IsDeleted    bool            `json:"IsDeleted"`
	Invoice      *xero.WireRef   `json:"Invoice"`
	CreditNote   *xero.WireRef   `json:"CreditNote"`
}

func (w *wireAllocation) allocation(creditNoteID string) Allocation {
	out := Allocation{
		AllocationID: w.AllocationID,
		CreditNoteID: creditNoteID,
		Amount:       w.Amount,
		Date:         w.Date.Time(),
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
	Contact          *xero.WireRef         `json:"Contact"`
	Total            decimal.Decimal       `json:"Total"`
	RemainingCredit  decimal.Decimal       `json:"RemainingCredit"`
	CurrencyRate     decimal.Decimal       `json:"CurrencyRate"`
	Date             xero.WireTime         `json:"Date"`
	UpdatedDateUTC   xero.WireTime         `json:"UpdatedDateUTC"`
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
		Date:             w.Date.Time(),
		UpdatedAt:        w.UpdatedDateUTC.Time(),
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
	CreditNotes []wireCreditNote     `json:"CreditNotes"`
	Pagination  *xero.WirePagination `json:"pagination"`
}

func (e *creditNotesEnvelope) creditNotes() []CreditNote {
	out := make([]CreditNote, 0, len(e.CreditNotes))
	for idx := range e.CreditNotes {
		out = append(out, e.CreditNotes[idx].creditNote())
	}
	return out
}

type allocationBody struct {
	Invoice xero.IDRef  `json:"Invoice"`
	Amount  xero.Amount `json:"Amount"`
	Date    string      `json:"Date,omitempty"`
}

type allocationWriteEnvelope struct {
	Allocations []allocationBody `json:"Allocations"`
}

type allocationsEnvelope struct {
	Allocations []wireAllocation `json:"Allocations"`
}

//nolint:gocritic // value inputs are the package contract the accounting adapter is written against.
func CreateCreditNote(
	ctx context.Context,
	c *xero.Client,
	key string,
	in CreditNoteInput,
) (*CreditNote, error) {
	body, err := in.body()
	if err != nil {
		return nil, err
	}
	return writeCreditNote(ctx, c, key, "", &body)
}

//nolint:gocritic // value inputs are the package contract the accounting adapter is written against.
func UpdateCreditNote(
	ctx context.Context,
	c *xero.Client,
	key, id string,
	in CreditNoteInput,
) (*CreditNote, error) {
	creditNoteID, err := xero.GUID(id)
	if err != nil {
		return nil, err
	}
	body, err := in.body()
	if err != nil {
		return nil, err
	}
	body.CreditNoteID = creditNoteID
	return writeCreditNote(ctx, c, key, creditNoteID, &body)
}

func VoidCreditNote(ctx context.Context, c *xero.Client, key, id string) (*CreditNote, error) {
	creditNoteID, err := xero.GUID(id)
	if err != nil {
		return nil, err
	}
	return writeCreditNote(ctx, c, key, creditNoteID, &creditNoteBody{
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

func writeCreditNote(
	ctx context.Context,
	c *xero.Client,
	key, creditNoteID string,
	body *creditNoteBody,
) (*CreditNote, error) {
	idem, err := xero.IdempotencyKey(key)
	if err != nil {
		return nil, err
	}
	req := &xero.Call{
		Endpoint: "credit-notes-create",
		Method:   http.MethodPut,
		Path:     xero.AccountingPath(creditNotesResource),
		Query:    url.Values{unitDPParam: {unitDecimalPlaces}},
		Key:      idem,
		Body:     creditNoteWriteEnvelope{CreditNotes: []creditNoteBody{*body}},
	}
	if creditNoteID != "" {
		req.Endpoint = "credit-notes-update"
		req.Method = http.MethodPost
		req.Path = xero.AccountingPath(creditNotesResource, creditNoteID)
	}

	var out creditNotesEnvelope
	req.Out = &out
	if err = c.Do(ctx, req); err != nil {
		return nil, err
	}
	if len(out.CreditNotes) == 0 {
		return nil, xeroerrors.ErrUnexpectedPayload
	}
	written := out.CreditNotes[0].creditNote()
	return &written, nil
}

func CreditNotesByID(ctx context.Context, c *xero.Client, ids []string) ([]CreditNote, error) {
	var out creditNotesEnvelope
	requested, err := readByIDs(ctx, c, "credit-notes-by-id", creditNotesResource, ids, &out)
	if err != nil {
		return nil, err
	}
	if !requested {
		return []CreditNote{}, nil
	}
	return out.creditNotes(), nil
}

func CreditNotes(
	ctx context.Context,
	c *xero.Client,
	page int,
	modifiedSince *time.Time,
) (*CreditNotePage, error) {
	query, err := xero.PageQuery(page)
	if err != nil {
		return nil, err
	}
	var out creditNotesEnvelope
	if err = xero.ReadPage(
		ctx,
		c, "credit-notes", creditNotesResource, query, modifiedSince, &out,
	); err != nil {
		return nil, err
	}
	notes := out.creditNotes()
	return &CreditNotePage{
		CreditNotes: notes,
		More:        xero.HasMore(out.Pagination, page, len(notes)),
	}, nil
}

//nolint:gocritic // value inputs are the package contract the accounting adapter is written against.
func FindCreditNotes(ctx context.Context, c *xero.Client, f InvoiceFilter) ([]CreditNote, error) {
	terms, err := f.terms(CreditNoteTypeReceivable, CreditNoteTypePayable)
	if err != nil {
		return nil, err
	}
	from, to := terms.dateClauses()
	base := url.Values{whereParam: {xero.WhereJoin(
		xero.WhereAnyOf("Type", xero.Optional(terms.kind), xero.WhereString),
		xero.WhereAnyOf("CreditNoteNumber", terms.numbers, xero.WhereString),
		xero.WhereAnyOf("Contact.ContactID", terms.contactIDs, xero.WhereGUID),
		xero.WhereAnyOf("Status", terms.statuses, xero.WhereString),
		from,
		to,
	)}}

	return xero.CollectPages(func(page int) ([]CreditNote, bool, error) {
		query, queryErr := xero.WithPage(base, page)
		if queryErr != nil {
			return nil, false, queryErr
		}
		var out creditNotesEnvelope
		if readErr := xero.ReadPage(
			ctx,
			c, "credit-notes-search", creditNotesResource, query, nil, &out,
		); readErr != nil {
			return nil, false, readErr
		}
		notes := out.creditNotes()
		return notes, xero.HasMore(out.Pagination, page, len(notes)), nil
	})
}

func Allocate(
	ctx context.Context,
	c *xero.Client,
	key, creditNoteID string,
	a AllocationInput,
) (*Allocation, error) {
	idem, err := xero.IdempotencyKey(key)
	if err != nil {
		return nil, err
	}
	noteID, err := xero.GUID(creditNoteID)
	if err != nil {
		return nil, err
	}
	invoiceID, err := xero.GUID(a.InvoiceID)
	if err != nil {
		return nil, err
	}
	if !a.Amount.IsPositive() {
		return nil, xeroerrors.ErrAmountNotPositive
	}

	var out allocationsEnvelope
	if err = c.Do(ctx, &xero.Call{
		Endpoint: "credit-note-allocate",
		Method:   http.MethodPut,
		Path:     xero.AccountingPath(creditNotesResource, noteID, allocationsResource),
		Key:      idem,
		Body: allocationWriteEnvelope{Allocations: []allocationBody{{
			Invoice: xero.IDRef{InvoiceID: invoiceID},
			Amount:  xero.Amount(a.Amount),
			Date:    xero.FormatDate(a.Date),
		}}},
		Out: &out,
	}); err != nil {
		return nil, err
	}
	if len(out.Allocations) == 0 {
		return nil, xeroerrors.ErrUnexpectedPayload
	}
	allocation := out.Allocations[0].allocation(noteID)
	return &allocation, nil
}

func DeleteAllocation(
	ctx context.Context,
	c *xero.Client,
	creditNoteID, allocationID string,
) error {
	noteID, err := xero.GUID(creditNoteID)
	if err != nil {
		return err
	}
	allocID, err := xero.GUID(allocationID)
	if err != nil {
		return err
	}
	return c.Do(ctx, &xero.Call{
		Endpoint: "credit-note-allocation-delete",
		Method:   http.MethodDelete,
		Path:     xero.AccountingPath(creditNotesResource, noteID, allocationsResource, allocID),
		Expected: []int{http.StatusOK, http.StatusNoContent},
	})

}
