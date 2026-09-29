package xero

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

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
	maxSearchPages           = 50
	unitDecimalPlaces        = "4"
	moneyDecimalPlaces       = 2
	idsParam                 = "IDs"
	whereParam               = "where"
	unitDPParam              = "unitdp"
	statusesParam            = "Statuses"
)

var ErrTooManyResults = errors.New("xero: the search matched more documents than can be read")

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

type number decimal.Decimal

func (n number) MarshalJSON() ([]byte, error) {
	return []byte(decimal.Decimal(n).String()), nil
}

func numberOf(value decimal.Decimal) *number {
	if value.IsZero() {
		return nil
	}
	out := number(value)
	return &out
}

type amount decimal.Decimal

func (a amount) MarshalJSON() ([]byte, error) {
	return []byte(decimal.Decimal(a).StringFixed(moneyDecimalPlaces)), nil
}

type idRef struct {
	ContactID    string `json:"ContactID,omitempty"`
	InvoiceID    string `json:"InvoiceID,omitempty"`
	CreditNoteID string `json:"CreditNoteID,omitempty"`
	AccountID    string `json:"AccountID,omitempty"`
	Code         string `json:"Code,omitempty"`
}

type lineBody struct {
	Description string  `json:"Description,omitempty"`
	Quantity    *number `json:"Quantity,omitempty"`
	UnitAmount  *number `json:"UnitAmount,omitempty"`
	LineAmount  *amount `json:"LineAmount,omitempty"`
	AccountCode string  `json:"AccountCode,omitempty"`
	ItemCode    string  `json:"ItemCode,omitempty"`
}

type wireRef struct {
	ContactID     string `json:"ContactID"`
	InvoiceID     string `json:"InvoiceID"`
	InvoiceNumber string `json:"InvoiceNumber"`
	CreditNoteID  string `json:"CreditNoteID"`
	Type          string `json:"Type"`
	AccountID     string `json:"AccountID"`
	Code          string `json:"Code"`
	Contact       *struct {
		ContactID string `json:"ContactID"`
	} `json:"Contact"`
}

func (r *wireRef) contactID() string {
	if r == nil || r.Contact == nil {
		return ""
	}
	return r.Contact.ContactID
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
	contact         *idRef
	number          string
	reference       string
	date            string
	currencyCode    string
	currencyRate    *number
	status          string
	lineAmountTypes string
	lines           []lineBody
}

func (s *documentSpec) fields(allowedTypes ...string) (documentFields, error) {
	kind := strings.ToUpper(strings.TrimSpace(s.kind))
	if !slices.Contains(allowedTypes, kind) {
		return documentFields{}, ErrUnknownType
	}
	contactID, err := guid(s.contactID)
	if err != nil {
		if errors.Is(err, ErrIDRequired) {
			return documentFields{}, ErrContactRequired
		}
		return documentFields{}, err
	}
	if s.date.IsZero() {
		return documentFields{}, ErrDateRequired
	}
	out := documentFields{
		kind:      kind,
		contact:   &idRef{ContactID: contactID},
		number:    strings.TrimSpace(s.number),
		reference: strings.TrimSpace(s.reference),
		date:      formatDate(s.date),
	}
	if !textWithin(out.number, MaxDocumentNumberLength) ||
		!textWithin(out.reference, MaxReferenceLength) {
		return documentFields{}, ErrFieldTooLong
	}
	if out.currencyCode, err = currencyCode(s.currencyCode); err != nil {
		return documentFields{}, err
	}
	if s.currencyRate.IsNegative() {
		return documentFields{}, ErrNegativeRate
	}
	out.currencyRate = numberOf(s.currencyRate)
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
		return "", ErrUnknownStatus
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
		return "", ErrUnknownAmountTypes
	}
}

