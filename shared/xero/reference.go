package xero

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	AccountStatusActive     = "ACTIVE"
	AccountStatusArchived   = "ARCHIVED"
	SystemAccountDebtors    = "DEBTORS"
	SystemAccountCreditors  = "CREDITORS"
	ContactStatusActive     = "ACTIVE"
	ContactStatusArchived   = "ARCHIVED"
	AddressTypeStreet       = "STREET"
	AddressTypePOBox        = "POBOX"
	MaxItemCodeLength       = 30
	MaxItemNameLength       = 50
	MaxDescriptionLength    = 4000
	MaxContactNameLength    = 255
	MaxContactNumberLength  = 50
	MaxEmailLength          = 255
	MaxTaxNumberLength      = 50
	MaxAddressLineLength    = 500
	MaxAddressPartLength    = 255
	MaxPostalCodeLength     = 50
	contactsResource        = "Contacts"
	itemsResource           = "Items"
	includeArchivedParam    = "includeArchived"
	includeArchivedParamYes = "true"
)

type Account struct {
	AccountID               string
	Code                    string
	Name                    string
	Type                    string
	Status                  string
	Class                   string
	SystemAccount           string
	BankAccountType         string
	CurrencyCode            string
	EnablePaymentsToAccount bool
	UpdatedAt               time.Time
}

func (a *Account) Active() bool {
	return strings.EqualFold(a.Status, AccountStatusActive)
}

func (a *Account) IsReceivable() bool {
	return strings.EqualFold(a.SystemAccount, SystemAccountDebtors)
}

func (a *Account) IsPayable() bool {
	return strings.EqualFold(a.SystemAccount, SystemAccountCreditors)
}

type Item struct {
	ItemID           string
	Code             string
	Name             string
	Description      string
	IsSold           bool
	IsPurchased      bool
	SalesUnitPrice   decimal.Decimal
	SalesAccountCode string
	UpdatedAt        time.Time
}

type ItemInput struct {
	Code             string
	Name             string
	Description      string
	SalesAccountCode string
}

type Address struct {
	AddressType  string `json:"AddressType,omitempty"`
	AddressLine1 string `json:"AddressLine1,omitempty"`
	AddressLine2 string `json:"AddressLine2,omitempty"`
	AddressLine3 string `json:"AddressLine3,omitempty"`
	AddressLine4 string `json:"AddressLine4,omitempty"`
	City         string `json:"City,omitempty"`
	Region       string `json:"Region,omitempty"`
	PostalCode   string `json:"PostalCode,omitempty"`
	Country      string `json:"Country,omitempty"`
}

type Contact struct {
	ContactID     string
	ContactNumber string
	Name          string
	FirstName     string
	LastName      string
	EmailAddress  string
	ContactStatus string
	TaxNumber     string
	IsCustomer    bool
	IsSupplier    bool
	Addresses     []Address
	UpdatedAt     time.Time
}

type ContactInput struct {
	Name          string
	ContactNumber string
	EmailAddress  string
	TaxNumber     string
	Addresses     []Address
}

type ContactPage struct {
	Contacts []Contact
	More     bool
}

type wirePagination struct {
	Page      int `json:"page"`
	PageSize  int `json:"pageSize"`
	PageCount int `json:"pageCount"`
	ItemCount int `json:"itemCount"`
}

func hasMore(pagination *wirePagination, page, received int) bool {
	if pagination != nil && pagination.PageCount > 0 {
		return page < pagination.PageCount
	}
	return received >= MaxPageSize
}

type wireAccount struct {
	AccountID               string   `json:"AccountID"`
	Code                    string   `json:"Code"`
	Name                    string   `json:"Name"`
	Type                    string   `json:"Type"`
	Status                  string   `json:"Status"`
	Class                   string   `json:"Class"`
	SystemAccount           string   `json:"SystemAccount"`
	BankAccountType         string   `json:"BankAccountType"`
	CurrencyCode            string   `json:"CurrencyCode"`
	EnablePaymentsToAccount bool     `json:"EnablePaymentsToAccount"`
	UpdatedDateUTC          wireTime `json:"UpdatedDateUTC"`
}

