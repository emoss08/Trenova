package auditlogresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/pkg/pagination"
)

func auditEntryConnectionToModel(
	result *pagination.CursorListResult[*audit.Entry],
) (*gqlmodel.AuditEntryConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *audit.Entry, cursor string) *gqlmodel.AuditEntryEdge {
			return &gqlmodel.AuditEntryEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.AuditEntryEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AuditEntryConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func legalHoldConnectionToModel(
	result *pagination.CursorListResult[*audit.LegalHold],
) (*gqlmodel.AuditLegalHoldConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *audit.LegalHold, cursor string) *gqlmodel.AuditLegalHoldEdge {
			return &gqlmodel.AuditLegalHoldEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AuditLegalHoldEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AuditLegalHoldConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func exportConnectionToModel(
	result *pagination.CursorListResult[*audit.Export],
) (*gqlmodel.AuditExportConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *audit.Export, cursor string) *gqlmodel.AuditExportEdge {
			return &gqlmodel.AuditExportEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AuditExportEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AuditExportConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func exportFiltersFromInput(input *gqlmodel.AuditExportFiltersInput) audit.ExportFilters {
	if input == nil {
		return audit.ExportFilters{}
	}
	filters := audit.ExportFilters{
		Resources:  input.Resources,
		Operations: input.Operations,
		Channels:   input.Channels,
	}
	for _, category := range input.Categories {
		filters.Categories = append(filters.Categories, string(category))
	}
	if input.PrincipalID != nil {
		filters.PrincipalID = *input.PrincipalID
	}
	if input.ResourceID != nil {
		filters.ResourceID = *input.ResourceID
	}
	return filters
}
