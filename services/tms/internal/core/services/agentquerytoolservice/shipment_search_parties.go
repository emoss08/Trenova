package agentquerytoolservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	partyMatchLimit = 10
	partyTokenLimit = 4
	partyTokenMin   = 3
)

var partyFillerWords = map[string]struct{}{
	"and": {}, "any": {}, "driver": {}, "drivers": {}, "driving": {}, "for": {}, "from": {},
	"going": {}, "has": {}, "headed": {}, "heading": {}, "hes": {}, "him": {}, "her": {},
	"shes": {}, "she": {},
	"into": {}, "its": {}, "load": {}, "loads": {}, "one": {}, "our": {}, "out": {},
	"shipment": {}, "shipments": {}, "that": {}, "thats": {}, "the": {}, "this": {},
	"truck": {}, "with": {}, "where": {}, "wheres": {}, "which": {},
}

type partyMatch struct {
	term string
	id   pulid.ID
	name string
}

type shipmentPartySources struct {
	customers   repositories.CustomerRepository
	locations   repositories.LocationRepository
	workers     repositories.WorkerRepository
	commodities repositories.CommodityRepository
	types       repositories.ShipmentTypeRepository
}

func (s shipmentPartySources) none() bool {
	return s.customers == nil && s.locations == nil && s.workers == nil &&
		s.commodities == nil && s.types == nil
}

type shipmentParties struct {
	customers   []partyMatch
	locations   []partyMatch
	workers     []partyMatch
	commodities []partyMatch
	types       []partyMatch
}

func (p *shipmentParties) empty() bool {
	return len(p.kinds()) == 0
}

func (p *shipmentParties) kinds() [][]partyMatch {
	kinds := make([][]partyMatch, 0, 5)
	for _, matches := range [][]partyMatch{
		p.customers, p.locations, p.workers, p.commodities, p.types,
	} {
		if len(matches) > 0 {
			kinds = append(kinds, matches)
		}
	}

	return kinds
}

func (p *shipmentParties) allFromDifferentWords() bool {
	kinds := p.kinds()
	if len(kinds) < 2 {
		return false
	}
	for idx, matches := range kinds {
		for _, other := range kinds[idx+1:] {
			for _, match := range matches {
				for _, candidate := range other {
					if match.term == candidate.term {
						return false
					}
				}
			}
		}
	}

	return true
}

// onlyPlacesFromDifferentWords reports a query naming two or more places and
// nothing else, such as "denver phoenix": a load through both, not either.
func (p *shipmentParties) onlyPlacesFromDifferentWords() bool {
	return len(p.kinds()) == 1 && len(partiesByTerm(p.locations)) > 1
}

// partiesByTerm groups matches by the word that found them, in the order the
// words came. "peak denver phoenix" names two places, and a load must stop at
// both: matching either filled the page with Peak's newest loads into Denver
// and left out the one load from Denver to Phoenix.
func partiesByTerm(matches []partyMatch) [][]partyMatch {
	groups := make([][]partyMatch, 0, partyTokenLimit)
	for _, match := range matches {
		idx := slices.IndexFunc(groups, func(group []partyMatch) bool {
			return group[0].term == match.term
		})
		if idx < 0 {
			groups = append(groups, []partyMatch{match})
			continue
		}
		groups[idx] = append(groups[idx], match)
	}

	return groups
}

func partyIDs(matches []partyMatch) []pulid.ID {
	ids := make([]pulid.ID, 0, len(matches))
	for _, match := range matches {
		if !slices.Contains(ids, match.id) {
			ids = append(ids, match.id)
		}
	}

	return ids
}

func partyNames(matches []partyMatch) string {
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		if !slices.Contains(names, match.name) {
			names = append(names, match.name)
		}
	}

	return strings.Join(names, ", ")
}

func partyTerms(query string) []string {
	query = strings.TrimSpace(query)
	terms := []string{query}
	for _, word := range strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return r == ' ' || r == ',' || r == '/' || r == '-' || r == '\''
	}) {
		if len(terms) > partyTokenLimit {
			break
		}
		if len(word) < partyTokenMin {
			continue
		}
		if _, filler := partyFillerWords[word]; filler {
			continue
		}
		if !slices.ContainsFunc(terms, func(term string) bool { return strings.EqualFold(term, word) }) {
			terms = append(terms, word)
		}
	}

	return terms
}

func (t *searchShipmentsTool) resolveParties(
	ctx context.Context,
	tenant pagination.TenantInfo,
	query string,
) (*shipmentParties, error) {
	parties := &shipmentParties{}
	if t.parties.none() {
		return parties, nil
	}

	for _, term := range partyTerms(query) {
		filter := &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: partyMatchLimit},
			Query:      term,
		}

		if t.parties.customers != nil {
			found, err := t.parties.customers.List(ctx, &repositories.ListCustomerRequest{Filter: filter})
			if err != nil {
				return nil, err
			}
			for _, item := range found.Items {
				parties.customers = append(parties.customers, partyMatch{
					term: term, id: item.ID, name: item.Name,
				})
			}
		}

		if t.parties.locations != nil {
			found, err := t.parties.locations.List(ctx, &repositories.ListLocationRequest{Filter: filter})
			if err != nil {
				return nil, err
			}
			for _, item := range found.Items {
				parties.locations = append(parties.locations, partyMatch{
					term: term, id: item.ID, name: item.Name + " (" + item.City + ")",
				})
			}
		}

		if t.parties.workers != nil {
			found, err := t.parties.workers.List(ctx, &repositories.ListWorkersRequest{Filter: filter})
			if err != nil {
				return nil, err
			}
			for _, item := range found.Items {
				parties.workers = append(parties.workers, partyMatch{
					term: term, id: item.ID, name: workerName(item),
				})
			}
		}

		if t.parties.commodities != nil {
			found, err := t.parties.commodities.List(ctx, &repositories.ListCommodityRequest{Filter: filter})
			if err != nil {
				return nil, err
			}
			for _, item := range found.Items {
				parties.commodities = append(parties.commodities, partyMatch{
					term: term, id: item.ID, name: item.Name,
				})
			}
		}

		if t.parties.types != nil {
			found, err := t.parties.types.List(ctx, &repositories.ListShipmentTypesRequest{Filter: filter})
			if err != nil {
				return nil, err
			}
			for _, item := range found.Items {
				parties.types = append(parties.types, partyMatch{
					term: term, id: item.ID, name: item.Code,
				})
			}
		}

		if term == query && !parties.empty() {
			break
		}
	}

	return parties, nil
}
