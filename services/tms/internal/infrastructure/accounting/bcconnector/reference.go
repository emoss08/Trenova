package bcconnector

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/businesscentral"
)

var (
	_ services.AccountingReferenceReader  = (*Connector)(nil)
	_ services.AccountingReferenceCreator = (*Connector)(nil)
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
	req *services.AccountingReferencePageRequest,
) (*services.AccountingReferencePage, error) {
	client, err := c.client(services.AccountingDocumentAuth{
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
	return &services.AccountingReferencePage{Objects: objects}, nil
}

func listReference(
	ctx context.Context,
	client *businesscentral.Client,
	kind accountingsync.ReferenceKind,
) ([]*accountingsync.AccountingReferenceObject, error) {
	switch kind {
	case accountingsync.ReferenceKindAccount:
		accounts, err := client.Accounts(ctx, nil)
		return convertAll(accounts, accountObjectOf), err
	case accountingsync.ReferenceKindItem:
		items, err := client.Items(ctx, nil)
		return convertAll(items, itemObjectOf), err
	case accountingsync.ReferenceKindCustomer, accountingsync.ReferenceKindVendor:
		partyKind := partyKindOf(kind)
		parties, err := client.Parties(ctx, partyKind, nil)
		return convertAll(parties, partyConverter(kind)), err
	case accountingsync.ReferenceKindTerm:
		terms, err := client.PaymentTerms(ctx)
		return convertAll(terms, termObjectOf), err
	case accountingsync.ReferenceKindPaymentMethod:
		return []*accountingsync.AccountingReferenceObject{}, nil
	default:
		return nil, fmt.Errorf("businesscentral: unknown reference kind %q", kind)
	}
}

func convertAll[T any](
	values []T,
	convert func(*T) *accountingsync.AccountingReferenceObject,
) []*accountingsync.AccountingReferenceObject {
	out := make([]*accountingsync.AccountingReferenceObject, 0, len(values))
	for idx := range values {
		out = append(out, convert(&values[idx]))
	}
	return out
}

func partyKindOf(kind accountingsync.ReferenceKind) businesscentral.PartyKind {
	if kind == accountingsync.ReferenceKindVendor {
		return businesscentral.PartyVendor
	}
	return businesscentral.PartyCustomer
}

func referenceKindOf(kind businesscentral.PartyKind) accountingsync.ReferenceKind {
	if kind == businesscentral.PartyVendor {
		return accountingsync.ReferenceKindVendor
	}
	return accountingsync.ReferenceKindCustomer
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
		return c.createItem(ctx, client, req.Item)
	case accountingsync.ReferenceKindCustomer, accountingsync.ReferenceKindVendor:
		if req.Party == nil {
			return nil, errPartyDraftRequired
		}
		party, createErr := createParty(ctx, client, partyKindOf(req.Kind), req.Party)
		if createErr != nil {
			return nil, createErr
		}
		return partyObjectOf(req.Kind, party), nil
	case accountingsync.ReferenceKindAccount,
		accountingsync.ReferenceKindTerm,
		accountingsync.ReferenceKindPaymentMethod:
		return nil, fmt.Errorf("businesscentral: %s records are not created from Trenova", req.Kind)
	default:
		return nil, fmt.Errorf("businesscentral: unknown reference kind %q", req.Kind)
	}
}

func (c *Connector) createItem(
	ctx context.Context,
	client *businesscentral.Client,
	draft *services.AccountingItemDraft,
) (*accountingsync.AccountingReferenceObject, error) {
	name := c.SanitizeName(accountingsync.ReferenceKindItem, draft.Name)
	items, err := client.Items(ctx, nil)
	if err != nil {
		return nil, err
	}
	for idx := range items {
		if strings.EqualFold(strings.TrimSpace(items[idx].DisplayName), name) {
			return nil, fmt.Errorf("%w: item %q", errDuplicateName, name)
		}
	}
	created, err := client.CreateItem(ctx, &businesscentral.ItemInput{
		DisplayName: name,
		Type:        businesscentral.ItemTypeService,
	})
	if err != nil {
		return nil, err
	}
	return itemObjectOf(created), nil
}

func createParty(
	ctx context.Context,
	client *businesscentral.Client,
	kind businesscentral.PartyKind,
	draft *services.AccountingPartyDraft,
) (*businesscentral.Party, error) {
	input := partyInputOf(draft)
	if input.DisplayName == "" {
		return nil, businesscentral.ErrNameRequired
	}
	existing, err := client.FindParties(ctx, kind, input.DisplayName)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return nil, fmt.Errorf("%w: %q", errDuplicateName, input.DisplayName)
	}
	return client.CreateParty(ctx, kind, input)
}

