package integration

import "github.com/emoss08/trenova/internal/core/domain/fuelpurchase"

// SupportsFuelCards reports whether the integration type is a fuel card feed the
// platform can actually drive. It is the single source of truth shared by the
// connector registry and the sync scheduler, so the two can never disagree about
// which organizations have a live feed.
func (t Type) SupportsFuelCards() bool {
	switch t {
	case TypeWEXFuel, TypeComdataFuel, TypeRampFuel:
		return true
	default:
		return false
	}
}

// CardProviders lists the card brands an integration type can carry. WEX and EFS
// are two brands on one network, so a single WEX connection may hold either.
func (t Type) CardProviders() []fuelpurchase.CardProvider {
	switch t {
	case TypeWEXFuel:
		return []fuelpurchase.CardProvider{
			fuelpurchase.CardProviderWEX,
			fuelpurchase.CardProviderEFS,
		}
	case TypeComdataFuel:
		return []fuelpurchase.CardProvider{fuelpurchase.CardProviderComdata}
	case TypeRampFuel:
		return []fuelpurchase.CardProvider{fuelpurchase.CardProviderRamp}
	default:
		return nil
	}
}

// HasEnabledFuelCards reports whether any of the organization's integrations is
// an enabled, supported fuel card feed.
func HasEnabledFuelCards(records []*Integration) bool {
	for _, record := range records {
		if record == nil {
			continue
		}
		if record.Category != CategoryFuelCards || !record.Enabled {
			continue
		}
		if record.Type.SupportsFuelCards() {
			return true
		}
	}

	return false
}
