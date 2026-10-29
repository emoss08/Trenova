package businesscentralref

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/businesscentralerrors"
	"github.com/shopspring/decimal"
)

const (
	CategoryAssets          = "Assets"
	CategoryLiabilities     = "Liabilities"
	CategoryEquity          = "Equity"
	CategoryIncome          = "Income"
	CategoryCostOfGoodsSold = "CostOfGoodsSold"
	CategoryExpense         = "Expense"
	AccountTypePosting      = "Posting"
	AccountTypeHeading      = "Heading"
	AccountTypeTotal        = "Total"
	ItemTypeInventory       = "Inventory"
	ItemTypeService         = "Service"
	ItemTypeNonInventory    = "Non-Inventory"
	BlockedShip             = "Ship"
	BlockedInvoice          = "Invoice"
	BlockedPayment          = "Payment"
	BlockedAll              = "All"
	maxEmailLength          = 80
	maxAddressLineLength    = 100
	maxCityLength           = 30
	maxStateLength          = 30
	maxPostalCodeLength     = 20
	maxCountryLength        = 10
	accountsEntity          = "accounts"
	itemsEntity             = "items"
	customersEntity         = "customers"
	vendorsEntity           = "vendors"
	paymentTermsEntity      = "paymentTerms"
	paymentMethodsEntity    = "paymentMethods"
	displayNameField        = "displayName"
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
		return "", businesscentralerrors.ErrUnknownKind
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
	ID                   string                   `json:"id"`
	Number               string                   `json:"number"`
	DisplayName          string                   `json:"displayName"`
	Category             string                   `json:"category"`
	SubCategory          string                   `json:"subCategory"`
	AccountType          string                   `json:"accountType"`
	Blocked              bool                     `json:"blocked"`
	DirectPosting        bool                     `json:"directPosting"`
	LastModifiedDateTime businesscentral.WireTime `json:"lastModifiedDateTime"`
}

func (w *wireAccount) account() Account {
	return Account{
		ID:            strings.ToLower(w.ID),
		Number:        w.Number,
		DisplayName:   w.DisplayName,
		Category:      businesscentral.EnumValue(w.Category),
		SubCategory:   w.SubCategory,
		AccountType:   businesscentral.EnumValue(w.AccountType),
		Blocked:       w.Blocked,
		DirectPosting: w.DirectPosting,
		LastModified:  w.LastModifiedDateTime.Time(),
	}
}

type wireItem struct {
	ID                   string                   `json:"id"`
	Number               string                   `json:"number"`
	DisplayName          string                   `json:"displayName"`
	Type                 string                   `json:"type"`
	Blocked              bool                     `json:"blocked"`
	UnitPrice            decimal.Decimal          `json:"unitPrice"`
	LastModifiedDateTime businesscentral.WireTime `json:"lastModifiedDateTime"`
}

func (w *wireItem) item() Item {
	return Item{
		ID:           strings.ToLower(w.ID),
		Number:       w.Number,
		DisplayName:  w.DisplayName,
		Type:         businesscentral.EnumValue(w.Type),
		Blocked:      w.Blocked,
		UnitPrice:    w.UnitPrice,
		LastModified: w.LastModifiedDateTime.Time(),
	}
}

type itemBody struct {
	DisplayName string `json:"displayName"`
	Type        string `json:"type"`
}

