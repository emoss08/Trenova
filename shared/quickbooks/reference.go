package quickbooks

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/restx"
)

type ReferenceKind string

const (
	KindAccount       = ReferenceKind("Account")
	KindItem          = ReferenceKind("Item")
	KindCustomer      = ReferenceKind("Customer")
	KindVendor        = ReferenceKind("Vendor")
	KindTerm          = ReferenceKind("Term")
	KindPaymentMethod = ReferenceKind("PaymentMethod")
)

const (
	MaxQueryResults    = 1000
	MaxRequestIDLength = 50
	MaxItemNameLength  = 100
	MaxPartyNameLength = 500
	itemTypeService    = "Service"
	faultDuplicateName = "6240"
)

var (
	ErrUnknownReferenceKind = errors.New("quickbooks: unknown reference kind")
	ErrInvalidPaging        = errors.New(
		"quickbooks: start position must be at least 1 and page size between 1 and 1000",
	)
	ErrRequestIDRequired = errors.New(
		"quickbooks: a request id of at most 50 characters is required",
	)
	ErrInvalidName = errors.New(
		"quickbooks: the name is empty, too long, or contains a colon, tab or line break",
	)
	ErrIncomeAccountRequired = errors.New("quickbooks: an item needs an income account")
)

func (k ReferenceKind) IsValid() bool {
	switch k {
	case KindAccount, KindItem, KindCustomer, KindVendor, KindTerm, KindPaymentMethod:
		return true
	default:
		return false
	}
}

func AllReferenceKinds() []ReferenceKind {
	return []ReferenceKind{
		KindAccount,
		KindItem,
		KindCustomer,
		KindVendor,
		KindTerm,
		KindPaymentMethod,
	}
}

type ReferenceObject struct {
	Kind               ReferenceKind
	ID                 string
	Name               string
	FullyQualifiedName string
	Number             string
	Description        string
	Classification     string
	AccountType        string
	AccountSubType     string
	ItemType           string
	TermType           string
	PaymentMethodType  string
	ParentID           string
	Active             bool
	CurrencyCode       string
	SyncToken          string
	LastUpdatedAt      int64
	CompanyName        string
	Email              string
	AddressLine1       string
	City               string
	State              string
	PostalCode         string
	IncomeAccountID    string
	DueDays            *int
	Is1099             bool
}

type ReferencePage struct {
	Objects   []ReferenceObject
	NextStart int
}

type ItemDraft struct {
	Name            string
	Description     string
	Sku             string
	IncomeAccountID string
}

type PartyDraft struct {
	DisplayName  string
	CompanyName  string
	Email        string
	AddressLine1 string
	City         string
	State        string
	PostalCode   string
	Country      string
	Is1099       bool
}

type refValue struct {
	Value string `json:"value"`
}

type metaData struct {
	LastUpdatedTime string `json:"LastUpdatedTime"`
}

type emailAddr struct {
	Address string `json:"Address"`
}

type physicalAddr struct {
	Line1                  string `json:"Line1,omitempty"`
	City                   string `json:"City,omitempty"`
	CountrySubDivisionCode string `json:"CountrySubDivisionCode,omitempty"`
	PostalCode             string `json:"PostalCode,omitempty"`
	Country                string `json:"Country,omitempty"`
}

type wireEntity struct {
	ID                 string        `json:"Id"`
	Name               string        `json:"Name"`
	DisplayName        string        `json:"DisplayName"`
	CompanyName        string        `json:"CompanyName"`
	FullyQualifiedName string        `json:"FullyQualifiedName"`
	AcctNum            string        `json:"AcctNum"`
	Sku                string        `json:"Sku"`
	Description        string        `json:"Description"`
	Classification     string        `json:"Classification"`
	AccountType        string        `json:"AccountType"`
	AccountSubType     string        `json:"AccountSubType"`
	Type               string        `json:"Type"`
	Active             bool          `json:"Active"`
	ParentRef          *refValue     `json:"ParentRef"`
	CurrencyRef        *refValue     `json:"CurrencyRef"`
	IncomeAccountRef   *refValue     `json:"IncomeAccountRef"`
	PrimaryEmailAddr   *emailAddr    `json:"PrimaryEmailAddr"`
	BillAddr           *physicalAddr `json:"BillAddr"`
	DueDays            *int          `json:"DueDays"`
	Vendor1099         bool          `json:"Vendor1099"`
	SyncToken          string        `json:"SyncToken"`
	MetaData           *metaData     `json:"MetaData"`
}

type queryResponse struct {
	Account       []wireEntity `json:"Account"`
	Item          []wireEntity `json:"Item"`
	Customer      []wireEntity `json:"Customer"`
	Vendor        []wireEntity `json:"Vendor"`
	Term          []wireEntity `json:"Term"`
	PaymentMethod []wireEntity `json:"PaymentMethod"`
}

