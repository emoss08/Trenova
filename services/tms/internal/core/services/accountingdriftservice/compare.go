package accountingdriftservice

import (
	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/money"
)

const (
	providerStatePosted  = "Posted"
	providerStateVoided  = "Voided"
	providerStateDeleted = "Deleted"
)

type comparison struct {
	record   *accountingsync.AccountingSyncRecord
	trenova  *repositories.AccountingDriftState
	provider *services.AccountingDocumentState
}

func comparesAmount(objectType accountingsync.SyncObjectType) bool {
	return objectType != accountingsync.SyncObjectCreditApplication
}

func (c *comparison) found() bool {
	return c.provider != nil && c.provider.Found
}

func (c *comparison) providerVoided() bool {
	return c.found() && c.provider.Voided
}

func (c *comparison) trenovaMinor() int64 {
	return c.trenova.AmountMinor + c.trenova.ReflectedMinor
}

func (c *comparison) providerMinor() int64 {
	if !c.found() {
		return 0
	}
	return money.MinorUnits(c.provider.Total.Abs())
}

func (c *comparison) providerState() string {
	switch {
	case !c.found():
		return providerStateDeleted
	case c.provider.Voided:
		return providerStateVoided
	default:
		return providerStatePosted
	}
}

func (c *comparison) kind() (accountingsync.DriftKind, bool) {
	trenovaVoided := c.trenova.Voided
	switch {
	case !c.found():
		return accountingsync.DriftDeletedInProvider, !trenovaVoided
	case c.providerVoided():
		return accountingsync.DriftVoidedInProvider, !trenovaVoided
	case trenovaVoided:
		return accountingsync.DriftStatusMismatch, true
	case comparesAmount(c.record.ObjectType) && c.trenovaMinor() != c.providerMinor():
		return accountingsync.DriftAmountMismatch, true
	default:
		return "", false
	}
}

func (c *comparison) observation(
	base *accountingsync.DriftObservation,
) *accountingsync.DriftObservation {
	kind, differs := c.kind()
	if !differs {
		return nil
	}
	obs := *base
	obs.ObjectType = c.record.ObjectType
	obs.ObjectID = c.record.ObjectID
	obs.ObjectNumber = c.trenova.Number
	obs.PartyID = c.trenova.PartyID
	obs.PartyName = c.trenova.PartyName
	obs.ExternalID = c.record.ExternalID
	obs.Kind = kind
	obs.CurrencyCode = c.trenova.CurrencyCode
	obs.TrenovaState = c.trenova.State
	obs.ProviderState = c.providerState()
	if comparesAmount(c.record.ObjectType) {
		trenovaMinor := c.trenovaMinor()
		obs.TrenovaMinor = &trenovaMinor
		if c.found() {
			providerMinor := c.providerMinor()
			obs.ProviderMinor = &providerMinor
		}
	}
	if c.found() {
		if c.provider.ModifiedAt > 0 {
			modifiedAt := c.provider.ModifiedAt
			obs.ProviderModifiedAt = &modifiedAt
		}
		obs.ProviderModifiedBy = c.provider.ModifiedBy
	}
	return &obs
}