type accountsEnvelope struct {
	Accounts []wireAccount `json:"Accounts"`
}

type wireSalesDetails struct {
	UnitPrice   decimal.NullDecimal `json:"UnitPrice"`
	AccountCode string              `json:"AccountCode"`
}

type wireItem struct {
	ItemID         string            `json:"ItemID"`
	Code           string            `json:"Code"`
	Name           string            `json:"Name"`
	Description    string            `json:"Description"`
	IsSold         bool              `json:"IsSold"`
	IsPurchased    bool              `json:"IsPurchased"`
	SalesDetails   *wireSalesDetails `json:"SalesDetails"`
	UpdatedDateUTC wireTime          `json:"UpdatedDateUTC"`
}

func (w *wireItem) item() Item {
	out := Item{
		ItemID:      w.ItemID,
		Code:        w.Code,
		Name:        w.Name,
		Description: w.Description,
		IsSold:      w.IsSold,
		IsPurchased: w.IsPurchased,
		UpdatedAt:   w.UpdatedDateUTC.time(),
	}
	if w.SalesDetails != nil {
		out.SalesAccountCode = w.SalesDetails.AccountCode
		if w.SalesDetails.UnitPrice.Valid {
			out.SalesUnitPrice = w.SalesDetails.UnitPrice.Decimal
		}
	}
	return out
}

type itemsEnvelope struct {
	Items []wireItem `json:"Items"`
}

type itemSalesBody struct {
	AccountCode string `json:"AccountCode,omitempty"`
}

type itemBody struct {
	Code         string         `json:"Code"`
	Name         string         `json:"Name"`
	Description  string         `json:"Description,omitempty"`
	IsSold       bool           `json:"IsSold"`
	SalesDetails *itemSalesBody `json:"SalesDetails,omitempty"`
}

type itemWriteEnvelope struct {
	Items []itemBody `json:"Items"`
}

type wireContact struct {
	ContactID      string    `json:"ContactID"`
	ContactNumber  string    `json:"ContactNumber"`
	Name           string    `json:"Name"`
	FirstName      string    `json:"FirstName"`
	LastName       string    `json:"LastName"`
	EmailAddress   string    `json:"EmailAddress"`
	ContactStatus  string    `json:"ContactStatus"`
	TaxNumber      string    `json:"TaxNumber"`
	IsCustomer     bool      `json:"IsCustomer"`
	IsSupplier     bool      `json:"IsSupplier"`
	Addresses      []Address `json:"Addresses"`
	UpdatedDateUTC wireTime  `json:"UpdatedDateUTC"`
}

func (w *wireContact) contact() Contact {
	return Contact{
		ContactID:     w.ContactID,
		ContactNumber: w.ContactNumber,
		Name:          w.Name,
		FirstName:     w.FirstName,
		LastName:      w.LastName,
		EmailAddress:  w.EmailAddress,
		ContactStatus: w.ContactStatus,
		TaxNumber:     w.TaxNumber,
		IsCustomer:    w.IsCustomer,
		IsSupplier:    w.IsSupplier,
		Addresses:     w.Addresses,
		UpdatedAt:     w.UpdatedDateUTC.time(),
	}
}

type contactsEnvelope struct {
	Contacts   []wireContact   `json:"Contacts"`
	Pagination *wirePagination `json:"pagination"`
}

type contactBody struct {
	ContactID     string    `json:"ContactID,omitempty"`
	Name          string    `json:"Name"`
	ContactNumber string    `json:"ContactNumber,omitempty"`
	EmailAddress  string    `json:"EmailAddress,omitempty"`
	TaxNumber     string    `json:"TaxNumber,omitempty"`
	Addresses     []Address `json:"Addresses,omitempty"`
}

type contactWriteEnvelope struct {
	Contacts []contactBody `json:"Contacts"`
}

