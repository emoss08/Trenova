package xeroconnector

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/xero"
)

var (
	_ services.AccountingReferenceReader  = (*Connector)(nil)
	_ services.AccountingReferenceCreator = (*Connector)(nil)
)

const (
	accountStatusDeleted = "DELETED"
	itemTypeService      = "Service"
)

var (
	errItemDraftRequired  = errors.New("xero: an item draft is required")
	errPartyDraftRequired = errors.New("xero: a customer or supplier draft is required")
)

func (c *Connector) MaxReferencePageSize() int {
	return xero.MaxPageSize
}

func (c *Connector) ListReference(
	ctx context.Context,
	req *services.AccountingReferencePageRequest,
) (*services.AccountingReferencePage, error) {
	client, err := c.client(services.AccountingDocumentAuth{
		RealmID:     req.RealmID,
		AccessToken: req.AccessToken,
	})
	if err != nil {
		return nil, err
	}

	switch req.Kind {
	case accountingsync.ReferenceKindAccount:
		accounts, listErr := client.Accounts(ctx, nil)
		if listErr != nil {
			return nil, listErr
		}
		c.codes.store(client.TenantID(), accounts)
		out := &services.AccountingReferencePage{
			Objects: make([]*accountingsync.AccountingReferenceObject, 0, len(accounts)),
		}
		for idx := range accounts {
			out.Objects = append(out.Objects, accountObjectOf(&accounts[idx]))
		}
		return out, nil
	case accountingsync.ReferenceKindItem:
		items, listErr := client.Items(ctx, nil)
		if listErr != nil {
			return nil, listErr
		}
		objects, mapErr := c.itemObjects(ctx, client, items)
		if mapErr != nil {
			return nil, mapErr
		}
		return &services.AccountingReferencePage{Objects: objects}, nil
	case accountingsync.ReferenceKindCustomer, accountingsync.ReferenceKindVendor:
		page := max(req.StartPosition, 1)
		contacts, listErr := client.Contacts(ctx, page, nil)
		if listErr != nil {
			return nil, listErr
		}
		out := &services.AccountingReferencePage{
			Objects: make([]*accountingsync.AccountingReferenceObject, 0, len(contacts.Contacts)),
		}
		for idx := range contacts.Contacts {
			contact := &contacts.Contacts[idx]
			if contactHolds(contact, req.Kind) {
				out.Objects = append(out.Objects, contactObjectOf(req.Kind, contact))
			}
		}
		if contacts.More {
			out.NextStart = page + 1
		}
		return out, nil
	case accountingsync.ReferenceKindTerm, accountingsync.ReferenceKindPaymentMethod:
		return &services.AccountingReferencePage{}, nil
	default:
		return nil, fmt.Errorf("xero: unknown reference kind %q", req.Kind)
	}
}

