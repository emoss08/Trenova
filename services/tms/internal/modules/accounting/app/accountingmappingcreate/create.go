package accountingmappingcreate

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	RequestIDPrefix    = "trn-"
	requestIDHashChars = 40
)

func CreationRequestID(row *accountingmapping.AccountingMapping) string {
	sum := sha256.Sum256([]byte(
		row.ConnectionID.String() + "|" + row.ID.String() + "|" + strconv.FormatInt(
			row.Version,
			10,
		),
	))
	return RequestIDPrefix + hex.EncodeToString(sum[:])[:requestIDHashChars]
}

const maxReasonRunes = 300

func chosenName(
	creator accountingconnectorports.AccountingReferenceCreator,
	kind accountingprovider.ReferenceKind,
	override string,
	fallbacks ...string,
) (string, error) {
	for _, candidate := range append([]string{override}, fallbacks...) {
		if name := creator.SanitizeName(kind, candidate); name != "" {
			return name, nil
		}
	}
	return "", errortypes.NewValidationError("name", errortypes.ErrRequired, "A name is required")
}

func customerParty(
	cus *customer.Customer,
	name string,
) *accountingconnectorports.AccountingPartyDraft {
	return &accountingconnectorports.AccountingPartyDraft{
		DisplayName:  name,
		CompanyName:  cus.Name,
		AddressLine1: cus.AddressLine1,
		City:         cus.City,
		State:        stateCode(cus.State),
		PostalCode:   cus.PostalCode,
		Country:      stateCountry(cus.State),
	}
}

func carrierParty(
	carr *carrier.Carrier,
	name string,
) *accountingconnectorports.AccountingPartyDraft {
	draft := &accountingconnectorports.AccountingPartyDraft{
		DisplayName: name,
		CompanyName: carr.Name,
		Email:       strings.TrimSpace(carr.Email),
		Is1099:      carr.Is1099Eligible,
	}
	applyCarrierAddress(draft, carr)
	return draft
}

func driverParty(wrk *worker.Worker, name string) *accountingconnectorports.AccountingPartyDraft {
	return &accountingconnectorports.AccountingPartyDraft{
		DisplayName:  name,
		Email:        strings.TrimSpace(wrk.Email),
		AddressLine1: wrk.AddressLine1,
		City:         wrk.City,
		State:        stateCode(wrk.State),
		PostalCode:   wrk.PostalCode,
		Country:      stateCountry(wrk.State),
		Is1099:       true,
	}
}

func applyCarrierAddress(
	draft *accountingconnectorports.AccountingPartyDraft,
	carr *carrier.Carrier,
) {
	if strings.TrimSpace(carr.RemitAddressLine1) != "" {
		draft.AddressLine1 = carr.RemitAddressLine1
		draft.City = carr.RemitCity
		draft.PostalCode = carr.RemitPostalCode
		draft.State = stateCode(carr.RemitState)
		draft.Country = stateCountry(carr.RemitState)
		return
	}
	draft.AddressLine1 = carr.AddressLine1
	draft.City = carr.City
	draft.PostalCode = carr.PostalCode
	draft.State = stateCode(carr.State)
	draft.Country = stateCountry(carr.State)
}

func stateCode(state *usstate.UsState) string {
	if state == nil {
		return ""
	}
	return state.Abbreviation
}

func stateCountry(state *usstate.UsState) string {
	if state == nil {
		return ""
	}
	return state.CountryIso3
}
