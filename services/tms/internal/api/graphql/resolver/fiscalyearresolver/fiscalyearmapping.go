package fiscalyearresolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
)

func fiscalYearColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.FiscalYearSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func fiscalYearConnectionToModel(
	result *pagination.CursorListResult[*fiscalyear.FiscalYear],
) (*gqlmodel.FiscalYearConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *fiscalyear.FiscalYear, cursor string) *gqlmodel.FiscalYearEdge {
			return &gqlmodel.FiscalYearEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.FiscalYearEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.FiscalYearConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func fiscalYearFromInput(
	input *gqlmodel.FiscalYearInput,
	authCtx *authctx.AuthContext,
) *fiscalyear.FiscalYear {
	entity := &fiscalyear.FiscalYear{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
	}
	applyFiscalYearInput(entity, input)

	return entity
}

func applyFiscalYearInput(entity *fiscalyear.FiscalYear, input *gqlmodel.FiscalYearInput) {
	entity.Status = input.Status
	entity.Year = input.Year
	entity.Name = input.Name
	entity.Description = gqlscalar.StringValue(input.Description)
	entity.StartDate = int64(input.StartDate)
	entity.EndDate = int64(input.EndDate)
	entity.IsCurrent = gqlscalar.BoolValue(input.IsCurrent)
	entity.IsCalendarYear = gqlscalar.BoolValue(input.IsCalendarYear)
	entity.AllowAdjustingEntries = gqlscalar.BoolValue(input.AllowAdjustingEntries)
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}
}
