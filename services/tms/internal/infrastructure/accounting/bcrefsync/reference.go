package bcrefsync

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/bcsession"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/businesscentralerrors"
	"github.com/emoss08/trenova/shared/businesscentralref"
)

type References struct {
	*bcsession.Session
}

const (
	maxNameLength       = 100
	maxEmailLength      = 80
	maxAddressLength    = 100
	maxCityLength       = 30
	maxStateLength      = 30
	maxPostalCodeLength = 20
	maxCountryLength    = 10
	dueDaysUnit         = "D"
)

func (c *References) MaxReferencePageSize() int {
	return businesscentral.MaxPageSize
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
	objects, err := listReference(ctx, client, req.Kind)
	if err != nil {
		return nil, err
	}
	return &accountingconnectorports.AccountingReferencePage{Objects: objects}, nil
}

func listReference(
	ctx context.Context,
	client *businesscentral.Client,
	kind accountingprovider.ReferenceKind,
) ([]*accountingmapping.AccountingReferenceObject, error) {
	switch kind {
	case accountingprovider.ReferenceKindAccount:
		accounts, err := businesscentralref.Accounts(ctx, client, nil)
		return convertAll(accounts, AccountObjectOf), err
	case accountingprovider.ReferenceKindItem:
		items, err := businesscentralref.Items(ctx, client, nil)
		return convertAll(items, ItemObjectOf), err
	case accountingprovider.ReferenceKindCustomer, accountingprovider.ReferenceKindVendor:
		partyKind := partyKindOf(kind)
		parties, err := businesscentralref.Parties(ctx, client, partyKind, nil)
		return convertAll(parties, partyConverter(kind)), err
	case accountingprovider.ReferenceKindTerm:
		terms, err := businesscentralref.PaymentTerms(ctx, client)
		return convertAll(terms, termObjectOf), err
	case accountingprovider.ReferenceKindPaymentMethod:
		return []*accountingmapping.AccountingReferenceObject{}, nil
	default:
		return nil, fmt.Errorf("businesscentral: unknown reference kind %q", kind)
	}
}

func convertAll[T any](
	values []T,
	convert func(*T) *accountingmapping.AccountingReferenceObject,
) []*accountingmapping.AccountingReferenceObject {
	out := make([]*accountingmapping.AccountingReferenceObject, 0, len(values))
	for idx := range values {
		out = append(out, convert(&values[idx]))
	}
	return out
}

func partyKindOf(kind accountingprovider.ReferenceKind) businesscentralref.PartyKind {
	if kind == accountingprovider.ReferenceKindVendor {
		return businesscentralref.PartyVendor
	}
	return businesscentralref.PartyCustomer
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
			return nil, bcsession.ErrItemDraftRequired
		}
		return c.createItem(ctx, client, req.Item)
	case accountingprovider.ReferenceKindCustomer, accountingprovider.ReferenceKindVendor:
		if req.Party == nil {
			return nil, bcsession.ErrPartyDraftRequired
		}
		party, createErr := CreateParty(ctx, client, partyKindOf(req.Kind), req.Party)
		if createErr != nil {
			return nil, createErr
		}
		return PartyObjectOf(req.Kind, party), nil
	case accountingprovider.ReferenceKindAccount,
		accountingprovider.ReferenceKindTerm,
		accountingprovider.ReferenceKindPaymentMethod:
		return nil, fmt.Errorf("businesscentral: %s records are not created from Trenova", req.Kind)
	default:
		return nil, fmt.Errorf("businesscentral: unknown reference kind %q", req.Kind)
	}
}

func (c *References) createItem(
	ctx context.Context,
	client *businesscentral.Client,
	draft *accountingconnectorports.AccountingItemDraft,
) (*accountingmapping.AccountingReferenceObject, error) {
	name := c.SanitizeName(accountingprovider.ReferenceKindItem, draft.Name)
	items, err := businesscentralref.Items(ctx, client, nil)
	if err != nil {
		return nil, err
	}
	for idx := range items {
		if strings.EqualFold(strings.TrimSpace(items[idx].DisplayName), name) {
			return nil, fmt.Errorf("%w: item %q", bcsession.ErrDuplicateName, name)
		}
	}
	created, err := businesscentralref.CreateItem(ctx, client, &businesscentralref.ItemInput{
		DisplayName: name,
		Type:        businesscentralref.ItemTypeService,
	})
	if err != nil {
		return nil, err
	}
	return ItemObjectOf(created), nil
}

func CreateParty(
	ctx context.Context,
	client *businesscentral.Client,
	kind businesscentralref.PartyKind,
	draft *accountingconnectorports.AccountingPartyDraft,
) (*businesscentralref.Party, error) {
	input := PartyInputOf(draft)
	if input.DisplayName == "" {
		return nil, businesscentralerrors.ErrNameRequired
	}
	existing, err := businesscentralref.FindParties(ctx, client, kind, input.DisplayName)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return nil, fmt.Errorf("%w: %q", bcsession.ErrDuplicateName, input.DisplayName)
	}
	return businesscentralref.CreateParty(ctx, client, kind, input)
}