type queryEnvelope struct {
	QueryResponse queryResponse `json:"QueryResponse"`
}

type createEnvelope struct {
	Item     *wireEntity `json:"Item"`
	Customer *wireEntity `json:"Customer"`
	Vendor   *wireEntity `json:"Vendor"`
}

type itemBody struct {
	Name             string    `json:"Name"`
	Type             string    `json:"Type"`
	Description      string    `json:"Description,omitempty"`
	Sku              string    `json:"Sku,omitempty"`
	IncomeAccountRef *refValue `json:"IncomeAccountRef"`
}

type partyBody struct {
	DisplayName      string        `json:"DisplayName"`
	CompanyName      string        `json:"CompanyName,omitempty"`
	PrimaryEmailAddr *emailAddr    `json:"PrimaryEmailAddr,omitempty"`
	BillAddr         *physicalAddr `json:"BillAddr,omitempty"`
	Vendor1099       *bool         `json:"Vendor1099,omitempty"`
}

func (c *Client) ListReference(
	ctx context.Context,
	kind ReferenceKind,
	startPosition, pageSize int,
) (*ReferencePage, error) {
	if !kind.IsValid() {
		return nil, ErrUnknownReferenceKind
	}
	if startPosition < 1 || pageSize < 1 || pageSize > MaxQueryResults {
		return nil, ErrInvalidPaging
	}

	query := c.query()
	query.Set(queryResource, "select * from "+string(kind)+
		" where Active in (true, false) startposition "+strconv.Itoa(startPosition)+
		" maxresults "+strconv.Itoa(pageSize))

	var out queryEnvelope
	if _, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: queryResource,
		Method:   http.MethodGet,
		Path:     c.companyPath(queryResource),
		Query:    query,
		Out:      &out,
	}); err != nil {
		return nil, err
	}

	wire := out.QueryResponse.entities(kind)
	page := &ReferencePage{Objects: make([]ReferenceObject, 0, len(wire))}
	for idx := range wire {
		page.Objects = append(page.Objects, wire[idx].toReference(kind))
	}
	if len(wire) == pageSize {
		page.NextStart = startPosition + pageSize
	}

	return page, nil
}

func (c *Client) CreateItem(
	ctx context.Context,
	requestID string,
	draft ItemDraft,
) (*ReferenceObject, error) {
	if err := validateRequestID(requestID); err != nil {
		return nil, err
	}
	if !validName(draft.Name, MaxItemNameLength) {
		return nil, ErrInvalidName
	}
	if strings.TrimSpace(draft.IncomeAccountID) == "" {
		return nil, ErrIncomeAccountRequired
	}

	return c.create(ctx, requestID, KindItem, "item", itemBody{
		Name:             strings.TrimSpace(draft.Name),
		Type:             itemTypeService,
		Description:      strings.TrimSpace(draft.Description),
		Sku:              strings.TrimSpace(draft.Sku),
		IncomeAccountRef: &refValue{Value: strings.TrimSpace(draft.IncomeAccountID)},
	})
}

func (c *Client) CreateCustomer(
	ctx context.Context,
	requestID string,
	draft *PartyDraft,
) (*ReferenceObject, error) {
	body, err := partyBodyOf(requestID, draft, false)
	if err != nil {
		return nil, err
	}
	return c.create(ctx, requestID, KindCustomer, "customer", body)
}

func (c *Client) CreateVendor(
	ctx context.Context,
	requestID string,
	draft *PartyDraft,
) (*ReferenceObject, error) {
	body, err := partyBodyOf(requestID, draft, true)
	if err != nil {
		return nil, err
	}
	return c.create(ctx, requestID, KindVendor, "vendor", body)
}

func (c *Client) create(
	ctx context.Context,
	requestID string,
	kind ReferenceKind,
	resource string,
	body any,
) (*ReferenceObject, error) {
	query := c.query()
	query.Set("requestid", requestID)

	var out createEnvelope
	if _, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: "create-" + resource,
		Method:   http.MethodPost,
		Path:     c.companyPath(resource),
		Query:    query,
		Body:     body,
		Out:      &out,
	}); err != nil {
		return nil, err
	}

	entity := out.entity(kind)
	if entity == nil {
		return nil, ErrUnexpectedPayload
	}
	created := entity.toReference(kind)
	return &created, nil
}

