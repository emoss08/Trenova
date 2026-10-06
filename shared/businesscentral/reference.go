package businesscentral

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	CategoryAssets            = "Assets"
	CategoryLiabilities       = "Liabilities"
	CategoryEquity            = "Equity"
	CategoryIncome            = "Income"
	CategoryCostOfGoodsSold   = "CostOfGoodsSold"
	CategoryExpense           = "Expense"
	AccountTypePosting        = "Posting"
	AccountTypeHeading        = "Heading"
	AccountTypeTotal          = "Total"
	ItemTypeInventory         = "Inventory"
	ItemTypeService           = "Service"
	ItemTypeNonInventory      = "Non-Inventory"
	BlockedShip               = "Ship"
	BlockedInvoice            = "Invoice"
	BlockedPayment            = "Payment"
	BlockedAll                = "All"
	maxDisplayNameLength      = 100
	maxEmailLength            = 80
	maxAddressLineLength      = 100
	maxCityLength             = 30
	maxStateLength            = 30
	maxPostalCodeLength       = 20
	maxCountryLength          = 10
	maxCurrencyCodeLength     = 10
	blankEnumValue            = "_x0020_"
	accountsEntity            = "accounts"
	itemsEntity               = "items"
	customersEntity           = "customers"
	vendorsEntity             = "vendors"
	paymentTermsEntity        = "paymentTerms"
	paymentMethodsEntity      = "paymentMethods"
	displayNameField          = "displayName"
	lastModifiedDateTimeField = "lastModifiedDateTime"
)

type PartyKind int

const (
	PartyCustomer PartyKind = iota + 1
	PartyVendor
)

func (k PartyKind) entity() (string, error) {
	switch k {
	case PartyCustomer:
		return customersEntity, nil
	case PartyVendor:
		return vendorsEntity, nil
	default:
		return "", ErrUnknownKind
	}
}

type Account struct {
	ID            string
	Number        string
	DisplayName   string
	Category      string
	SubCategory   string
	AccountType   string
	Blocked       bool
	DirectPosting bool
	LastModified  time.Time
}

func (a *Account) Postable() bool {
	return !a.Blocked && strings.EqualFold(a.AccountType, AccountTypePosting)
}

type Item struct {
	ID           string
	Number       string
	DisplayName  string
	Type         string
	Blocked      bool
	UnitPrice    decimal.Decimal
	LastModified time.Time
}

type ItemInput struct {
	DisplayName string
	Type        string
}

type Party struct {
	ID             string
	Number         string
	DisplayName    string
	Email          string
	AddressLine1   string
	City           string
	State          string
	PostalCode     string
	Country        string
	CurrencyCode   string
	PaymentTermsID string
	Blocked        string
	LastModified   time.Time
	ETag           string
}

func (p *Party) IsBlocked() bool {
	return p.Blocked != ""
}

type PartyInput struct {
	DisplayName    string
	Email          string
	AddressLine1   string
	City           string
	State          string
	PostalCode     string
	Country        string
	CurrencyCode   string
	PaymentTermsID string
}

type PaymentTerm struct {
	ID                 string
	Code               string
	DisplayName        string
	DueDateCalculation string
}

type PaymentMethod struct {
	ID          string
	Code        string
	DisplayName string
}

type wireAccount struct {
	ID                   string   `json:"id"`
	Number               string   `json:"number"`
	DisplayName          string   `json:"displayName"`
	Category             string   `json:"category"`
	SubCategory          string   `json:"subCategory"`
	AccountType          string   `json:"accountType"`
	Blocked              bool     `json:"blocked"`
	DirectPosting        bool     `json:"directPosting"`
	LastModifiedDateTime wireTime `json:"lastModifiedDateTime"`
}

func (w *wireAccount) account() Account {
	return Account{
		ID:            strings.ToLower(w.ID),
		Number:        w.Number,
		DisplayName:   w.DisplayName,
		Category:      enumValue(w.Category),
		SubCategory:   w.SubCategory,
		AccountType:   enumValue(w.AccountType),
		Blocked:       w.Blocked,
		DirectPosting: w.DirectPosting,
		LastModified:  w.LastModifiedDateTime.time(),
	}
}

type wireItem struct {
	ID                   string          `json:"id"`
	Number               string          `json:"number"`
	DisplayName          string          `json:"displayName"`
	Type                 string          `json:"type"`
	Blocked              bool            `json:"blocked"`
	UnitPrice            decimal.Decimal `json:"unitPrice"`
	LastModifiedDateTime wireTime        `json:"lastModifiedDateTime"`
}

func (w *wireItem) item() Item {
	return Item{
		ID:           strings.ToLower(w.ID),
		Number:       w.Number,
		DisplayName:  w.DisplayName,
		Type:         enumValue(w.Type),
		Blocked:      w.Blocked,
		UnitPrice:    w.UnitPrice,
		LastModified: w.LastModifiedDateTime.time(),
	}
}