func (c *References) IsDuplicateName(err error) bool {
	return errors.Is(err, bcsession.ErrDuplicateName) || businesscentralerrors.IsDuplicate(err)
}

func (c *References) SanitizeName(_ accountingprovider.ReferenceKind, name string) string {
	return CleanText(name, maxNameLength)
}

func PartyInputOf(
	draft *accountingconnectorports.AccountingPartyDraft,
) *businesscentralref.PartyInput {
	return &businesscentralref.PartyInput{
		DisplayName:  CleanText(draft.DisplayName, maxNameLength),
		Email:        CleanText(draft.Email, maxEmailLength),
		AddressLine1: CleanText(draft.AddressLine1, maxAddressLength),
		City:         CleanText(draft.City, maxCityLength),
		State:        CleanText(draft.State, maxStateLength),
		PostalCode:   CleanText(draft.PostalCode, maxPostalCodeLength),
		Country:      CleanText(draft.Country, maxCountryLength),
	}
}

func AccountObjectOf(
	account *businesscentralref.Account,
) *accountingmapping.AccountingReferenceObject {
	return &accountingmapping.AccountingReferenceObject{
		Kind:              accountingprovider.ReferenceKindAccount,
		ExternalID:        account.ID,
		Name:              account.DisplayName,
		Number:            account.Number,
		Classification:    account.Category,
		AccountType:       account.AccountType,
		AccountSubType:    account.SubCategory,
		AccountClass:      AccountClassOf(account),
		Active:            account.Postable(),
		ProviderUpdatedAt: unixOf(account.LastModified),
	}
}

func AccountClassOf(account *businesscentralref.Account) accountingmapping.AccountClass {
	sub := strings.ToLower(account.SubCategory)
	switch account.Category {
	case businesscentralref.CategoryIncome:
		return accountingmapping.AccountClassIncome
	case businesscentralref.CategoryCostOfGoodsSold:
		return accountingmapping.AccountClassCostOfSales
	case businesscentralref.CategoryExpense:
		return accountingmapping.AccountClassExpense
	case businesscentralref.CategoryEquity:
		return accountingmapping.AccountClassEquity
	case businesscentralref.CategoryAssets:
		switch {
		case strings.Contains(sub, "receivable"):
			return accountingmapping.AccountClassReceivable
		case strings.Contains(sub, "cash"), strings.Contains(sub, "bank"):
			return accountingmapping.AccountClassBank
		default:
			return accountingmapping.AccountClassAsset
		}
	case businesscentralref.CategoryLiabilities:
		if strings.Contains(sub, "payable") {
			return accountingmapping.AccountClassPayable
		}
		return accountingmapping.AccountClassLiability
	default:
		return ""
	}
}

func ItemObjectOf(item *businesscentralref.Item) *accountingmapping.AccountingReferenceObject {
	name := item.DisplayName
	if name == "" {
		name = item.Number
	}
	return &accountingmapping.AccountingReferenceObject{
		Kind:              accountingprovider.ReferenceKindItem,
		ExternalID:        item.ID,
		Name:              name,
		Number:            item.Number,
		ItemType:          item.Type,
		Active:            !item.Blocked,
		ProviderUpdatedAt: unixOf(item.LastModified),
	}
}

func partyConverter(
	kind accountingprovider.ReferenceKind,
) func(*businesscentralref.Party) *accountingmapping.AccountingReferenceObject {
	return func(party *businesscentralref.Party) *accountingmapping.AccountingReferenceObject {
		return PartyObjectOf(kind, party)
	}
}

func PartyObjectOf(
	kind accountingprovider.ReferenceKind,
	party *businesscentralref.Party,
) *accountingmapping.AccountingReferenceObject {
	return &accountingmapping.AccountingReferenceObject{
		Kind:              kind,
		ExternalID:        party.ID,
		Name:              party.DisplayName,
		Number:            party.Number,
		Active:            !party.IsBlocked(),
		CompanyName:       party.DisplayName,
		Email:             party.Email,
		AddressLine1:      party.AddressLine1,
		City:              party.City,
		State:             party.State,
		PostalCode:        party.PostalCode,
		CurrencyCode:      party.CurrencyCode,
		ProviderUpdatedAt: unixOf(party.LastModified),
	}
}

func termObjectOf(
	term *businesscentralref.PaymentTerm,
) *accountingmapping.AccountingReferenceObject {
	name := term.DisplayName
	if name == "" {
		name = term.Code
	}
	return &accountingmapping.AccountingReferenceObject{
		Kind:       accountingprovider.ReferenceKindTerm,
		ExternalID: term.ID,
		Name:       name,
		Number:     term.Code,
		Active:     true,
		DueDays:    dueDaysOf(term.DueDateCalculation),
	}
}

func dueDaysOf(calculation string) *int {
	value := strings.ToUpper(strings.TrimSpace(calculation))
	if !strings.HasSuffix(value, dueDaysUnit) {
		return nil
	}
	days, err := strconv.Atoi(strings.TrimSuffix(value, dueDaysUnit))
	if err != nil || days < 0 {
		return nil
	}
	return &days
}