func (c *Connector) IsDuplicateName(err error) bool {
	return errors.Is(err, errDuplicateName) || businesscentral.IsDuplicate(err)
}

func (c *Connector) SanitizeName(_ accountingsync.ReferenceKind, name string) string {
	return cleanText(name, maxNameLength)
}

func partyInputOf(draft *services.AccountingPartyDraft) *businesscentral.PartyInput {
	return &businesscentral.PartyInput{
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
	auth       services.AccountingDocumentAuth
	kind       businesscentral.PartyKind
	externalID string
	party      *services.AccountingPartyDraft
}

func (c *Connector) upsertParty(
	ctx context.Context,
	write *partyWrite,
) (*services.AccountingDocumentResult, error) {
	client, err := c.client(write.auth)
	if err != nil {
		return nil, err
	}
	var saved *businesscentral.Party
	if id := strings.TrimSpace(write.externalID); id != "" {
		saved, err = updateParty(ctx, client, write.kind, id, write.party)
	} else {
		saved, err = createParty(ctx, client, write.kind, write.party)
	}
	if err != nil {
		return nil, err
	}
	return &services.AccountingDocumentResult{
		ExternalID: saved.ID,
		DocNumber:  saved.DisplayName,
		Refs:       map[string]string{},
	}, nil
}

func updateParty(
	ctx context.Context,
	client *businesscentral.Client,
	kind businesscentral.PartyKind,
	id string,
	draft *services.AccountingPartyDraft,
) (*businesscentral.Party, error) {
	current, err := client.Party(ctx, kind, id)
	if err != nil {
		return nil, err
	}
	input := partyInputOf(draft)
	input.CurrencyCode = current.CurrencyCode
	input.PaymentTermsID = current.PaymentTermsID
	return client.UpdateParty(ctx, &businesscentral.PartyUpdate{
		Kind:    kind,
		PartyID: current.ID,
		ETag:    current.ETag,
		Input:   *input,
	})
}

func accountObjectOf(account *businesscentral.Account) *accountingsync.AccountingReferenceObject {
	return &accountingsync.AccountingReferenceObject{
		Kind:              accountingsync.ReferenceKindAccount,
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

func AccountClassOf(account *businesscentral.Account) accountingsync.AccountClass {
	sub := strings.ToLower(account.SubCategory)
	switch account.Category {
	case businesscentral.CategoryIncome:
		return accountingsync.AccountClassIncome
	case businesscentral.CategoryCostOfGoodsSold:
		return accountingsync.AccountClassCostOfSales
	case businesscentral.CategoryExpense:
		return accountingsync.AccountClassExpense
	case businesscentral.CategoryEquity:
		return accountingsync.AccountClassEquity
	case businesscentral.CategoryAssets:
		switch {
		case strings.Contains(sub, "receivable"):
			return accountingsync.AccountClassReceivable
		case strings.Contains(sub, "cash"), strings.Contains(sub, "bank"):
			return accountingsync.AccountClassBank
		default:
			return accountingsync.AccountClassAsset
		}
	case businesscentral.CategoryLiabilities:
		if strings.Contains(sub, "payable") {
			return accountingsync.AccountClassPayable
		}
		return accountingsync.AccountClassLiability
	default:
		return ""
	}
}

func itemObjectOf(item *businesscentral.Item) *accountingsync.AccountingReferenceObject {
	name := item.DisplayName
	if name == "" {
		name = item.Number
	}
	return &accountingsync.AccountingReferenceObject{
		Kind:              accountingsync.ReferenceKindItem,
		ExternalID:        item.ID,
		Name:              name,
		Number:            item.Number,
		ItemType:          item.Type,
		Active:            !item.Blocked,
		ProviderUpdatedAt: unixOf(item.LastModified),
	}
}

func partyConverter(
	kind accountingsync.ReferenceKind,
) func(*businesscentral.Party) *accountingsync.AccountingReferenceObject {
	return func(party *businesscentral.Party) *accountingsync.AccountingReferenceObject {
		return partyObjectOf(kind, party)
	}
}

func partyObjectOf(
	kind accountingsync.ReferenceKind,
	party *businesscentral.Party,
) *accountingsync.AccountingReferenceObject {
	return &accountingsync.AccountingReferenceObject{
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

func termObjectOf(term *businesscentral.PaymentTerm) *accountingsync.AccountingReferenceObject {
	name := term.DisplayName
	if name == "" {
		name = term.Code
	}
	return &accountingsync.AccountingReferenceObject{
		Kind:       accountingsync.ReferenceKindTerm,
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
