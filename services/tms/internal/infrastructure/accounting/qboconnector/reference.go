package qboconnector

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/emoss08/trenova/shared/quickbooksref"
)

var (
	_ accountingconnectorports.AccountingReferenceReader  = (*Connector)(nil)
	_ accountingconnectorports.AccountingReferenceCreator = (*Connector)(nil)
)

func (c *Connector) MaxReferencePageSize() int {
	return quickbooksref.MaxQueryResults
}

func (c *Connector) ListReference(
	ctx context.Context,
	req *accountingconnectorports.AccountingReferencePageRequest,
) (*accountingconnectorports.AccountingReferencePage, error) {
	kind, err := providerKind(req.Kind)
	if err != nil {
		return nil, err
	}
	client, err := quickbooks.New(c.env, req.RealmID, req.AccessToken, c.apiOpts...)
	if err != nil {
		return nil, err
	}

	page, err := quickbooksref.ListReference(ctx, client, kind, req.StartPosition, req.PageSize)
	if err != nil {
		return nil, err
	}

	out := &accountingconnectorports.AccountingReferencePage{
		Objects:   make([]*accountingmapping.AccountingReferenceObject, 0, len(page.Objects)),
		NextStart: page.NextStart,
	}
	for idx := range page.Objects {
		out.Objects = append(out.Objects, referenceObjectOf(req.Kind, &page.Objects[idx]))
	}
	return out, nil
}

func (c *Connector) CreateReference(
	ctx context.Context,
	req *accountingconnectorports.AccountingCreateReferenceRequest,
) (*accountingmapping.AccountingReferenceObject, error) {
	client, err := quickbooks.New(c.env, req.RealmID, req.AccessToken, c.apiOpts...)
	if err != nil {
		return nil, err
	}

	var created *quickbooksref.ReferenceObject
	switch req.Kind {
	case accountingprovider.ReferenceKindItem:
		if req.Item == nil {
			return nil, errors.New("quickbooks: an item draft is required")
		}
		created, err = quickbooksref.CreateItem(ctx, client, req.RequestID, quickbooksref.ItemDraft{
			Name:            req.Item.Name,
			Description:     req.Item.Description,
			Sku:             req.Item.Sku,
			IncomeAccountID: req.Item.IncomeAccountID,
		})
	case accountingprovider.ReferenceKindCustomer:
		created, err = quickbooksref.CreateCustomer(
			ctx,
			client,
			req.RequestID,
			partyDraftOf(req.Party),
		)
	case accountingprovider.ReferenceKindVendor:
		created, err = quickbooksref.CreateVendor(
			ctx,
			client,
			req.RequestID,
			partyDraftOf(req.Party),
		)
	case accountingprovider.ReferenceKindAccount,
		accountingprovider.ReferenceKindTerm,
		accountingprovider.ReferenceKindPaymentMethod:
		return nil, fmt.Errorf("quickbooks: %s records are not created from Trenova", req.Kind)
	default:
		return nil, fmt.Errorf("quickbooks: unknown reference kind %q", req.Kind)
	}
	if err != nil {
		return nil, err
	}

	return referenceObjectOf(req.Kind, created), nil
}

func (c *Connector) IsDuplicateName(err error) bool {
	return quickbooksref.IsDuplicateName(err)
}

func (c *Connector) SanitizeName(kind accountingprovider.ReferenceKind, name string) string {
	if kind == accountingprovider.ReferenceKindItem {
		return quickbooksref.SanitizeName(name, quickbooksref.MaxItemNameLength)
	}
	return quickbooksref.SanitizeName(name, quickbooksref.MaxPartyNameLength)
}

func partyDraftOf(draft *accountingconnectorports.AccountingPartyDraft) *quickbooksref.PartyDraft {
	if draft == nil {
		return nil
	}
	return &quickbooksref.PartyDraft{
		DisplayName:  draft.DisplayName,
		CompanyName:  draft.CompanyName,
		Email:        draft.Email,
		AddressLine1: draft.AddressLine1,
		City:         draft.City,
		State:        draft.State,
		PostalCode:   draft.PostalCode,
		Country:      draft.Country,
		Is1099:       draft.Is1099,
	}
}

