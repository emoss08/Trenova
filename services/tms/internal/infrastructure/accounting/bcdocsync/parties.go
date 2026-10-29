package bcdocsync

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/bcrefsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/businesscentralref"
)

type partyWrite struct {
	auth       accountingconnectorports.AccountingDocumentAuth
	kind       businesscentralref.PartyKind
	externalID string
	party      *accountingconnectorports.AccountingPartyDraft
}

func (c *Documents) upsertParty(
	ctx context.Context,
	write *partyWrite,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	client, err := c.Client(write.auth)
	if err != nil {
		return nil, err
	}
	var saved *businesscentralref.Party
	if id := strings.TrimSpace(write.externalID); id != "" {
		saved, err = updateParty(ctx, client, write.kind, id, write.party)
	} else {
		saved, err = bcrefsync.CreateParty(ctx, client, write.kind, write.party)
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
	input := bcrefsync.PartyInputOf(draft)
	input.CurrencyCode = current.CurrencyCode
	input.PaymentTermsID = current.PaymentTermsID
	return businesscentralref.UpdateParty(ctx, client, &businesscentralref.PartyUpdate{
		Kind:    kind,
		PartyID: current.ID,
		ETag:    current.ETag,
		Input:   *input,
	})
}
