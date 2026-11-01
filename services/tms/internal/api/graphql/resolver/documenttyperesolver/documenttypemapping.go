package documenttyperesolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
)

func documentTypeColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.DocumentTypeSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func documentTypeConnectionToModel(
	result *pagination.CursorListResult[*documenttype.DocumentType],
) (*gqlmodel.DocumentTypeConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *documenttype.DocumentType, cursor string) *gqlmodel.DocumentTypeEdge {
			return &gqlmodel.DocumentTypeEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.DocumentTypeEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.DocumentTypeConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func documentTypeFromInput(
	input *gqlmodel.DocumentTypeInput,
	authCtx *authctx.AuthContext,
) *documenttype.DocumentType {
	entity := &documenttype.DocumentType{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
	}
	applyDocumentTypeInput(entity, input)

	return entity
}

func applyDocumentTypeInput(entity *documenttype.DocumentType, input *gqlmodel.DocumentTypeInput) {
	entity.Code = input.Code
	entity.Name = input.Name
	entity.Description = gqlscalar.StringValue(input.Description)
	entity.Color = gqlscalar.StringValue(input.Color)
	entity.DocumentClassification = input.DocumentClassification
	entity.DocumentCategory = input.DocumentCategory
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}
}