type itemBody struct {
	DisplayName string `json:"displayName"`
	Type        string `json:"type"`
}

type wireParty struct {
	ID                   string   `json:"id"`
	Number               string   `json:"number"`
	DisplayName          string   `json:"displayName"`
	Email                string   `json:"email"`
	AddressLine1         string   `json:"addressLine1"`
	City                 string   `json:"city"`
	State                string   `json:"state"`
	PostalCode           string   `json:"postalCode"`
	Country              string   `json:"country"`
	CurrencyCode         string   `json:"currencyCode"`
	PaymentTermsID       string   `json:"paymentTermsId"`
	Blocked              string   `json:"blocked"`
	LastModifiedDateTime wireTime `json:"lastModifiedDateTime"`
	ETag                 string   `json:"@odata.etag"`
}

func (w *wireParty) party() Party {
	return Party{
		ID:             strings.ToLower(w.ID),
		Number:         w.Number,
		DisplayName:    w.DisplayName,
		Email:          w.Email,
		AddressLine1:   w.AddressLine1,
		City:           w.City,
		State:          w.State,
		PostalCode:     w.PostalCode,
		Country:        w.Country,
		CurrencyCode:   strings.ToUpper(strings.TrimSpace(w.CurrencyCode)),
		PaymentTermsID: emptyGUID(w.PaymentTermsID),
		Blocked:        enumValue(w.Blocked),
		LastModified:   w.LastModifiedDateTime.time(),
		ETag:           w.ETag,
	}
}

type partyBody struct {
	DisplayName    string `json:"displayName"`
	Email          string `json:"email,omitempty"`
	AddressLine1   string `json:"addressLine1,omitempty"`
	City           string `json:"city,omitempty"`
	State          string `json:"state,omitempty"`
	PostalCode     string `json:"postalCode,omitempty"`
	Country        string `json:"country,omitempty"`
	CurrencyCode   string `json:"currencyCode,omitempty"`
	PaymentTermsID string `json:"paymentTermsId,omitempty"`
}

type wirePaymentTerm struct {
	ID                 string `json:"id"`
	Code               string `json:"code"`
	DisplayName        string `json:"displayName"`
	DueDateCalculation string `json:"dueDateCalculation"`
}

func (w *wirePaymentTerm) term() PaymentTerm {
	return PaymentTerm{
		ID:                 strings.ToLower(w.ID),
		Code:               w.Code,
		DisplayName:        w.DisplayName,
		DueDateCalculation: w.DueDateCalculation,
	}
}

type wirePaymentMethod struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	DisplayName string `json:"displayName"`
}

func (w *wirePaymentMethod) method() PaymentMethod {
	return PaymentMethod{ID: strings.ToLower(w.ID), Code: w.Code, DisplayName: w.DisplayName}
}

func (c *Client) Accounts(ctx context.Context, modifiedSince *time.Time) ([]Account, error) {
	return collect(ctx, c.core, &listCall{
		endpoint: "accounts",
		path:     c.collectionPath(accountsEntity),
		filter:   filterModifiedSince(modifiedSince),
	}, (*wireAccount).account)
}

func (c *Client) Items(ctx context.Context, modifiedSince *time.Time) ([]Item, error) {
	return collect(ctx, c.core, &listCall{
		endpoint: "items",
		path:     c.collectionPath(itemsEntity),
		filter:   filterModifiedSince(modifiedSince),
	}, (*wireItem).item)
}

func (c *Client) CreateItem(ctx context.Context, in *ItemInput) (*Item, error) {
	body, err := in.body()
	if err != nil {
		return nil, err
	}
	return fetchOne(ctx, c.core, &call{
		endpoint: "items-create",
		method:   http.MethodPost,
		path:     c.collectionPath(itemsEntity),
		body:     body,
		expected: []int{http.StatusCreated},
	}, (*wireItem).item)
}

func (in *ItemInput) body() (itemBody, error) {
	name, err := cleanText(in.DisplayName, maxDisplayNameLength)
	if err != nil {
		return itemBody{}, err
	}
	if name == "" {
		return itemBody{}, ErrNameRequired
	}
	kind := strings.TrimSpace(in.Type)
	switch {
	case kind == "":
		kind = ItemTypeService
	case strings.EqualFold(kind, ItemTypeService):
		kind = ItemTypeService
	case strings.EqualFold(kind, ItemTypeInventory):
		kind = ItemTypeInventory
	case strings.EqualFold(kind, ItemTypeNonInventory):
		kind = ItemTypeNonInventory
	default:
		return itemBody{}, ErrUnknownType
	}
	return itemBody{DisplayName: name, Type: kind}, nil
}