func providerKind(kind accountingprovider.ReferenceKind) (quickbooksref.ReferenceKind, error) {
	switch kind {
	case accountingprovider.ReferenceKindAccount:
		return quickbooksref.KindAccount, nil
	case accountingprovider.ReferenceKindItem:
		return quickbooksref.KindItem, nil
	case accountingprovider.ReferenceKindCustomer:
		return quickbooksref.KindCustomer, nil
	case accountingprovider.ReferenceKindVendor:
		return quickbooksref.KindVendor, nil
	case accountingprovider.ReferenceKindTerm:
		return quickbooksref.KindTerm, nil
	case accountingprovider.ReferenceKindPaymentMethod:
		return quickbooksref.KindPaymentMethod, nil
	default:
		return "", fmt.Errorf("quickbooks: unknown reference kind %q", kind)
	}
}

func referenceObjectOf(
	kind accountingprovider.ReferenceKind,
	obj *quickbooksref.ReferenceObject,
) *accountingmapping.AccountingReferenceObject {
	out := &accountingmapping.AccountingReferenceObject{
		Kind:                    kind,
		ExternalID:              obj.ID,
		Name:                    obj.Name,
		FullyQualifiedName:      obj.FullyQualifiedName,
		Number:                  obj.Number,
		Description:             obj.Description,
		Classification:          obj.Classification,
		AccountType:             obj.AccountType,
		AccountSubType:          obj.AccountSubType,
		AccountClass:            AccountClassOf(obj.AccountType, obj.AccountSubType),
		ItemType:                obj.ItemType,
		ParentExternalID:        obj.ParentID,
		Active:                  obj.Active,
		CurrencyCode:            obj.CurrencyCode,
		SyncToken:               obj.SyncToken,
		CompanyName:             obj.CompanyName,
		Email:                   obj.Email,
		AddressLine1:            obj.AddressLine1,
		City:                    obj.City,
		State:                   obj.State,
		PostalCode:              obj.PostalCode,
		IncomeAccountExternalID: obj.IncomeAccountID,
		DueDays:                 obj.DueDays,
		Is1099:                  obj.Is1099,
	}
	switch kind {
	case accountingprovider.ReferenceKindTerm:
		out.SubType = obj.TermType
	case accountingprovider.ReferenceKindPaymentMethod:
		out.SubType = obj.PaymentMethodType
	case accountingprovider.ReferenceKindAccount,
		accountingprovider.ReferenceKindItem,
		accountingprovider.ReferenceKindCustomer,
		accountingprovider.ReferenceKindVendor:
	}
	if obj.LastUpdatedAt > 0 {
		updated := obj.LastUpdatedAt
		out.ProviderUpdatedAt = &updated
	}
	return out
}

const undepositedFundsSubType = "UndepositedFunds"

var accountClasses = map[string]accountingmapping.AccountClass{
	"Accounts Receivable":     accountingmapping.AccountClassReceivable,
	"Accounts Payable":        accountingmapping.AccountClassPayable,
	"Bank":                    accountingmapping.AccountClassBank,
	"Income":                  accountingmapping.AccountClassIncome,
	"Other Income":            accountingmapping.AccountClassOtherIncome,
	"Cost of Goods Sold":      accountingmapping.AccountClassCostOfSales,
	"Expense":                 accountingmapping.AccountClassExpense,
	"Other Expense":           accountingmapping.AccountClassOtherExpense,
	"Other Current Asset":     accountingmapping.AccountClassAsset,
	"Fixed Asset":             accountingmapping.AccountClassAsset,
	"Other Asset":             accountingmapping.AccountClassAsset,
	"Credit Card":             accountingmapping.AccountClassLiability,
	"Other Current Liability": accountingmapping.AccountClassLiability,
	"Long Term Liability":     accountingmapping.AccountClassLiability,
	"Equity":                  accountingmapping.AccountClassEquity,
}

func AccountClassOf(accountType, subType string) accountingmapping.AccountClass {
	if subType == undepositedFundsSubType {
		return accountingmapping.AccountClassUndepositedFunds
	}
	return accountClasses[accountType]
}
