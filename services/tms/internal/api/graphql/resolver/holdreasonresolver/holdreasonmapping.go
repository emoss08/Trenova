package holdreasonresolver

import (
	"context"

	"github.com/emoss08/trenova/shared/intutils"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
)

func holdReasonColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.HoldReasonSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func holdReasonConnectionToModel(
	result *pagination.CursorListResult[*holdreason.HoldReason],
) (*gqlmodel.HoldReasonConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *holdreason.HoldReason, cursor string) *gqlmodel.HoldReasonEdge {
			return &gqlmodel.HoldReasonEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.HoldReasonEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.HoldReasonConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func holdReasonFromInput(
	input *gqlmodel.HoldReasonInput,
	authCtx *authctx.AuthContext,
) *holdreason.HoldReason {
	entity := &holdreason.HoldReason{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
	}
	applyHoldReasonInput(entity, input)

	return entity
}

func applyHoldReasonInput(entity *holdreason.HoldReason, input *gqlmodel.HoldReasonInput) {
	entity.Type = input.Type
	entity.Code = input.Code
	entity.Label = input.Label
	entity.Description = gqlscalar.StringValue(input.Description)
	entity.Active = input.Active
	entity.DefaultSeverity = input.DefaultSeverity
	entity.DefaultBlocksDispatch = input.DefaultBlocksDispatch
	entity.DefaultBlocksDelivery = input.DefaultBlocksDelivery
	entity.DefaultBlocksBilling = input.DefaultBlocksBilling
	entity.DefaultVisibleToCustomer = input.DefaultVisibleToCustomer
	entity.SortOrder = intutils.SafeToInt32(input.SortOrder)
	entity.ExternalMap = input.ExternalMap
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}
}
