package resolver

import (
	"context"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/sliceutils"
)

// jurisdictionCountryRank keeps the picker in the order a dispatcher reads a fuel-tax
// return: the home country first, then the two it shares IFTA with.
var jurisdictionCountryRank = map[string]int{"US": 0, "CA": 1, "MX": 2}

// resolveIFTAJurisdictionSelectOptions pages in memory because the jurisdiction
// table is a fixed reference list of roughly seventy rows shared by every
// tenant, with no organization column to scope a query by.
func (r *Resolver) resolveIFTAJurisdictionSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	jurisdictions, err := r.iftaService.ListJurisdictions(
		ctx,
		&repositories.ListJurisdictionsRequest{
			MembersOnly: selectOptionBoolFilter(req.filters, "membersOnly"),
		},
	)
	if err != nil {
		return nil, err
	}

	sortJurisdictions(jurisdictions)

	if len(req.ids) > 0 {
		wanted := make(map[string]struct{}, len(req.ids))
		for _, id := range req.ids {
			wanted[id.String()] = struct{}{}
		}
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, jurisdiction := range jurisdictions {
			if _, ok := wanted[jurisdiction.ID.String()]; ok {
				items = append(items, iftaJurisdictionSelectOptionItem(jurisdiction))
			}
		}

		return selectOptionConnection(items, len(items), 0)
	}

	matches := matchingJurisdictions(jurisdictions, req.selectQuery.Query)
	offset := req.selectQuery.Pagination.SafeOffset()
	page := sliceutils.Page(matches, offset, req.selectQuery.Pagination.SafeLimit())

	items := make([]selectOptionConnectionItem, 0, len(page))
	for _, jurisdiction := range page {
		items = append(items, iftaJurisdictionSelectOptionItem(jurisdiction))
	}

	return selectOptionConnection(items, len(matches), offset)
}

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

func iftaJurisdictionSelectOptionItem(entity *ifta.Jurisdiction) selectOptionConnectionItem {
	description := entity.Name
	if !entity.IsIftaMember {
		description += " · Not an IFTA member"
	}

	return selectOptionConnectionItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Code + " — " + entity.Name,
			Description: &description,
			Meta: map[string]any{
				"code":         entity.Code,
				"name":         entity.Name,
				"countryCode":  entity.CountryCode,
				"isIftaMember": entity.IsIftaMember,
				"hasSurcharge": entity.HasSurcharge,
			},
		},
		entity.CreatedAt,
		entity.ID,
	)
}