func (c *Client) Accounts(ctx context.Context, modifiedSince *time.Time) ([]Account, error) {
	var out accountsEnvelope
	if err := c.do(ctx, &call{
		endpoint:      "accounts",
		method:        http.MethodGet,
		path:          accountingPath("Accounts"),
		modifiedSince: modifiedSince,
		out:           &out,
		expected:      []int{http.StatusOK, http.StatusNotModified},
	}); err != nil {
		return nil, err
	}

	accounts := make([]Account, 0, len(out.Accounts))
	for idx := range out.Accounts {
		wire := &out.Accounts[idx]
		accounts = append(accounts, Account{
			AccountID:               wire.AccountID,
			Code:                    wire.Code,
			Name:                    wire.Name,
			Type:                    wire.Type,
			Status:                  wire.Status,
			Class:                   wire.Class,
			SystemAccount:           wire.SystemAccount,
			BankAccountType:         wire.BankAccountType,
			CurrencyCode:            wire.CurrencyCode,
			EnablePaymentsToAccount: wire.EnablePaymentsToAccount,
			UpdatedAt:               wire.UpdatedDateUTC.time(),
		})
	}
	return accounts, nil
}

func (c *Client) Items(ctx context.Context, modifiedSince *time.Time) ([]Item, error) {
	var out itemsEnvelope
	if err := c.do(ctx, &call{
		endpoint:      "items",
		method:        http.MethodGet,
		path:          accountingPath(itemsResource),
		modifiedSince: modifiedSince,
		out:           &out,
		expected:      []int{http.StatusOK, http.StatusNotModified},
	}); err != nil {
		return nil, err
	}

	items := make([]Item, 0, len(out.Items))
	for idx := range out.Items {
		items = append(items, out.Items[idx].item())
	}
	return items, nil
}

func (c *Client) CreateItem(ctx context.Context, key string, item ItemInput) (*Item, error) {
	idem, err := idempotencyKey(key)
	if err != nil {
		return nil, err
	}
	body, err := item.body()
	if err != nil {
		return nil, err
	}

	var out itemsEnvelope
	if err = c.do(ctx, &call{
		endpoint: "items-create",
		method:   http.MethodPut,
		path:     accountingPath(itemsResource),
		key:      idem,
		body:     itemWriteEnvelope{Items: []itemBody{body}},
		out:      &out,
	}); err != nil {
		return nil, err
	}
	if len(out.Items) == 0 {
		return nil, ErrUnexpectedPayload
	}
	created := out.Items[0].item()
	return &created, nil
}

