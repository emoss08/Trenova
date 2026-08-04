package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/pkg/pagination"
)

func jurisdictionRuleConnectionToModel(
	result *pagination.CursorListResult[*jurisdictionrule.JurisdictionRule],
) (*gqlmodel.JurisdictionRuleConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(
			node *jurisdictionrule.JurisdictionRule,
			cursor string,
		) *gqlmodel.JurisdictionRuleEdge {
			return &gqlmodel.JurisdictionRuleEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.JurisdictionRuleEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.JurisdictionRuleConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
