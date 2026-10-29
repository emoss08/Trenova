package xerorefsync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/xerosession"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/emoss08/trenova/shared/xeroerrors"
	"github.com/emoss08/trenova/shared/xeroref"
)

type References struct {
	*xerosession.Session
}

const (
	itemTypeService = "Service"
)

func (c *References) MaxReferencePageSize() int {
	return xero.MaxPageSize
}

func (c *References) ListReference(
	ctx context.Context,
	req *accountingconnectorports.AccountingReferencePageRequest,
) (*accountingconnectorports.AccountingReferencePage, error) {
	client, err := c.Client(accountingconnectorports.AccountingDocumentAuth{
		RealmID:     req.RealmID,
		AccessToken: req.AccessToken,
	})
	if err != nil {
		return nil, err
	}

	switch req.Kind {
	case accountingprovider.ReferenceKindAccount:
		accounts, listErr := xeroref.Accounts(ctx, client, nil)
		if listErr != nil {
			return nil, listErr
		}
		c.Codes.Store(client.TenantID(), accounts)
		out := &accountingconnectorports.AccountingReferencePage{
			Objects: make([]*accountingmapping.AccountingReferenceObject, 0, len(accounts)),
		}
		for idx := range accounts {
			out.Objects = append(out.Objects, AccountObjectOf(&accounts[idx]))
		}
		return out, nil
	case accountingprovider.ReferenceKindItem:
		items, listErr := xeroref.Items(ctx, client, nil)
		if listErr != nil {
			return nil, listErr
		}
		objects, mapErr := c.ItemObjects(ctx, client, items)
		if mapErr != nil {
			return nil, mapErr
		}
		return &accountingconnectorports.AccountingReferencePage{Objects: objects}, nil
	case accountingprovider.ReferenceKindCustomer, accountingprovider.ReferenceKindVendor:
		page := max(req.StartPosition, 1)
		contacts, listErr := xeroref.Contacts(ctx, client, page, nil)
		if listErr != nil {
			return nil, listErr
		}
		out := &accountingconnectorports.AccountingReferencePage{
			Objects: make(
				[]*accountingmapping.AccountingReferenceObject,
				0,
				len(contacts.Contacts),
			),
		}
		for idx := range contacts.Contacts {
			contact := &contacts.Contacts[idx]
			if ContactHolds(contact, req.Kind) {
				out.Objects = append(out.Objects, ContactObjectOf(req.Kind, contact))
			}
		}
		if contacts.More {
			out.NextStart = page + 1
		}
		return out, nil
	case accountingprovider.ReferenceKindTerm, accountingprovider.ReferenceKindPaymentMethod:
		return &accountingconnectorports.AccountingReferencePage{}, nil
	default:
		return nil, fmt.Errorf("xero: unknown reference kind %q", req.Kind)
	}
}