func (in *ItemInput) body() (itemBody, error) {
	code := strings.TrimSpace(in.Code)
	if code == "" || !textWithin(code, MaxItemCodeLength) {
		return itemBody{}, ErrItemCodeInvalid
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || !textWithin(name, MaxItemNameLength) {
		return itemBody{}, ErrItemNameInvalid
	}
	description := strings.TrimSpace(in.Description)
	if !textWithin(description, MaxDescriptionLength) {
		return itemBody{}, ErrFieldTooLong
	}

	body := itemBody{Code: code, Name: name, Description: description, IsSold: true}
	if account := strings.TrimSpace(in.SalesAccountCode); account != "" {
		body.SalesDetails = &itemSalesBody{AccountCode: account}
	}
	return body, nil
}

func (c *Client) Contacts(
	ctx context.Context,
	page int,
	modifiedSince *time.Time,
) (*ContactPage, error) {
	query, err := pageQuery(page)
	if err != nil {
		return nil, err
	}
	query.Set(includeArchivedParam, includeArchivedParamYes)

	var out contactsEnvelope
	if err = c.do(ctx, &call{
		endpoint:      "contacts",
		method:        http.MethodGet,
		path:          accountingPath(contactsResource),
		query:         query,
		modifiedSince: modifiedSince,
		out:           &out,
		expected:      []int{http.StatusOK, http.StatusNotModified},
	}); err != nil {
		return nil, err
	}

	contacts := make([]Contact, 0, len(out.Contacts))
	for idx := range out.Contacts {
		contacts = append(contacts, out.Contacts[idx].contact())
	}
	return &ContactPage{
		Contacts: contacts,
		More:     hasMore(out.Pagination, page, len(contacts)),
	}, nil
}

//nolint:gocritic // value inputs are the package contract the accounting adapter is written against.
func (c *Client) CreateContact(ctx context.Context, key string, in ContactInput) (*Contact, error) {
	return c.writeContact(ctx, key, "", &in)
}

//nolint:gocritic // value inputs are the package contract the accounting adapter is written against.
func (c *Client) UpdateContact(
	ctx context.Context,
	key, contactID string,
	in ContactInput,
) (*Contact, error) {
	id, err := guid(contactID)
	if err != nil {
		return nil, err
	}
	return c.writeContact(ctx, key, id, &in)
}

func (c *Client) writeContact(
	ctx context.Context,
	key, contactID string,
	in *ContactInput,
) (*Contact, error) {
	idem, err := idempotencyKey(key)
	if err != nil {
		return nil, err
	}
	body, err := in.body()
	if err != nil {
		return nil, err
	}
	body.ContactID = contactID

	req := &call{
		endpoint: "contacts-create",
		method:   http.MethodPut,
		path:     accountingPath(contactsResource),
		key:      idem,
		body:     contactWriteEnvelope{Contacts: []contactBody{body}},
	}
	if contactID != "" {
		req.endpoint = "contacts-update"
		req.method = http.MethodPost
		req.path = accountingPath(contactsResource, contactID)
	}

	var out contactsEnvelope
	req.out = &out
	if err = c.do(ctx, req); err != nil {
		return nil, err
	}
	if len(out.Contacts) == 0 {
		return nil, ErrUnexpectedPayload
	}
	written := out.Contacts[0].contact()
	return &written, nil
}

func (in *ContactInput) body() (contactBody, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || !textWithin(name, MaxContactNameLength) {
		return contactBody{}, ErrContactNameInvalid
	}
	body := contactBody{
		Name:          name,
		ContactNumber: strings.TrimSpace(in.ContactNumber),
		EmailAddress:  strings.TrimSpace(in.EmailAddress),
		TaxNumber:     strings.TrimSpace(in.TaxNumber),
	}
	if !textWithin(body.ContactNumber, MaxContactNumberLength) ||
		!textWithin(body.EmailAddress, MaxEmailLength) ||
		!textWithin(body.TaxNumber, MaxTaxNumberLength) {
		return contactBody{}, ErrFieldTooLong
	}
	if len(in.Addresses) > 0 {
		body.Addresses = make([]Address, 0, len(in.Addresses))
		for idx := range in.Addresses {
			address, err := cleanAddress(&in.Addresses[idx])
			if err != nil {
				return contactBody{}, err
			}
			body.Addresses = append(body.Addresses, address)
		}
	}
	return body, nil
}

func cleanAddress(in *Address) (Address, error) {
	out := Address{
		AddressType:  strings.ToUpper(strings.TrimSpace(in.AddressType)),
		AddressLine1: strings.TrimSpace(in.AddressLine1),
		AddressLine2: strings.TrimSpace(in.AddressLine2),
		AddressLine3: strings.TrimSpace(in.AddressLine3),
		AddressLine4: strings.TrimSpace(in.AddressLine4),
		City:         strings.TrimSpace(in.City),
		Region:       strings.TrimSpace(in.Region),
		PostalCode:   strings.TrimSpace(in.PostalCode),
		Country:      strings.TrimSpace(in.Country),
	}
	if out.AddressType == "" {
		out.AddressType = AddressTypeStreet
	}
	if out.AddressType != AddressTypeStreet && out.AddressType != AddressTypePOBox {
		return Address{}, ErrUnknownType
	}
	if !textWithin(out.AddressLine1, MaxAddressLineLength) ||
		!textWithin(out.AddressLine2, MaxAddressLineLength) ||
		!textWithin(out.AddressLine3, MaxAddressLineLength) ||
		!textWithin(out.AddressLine4, MaxAddressLineLength) ||
		!textWithin(out.City, MaxAddressPartLength) ||
		!textWithin(out.Region, MaxAddressPartLength) ||
		!textWithin(out.Country, MaxAddressPartLength) ||
		!textWithin(out.PostalCode, MaxPostalCodeLength) {
		return Address{}, ErrFieldTooLong
	}
	return out, nil
}
