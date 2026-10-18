package agentsafetyresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/pulid"
)

func agentToolSafetyPageToModel(
	page *agentports.AgentToolSafetyPage,
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
	page *agentports.AgentToolPolicyPage,
) *gqlmodel.AgentToolPolicyConnection {
	out := &gqlmodel.AgentToolPolicyConnection{
		Edges:      make([]*gqlmodel.AgentToolPolicyEdge, 0, len(page.Edges)),
		PageInfo:   &gqlmodel.PageInfo{HasNextPage: page.HasNextPage},
		TotalCount: page.TotalCount,
	}
	for idx := range page.Edges {
		edge := &page.Edges[idx]
		node := ToolPolicyViewToModel(&edge.View)
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