func (c *Client) Parties(
	ctx context.Context,
	kind PartyKind,
	modifiedSince *time.Time,
) ([]Party, error) {
	entity, err := kind.entity()
	if err != nil {
		return nil, err
	}
	return collect(ctx, c.core, &listCall{
		endpoint: entity,
		path:     c.collectionPath(entity),
		filter:   filterModifiedSince(modifiedSince),
	}, (*wireParty).party)
}

func (c *Client) FindParties(
	ctx context.Context,
	kind PartyKind,
	displayName string,
) ([]Party, error) {
	entity, err := kind.entity()
	if err != nil {
		return nil, err
	}
	literal, err := odataString(strings.TrimSpace(displayName))
	if err != nil {
		return nil, err
	}
	return collect(ctx, c.core, &listCall{
		endpoint: entity + suffixSearch,
		path:     c.collectionPath(entity),
		filter:   filterEquals(displayNameField, literal),
	}, (*wireParty).party)
}

func (c *Client) Party(ctx context.Context, kind PartyKind, partyID string) (*Party, error) {
	entity, err := kind.entity()
	if err != nil {
		return nil, err
	}
	id, err := guid(partyID)
	if err != nil {
		return nil, err
	}
	return fetchOne(ctx, c.core, &call{
		endpoint: entity + suffixGet,
		method:   http.MethodGet,
		path:     c.entityPath(entity, id),
	}, (*wireParty).party)
}

func (c *Client) CreateParty(ctx context.Context, kind PartyKind, in *PartyInput) (*Party, error) {
	entity, err := kind.entity()
	if err != nil {
		return nil, err
	}
	body, err := in.body()
	if err != nil {
		return nil, err
	}
	return fetchOne(ctx, c.core, &call{
		endpoint: entity + suffixCreate,
		method:   http.MethodPost,
		path:     c.collectionPath(entity),
		body:     body,
		expected: []int{http.StatusCreated},
	}, (*wireParty).party)
}

type PartyUpdate struct {
	Kind    PartyKind
	PartyID string
	ETag    string
	Input   PartyInput
}

func (c *Client) UpdateParty(ctx context.Context, update *PartyUpdate) (*Party, error) {
	entity, err := update.Kind.entity()
	if err != nil {
		return nil, err
	}
	id, err := guid(update.PartyID)
	if err != nil {
		return nil, err
	}
	match, err := ifMatch(update.ETag)
	if err != nil {
		return nil, err
	}
	body, err := update.Input.body()
	if err != nil {
		return nil, err
	}
	return fetchOne(ctx, c.core, &call{
		endpoint: entity + suffixUpdate,
		method:   http.MethodPatch,
		path:     c.entityPath(entity, id),
		ifMatch:  match,
		body:     body,
	}, (*wireParty).party)
}

func (in *PartyInput) body() (partyBody, error) {
	name, err := cleanText(in.DisplayName, maxDisplayNameLength)
	if err != nil {
		return partyBody{}, err
	}
	if name == "" {
		return partyBody{}, ErrNameRequired
	}
	body := partyBody{DisplayName: name}
	fields := []struct {
		target *string
		raw    string
		limit  int
	}{
		{&body.Email, in.Email, maxEmailLength},
		{&body.AddressLine1, in.AddressLine1, maxAddressLineLength},
		{&body.City, in.City, maxCityLength},
		{&body.State, in.State, maxStateLength},
		{&body.PostalCode, in.PostalCode, maxPostalCodeLength},
		{&body.Country, in.Country, maxCountryLength},
	}
	for idx := range fields {
		if *fields[idx].target, err = cleanText(fields[idx].raw, fields[idx].limit); err != nil {
			return partyBody{}, err
		}
	}
	if body.CurrencyCode, err = currencyCode(in.CurrencyCode); err != nil {
		return partyBody{}, err
	}
	if body.PaymentTermsID, err = optionalGUID(in.PaymentTermsID); err != nil {
		return partyBody{}, err
	}
	return body, nil
}

func (c *Client) PaymentTerms(ctx context.Context) ([]PaymentTerm, error) {
	return collect(ctx, c.core, &listCall{
		endpoint: "payment-terms",
		path:     c.collectionPath(paymentTermsEntity),
	}, (*wirePaymentTerm).term)
}

func (c *Client) PaymentMethods(ctx context.Context) ([]PaymentMethod, error) {
	return collect(ctx, c.core, &listCall{
		endpoint: "payment-methods",
		path:     c.collectionPath(paymentMethodsEntity),
	}, (*wirePaymentMethod).method)
}

func enumValue(raw string) string {
	value := strings.TrimSpace(raw)
	if value == blankEnumValue {
		return ""
	}
	return value
}

func emptyGUID(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "00000000-0000-0000-0000-000000000000" {
		return ""
	}
	return value
}

func currencyCode(raw string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if !textWithin(code, maxCurrencyCodeLength) || hasControl(code) {
		return "", ErrCurrencyCodeInvalid
	}
	return code, nil
}