type wireParty struct {
	ID                   string                   `json:"id"`
	Number               string                   `json:"number"`
	DisplayName          string                   `json:"displayName"`
	Email                string                   `json:"email"`
	AddressLine1         string                   `json:"addressLine1"`
	City                 string                   `json:"city"`
	State                string                   `json:"state"`
	PostalCode           string                   `json:"postalCode"`
	Country              string                   `json:"country"`
	CurrencyCode         string                   `json:"currencyCode"`
	PaymentTermsID       string                   `json:"paymentTermsId"`
	Blocked              string                   `json:"blocked"`
	LastModifiedDateTime businesscentral.WireTime `json:"lastModifiedDateTime"`
	ETag                 string                   `json:"@odata.etag"`
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
		PaymentTermsID: businesscentral.EmptyGUID(w.PaymentTermsID),
		Blocked:        businesscentral.EnumValue(w.Blocked),
		LastModified:   w.LastModifiedDateTime.Time(),
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

func Accounts(
	ctx context.Context,
	c *businesscentral.Client,
	modifiedSince *time.Time,
) ([]Account, error) {
	return businesscentral.Collect(ctx, c.Core(), &businesscentral.ListCall{
		Endpoint: "accounts",
		Path:     c.CollectionPath(accountsEntity),
		Filter:   businesscentral.FilterModifiedSince(modifiedSince),
	}, (*wireAccount).account)
}

func Items(
	ctx context.Context,
	c *businesscentral.Client,
	modifiedSince *time.Time,
) ([]Item, error) {
	return businesscentral.Collect(ctx, c.Core(), &businesscentral.ListCall{
		Endpoint: "items",
		Path:     c.CollectionPath(itemsEntity),
		Filter:   businesscentral.FilterModifiedSince(modifiedSince),
	}, (*wireItem).item)
}

func CreateItem(ctx context.Context, c *businesscentral.Client, in *ItemInput) (*Item, error) {
	body, err := in.body()
	if err != nil {
		return nil, err
	}
	return businesscentral.FetchOne(ctx, c.Core(), &businesscentral.Call{
		Endpoint: "items-create",
		Method:   http.MethodPost,
		Path:     c.CollectionPath(itemsEntity),
		Body:     body,
		Expected: []int{http.StatusCreated},
	}, (*wireItem).item)
}

func (in *ItemInput) body() (itemBody, error) {
	name, err := businesscentral.CleanText(in.DisplayName, businesscentral.MaxDisplayNameLength)
	if err != nil {
		return itemBody{}, err
	}
	if name == "" {
		return itemBody{}, businesscentralerrors.ErrNameRequired
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
		return itemBody{}, businesscentralerrors.ErrUnknownType
	}
	return itemBody{DisplayName: name, Type: kind}, nil
}

func Parties(
	ctx context.Context,
	c *businesscentral.Client,
	kind PartyKind,
	modifiedSince *time.Time,
) ([]Party, error) {
	entity, err := kind.entity()
	if err != nil {
		return nil, err
	}
	return businesscentral.Collect(ctx, c.Core(), &businesscentral.ListCall{
		Endpoint: entity,
		Path:     c.CollectionPath(entity),
		Filter:   businesscentral.FilterModifiedSince(modifiedSince),
	}, (*wireParty).party)
}

func FindParties(
	ctx context.Context,
	c *businesscentral.Client,
	kind PartyKind,
	displayName string,
) ([]Party, error) {
	entity, err := kind.entity()
	if err != nil {
		return nil, err
	}
	literal, err := businesscentral.ODataString(strings.TrimSpace(displayName))
	if err != nil {
		return nil, err
	}
	return businesscentral.Collect(ctx, c.Core(), &businesscentral.ListCall{
		Endpoint: entity + businesscentral.SuffixSearch,
		Path:     c.CollectionPath(entity),
		Filter:   businesscentral.FilterEquals(displayNameField, literal),
	}, (*wireParty).party)
}

func ReadParty(
	ctx context.Context,
	c *businesscentral.Client,
	kind PartyKind,
	partyID string,
) (*Party, error) {
	entity, err := kind.entity()
	if err != nil {
		return nil, err
	}
	id, err := businesscentral.GUID(partyID)
	if err != nil {
		return nil, err
	}
	return businesscentral.FetchOne(ctx, c.Core(), &businesscentral.Call{
		Endpoint: entity + businesscentral.SuffixGet,
		Method:   http.MethodGet,
		Path:     c.EntityPath(entity, id),
	}, (*wireParty).party)
}

func CreateParty(
	ctx context.Context,
	c *businesscentral.Client,
	kind PartyKind,
	in *PartyInput,
) (*Party, error) {
	entity, err := kind.entity()
	if err != nil {
		return nil, err
	}
	body, err := in.body()
	if err != nil {
		return nil, err
	}
	return businesscentral.FetchOne(ctx, c.Core(), &businesscentral.Call{
		Endpoint: entity + businesscentral.SuffixCreate,
		Method:   http.MethodPost,
		Path:     c.CollectionPath(entity),
		Body:     body,
		Expected: []int{http.StatusCreated},
	}, (*wireParty).party)
}

type PartyUpdate struct {
	Kind    PartyKind
	PartyID string
	ETag    string
	Input   PartyInput
}

func UpdateParty(
	ctx context.Context,
	c *businesscentral.Client,
	update *PartyUpdate,
) (*Party, error) {
	entity, err := update.Kind.entity()
	if err != nil {
		return nil, err
	}
	id, err := businesscentral.GUID(update.PartyID)
	if err != nil {
		return nil, err
	}
	match, err := businesscentral.IfMatch(update.ETag)
	if err != nil {
		return nil, err
	}
	body, err := update.Input.body()
	if err != nil {
		return nil, err
	}
	return businesscentral.FetchOne(ctx, c.Core(), &businesscentral.Call{
		Endpoint: entity + businesscentral.SuffixUpdate,
		Method:   http.MethodPatch,
		Path:     c.EntityPath(entity, id),
		IfMatch:  match,
		Body:     body,
	}, (*wireParty).party)
}

func (in *PartyInput) body() (partyBody, error) {
	name, err := businesscentral.CleanText(in.DisplayName, businesscentral.MaxDisplayNameLength)
	if err != nil {
		return partyBody{}, err
	}
	if name == "" {
		return partyBody{}, businesscentralerrors.ErrNameRequired
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
		if *fields[idx].target, err = businesscentral.CleanText(fields[idx].raw, fields[idx].limit); err != nil {
			return partyBody{}, err
		}
	}
	if body.CurrencyCode, err = businesscentral.CurrencyCode(in.CurrencyCode); err != nil {
		return partyBody{}, err
	}
	if body.PaymentTermsID, err = businesscentral.OptionalGUID(in.PaymentTermsID); err != nil {
		return partyBody{}, err
	}
	return body, nil
}

func PaymentTerms(ctx context.Context, c *businesscentral.Client) ([]PaymentTerm, error) {
	return businesscentral.Collect(ctx, c.Core(), &businesscentral.ListCall{
		Endpoint: "payment-terms",
		Path:     c.CollectionPath(paymentTermsEntity),
	}, (*wirePaymentTerm).term)
}

func PaymentMethods(ctx context.Context, c *businesscentral.Client) ([]PaymentMethod, error) {
	return businesscentral.Collect(ctx, c.Core(), &businesscentral.ListCall{
		Endpoint: "payment-methods",
		Path:     c.CollectionPath(paymentMethodsEntity),
	}, (*wirePaymentMethod).method)
}
