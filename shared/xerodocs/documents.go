package xerodocs

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/xero"
	"github.com/emoss08/trenova/shared/xeroerrors"
	"github.com/emoss08/trenova/shared/xeroref"
	"github.com/shopspring/decimal"
)

const (
	InvoiceTypeReceivable    = "ACCREC"
	InvoiceTypePayable       = "ACCPAY"
	CreditNoteTypeReceivable = "ACCRECCREDIT"
	CreditNoteTypePayable    = "ACCPAYCREDIT"
	StatusDraft              = "DRAFT"
	StatusSubmitted          = "SUBMITTED"
	StatusAuthorised         = "AUTHORISED"
	StatusPaid               = "PAID"
	StatusVoided             = "VOIDED"
	StatusDeleted            = "DELETED"
	LineAmountsNoTax         = "NoTax"
	LineAmountsExclusive     = "Exclusive"
	LineAmountsInclusive     = "Inclusive"
	MaxDocumentNumberLength  = 255
	MaxReferenceLength       = 255
	unitDecimalPlaces        = "4"
	idsParam                 = "IDs"
	unitDPParam              = "unitdp"
)

type LineItem struct {
	Description string
	Quantity    decimal.Decimal
	UnitAmount  decimal.Decimal
	LineAmount  decimal.Decimal
	AccountCode string
	ItemCode    string
}

type DocumentPayment struct {
	PaymentID string
	Amount    decimal.Decimal
}

type AppliedCredit struct {
	CreditNoteID string
	Amount       decimal.Decimal
}

type InvoiceFilter struct {
	Type       string
	Numbers    []string
	ContactIDs []string
	Statuses   []string
	DateFrom   time.Time
	DateTo     time.Time
}

type lineBody struct {
	Description string       `json:"Description,omitempty"`
	Quantity    *xero.Number `json:"Quantity,omitempty"`
	UnitAmount  *xero.Number `json:"UnitAmount,omitempty"`
	LineAmount  *xero.Amount `json:"LineAmount,omitempty"`
	AccountCode string       `json:"AccountCode,omitempty"`
	ItemCode    string       `json:"ItemCode,omitempty"`
}

type wireDocumentPayment struct {
	PaymentID string          `json:"PaymentID"`
	Amount    decimal.Decimal `json:"Amount"`
}

func documentPayments(wire []wireDocumentPayment) []DocumentPayment {
	if len(wire) == 0 {
		return nil
	}
	out := make([]DocumentPayment, 0, len(wire))
	for idx := range wire {
		out = append(out, DocumentPayment{PaymentID: wire[idx].PaymentID, Amount: wire[idx].Amount})
	}
	return out
}

type documentSpec struct {
	kind            string
	contactID       string
	number          string
	reference       string
	date            time.Time
	currencyCode    string
	currencyRate    decimal.Decimal
	status          string
	lineAmountTypes string
	lines           []LineItem
}

type documentFields struct {
	kind            string
	contact         *xero.IDRef
	number          string
	reference       string
	date            string
	currencyCode    string
	currencyRate    *xero.Number
	status          string
	lineAmountTypes string
	lines           []lineBody
}

func (s *documentSpec) fields(allowedTypes ...string) (documentFields, error) {
	kind := strings.ToUpper(strings.TrimSpace(s.kind))
	if !slices.Contains(allowedTypes, kind) {
		return documentFields{}, xeroerrors.ErrUnknownType
	}
	contactID, err := xero.GUID(s.contactID)
	if err != nil {
		if errors.Is(err, xeroerrors.ErrIDRequired) {
			return documentFields{}, xeroerrors.ErrContactRequired
		}
		return documentFields{}, err
	}
	if s.date.IsZero() {
		return documentFields{}, xeroerrors.ErrDateRequired
	}
	out := documentFields{
		kind:      kind,
		contact:   &xero.IDRef{ContactID: contactID},
		number:    strings.TrimSpace(s.number),
		reference: strings.TrimSpace(s.reference),
		date:      xero.FormatDate(s.date),
	}
	if !xero.TextWithin(out.number, MaxDocumentNumberLength) ||
		!xero.TextWithin(out.reference, MaxReferenceLength) {
		return documentFields{}, xeroerrors.ErrFieldTooLong
	}
	if out.currencyCode, err = xero.CurrencyCode(s.currencyCode); err != nil {
		return documentFields{}, err
	}
	if s.currencyRate.IsNegative() {
		return documentFields{}, xeroerrors.ErrNegativeRate
	}
	out.currencyRate = xero.NumberOf(s.currencyRate)
	if out.status, err = writableStatus(s.status); err != nil {
		return documentFields{}, err
	}
	if out.lineAmountTypes, err = lineAmountTypes(s.lineAmountTypes); err != nil {
		return documentFields{}, err
	}
	if out.lines, err = lineBodies(s.lines); err != nil {
		return documentFields{}, err
	}
	return out, nil
}

func writableStatus(raw string) (string, error) {
	status := strings.ToUpper(strings.TrimSpace(raw))
	switch status {
	case "":
		return StatusAuthorised, nil
	case StatusDraft, StatusSubmitted, StatusAuthorised:
		return status, nil
	default:
		return "", xeroerrors.ErrUnknownStatus
	}
}

