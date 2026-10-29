package xerosession

import (
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/xeroref"
)

var (
	ErrItemDraftRequired  = errors.New("xero: an item draft is required")
	ErrPartyDraftRequired = errors.New("xero: a customer or supplier draft is required")
)

func ContactInputOf(draft *accountingconnectorports.AccountingPartyDraft) xeroref.ContactInput {
	input := xeroref.ContactInput{
		Name: strings.TrimSpace(
			stringutils.OneLine(draft.DisplayName, xeroref.MaxContactNameLength),
		),
		EmailAddress: strings.TrimSpace(draft.Email),
	}
	address := xeroref.Address{
		AddressType:  xeroref.AddressTypeStreet,
		AddressLine1: strings.TrimSpace(draft.AddressLine1),
		City:         strings.TrimSpace(draft.City),
		Region:       strings.TrimSpace(draft.State),
		PostalCode:   strings.TrimSpace(draft.PostalCode),
		Country:      strings.TrimSpace(draft.Country),
	}
	if address.AddressLine1 != "" || address.City != "" || address.Region != "" ||
		address.PostalCode != "" || address.Country != "" {
		input.Addresses = []xeroref.Address{address}
	}
	return input
}
