package settlementselectoptions

import (
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// jurisdictionCountryRank keeps the picker in the order a dispatcher reads a fuel-tax
// return: the home country first, then the two it shares IFTA with.
var jurisdictionCountryRank = map[string]int{"US": 0, "CA": 1, "MX": 2}

func sortJurisdictions(jurisdictions []*ifta.Jurisdiction) {
	sort.SliceStable(jurisdictions, func(i, j int) bool {
		left, right := jurisdictions[i], jurisdictions[j]
		leftRank := jurisdictionCountryRank[left.CountryCode]
		rightRank := jurisdictionCountryRank[right.CountryCode]
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		if left.SortOrder != right.SortOrder {
			return left.SortOrder < right.SortOrder
		}
		return left.Code < right.Code
	})
}

func matchingJurisdictions(
	jurisdictions []*ifta.Jurisdiction,
	query string,
) []*ifta.Jurisdiction {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return jurisdictions
	}

	matches := make([]*ifta.Jurisdiction, 0, len(jurisdictions))
	for _, jurisdiction := range jurisdictions {
		if strings.Contains(strings.ToLower(jurisdiction.Code), needle) ||
			strings.Contains(strings.ToLower(jurisdiction.Name), needle) {
			matches = append(matches, jurisdiction)
		}
	}

	return matches
}

func iftaJurisdictionSelectOptionItem(entity *ifta.Jurisdiction) selectoption.Item {
	description := entity.Name
	if !entity.IsIftaMember {
		description += " · Not an IFTA member"
	}

	return selectoption.ItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Code + " — " + entity.Name,
			Description: &description,
			Meta: map[string]any{
				selectoption.MetaKeyCode: entity.Code,
				"name":                   entity.Name,
				"countryCode":            entity.CountryCode,
				"isIftaMember":           entity.IsIftaMember,
				"hasSurcharge":           entity.HasSurcharge,
			},
		},
		entity.CreatedAt,
		entity.ID,
	)
}

func matchingIFTAFuelTypes(query string, iftaOnly bool) []domaintypes.IFTAFuelType {
	needle := strings.ToLower(strings.TrimSpace(query))
	fuelTypes := domaintypes.AllIFTAFuelTypes()

	matches := make([]domaintypes.IFTAFuelType, 0, len(fuelTypes))
	for _, fuelType := range fuelTypes {
		if iftaOnly && !fuelType.CountsForIFTA() {
			continue
		}
		if needle != "" && !iftaFuelTypeMatches(fuelType, needle) {
			continue
		}
		matches = append(matches, fuelType)
	}

	return matches
}

func iftaFuelTypeMatches(fuelType domaintypes.IFTAFuelType, needle string) bool {
	return strings.Contains(strings.ToLower(fuelType.String()), needle) ||
		strings.Contains(strings.ToLower(fuelType.Label()), needle)
}

func iftaFuelTypeID(fuelType domaintypes.IFTAFuelType) pulid.ID {
	return pulid.ID(fuelType.String())
}

func iftaFuelTypeSelectOptionItem(
	fuelType domaintypes.IFTAFuelType,
) selectoption.Item {
	return selectoption.ItemFor(
		iftaFuelTypeSelectOption(fuelType),
		0,
		iftaFuelTypeID(fuelType),
	)
}

func iftaFuelTypeSelectOption(fuelType domaintypes.IFTAFuelType) *gqlmodel.SelectOption {
	description := iftaFuelTypeDescription(fuelType)
	return &gqlmodel.SelectOption{
		ID:          fuelType.String(),
		Label:       fuelType.Label(),
		Description: &description,
		Meta: map[string]any{
			"countsForIfta": fuelType.CountsForIFTA(),
			"gaseous":       fuelType.IsGaseous(),
		},
	}
}

func iftaFuelTypeDescription(fuelType domaintypes.IFTAFuelType) string {
	switch {
	case !fuelType.CountsForIFTA():
		return "Not reported on the quarterly IFTA return"
	case fuelType.IsGaseous():
		return "Reported on the quarterly IFTA return in gallon equivalents"
	default:
		return "Reported on the quarterly IFTA return"
	}
}
