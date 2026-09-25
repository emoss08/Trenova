package qboconnector

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/quickbooks"
)

var (
	_ services.AccountingReferenceReader  = (*Connector)(nil)
	_ services.AccountingReferenceCreator = (*Connector)(nil)
)

func (c *Connector) MaxReferencePageSize() int {
	return quickbooks.MaxQueryResults
}

func (c *Connector) ListReference(
	ctx context.Context,
	req *services.AccountingReferencePageRequest,
) (*services.AccountingReferencePage, error) {
	kind, err := providerKind(req.Kind)
	if err != nil {
		return nil, err
	}
	client, err := quickbooks.New(c.env, req.RealmID, req.AccessToken, c.apiOpts...)
	if err != nil {
		return nil, err
	}

	page, err := client.ListReference(ctx, kind, req.StartPosition, req.PageSize)
	if err != nil {
		return nil, err
	}

	out := &services.AccountingReferencePage{
		Objects:   make([]*accountingsync.AccountingReferenceObject, 0, len(page.Objects)),
		NextStart: page.NextStart,
	}
	for idx := range page.Objects {
		out.Objects = append(out.Objects, referenceObjectOf(req.Kind, &page.Objects[idx]))
	}
	return out, nil
}

func (c *Connector) CreateReference(
	ctx context.Context,
	req *services.AccountingCreateReferenceRequest,
) (*accountingsync.AccountingReferenceObject, error) {
	client, err := quickbooks.New(c.env, req.RealmID, req.AccessToken, c.apiOpts...)
	if err != nil {
		return nil, err
	}

	var created *quickbooks.ReferenceObject
	switch req.Kind {
	case accountingsync.ReferenceKindItem:
		if req.Item == nil {
			return nil, errors.New("quickbooks: an item draft is required")
		}
		created, err = client.CreateItem(ctx, req.RequestID, quickbooks.ItemDraft{
			Name:            req.Item.Name,
			Description:     req.Item.Description,
			Sku:             req.Item.Sku,
			IncomeAccountID: req.Item.IncomeAccountID,
		})
	case accountingsync.ReferenceKindCustomer:
		created, err = client.CreateCustomer(ctx, req.RequestID, partyDraftOf(req.Party))
	case accountingsync.ReferenceKindVendor:
		created, err = client.CreateVendor(ctx, req.RequestID, partyDraftOf(req.Party))
	case accountingsync.ReferenceKindAccount,
		accountingsync.ReferenceKindTerm,
		accountingsync.ReferenceKindPaymentMethod:
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
	return quickbooks.IsDuplicateName(err)
}

func (c *Connector) SanitizeName(kind accountingsync.ReferenceKind, name string) string {
	if kind == accountingsync.ReferenceKindItem {
		return quickbooks.SanitizeName(name, quickbooks.MaxItemNameLength)
	}
	return quickbooks.SanitizeName(name, quickbooks.MaxPartyNameLength)
}

func partyDraftOf(draft *services.AccountingPartyDraft) *quickbooks.PartyDraft {
	if draft == nil {
		return nil
	}
	return &quickbooks.PartyDraft{
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

func providerKind(kind accountingsync.ReferenceKind) (quickbooks.ReferenceKind, error) {
	switch kind {
	case accountingsync.ReferenceKindAccount:
		return quickbooks.KindAccount, nil
	case accountingsync.ReferenceKindItem:
		return quickbooks.KindItem, nil
	case accountingsync.ReferenceKindCustomer:
		return quickbooks.KindCustomer, nil
	case accountingsync.ReferenceKindVendor:
		return quickbooks.KindVendor, nil
	case accountingsync.ReferenceKindTerm:
		return quickbooks.KindTerm, nil
	case accountingsync.ReferenceKindPaymentMethod:
		return quickbooks.KindPaymentMethod, nil
	default:
		return "", fmt.Errorf("quickbooks: unknown reference kind %q", kind)
	}
}

func referenceObjectOf(
	kind accountingsync.ReferenceKind,
	obj *quickbooks.ReferenceObject,
) *accountingsync.AccountingReferenceObject {
	out := &accountingsync.AccountingReferenceObject{
		Kind:                    kind,
		ExternalID:              obj.ID,
		Name:                    obj.Name,
		FullyQualifiedName:      obj.FullyQualifiedName,
		Number:                  obj.Number,
		Description:             obj.Description,
		Classification:          obj.Classification,
		AccountType:             obj.AccountType,
		AccountSubType:          obj.AccountSubType,
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
	case accountingsync.ReferenceKindTerm:
		out.SubType = obj.TermType
	case accountingsync.ReferenceKindPaymentMethod:
		out.SubType = obj.PaymentMethodType
	case accountingsync.ReferenceKindAccount,
		accountingsync.ReferenceKindItem,
		accountingsync.ReferenceKindCustomer,
		accountingsync.ReferenceKindVendor:
	}
	if obj.LastUpdatedAt > 0 {
		updated := obj.LastUpdatedAt
		out.ProviderUpdatedAt = &updated
	}
	return out
}
