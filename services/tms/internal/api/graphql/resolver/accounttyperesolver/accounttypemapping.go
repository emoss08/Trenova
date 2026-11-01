package accounttyperesolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
)

func accountTypeColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AccountTypeSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func accountTypeConnectionToModel(
	result *pagination.CursorListResult[*accounttype.AccountType],
) (*gqlmodel.AccountTypeConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *accounttype.AccountType, cursor string) *gqlmodel.AccountTypeEdge {
			return &gqlmodel.AccountTypeEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.AccountTypeEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AccountTypeConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func accountTypeFromInput(
	input *gqlmodel.AccountTypeInput,
	authCtx *authctx.AuthContext,
) *accounttype.AccountType {
	entity := &accounttype.AccountType{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
	}
	applyAccountTypeInput(entity, input)

	return entity
}

func applyAccountTypeInput(entity *accounttype.AccountType, input *gqlmodel.AccountTypeInput) {
	entity.Status = input.Status
	entity.Code = input.Code
	entity.Name = input.Name
	entity.Description = gqlscalar.StringValue(input.Description)
	entity.Category = input.Category
	entity.Color = gqlscalar.StringValue(input.Color)
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}
}

func applyAccountTypePatch(
	entity *accounttype.AccountType,
	input *gqlmodel.AccountTypePatchInput,
) error {
	if err := gqlscalar.ApplyRequired(
		"status",
		"Status",
		input.Status,
		&entity.Status,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired("code", "Code", input.Code, &entity.Code); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired("name", "Name", input.Name, &entity.Name); err != nil {
		return err
	}
	gqlscalar.ApplyClearable(input.Description, &entity.Description)
	if err := gqlscalar.ApplyRequired(
		"category",
		"Category",
		input.Category,
		&entity.Category,
	); err != nil {
		return err
	}
	gqlscalar.ApplyClearable(input.Color, &entity.Color)

	return gqlscalar.ApplyRequiredInt("version", "Version", input.Version, &entity.Version)
}
