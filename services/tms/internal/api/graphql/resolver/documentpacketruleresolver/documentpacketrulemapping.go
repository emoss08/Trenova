package documentpacketruleresolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentpacketrule"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
)

func documentPacketRuleColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.DocumentPacketRuleSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func documentPacketRuleConnectionToModel(
	result *pagination.CursorListResult[*documentpacketrule.DocumentPacketRule],
) (*gqlmodel.DocumentPacketRuleConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *documentpacketrule.DocumentPacketRule, cursor string) *gqlmodel.DocumentPacketRuleEdge {
			return &gqlmodel.DocumentPacketRuleEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.DocumentPacketRuleEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.DocumentPacketRuleConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func documentPacketRuleFromInput(
	input *gqlmodel.DocumentPacketRuleInput,
	authCtx *authctx.AuthContext,
) (*documentpacketrule.DocumentPacketRule, error) {
	entity := &documentpacketrule.DocumentPacketRule{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
	}
	if err := applyDocumentPacketRuleInput(entity, input); err != nil {
		return nil, err
	}

	return entity, nil
}

func applyDocumentPacketRuleInput(
	entity *documentpacketrule.DocumentPacketRule,
	input *gqlmodel.DocumentPacketRuleInput,
) error {
	documentTypeID, err := gqlid.RequiredID("documentTypeId", input.DocumentTypeID)
	if err != nil {
		return err
	}

	entity.ResourceType = input.ResourceType
	entity.DocumentTypeID = documentTypeID
	entity.Required = input.Required
	entity.AllowMultiple = input.AllowMultiple
	entity.DisplayOrder = input.DisplayOrder
	entity.ExpirationRequired = input.ExpirationRequired
	entity.ExpirationWarningDays = input.ExpirationWarningDays
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	return nil
}