func (c *References) ItemObjects(
	ctx context.Context,
	client *xero.Client,
	items []xeroref.Item,
) ([]*accountingmapping.AccountingReferenceObject, error) {
	objects := make([]*accountingmapping.AccountingReferenceObject, 0, len(items))
	for idx := range items {
		item := &items[idx]
		obj := itemObjectOf(item)
		if item.SalesAccountCode != "" {
			accountID, ok, err := c.Codes.IDFor(ctx, client, item.SalesAccountCode)
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

func (c *References) CreateReference(
	ctx context.Context,
	req *accountingconnectorports.AccountingCreateReferenceRequest,
) (*accountingmapping.AccountingReferenceObject, error) {
	client, err := c.Client(accountingconnectorports.AccountingDocumentAuth{
		RealmID:     req.RealmID,
		AccessToken: req.AccessToken,
	})
	if err != nil {
		return nil, err
	}

	switch req.Kind {
	case accountingprovider.ReferenceKindItem:
		if req.Item == nil {
			return nil, xerosession.ErrItemDraftRequired
		}
		input := xeroref.ItemInput{
			Code: itemCode(req.Item),
			Name: c.SanitizeName(accountingprovider.ReferenceKindItem, req.Item.Name),
			Description: stringutils.TruncateRunes(
				req.Item.Description,
				xeroref.MaxDescriptionLength,
			),
		}
		if strings.TrimSpace(req.Item.IncomeAccountID) != "" {
			code, codeErr := c.AccountCode(
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
		created, createErr := xeroref.CreateItem(ctx, client, req.RequestID, input)
		if createErr != nil {
			return nil, createErr
		}
		obj := itemObjectOf(created)
		obj.IncomeAccountExternalID = req.Item.IncomeAccountID
		return obj, nil
	case accountingprovider.ReferenceKindCustomer, accountingprovider.ReferenceKindVendor:
		if req.Party == nil {
			return nil, xerosession.ErrPartyDraftRequired
		}
		created, createErr := xeroref.CreateContact(
			ctx,
			client,
			req.RequestID,
			xerosession.ContactInputOf(req.Party),
		)
		if createErr != nil {
			return nil, createErr
		}
		return ContactObjectOf(req.Kind, created), nil
	case accountingprovider.ReferenceKindAccount,
		accountingprovider.ReferenceKindTerm,
		accountingprovider.ReferenceKindPaymentMethod:
		return nil, fmt.Errorf("xero: %s records are not created from Trenova", req.Kind)
	default:
		return nil, fmt.Errorf("xero: unknown reference kind %q", req.Kind)
	}
}

func (c *References) IsDuplicateName(err error) bool {
	return xeroerrors.IsDuplicateNumber(err)
}

func (c *References) SanitizeName(kind accountingprovider.ReferenceKind, name string) string {
	if kind == accountingprovider.ReferenceKindItem {
		return strings.TrimSpace(stringutils.OneLine(name, xeroref.MaxItemNameLength))
	}
	return strings.TrimSpace(stringutils.OneLine(name, xeroref.MaxContactNameLength))
}

func itemCode(draft *accountingconnectorports.AccountingItemDraft) string {
	source := strings.TrimSpace(draft.Sku)
	if source == "" {
		source = draft.Name
	}
	return strings.TrimSpace(stringutils.OneLine(source, xeroref.MaxItemCodeLength))
}

func ContactHolds(contact *xeroref.Contact, kind accountingprovider.ReferenceKind) bool {
	if !contact.IsCustomer && !contact.IsSupplier {
		return true
	}
	if kind == accountingprovider.ReferenceKindCustomer {
		return contact.IsCustomer
	}
	return contact.IsSupplier
}

func ContactObjectOf(
	kind accountingprovider.ReferenceKind,
	contact *xeroref.Contact,
) *accountingmapping.AccountingReferenceObject {
	out := &accountingmapping.AccountingReferenceObject{
		Kind:        kind,
		ExternalID:  contact.ContactID,
		Name:        contact.Name,
		Number:      contact.ContactNumber,
		Active:      !strings.EqualFold(contact.ContactStatus, xeroref.ContactStatusArchived),
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

func primaryAddress(addresses []xeroref.Address) (*xeroref.Address, bool) {
	var fallback *xeroref.Address
	for idx := range addresses {
		address := &addresses[idx]
		if address.AddressLine1 == "" && address.City == "" && address.PostalCode == "" {
			continue
		}
		if strings.EqualFold(address.AddressType, xeroref.AddressTypeStreet) {
			return address, true
		}
		if fallback == nil {
			fallback = address
		}
	}
	return fallback, fallback != nil
}

func itemObjectOf(item *xeroref.Item) *accountingmapping.AccountingReferenceObject {
	name := item.Name
	if name == "" {
		name = item.Code
	}
	return &accountingmapping.AccountingReferenceObject{
		Kind:              accountingprovider.ReferenceKindItem,
		ExternalID:        item.ItemID,
		Name:              name,
		Number:            item.Code,
		Description:       item.Description,
		ItemType:          itemTypeService,
		Active:            true,
		ProviderUpdatedAt: unixOf(item.UpdatedAt),
	}
}

func AccountObjectOf(account *xeroref.Account) *accountingmapping.AccountingReferenceObject {
	subType := account.SystemAccount
	if subType == "" {
		subType = account.BankAccountType
	}
	return &accountingmapping.AccountingReferenceObject{
		Kind:              accountingprovider.ReferenceKindAccount,
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

var accountClasses = map[string]accountingmapping.AccountClass{
	"BANK":                    accountingmapping.AccountClassBank,
	"CURRENT":                 accountingmapping.AccountClassAsset,
	"FIXED":                   accountingmapping.AccountClassAsset,
	"INVENTORY":               accountingmapping.AccountClassAsset,
	"NONCURRENT":              accountingmapping.AccountClassAsset,
	"PREPAYMENT":              accountingmapping.AccountClassAsset,
	"CURRLIAB":                accountingmapping.AccountClassLiability,
	"LIABILITY":               accountingmapping.AccountClassLiability,
	"TERMLIAB":                accountingmapping.AccountClassLiability,
	"PAYGLIABILITY":           accountingmapping.AccountClassLiability,
	"SUPERANNUATIONLIABILITY": accountingmapping.AccountClassLiability,
	"WAGESPAYABLELIABILITY":   accountingmapping.AccountClassLiability,
	"EQUITY":                  accountingmapping.AccountClassEquity,
	"REVENUE":                 accountingmapping.AccountClassIncome,
	"SALES":                   accountingmapping.AccountClassIncome,
	"OTHERINCOME":             accountingmapping.AccountClassOtherIncome,
	"DIRECTCOSTS":             accountingmapping.AccountClassCostOfSales,
	"EXPENSE":                 accountingmapping.AccountClassExpense,
	"OVERHEADS":               accountingmapping.AccountClassExpense,
	"DEPRECIATN":              accountingmapping.AccountClassExpense,
	"SUPERANNUATIONEXPENSE":   accountingmapping.AccountClassExpense,
	"WAGESEXPENSE":            accountingmapping.AccountClassExpense,
}

func AccountClassOf(account *xeroref.Account) accountingmapping.AccountClass {
	switch {
	case account.IsReceivable():
		return accountingmapping.AccountClassReceivable
	case account.IsPayable():
		return accountingmapping.AccountClassPayable
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
