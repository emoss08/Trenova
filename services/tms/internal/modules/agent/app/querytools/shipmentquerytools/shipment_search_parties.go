package shipmentquerytools

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

type PartySources struct {
	Customers   customerports.CustomerRepository
	Locations   locationports.LocationRepository
	workers     workerports.WorkerRepository
	Commodities commodityports.CommodityRepository
	Types       shipmenttypeports.ShipmentTypeRepository
	Orders      shipmentports.OrderReferenceMatcher
}

func (s *PartySources) none() bool {
	return s.Customers == nil && s.Locations == nil && s.workers == nil &&
		s.Commodities == nil && s.Types == nil && s.Orders == nil
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

func orderReferenceName(match *shipmentports.OrderReferenceMatch) string {
	if match.PONumber == "" {
		return match.OrderNumber
	}

	return match.OrderNumber + " (PO " + match.PONumber + ")"
}
