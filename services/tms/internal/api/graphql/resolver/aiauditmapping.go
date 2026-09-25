package resolver

import (
	"context"
	"fmt"
	"slices"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// aiAuditEventComputedColumns are the columns each computed field of an
// event is worked out from: the arguments a reader sees are the recorded
// arguments cut to their sensitivity, and the rest are read by an id.
var aiAuditEventComputedColumns = map[string][]buncolgen.Column{
	"onBehalfOf": {buncolgen.AIAuditEventColumns.OnBehalfOfUserID},
	"decidedBy":  {buncolgen.AIAuditEventColumns.DecidedByUserID},
	"agent":      {buncolgen.AIAuditEventColumns.AgentDefinitionID},
	"traceUrl":   {buncolgen.AIAuditEventColumns.TraceID},
	"arguments": {
		buncolgen.AIAuditEventColumns.Arguments,
		buncolgen.AIAuditEventColumns.ArgumentSensitivity,
	},
	"auditEntries": {buncolgen.AIAuditEventColumns.ID},
}

// aiAuditEventColumns are the columns a trail page selects for the fields
// asked for, with the columns each computed field reads.
func aiAuditEventColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AIAuditEventSpec,
		func(path string) bool { return graphql.FieldRequested(ctx, path) },
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)
	if len(selection.Columns) == 0 {
		return selection.Columns
	}

	columns := selection.Columns
	for special, needed := range aiAuditEventComputedColumns {
		if !selection.HasSpecial(special) {
			continue
		}
		for _, column := range needed {
			if !slices.Contains(columns, column.String()) {
				columns = append(columns, column.String())
			}
		}
	}

	return columns
}

func aiAuditEventConnectionToModel(
	result *pagination.CursorListResult[*aiaudit.AIAuditEvent],
) (*gqlmodel.AIAuditEventConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *aiaudit.AIAuditEvent, cursor string) *gqlmodel.AIAuditEventEdge {
			return &gqlmodel.AIAuditEventEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AIAuditEventEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, fmt.Errorf("build AI audit event connection: %w", err)
	}

	return &gqlmodel.AIAuditEventConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func aiAuditExportConnectionToModel(
	result *pagination.CursorListResult[*aiaudit.AIAuditExport],
) (*gqlmodel.AIAuditExportConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *aiaudit.AIAuditExport, cursor string) *gqlmodel.AIAuditExportEdge {
			return &gqlmodel.AIAuditExportEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AIAuditExportEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, fmt.Errorf("build AI audit export connection: %w", err)
	}

	return &gqlmodel.AIAuditExportConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

// loadAuditUser reads a person the trail names. A person since removed from
// the organization is not an error: the trail keeps their name.
func loadAuditUser(ctx context.Context, id pulid.ID) (*tenant.User, error) {
	user, err := loadUser(ctx, id)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil
		}

		return nil, err
	}

	return user, nil
}
