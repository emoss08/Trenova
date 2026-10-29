package bcconnector

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/businesscentralerrors"
	"github.com/emoss08/trenova/shared/businesscentralref"
)

var (
	_ accountingconnectorports.AccountingReferenceReader  = (*Connector)(nil)
	_ accountingconnectorports.AccountingReferenceCreator = (*Connector)(nil)
)

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

var (
	errItemDraftRequired  = errors.New("businesscentral: an item draft is required")
	errPartyDraftRequired = errors.New("businesscentral: a customer or vendor draft is required")
)

func (c *Connector) MaxReferencePageSize() int {
	return businesscentral.MaxPageSize
}

func (c *Connector) ListReference(
	ctx context.Context,
	req *accountingconnectorports.AccountingReferencePageRequest,
) (*accountingconnectorports.AccountingReferencePage, error) {
	client, err := c.client(accountingconnectorports.AccountingDocumentAuth{
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
		return convertAll(accounts, accountObjectOf), err
	case accountingprovider.ReferenceKindItem:
		items, err := businesscentralref.Items(ctx, client, nil)
		return convertAll(items, itemObjectOf), err
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

func referenceKindOf(kind businesscentralref.PartyKind) accountingprovider.ReferenceKind {
	if kind == businesscentralref.PartyVendor {
		return accountingprovider.ReferenceKindVendor
	}
	return accountingprovider.ReferenceKindCustomer
}

func (c *Connector) CreateReference(
	ctx context.Context,
	req *accountingconnectorports.AccountingCreateReferenceRequest,
) (*accountingmapping.AccountingReferenceObject, error) {
	client, err := c.client(accountingconnectorports.AccountingDocumentAuth{
		RealmID:     req.RealmID,
		AccessToken: req.AccessToken,
	})
	if err != nil {
		return nil, err
	}
	switch req.Kind {
	case accountingprovider.ReferenceKindItem:
		if req.Item == nil {
			return nil, errItemDraftRequired
		}
		return c.createItem(ctx, client, req.Item)
	case accountingprovider.ReferenceKindCustomer, accountingprovider.ReferenceKindVendor:
		if req.Party == nil {
			return nil, errPartyDraftRequired
		}
		party, createErr := createParty(ctx, client, partyKindOf(req.Kind), req.Party)
		if createErr != nil {
			return nil, createErr
		}
		return partyObjectOf(req.Kind, party), nil
	case accountingprovider.ReferenceKindAccount,
		accountingprovider.ReferenceKindTerm,
		accountingprovider.ReferenceKindPaymentMethod:
		return nil, fmt.Errorf("businesscentral: %s records are not created from Trenova", req.Kind)
	default:
		return nil, fmt.Errorf("businesscentral: unknown reference kind %q", req.Kind)
	}
}

func (c *Connector) createItem(
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
			return nil, fmt.Errorf("%w: item %q", errDuplicateName, name)
		}
	}
	created, err := businesscentralref.CreateItem(ctx, client, &businesscentralref.ItemInput{
		DisplayName: name,
		Type:        businesscentralref.ItemTypeService,
	})
	if err != nil {
		return nil, err
	}
	return itemObjectOf(created), nil
}

func createParty(
	ctx context.Context,
	client *businesscentral.Client,
	kind businesscentralref.PartyKind,
	draft *accountingconnectorports.AccountingPartyDraft,
) (*businesscentralref.Party, error) {
	input := partyInputOf(draft)
	if input.DisplayName == "" {
		return nil, businesscentralerrors.ErrNameRequired
	}
	existing, err := businesscentralref.FindParties(ctx, client, kind, input.DisplayName)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return nil, fmt.Errorf("%w: %q", errDuplicateName, input.DisplayName)
	}
	return businesscentralref.CreateParty(ctx, client, kind, input)
}

func (c *Connector) IsDuplicateName(err error) bool {
	return errors.Is(err, errDuplicateName) || businesscentralerrors.IsDuplicate(err)
}

func (c *Connector) SanitizeName(_ accountingprovider.ReferenceKind, name string) string {
	return cleanText(name, maxNameLength)
}

func partyInputOf(
	draft *accountingconnectorports.AccountingPartyDraft,
) *businesscentralref.PartyInput {
	return &businesscentralref.PartyInput{
		DisplayName:  cleanText(draft.DisplayName, maxNameLength),
		Email:        cleanText(draft.Email, maxEmailLength),
		AddressLine1: cleanText(draft.AddressLine1, maxAddressLength),
		City:         cleanText(draft.City, maxCityLength),
		State:        cleanText(draft.State, maxStateLength),
		PostalCode:   cleanText(draft.PostalCode, maxPostalCodeLength),
		Country:      cleanText(draft.Country, maxCountryLength),
	}
}

type partyWrite struct {
	auth       accountingconnectorports.AccountingDocumentAuth
	kind       businesscentralref.PartyKind
	externalID string
	party      *accountingconnectorports.AccountingPartyDraft
}

func (c *Connector) upsertParty(
	ctx context.Context,
	write *partyWrite,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	client, err := c.client(write.auth)
	if err != nil {
		return nil, err
	}
	var saved *businesscentralref.Party
	if id := strings.TrimSpace(write.externalID); id != "" {
		saved, err = updateParty(ctx, client, write.kind, id, write.party)
	} else {
		saved, err = createParty(ctx, client, write.kind, write.party)
	}
	if err != nil {
		return nil, err
	}
	return &accountingconnectorports.AccountingDocumentResult{
		ExternalID: saved.ID,
		DocNumber:  saved.DisplayName,
		Refs:       map[string]string{},
	}, nil
}

func updateParty(
	ctx context.Context,
	client *businesscentral.Client,
	kind businesscentralref.PartyKind,
	id string,
	draft *accountingconnectorports.AccountingPartyDraft,
) (*businesscentralref.Party, error) {
	current, err := businesscentralref.ReadParty(ctx, client, kind, id)
	if err != nil {
		return nil, err
	}
	input := partyInputOf(draft)
	input.CurrencyCode = current.CurrencyCode
	input.PaymentTermsID = current.PaymentTermsID
	return businesscentralref.UpdateParty(ctx, client, &businesscentralref.PartyUpdate{
		Kind:    kind,
		PartyID: current.ID,
		ETag:    current.ETag,
		Input:   *input,
	})
}

func accountObjectOf(
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

func itemObjectOf(item *businesscentralref.Item) *accountingmapping.AccountingReferenceObject {
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
		return partyObjectOf(kind, party)
	}
}

func partyObjectOf(
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