func (c *Connector) itemObjects(
	ctx context.Context,
	client *xero.Client,
	items []xero.Item,
) ([]*accountingsync.AccountingReferenceObject, error) {
	objects := make([]*accountingsync.AccountingReferenceObject, 0, len(items))
	for idx := range items {
		item := &items[idx]
		obj := itemObjectOf(item)
		if item.SalesAccountCode != "" {
			accountID, ok, err := c.codes.idFor(ctx, client, item.SalesAccountCode)
			if err != nil {
				return nil, err
			}
			if ok {
				obj.IncomeAccountExternalID = accountID
			}
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

func (c *Connector) CreateReference(
	ctx context.Context,
	req *services.AccountingCreateReferenceRequest,
) (*accountingsync.AccountingReferenceObject, error) {
	client, err := c.client(services.AccountingDocumentAuth{
		RealmID:     req.RealmID,
		AccessToken: req.AccessToken,
	})
	if err != nil {
		return nil, err
	}

	switch req.Kind {
	case accountingsync.ReferenceKindItem:
		if req.Item == nil {
			return nil, errItemDraftRequired
		}
		input := xero.ItemInput{
			Code:        itemCode(req.Item),
			Name:        c.SanitizeName(accountingsync.ReferenceKindItem, req.Item.Name),
			Description: stringutils.TruncateRunes(req.Item.Description, xero.MaxDescriptionLength),
		}
		if strings.TrimSpace(req.Item.IncomeAccountID) != "" {
			code, codeErr := c.accountCode(
				ctx,
				client,
				req.Item.IncomeAccountID,
				"The item's income account",
			)
			if codeErr != nil {
				return nil, codeErr
			}
			input.SalesAccountCode = code
		}
		created, createErr := client.CreateItem(ctx, req.RequestID, input)
		if createErr != nil {
			return nil, createErr
		}
		obj := itemObjectOf(created)
		obj.IncomeAccountExternalID = req.Item.IncomeAccountID
		return obj, nil
	case accountingsync.ReferenceKindCustomer, accountingsync.ReferenceKindVendor:
		if req.Party == nil {
			return nil, errPartyDraftRequired
		}
		created, createErr := client.CreateContact(ctx, req.RequestID, contactInputOf(req.Party))
		if createErr != nil {
			return nil, createErr
		}
		return contactObjectOf(req.Kind, created), nil
	case accountingsync.ReferenceKindAccount,
		accountingsync.ReferenceKindTerm,
		accountingsync.ReferenceKindPaymentMethod:
		return nil, fmt.Errorf("xero: %s records are not created from Trenova", req.Kind)
	default:
		return nil, fmt.Errorf("xero: unknown reference kind %q", req.Kind)
	}
}

func (c *Connector) IsDuplicateName(err error) bool {
	return xero.IsDuplicateNumber(err)
}

func (c *Connector) SanitizeName(kind accountingsync.ReferenceKind, name string) string {
	if kind == accountingsync.ReferenceKindItem {
		return strings.TrimSpace(stringutils.OneLine(name, xero.MaxItemNameLength))
	}
	return strings.TrimSpace(stringutils.OneLine(name, xero.MaxContactNameLength))
}

func itemCode(draft *services.AccountingItemDraft) string {
	source := strings.TrimSpace(draft.Sku)
	if source == "" {
		source = draft.Name
	}
	return strings.TrimSpace(stringutils.OneLine(source, xero.MaxItemCodeLength))
}

func contactInputOf(draft *services.AccountingPartyDraft) xero.ContactInput {
	input := xero.ContactInput{
		Name: strings.TrimSpace(
			stringutils.OneLine(draft.DisplayName, xero.MaxContactNameLength),
		),
		EmailAddress: strings.TrimSpace(draft.Email),
	}
	address := xero.Address{
		AddressType:  xero.AddressTypeStreet,
		AddressLine1: strings.TrimSpace(draft.AddressLine1),
		City:         strings.TrimSpace(draft.City),
		Region:       strings.TrimSpace(draft.State),
		PostalCode:   strings.TrimSpace(draft.PostalCode),
		Country:      strings.TrimSpace(draft.Country),
	}
	if address.AddressLine1 != "" || address.City != "" || address.Region != "" ||
		address.PostalCode != "" || address.Country != "" {
		input.Addresses = []xero.Address{address}
	}
	return input
}

func contactHolds(contact *xero.Contact, kind accountingsync.ReferenceKind) bool {
	if !contact.IsCustomer && !contact.IsSupplier {
		return true
	}
	if kind == accountingsync.ReferenceKindCustomer {
		return contact.IsCustomer
	}
	return contact.IsSupplier
}

func contactObjectOf(
	kind accountingsync.ReferenceKind,
	contact *xero.Contact,
) *accountingsync.AccountingReferenceObject {
	out := &accountingsync.AccountingReferenceObject{
		Kind:        kind,
		ExternalID:  contact.ContactID,
		Name:        contact.Name,
		Number:      contact.ContactNumber,
		Active:      !strings.EqualFold(contact.ContactStatus, xero.ContactStatusArchived),
		CompanyName: contact.Name,
		Email:       contact.EmailAddress,
	}
	if address, ok := primaryAddress(contact.Addresses); ok {
		out.AddressLine1 = address.AddressLine1
		out.City = address.City
		out.State = address.Region
		out.PostalCode = address.PostalCode
	}
	out.ProviderUpdatedAt = unixOf(contact.UpdatedAt)
	return out
}

func primaryAddress(addresses []xero.Address) (*xero.Address, bool) {
	var fallback *xero.Address
	for idx := range addresses {
		address := &addresses[idx]
		if address.AddressLine1 == "" && address.City == "" && address.PostalCode == "" {
			continue
		}
		if strings.EqualFold(address.AddressType, xero.AddressTypeStreet) {
			return address, true
		}
		if fallback == nil {
			fallback = address
		}
	}
	return fallback, fallback != nil
}

func itemObjectOf(item *xero.Item) *accountingsync.AccountingReferenceObject {
	name := item.Name
	if name == "" {
		name = item.Code
	}
	return &accountingsync.AccountingReferenceObject{
		Kind:              accountingsync.ReferenceKindItem,
		ExternalID:        item.ItemID,
		Name:              name,
		Number:            item.Code,
		Description:       item.Description,
		ItemType:          itemTypeService,
		Active:            true,
		ProviderUpdatedAt: unixOf(item.UpdatedAt),
	}
}

func accountObjectOf(account *xero.Account) *accountingsync.AccountingReferenceObject {
	subType := account.SystemAccount
	if subType == "" {
		subType = account.BankAccountType
	}
	return &accountingsync.AccountingReferenceObject{
		Kind:              accountingsync.ReferenceKindAccount,
		ExternalID:        account.AccountID,
		Name:              account.Name,
		Number:            account.Code,
		Classification:    account.Class,
		AccountType:       account.Type,
		AccountSubType:    subType,
		AccountClass:      AccountClassOf(account),
		Active:            account.Active(),
		CurrencyCode:      account.CurrencyCode,
		ProviderUpdatedAt: unixOf(account.UpdatedAt),
	}
}

var accountClasses = map[string]accountingsync.AccountClass{
	"BANK":                    accountingsync.AccountClassBank,
	"CURRENT":                 accountingsync.AccountClassAsset,
	"FIXED":                   accountingsync.AccountClassAsset,
	"INVENTORY":               accountingsync.AccountClassAsset,
	"NONCURRENT":              accountingsync.AccountClassAsset,
	"PREPAYMENT":              accountingsync.AccountClassAsset,
	"CURRLIAB":                accountingsync.AccountClassLiability,
	"LIABILITY":               accountingsync.AccountClassLiability,
	"TERMLIAB":                accountingsync.AccountClassLiability,
	"PAYGLIABILITY":           accountingsync.AccountClassLiability,
	"SUPERANNUATIONLIABILITY": accountingsync.AccountClassLiability,
	"WAGESPAYABLELIABILITY":   accountingsync.AccountClassLiability,
	"EQUITY":                  accountingsync.AccountClassEquity,
	"REVENUE":                 accountingsync.AccountClassIncome,
	"SALES":                   accountingsync.AccountClassIncome,
	"OTHERINCOME":             accountingsync.AccountClassOtherIncome,
	"DIRECTCOSTS":             accountingsync.AccountClassCostOfSales,
	"EXPENSE":                 accountingsync.AccountClassExpense,
	"OVERHEADS":               accountingsync.AccountClassExpense,
	"DEPRECIATN":              accountingsync.AccountClassExpense,
	"SUPERANNUATIONEXPENSE":   accountingsync.AccountClassExpense,
	"WAGESEXPENSE":            accountingsync.AccountClassExpense,
}

func AccountClassOf(account *xero.Account) accountingsync.AccountClass {
	switch {
	case account.IsReceivable():
		return accountingsync.AccountClassReceivable
	case account.IsPayable():
		return accountingsync.AccountClassPayable
	default:
		return accountClasses[strings.ToUpper(strings.TrimSpace(account.Type))]
	}
}

func unixOf(t time.Time) *int64 {
	if t.IsZero() {
		return nil
	}
	value := t.Unix()
	return &value
}