func lineAmountTypes(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	switch {
	case trimmed == "":
		return LineAmountsNoTax, nil
	case strings.EqualFold(trimmed, LineAmountsNoTax):
		return LineAmountsNoTax, nil
	case strings.EqualFold(trimmed, LineAmountsExclusive):
		return LineAmountsExclusive, nil
	case strings.EqualFold(trimmed, LineAmountsInclusive):
		return LineAmountsInclusive, nil
	default:
		return "", xeroerrors.ErrUnknownAmountTypes
	}
}

func lineBodies(lines []LineItem) ([]lineBody, error) {
	if len(lines) == 0 {
		return nil, xeroerrors.ErrLinesRequired
	}
	out := make([]lineBody, 0, len(lines))
	for idx := range lines {
		line := &lines[idx]
		body := lineBody{
			Description: strings.TrimSpace(line.Description),
			Quantity:    xero.NumberOf(line.Quantity),
			UnitAmount:  xero.NumberOf(line.UnitAmount),
			AccountCode: strings.TrimSpace(line.AccountCode),
			ItemCode:    strings.TrimSpace(line.ItemCode),
		}
		if body.Description == "" && body.ItemCode == "" {
			return nil, xeroerrors.ErrLineDescription
		}
		if !xero.TextWithin(body.Description, xeroref.MaxDescriptionLength) ||
			!xero.TextWithin(body.ItemCode, xeroref.MaxItemCodeLength) {
			return nil, xeroerrors.ErrFieldTooLong
		}
		if !line.LineAmount.IsZero() {
			value := xero.Amount(line.LineAmount)
			body.LineAmount = &value
		}
		out = append(out, body)
	}
	return out, nil
}

func readByIDs(
	ctx context.Context,
	c *xero.Client,
	endpoint, resource string,
	ids []string,
	out any,
) (bool, error) {
	unique, err := xero.GUIDs(ids, xero.MaxIDsPerRead)
	if err != nil {
		return false, err
	}
	if len(unique) == 0 {
		return false, nil
	}
	return true, c.Do(ctx, &xero.Call{
		Endpoint: endpoint,
		Method:   http.MethodGet,
		Path:     xero.AccountingPath(resource),
		Query:    url.Values{idsParam: {strings.Join(unique, ",")}},
		Out:      out,
	})

}

type filterTerms struct {
	kind       string
	numbers    []string
	contactIDs []string
	statuses   []string
	dateFrom   string
	dateTo     string
}

func (f *InvoiceFilter) terms(allowedTypes ...string) (filterTerms, error) {
	var out filterTerms
	if kind := strings.ToUpper(strings.TrimSpace(f.Type)); kind != "" {
		if !slices.Contains(allowedTypes, kind) {
			return filterTerms{}, xeroerrors.ErrUnknownType
		}
		out.kind = kind
	}
	numbers, err := searchNumbers(f.Numbers)
	if err != nil {
		return filterTerms{}, err
	}
	out.numbers = numbers
	if out.contactIDs, err = xero.GUIDs(f.ContactIDs, xero.MaxIDsPerRead); err != nil {
		return filterTerms{}, err
	}
	if out.statuses, err = searchStatuses(f.Statuses); err != nil {
		return filterTerms{}, err
	}
	if !f.DateFrom.IsZero() {
		out.dateFrom = whereDate(f.DateFrom)
	}
	if !f.DateTo.IsZero() {
		out.dateTo = whereDate(f.DateTo)
	}
	if out.kind == "" && len(out.numbers) == 0 && len(out.contactIDs) == 0 &&
		len(out.statuses) == 0 && out.dateFrom == "" && out.dateTo == "" {
		return filterTerms{}, xeroerrors.ErrEmptyFilter
	}
	return out, nil
}

func searchNumbers(raw []string) ([]string, error) {
	out := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, value := range raw {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || !xero.TextWithin(trimmed, MaxDocumentNumberLength) ||
			strings.ContainsAny(trimmed, ",\"\\") || xero.HasControl(trimmed) {
			return nil, xeroerrors.ErrInvalidFilter
		}
		if _, dup := seen[trimmed]; dup {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) > xero.MaxIDsPerRead {
		return nil, xeroerrors.ErrTooManyIDs
	}
	return out, nil
}

func searchStatuses(raw []string) ([]string, error) {
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		status := strings.ToUpper(strings.TrimSpace(value))
		switch status {
		case StatusDraft, StatusSubmitted, StatusAuthorised, StatusPaid, StatusVoided,
			StatusDeleted:
		default:
			return nil, xeroerrors.ErrUnknownStatus
		}
		if !slices.Contains(out, status) {
			out = append(out, status)
		}
	}
	return out, nil
}

func whereDate(t time.Time) string {
	return fmt.Sprintf("DateTime(%d,%02d,%02d)", t.Year(), int(t.Month()), t.Day())
}

func (t *filterTerms) dateClauses() (from, to string) {
	if t.dateFrom != "" {
		from = "Date>=" + t.dateFrom
	}
	if t.dateTo != "" {
		to = "Date<=" + t.dateTo
	}
	return from, to
}
