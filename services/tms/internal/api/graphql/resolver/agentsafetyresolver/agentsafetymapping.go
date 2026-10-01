package agentsafetyresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentsafetyservice"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/memtable"
	"github.com/emoss08/trenova/shared/pulid"
)

func toolPolicyViewsToModel(
	views []services.AgentToolPolicyView,
) []*gqlmodel.AgentToolPolicy {
	out := make([]*gqlmodel.AgentToolPolicy, 0, len(views))
	for idx := range views {
		out = append(out, base.ToolPolicyViewToModel(&views[idx]))
	}

	return out
}

func toolPolicyConnectionRequest(
	input *gqlmodel.AgentToolPolicyConnectionInput,
) memtable.Request {
	req := memtable.Request{
		First: base.IntValue(input.First),
		After: base.StringValue(input.After),
		Query: base.StringValue(input.Query),
	}
	if input.Egress != nil {
		req.FieldFilters = append(req.FieldFilters, domaintypes.FieldFilter{
			Field:    "egress",
			Operator: dbtype.OpEqual,
			Value:    input.Egress.String(),
		})
	}
	if input.Resource != nil {
		req.FieldFilters = append(req.FieldFilters, domaintypes.FieldFilter{
			Field:    "resource",
			Operator: dbtype.OpEqual,
			Value:    *input.Resource,
		})
	}
	if input.Kind != nil {
		req.FieldFilters = append(req.FieldFilters, domaintypes.FieldFilter{
			Field:    "kind",
			Operator: dbtype.OpEqual,
			Value:    input.Kind.String(),
		})
	}
	if input.RunsWithoutPerson != nil {
		req.FieldFilters = append(req.FieldFilters, domaintypes.FieldFilter{
			Field:    agentsafetyservice.FieldRunsWithoutPerson,
			Operator: dbtype.OpEqual,
			Value:    *input.RunsWithoutPerson,
		})
	}

	return req
}

func agentToolSafetyPageToModel(
	page *services.AgentToolSafetyPage,
) *gqlmodel.AgentToolSafetyConnection {
	out := &gqlmodel.AgentToolSafetyConnection{
		Edges:      make([]*gqlmodel.AgentToolSafetyEdge, 0, len(page.Edges)),
		PageInfo:   &gqlmodel.PageInfo{HasNextPage: page.HasNextPage},
		TotalCount: page.TotalCount,
	}
	for idx := range page.Edges {
		edge := &page.Edges[idx]
		out.Edges = append(out.Edges, &gqlmodel.AgentToolSafetyEdge{
			Node:   &edge.Node,
			Cursor: edge.Cursor,
		})
	}
	if count := len(out.Edges); count > 0 {
		endCursor := out.Edges[count-1].Cursor
		out.PageInfo.EndCursor = &endCursor
	}

	return out
}

func toolPolicyPageToModel(
	page *services.AgentToolPolicyPage,
) *gqlmodel.AgentToolPolicyConnection {
	out := &gqlmodel.AgentToolPolicyConnection{
		Edges:      make([]*gqlmodel.AgentToolPolicyEdge, 0, len(page.Edges)),
		PageInfo:   &gqlmodel.PageInfo{HasNextPage: page.HasNextPage},
		TotalCount: page.TotalCount,
	}
	for idx := range page.Edges {
		edge := &page.Edges[idx]
		node := base.ToolPolicyViewToModel(&edge.View)
		node.RunsWithoutPerson = edge.RunsWithoutPerson
		out.Edges = append(out.Edges, &gqlmodel.AgentToolPolicyEdge{
			Node:   node,
			Cursor: edge.Cursor,
		})
	}
	if count := len(out.Edges); count > 0 {
		endCursor := out.Edges[count-1].Cursor
		out.PageInfo.EndCursor = &endCursor
	}

	return out
}

func agentSafetyIDs(ids []string) ([]pulid.ID, error) {
	if ids == nil {
		return nil, nil
	}

	return base.ParseIDs(ids)
}
