package reportresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/tableexportservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func tableExportView(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	input *gqlmodel.TableExportViewInput,
) (tableexportservice.View, error) {
	if input == nil {
		return tableexportservice.View{}, errortypes.NewValidationError(
			"view",
			errortypes.ErrRequired,
			"Say which table to export",
		)
	}

	view := tableexportservice.View{
		Resource: permission.Resource(input.Resource),
		Columns:  make([]tableexportservice.Column, 0, len(input.Columns)),
	}
	for _, column := range input.Columns {
		view.Columns = append(view.Columns, tableexportservice.Column{
			Field: column.Field,
			Label: column.Label,
		})
	}

	if input.Filter != nil {
		connection, err := base.DataTableConnectionFromGraphQL(
			ctx,
			input.Filter,
			base.TenantInfo(authCtx),
		)
		if err != nil {
			return tableexportservice.View{}, err
		}
		view.Query = connection.Filter.Query
		view.FieldFilters = connection.Filter.FieldFilters
		view.FilterGroups = connection.Filter.FilterGroups
		view.Sort = connection.Filter.Sort
	}

	return view, nil
}

func tableExportResultToModel(result *tableexportservice.Result) *gqlmodel.TableExportResult {
	model := &gqlmodel.TableExportResult{
		DefinitionID:   result.DefinitionID.String(),
		SkippedColumns: result.Skipped,
	}
	if model.SkippedColumns == nil {
		model.SkippedColumns = []string{}
	}
	if result.Run != nil {
		model.Run = reportRunToModel(result.Run)
	}
	if result.Schedule != nil {
		model.Schedule = reportScheduleToModel(result.Schedule)
	}
	return model
}
