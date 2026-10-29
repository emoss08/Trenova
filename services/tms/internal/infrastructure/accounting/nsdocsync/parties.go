package nsdocsync

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsrefsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/netsuite"
)

const (
	sublistAddressBook = "addressBook"
)

type partyWrite struct {
	auth       accountingconnectorports.AccountingDocumentAuth
	kind       nsrefsync.PartyKind
	requestID  string
	externalID string
	party      *accountingconnectorports.AccountingPartyDraft
}

func (c *Documents) upsertParty(
	ctx context.Context,
	w *partyWrite,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	s, err := c.Open(w.auth)
	if err != nil {
		return nil, err
	}
	body := nsrefsync.PartyBodyOf(w.kind, w.party)
	if body.CompanyName == "" {
		return nil, nsbinding.ErrNameRequired
	}
	id := strings.TrimSpace(w.externalID)
	if id != "" {
		var replace []string
		if body.AddressBook != nil {
			replace = []string{sublistAddressBook}
		}
		if err = s.Client.Patch(ctx, netsuite.RecordPatch{
			Type:    w.kind.recordType(),
			ID:      id,
			Body:    body,
			Replace: replace,
		}); err != nil {
			return nil, err
		}
	} else if id, err = nsrefsync.CreateParty(ctx, s, &nsrefsync.PartyCreate{
		kind:      w.kind,
		requestID: w.requestID,
		draft:     w.party,
	}); err != nil {
		return nil, err
	}
	return &accountingconnectorports.AccountingDocumentResult{
		ExternalID: id,
		DocNumber:  body.CompanyName,
		Refs:       map[string]string{},
	}, nil
}
