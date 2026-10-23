package shipmentboardresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/agent/ports/shipmentbriefports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func scopeRequestFromGraphQL(
	input *gqlmodel.ShipmentBoardScopeInput,
	tenantInfo pagination.TenantInfo,
	margins bool,
) *shipmentports.ShipmentBoardScopeRequest {
	return &shipmentports.ShipmentBoardScopeRequest{
		Filter: base.QueryOptionsFromGraphQL(&base.GqlListOptions{
			TenantInfo:   tenantInfo,
			Query:        base.StringValue(input.Query),
			FieldFilters: input.FieldFilters,
			FilterGroups: input.FilterGroups,
		}),
		QuickFilters: base.QuickFilterSpecsFromGraphQL(input.QuickFilters),
		Timezone:     input.Timezone,
		Margins:      margins,
	}
}

func capabilitiesToModel(
	caps *shipmentports.ShipmentBoardCapabilities,
) *gqlmodel.ShipmentBoardCapabilities {
	return &gqlmodel.ShipmentBoardCapabilities{
		Ai:            caps.AI,
		OperationType: caps.OperationType,
		Hos:           caps.HOS,
		Maps:          caps.Maps,
	}
}

func stageSummariesToModel(
	rows []*shipmentports.ShipmentStageSummary,
) []*gqlmodel.ShipmentStageSummary {
	out := make([]*gqlmodel.ShipmentStageSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, &gqlmodel.ShipmentStageSummary{
			Stage:   row.Stage,
			Rank:    row.Rank,
			Count:   row.Count,
			Revenue: base.DecimalString(row.Revenue),
		})
	}
	return out
}

func boardGroupsToModel(rows []*shipmentports.ShipmentBoardGroup) []*gqlmodel.ShipmentBoardGroup {
	out := make([]*gqlmodel.ShipmentBoardGroup, 0, len(rows))
	for _, row := range rows {
		out = append(out, &gqlmodel.ShipmentBoardGroup{
			Key:     row.Key,
			Label:   row.Label,
			Count:   row.Count,
			Revenue: base.DecimalString(row.Revenue),
		})
	}
	return out
}

func quickFilterCountsToModel(
	rows []*shipmentports.ShipmentQuickFilterCount,
) []*gqlmodel.ShipmentQuickFilterCount {
	out := make([]*gqlmodel.ShipmentQuickFilterCount, 0, len(rows))
	for _, row := range rows {
		out = append(out, &gqlmodel.ShipmentQuickFilterCount{Filter: row.Filter, Count: row.Count})
	}
	return out
}

func briefingToModel(b *shipmentbriefports.ShipmentBriefing) *gqlmodel.ShipmentBriefing {
	segments := make([]*gqlmodel.ShipmentBriefingSegment, 0, len(b.Segments))
	for _, segment := range b.Segments {
		segments = append(segments, &gqlmodel.ShipmentBriefingSegment{
			Text:   segment.Text,
			Filter: segment.Filter,
		})
	}

	return &gqlmodel.ShipmentBriefing{
		Segments:    segments,
		Narrated:    b.Narrated,
		GeneratedAt: int(b.GeneratedAt),
		Generation:  b.Generation,
	}
}