func lineBodies(lines []LineItem) ([]lineBody, error) {
	if len(lines) == 0 {
		return nil, ErrLinesRequired
	}
	out := make([]lineBody, 0, len(lines))
	for idx := range lines {
		line := &lines[idx]
		body := lineBody{
			Description: strings.TrimSpace(line.Description),
			Quantity:    numberOf(line.Quantity),
			UnitAmount:  numberOf(line.UnitAmount),
			AccountCode: strings.TrimSpace(line.AccountCode),
			ItemCode:    strings.TrimSpace(line.ItemCode),
		}
		if body.Description == "" && body.ItemCode == "" {
			return nil, ErrLineDescription
		}
		if !textWithin(body.Description, MaxDescriptionLength) ||
			!textWithin(body.ItemCode, MaxItemCodeLength) {
			return nil, ErrFieldTooLong
		}
		if !line.LineAmount.IsZero() {
			value := amount(line.LineAmount)
			body.LineAmount = &value
		}
		out = append(out, body)
	}
	return out, nil
}

func (c *Client) readByIDs(
	ctx context.Context,
	endpoint, resource string,
	ids []string,
	out any,
) (bool, error) {
	unique, err := guids(ids, MaxIDsPerRead)
	if err != nil {
		return false, err
	}
	if len(unique) == 0 {
		return false, nil
	}
	return true, c.do(ctx, &call{
		endpoint: endpoint,
		method:   http.MethodGet,
		path:     accountingPath(resource),
		query:    url.Values{idsParam: {strings.Join(unique, ",")}},
		out:      out,
	})
}

func (c *Client) readPage(
	ctx context.Context,
	endpoint, resource string,
	query url.Values,
	modifiedSince *time.Time,
	out any,
) error {
	return c.do(ctx, &call{
		endpoint:      endpoint,
		method:        http.MethodGet,
		path:          accountingPath(resource),
		query:         query,
		modifiedSince: modifiedSince,
		out:           out,
		expected:      []int{http.StatusOK, http.StatusNotModified},
	})
}

func collectPages[T any](fetch func(page int) ([]T, bool, error)) ([]T, error) {
	var all []T
	for page := 1; ; page++ {
		if page > maxSearchPages {
			return nil, ErrTooManyResults
		}
		items, more, err := fetch(page)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if !more {
			if all == nil {
				all = []T{}
			}
			return all, nil
		}
	}
}

func withPage(base url.Values, page int) (url.Values, error) {
	query, err := pageQuery(page)
	if err != nil {
		return nil, err
	}
	for name, values := range base {
		query[name] = values
	}
	return query, nil
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
			return filterTerms{}, ErrUnknownType
		}
		out.kind = kind
	}
	numbers, err := searchNumbers(f.Numbers)
	if err != nil {
		return filterTerms{}, err
	}
	out.numbers = numbers
	if out.contactIDs, err = guids(f.ContactIDs, MaxIDsPerRead); err != nil {
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
		return filterTerms{}, ErrEmptyFilter
	}
	return out, nil
}

func searchNumbers(raw []string) ([]string, error) {
	out := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, value := range raw {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || !textWithin(trimmed, MaxDocumentNumberLength) ||
			strings.ContainsAny(trimmed, ",\"\\") || hasControl(trimmed) {
			return nil, ErrInvalidFilter
		}
		if _, dup := seen[trimmed]; dup {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) > MaxIDsPerRead {
		return nil, ErrTooManyIDs
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
			return nil, ErrUnknownStatus
		}
		if !slices.Contains(out, status) {
			out = append(out, status)
		}
	}
	return out, nil
}

func hasControl(value string) bool {
	for idx := range len(value) {
		if value[idx] < ' ' || value[idx] == 0x7f {
			return true
		}
	}
	return false
}

func whereDate(t time.Time) string {
	return fmt.Sprintf("DateTime(%d,%02d,%02d)", t.Year(), int(t.Month()), t.Day())
}

func whereString(value string) string {
	return strconv.Quote(value)
}

func whereAnyOf(field string, values []string, literal func(string) string) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, field+"=="+literal(value))
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return "(" + strings.Join(parts, " OR ") + ")"
}

func whereGUID(value string) string {
	return "guid(" + strconv.Quote(value) + ")"
}

func whereJoin(clauses ...string) string {
	kept := make([]string, 0, len(clauses))
	for _, clause := range clauses {
		if clause != "" {
			kept = append(kept, clause)
		}
	}
	return strings.Join(kept, " AND ")
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
