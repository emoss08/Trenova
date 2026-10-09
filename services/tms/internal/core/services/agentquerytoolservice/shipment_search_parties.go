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
	customers repositories.CustomerRepository
	locations repositories.LocationRepository
	workers   repositories.WorkerRepository
}

func (s shipmentPartySources) none() bool {
	return s.customers == nil && s.locations == nil && s.workers == nil
}

type shipmentParties struct {
	customers []partyMatch
	locations []partyMatch
	workers   []partyMatch
}

func (p *shipmentParties) empty() bool {
	return len(p.customers) == 0 && len(p.locations) == 0 && len(p.workers) == 0
}

func (p *shipmentParties) kinds() [][]partyMatch {
	kinds := make([][]partyMatch, 0, 3)
	for _, matches := range [][]partyMatch{p.customers, p.locations, p.workers} {
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

		if term == query && !parties.empty() {
			break
		}
	}

	return parties, nil
}
