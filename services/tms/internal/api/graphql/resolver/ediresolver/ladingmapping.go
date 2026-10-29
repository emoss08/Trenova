package ediresolver

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/pkg/lading"
)

func ladingInterchangeToModel(manifest *lading.Check) *gqlmodel.EDILadingInterchange {
	if manifest.Partner == nil || manifest.Partner.Interchange == nil {
		return nil
	}
	interchange := manifest.Partner.Interchange
	model := &gqlmodel.EDILadingInterchange{
		Version:    optionalText(interchange.Version),
		Usage:      optionalText(interchange.Usage),
		Separators: interchange.Separators,
	}
	if model.Separators == nil {
		model.Separators = []string{}
	}
	if interchange.Them != nil {
		model.TheirQualifier = optionalText(interchange.Them.Qualifier)
		model.TheirID = optionalText(interchange.Them.ID)
	}
	if interchange.Us != nil {
		model.OurQualifier = optionalText(interchange.Us.Qualifier)
		model.OurID = optionalText(interchange.Us.ID)
	}
	if interchange.Group != nil {
		model.TheirGroupCode = optionalText(interchange.Group.Theirs)
		model.OurGroupCode = optionalText(interchange.Group.Ours)
	}
	return model
}

func ladingExchangesToModel(manifest *lading.Check) []*gqlmodel.EDILadingExchange {
	exchanges := make([]*gqlmodel.EDILadingExchange, 0, len(manifest.Maps))
	for index := range manifest.Maps {
		item := &manifest.Maps[index]
		if item.Set == "" {
			continue
		}
		direction := edidocument.DocumentDirectionOutbound
		if item.Direction == lading.DirectionInbound {
			direction = edidocument.DocumentDirectionInbound
		}
		ack, within := acknowledgmentOf(manifest, item)
		exchange := &gqlmodel.EDILadingExchange{
			TransactionSet: item.Set,
			Direction:      direction,
			Map:            item.Name,
			Acknowledgment: optionalText(ack),
		}
		if ack != "" && within > 0 {
			seconds := int(within)
			exchange.AcknowledgmentWithin = &seconds
		}
		exchanges = append(exchanges, exchange)
	}
	slices.SortStableFunc(exchanges, func(a, b *gqlmodel.EDILadingExchange) int {
		return cmp.Or(
			cmp.Compare(a.TransactionSet, b.TransactionSet),
			cmp.Compare(a.Direction, b.Direction),
		)
	})
	return exchanges
}

// acknowledgmentOf is the acknowledgment that answers a map's set, and how long the flow waits
// for it: the one the receive clause sends back for an inbound map, the one the sending flow's
// expect clause waits on for an outbound one.
func acknowledgmentOf(manifest *lading.Check, item *lading.Map) (set string, within int64) {
	for flowIndex := range manifest.Flows {
		flow := &manifest.Flows[flowIndex]
		sends := false
		var expect *lading.Clause
		for clauseIndex := range flow.Clauses {
			clause := &flow.Clauses[clauseIndex]
			switch {
			case clause.Kind == lading.ClauseReceive && item.Direction == lading.DirectionInbound &&
				clause.Map == item.Name:
				if clause.Acknowledge == lading.AcknowledgeNone {
					return "", 0
				}
				return clause.Acknowledge, 0
			case (clause.Kind == lading.ClauseSend || clause.Kind == lading.ClauseRespond) &&
				clause.Map == item.Name:
				sends = true
			case clause.Kind == lading.ClauseExpect:
				expect = clause
			}
		}
		if sends && item.Direction == lading.DirectionOutbound && expect != nil {
			return expect.Set, expect.Within
		}
	}
	return "", 0
}
