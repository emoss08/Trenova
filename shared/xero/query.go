package xero

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	maxSearchPages     = 50
	moneyDecimalPlaces = 2
	WhereParam         = "where"
	StatusesParam      = "Statuses"
)

var ErrTooManyResults = errors.New("xero: the search matched more documents than can be read")

type Number decimal.Decimal

func (n Number) MarshalJSON() ([]byte, error) {
	return []byte(decimal.Decimal(n).String()), nil
}

func NumberOf(value decimal.Decimal) *Number {
	if value.IsZero() {
		return nil
	}
	out := Number(value)
	return &out
}

type Amount decimal.Decimal

func (a Amount) MarshalJSON() ([]byte, error) {
	return []byte(decimal.Decimal(a).StringFixed(moneyDecimalPlaces)), nil
}

type IDRef struct {
	ContactID    string `json:"ContactID,omitempty"`
	InvoiceID    string `json:"InvoiceID,omitempty"`
	CreditNoteID string `json:"CreditNoteID,omitempty"`
	AccountID    string `json:"AccountID,omitempty"`
	Code         string `json:"Code,omitempty"`
}

type WireRef struct {
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

func (r *WireRef) NestedContactID() string {
	if r == nil || r.Contact == nil {
		return ""
	}
	return r.Contact.ContactID
}

func ReadPage(
	ctx context.Context,
	c *Client,
	endpoint, resource string,
	query url.Values,
	modifiedSince *time.Time,
	out any,
) error {
	return c.Do(ctx, &Call{
		Endpoint:      endpoint,
		Method:        http.MethodGet,
		Path:          AccountingPath(resource),
		Query:         query,
		ModifiedSince: modifiedSince,
		Out:           out,
		Expected:      []int{http.StatusOK, http.StatusNotModified},
	})

}

func CollectPages[T any](fetch func(page int) ([]T, bool, error)) ([]T, error) {
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

func WithPage(base url.Values, page int) (url.Values, error) {
	query, err := PageQuery(page)
	if err != nil {
		return nil, err
	}
	for name, values := range base {
		query[name] = values
	}
	return query, nil
}

func HasControl(value string) bool {
	for idx := range len(value) {
		if value[idx] < ' ' || value[idx] == 0x7f {
			return true
		}
	}
	return false
}

func WhereString(value string) string {
	return strconv.Quote(value)
}

func WhereAnyOf(field string, values []string, literal func(string) string) string {
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

func WhereGUID(value string) string {
	return "guid(" + strconv.Quote(value) + ")"
}

func WhereJoin(clauses ...string) string {
	kept := make([]string, 0, len(clauses))
	for _, clause := range clauses {
		if clause != "" {
			kept = append(kept, clause)
		}
	}
	return strings.Join(kept, " AND ")
}

func Optional(value string) []string {
	if value == "" {
		return nil
	}
	return []string{value}
}

type WirePagination struct {
	Page      int `json:"page"`
	PageSize  int `json:"pageSize"`
	PageCount int `json:"pageCount"`
	ItemCount int `json:"itemCount"`
}

func HasMore(pagination *WirePagination, page, received int) bool {
	if pagination != nil && pagination.PageCount > 0 {
		return page < pagination.PageCount
	}
	return received >= MaxPageSize
}