func partyBodyOf(requestID string, draft *PartyDraft, vendor bool) (partyBody, error) {
	if draft == nil {
		return partyBody{}, ErrInvalidName
	}
	if err := validateRequestID(requestID); err != nil {
		return partyBody{}, err
	}
	if !validName(draft.DisplayName, MaxPartyNameLength) {
		return partyBody{}, ErrInvalidName
	}

	body := partyBody{
		DisplayName: strings.TrimSpace(draft.DisplayName),
		CompanyName: strings.TrimSpace(draft.CompanyName),
	}
	if email := strings.TrimSpace(draft.Email); email != "" {
		body.PrimaryEmailAddr = &emailAddr{Address: email}
	}
	addr := physicalAddr{
		Line1:                  strings.TrimSpace(draft.AddressLine1),
		City:                   strings.TrimSpace(draft.City),
		CountrySubDivisionCode: strings.TrimSpace(draft.State),
		PostalCode:             strings.TrimSpace(draft.PostalCode),
		Country:                strings.TrimSpace(draft.Country),
	}
	if addr != (physicalAddr{}) {
		body.BillAddr = &addr
	}
	if vendor && draft.Is1099 {
		is1099 := true
		body.Vendor1099 = &is1099
	}

	return body, nil
}

func validateRequestID(requestID string) error {
	trimmed := strings.TrimSpace(requestID)
	if trimmed == "" || len(trimmed) > MaxRequestIDLength || trimmed != requestID {
		return ErrRequestIDRequired
	}
	return nil
}

func validName(name string, maxLength int) bool {
	trimmed := strings.TrimSpace(name)
	return trimmed != "" &&
		utf8.RuneCountInString(trimmed) <= maxLength &&
		!strings.ContainsAny(trimmed, ":\t\n\r")
}

func SanitizeName(value string, maxLength int) string {
	replaced := strings.NewReplacer(":", " - ", "\t", " ", "\n", " ", "\r", " ").Replace(value)
	collapsed := strings.Join(strings.Fields(replaced), " ")
	runes := []rune(collapsed)
	if maxLength > 0 && len(runes) > maxLength {
		return strings.TrimSpace(string(runes[:maxLength]))
	}
	return collapsed
}

func IsDuplicateName(err error) bool {
	var fault *FaultError
	if !errors.As(err, &fault) {
		return false
	}
	for idx := range fault.Errors {
		if fault.Errors[idx].Code == faultDuplicateName {
			return true
		}
	}
	return false
}

func (q *queryResponse) entities(kind ReferenceKind) []wireEntity {
	switch kind {
	case KindAccount:
		return q.Account
	case KindItem:
		return q.Item
	case KindCustomer:
		return q.Customer
	case KindVendor:
		return q.Vendor
	case KindTerm:
		return q.Term
	case KindPaymentMethod:
		return q.PaymentMethod
	default:
		return nil
	}
}

func (e *createEnvelope) entity(kind ReferenceKind) *wireEntity {
	switch kind {
	case KindItem:
		return e.Item
	case KindCustomer:
		return e.Customer
	case KindVendor:
		return e.Vendor
	case KindAccount, KindTerm, KindPaymentMethod:
		return nil
	default:
		return nil
	}
}

func (w *wireEntity) toReference(kind ReferenceKind) ReferenceObject {
	obj := ReferenceObject{
		Kind:               kind,
		ID:                 w.ID,
		Name:               w.Name,
		FullyQualifiedName: w.FullyQualifiedName,
		Description:        w.Description,
		Active:             w.Active,
		SyncToken:          w.SyncToken,
		CompanyName:        w.CompanyName,
		DueDays:            w.DueDays,
		Is1099:             w.Vendor1099,
	}
	if w.DisplayName != "" {
		obj.Name = w.DisplayName
	}
	if obj.FullyQualifiedName == "" {
		obj.FullyQualifiedName = obj.Name
	}

	switch kind {
	case KindAccount:
		obj.Number = w.AcctNum
		obj.Classification = w.Classification
		obj.AccountType = w.AccountType
		obj.AccountSubType = w.AccountSubType
	case KindItem:
		obj.Number = w.Sku
		obj.ItemType = w.Type
	case KindVendor:
		obj.Number = w.AcctNum
	case KindTerm:
		obj.TermType = w.Type
	case KindPaymentMethod:
		obj.PaymentMethodType = w.Type
	case KindCustomer:
	}

	if w.ParentRef != nil {
		obj.ParentID = w.ParentRef.Value
	}
	if w.CurrencyRef != nil {
		obj.CurrencyCode = w.CurrencyRef.Value
	}
	if w.IncomeAccountRef != nil {
		obj.IncomeAccountID = w.IncomeAccountRef.Value
	}
	if w.PrimaryEmailAddr != nil {
		obj.Email = w.PrimaryEmailAddr.Address
	}
	if w.BillAddr != nil {
		obj.AddressLine1 = w.BillAddr.Line1
		obj.City = w.BillAddr.City
		obj.State = w.BillAddr.CountrySubDivisionCode
		obj.PostalCode = w.BillAddr.PostalCode
	}
	if w.MetaData != nil && w.MetaData.LastUpdatedTime != "" {
		if updated, err := time.Parse(time.RFC3339, w.MetaData.LastUpdatedTime); err == nil {
			obj.LastUpdatedAt = updated.Unix()
		}
	}

	return obj
}
