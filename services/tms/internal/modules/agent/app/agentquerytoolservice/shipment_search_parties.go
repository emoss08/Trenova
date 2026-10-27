package agentquerytoolservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/shipmenttypeports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
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
	customers   customerports.CustomerRepository
	locations   locationports.LocationRepository
	workers     workerports.WorkerRepository
	commodities commodityports.CommodityRepository
	types       shipmenttypeports.ShipmentTypeRepository
	orders      shipmentports.OrderReferenceMatcher
}

func (s shipmentPartySources) none() bool {
	return s.customers == nil && s.locations == nil && s.workers == nil &&
		s.commodities == nil && s.types == nil && s.orders == nil
}

type shipmentParties struct {
	terms       []string
	customers   []partyMatch
	locations   []partyMatch
	workers     []partyMatch
	commodities []partyMatch
	types       []partyMatch
	orders      []partyMatch
}

func (p *shipmentParties) empty() bool {
	return len(p.customers) == 0 && len(p.locations) == 0 && len(p.workers) == 0 &&
		len(p.commodities) == 0 && len(p.types) == 0 && len(p.orders) == 0
}

type partyKind struct {
	label   string
	matches func(p *shipmentParties) []partyMatch
	into    func(group *shipmentports.ShipmentPartyGroup, ids []pulid.ID)
}

var partyKinds = []partyKind{
	{
		label:   "customer",
		matches: func(p *shipmentParties) []partyMatch { return p.customers },
		into: func(g *shipmentports.ShipmentPartyGroup, ids []pulid.ID) {
			g.CustomerIDs = append(g.CustomerIDs, ids...)
		},
	},
	{
		label:   "with a stop at",
		matches: func(p *shipmentParties) []partyMatch { return p.locations },
		into: func(g *shipmentports.ShipmentPartyGroup, ids []pulid.ID) {
			g.StopLocationIDs = append(g.StopLocationIDs, ids...)
		},
	},
	{
		label:   "driven by",
		matches: func(p *shipmentParties) []partyMatch { return p.workers },
		into: func(g *shipmentports.ShipmentPartyGroup, ids []pulid.ID) {
			g.WorkerIDs = append(g.WorkerIDs, ids...)
		},
	},
	{
		label:   "carrying",
		matches: func(p *shipmentParties) []partyMatch { return p.commodities },
		into: func(g *shipmentports.ShipmentPartyGroup, ids []pulid.ID) {
			g.CommodityIDs = append(g.CommodityIDs, ids...)
		},
	},
	{
		label:   "of type",
		matches: func(p *shipmentParties) []partyMatch { return p.types },
		into: func(g *shipmentports.ShipmentPartyGroup, ids []pulid.ID) {
			g.ShipmentTypeIDs = append(g.ShipmentTypeIDs, ids...)
		},
	},
	{
		label:   "on order",
		matches: func(p *shipmentParties) []partyMatch { return p.orders },
		into: func(g *shipmentports.ShipmentPartyGroup, ids []pulid.ID) {
			g.OrderIDs = append(g.OrderIDs, ids...)
		},
	},
}

const stopKind = 1

type wordReading struct {
	kind    int
	matches []partyMatch
}

func (p *shipmentParties) readingsOf(term string) []wordReading {
	readings := make([]wordReading, 0, len(partyKinds))
	for kind := range partyKinds {
		var found []partyMatch
		for _, match := range partyKinds[kind].matches(p) {
			if match.term == term {
				found = append(found, match)
			}
		}
		if len(found) > 0 {
			readings = append(readings, wordReading{kind: kind, matches: found})
		}
	}

	return readings
}

func partyGroups(
	parties *shipmentParties,
	criteria *filtercatalog.Criteria,
) []shipmentports.ShipmentPartyGroup {
	byKind := make([][]partyMatch, len(partyKinds))
	stops := make([][]partyMatch, 0, partyTokenLimit)
	several := make([][]wordReading, 0, partyTokenLimit)
	terms := make([]string, 0, partyTokenLimit)
	for _, term := range parties.terms {
		readings := parties.readingsOf(term)
		switch {
		case len(readings) > 1:
			several = append(several, readings)
			terms = append(terms, term)
		case len(readings) == 1 && readings[0].kind == stopKind:
			stops = append(stops, readings[0].matches)
		case len(readings) == 1:
			byKind[readings[0].kind] = append(byKind[readings[0].kind], readings[0].matches...)
		}
	}

	groups := make(
		[]shipmentports.ShipmentPartyGroup,
		0,
		len(partyKinds)+len(stops)+len(several),
	)
	for kind, matches := range byKind {
		if len(matches) == 0 {
			continue
		}
		var group shipmentports.ShipmentPartyGroup
		partyKinds[kind].into(&group, partyIDs(matches))
		groups = append(groups, group)
		criteria.Field(partyKinds[kind].label, partyNames(matches))
	}
	for _, matches := range stops {
		groups = append(groups, shipmentports.ShipmentPartyGroup{
			StopLocationIDs: partyIDs(matches),
		})
		criteria.Field(partyKinds[stopKind].label, partyNames(matches))
	}
	for idx, readings := range several {
		var group shipmentports.ShipmentPartyGroup
		described := make([]string, 0, len(readings))
		for _, reading := range readings {
			partyKinds[reading.kind].into(&group, partyIDs(reading.matches))
			described = append(described,
				partyKinds[reading.kind].label+" "+partyNames(reading.matches))
		}
		groups = append(groups, group)
		criteria.Field(fmt.Sprintf("%q as", terms[idx]), strings.Join(described, "; or "))
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
		if !slices.ContainsFunc(
			terms,
			func(term string) bool { return strings.EqualFold(term, word) },
		) {
			terms = append(terms, word)
		}
	}

	return terms
}

type partySource struct {
	into   *[]partyMatch
	lookup func(context.Context, *pagination.QueryOptions, string) ([]partyMatch, error)
}

func (t *shipmentListTool) partySources(parties *shipmentParties) []partySource {
	sources := make([]partySource, 0, 5)
	if t.parties.customers != nil {
		sources = append(sources, partySource{into: &parties.customers, lookup: t.customerParties})
	}
	if t.parties.locations != nil {
		sources = append(sources, partySource{into: &parties.locations, lookup: t.locationParties})
	}
	if t.parties.workers != nil {
		sources = append(sources, partySource{into: &parties.workers, lookup: t.workerParties})
	}
	if t.parties.commodities != nil {
		sources = append(sources, partySource{
			into:   &parties.commodities,
			lookup: t.commodityParties,
		})
	}
	if t.parties.types != nil {
		sources = append(sources, partySource{into: &parties.types, lookup: t.typeParties})
	}
	if t.parties.orders != nil {
		sources = append(sources, partySource{into: &parties.orders, lookup: t.orderParties})
	}

	return sources
}

func (t *shipmentListTool) resolveParties(
	ctx context.Context,
	tenant pagination.TenantInfo,
	query string,
) (*shipmentParties, error) {
	parties := &shipmentParties{}
	if t.parties.none() {
		return parties, nil
	}

	sources := t.partySources(parties)
	for _, term := range partyTerms(query) {
		filter := &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: partyMatchLimit},
			Query:      term,
		}
		named := false
		for _, source := range sources {
			found, err := source.lookup(ctx, filter, term)
			if err != nil {
				return nil, err
			}
			*source.into = append(*source.into, found...)
			named = named || len(found) > 0
		}
		if named {
			parties.terms = append(parties.terms, term)
		}
		if term == query && named {
			break
		}
	}

	return parties, nil
}

func (t *shipmentListTool) customerParties(
	ctx context.Context,
	filter *pagination.QueryOptions,
	term string,
) ([]partyMatch, error) {
	found, err := t.parties.customers.List(ctx, &customerports.ListCustomerRequest{Filter: filter})
	if err != nil {
		return nil, err
	}

	matches := make([]partyMatch, 0, len(found.Items))
	for _, item := range found.Items {
		matches = append(matches, partyMatch{term: term, id: item.ID, name: item.Name})
	}

	return matches, nil
}

func (t *shipmentListTool) locationParties(
	ctx context.Context,
	filter *pagination.QueryOptions,
	term string,
) ([]partyMatch, error) {
	found, err := t.parties.locations.List(ctx, &locationports.ListLocationRequest{Filter: filter})
	if err != nil {
		return nil, err
	}

	matches := make([]partyMatch, 0, len(found.Items))
	for _, item := range found.Items {
		matches = append(matches, partyMatch{
			term: term, id: item.ID, name: item.Name + " (" + item.City + ")",
		})
	}

	return matches, nil
}

func (t *shipmentListTool) workerParties(
	ctx context.Context,
	filter *pagination.QueryOptions,
	term string,
) ([]partyMatch, error) {
	found, err := t.parties.workers.List(ctx, &workerports.ListWorkersRequest{Filter: filter})
	if err != nil {
		return nil, err
	}

	matches := make([]partyMatch, 0, len(found.Items))
	for _, item := range found.Items {
		matches = append(matches, partyMatch{term: term, id: item.ID, name: workerName(item)})
	}

	return matches, nil
}

func (t *shipmentListTool) commodityParties(
	ctx context.Context,
	filter *pagination.QueryOptions,
	term string,
) ([]partyMatch, error) {
	found, err := t.parties.commodities.List(
		ctx,
		&commodityports.ListCommodityRequest{Filter: filter},
	)
	if err != nil {
		return nil, err
	}

	matches := make([]partyMatch, 0, len(found.Items))
	for _, item := range found.Items {
		matches = append(matches, partyMatch{term: term, id: item.ID, name: item.Name})
	}

	return matches, nil
}

func (t *shipmentListTool) typeParties(
	ctx context.Context,
	filter *pagination.QueryOptions,
	term string,
) ([]partyMatch, error) {
	found, err := t.parties.types.List(
		ctx,
		&shipmenttypeports.ListShipmentTypesRequest{Filter: filter},
	)
	if err != nil {
		return nil, err
	}

	matches := make([]partyMatch, 0, len(found.Items))
	for _, item := range found.Items {
		matches = append(matches, partyMatch{term: term, id: item.ID, name: item.Code})
	}

	return matches, nil
}

func (t *shipmentListTool) orderParties(
	ctx context.Context,
	filter *pagination.QueryOptions,
	term string,
) ([]partyMatch, error) {
	found, err := t.parties.orders.MatchOrderReferences(
		ctx,
		&shipmentports.MatchOrderReferencesRequest{
			Query: filter.Query,
			Limit: filter.Pagination.Limit,
		},
	)
	if err != nil {
		return nil, err
	}

	matches := make([]partyMatch, 0, len(found))
	for idx := range found {
		matches = append(matches, partyMatch{
			term: term,
			id:   found[idx].OrderID,
			name: orderReferenceName(&found[idx]),
		})
	}

	return matches, nil
}

func orderReferenceName(match *shipmentports.OrderReferenceMatch) string {
	if match.PONumber == "" {
		return match.OrderNumber
	}

	return match.OrderNumber + " (PO " + match.PONumber + ")"
}
