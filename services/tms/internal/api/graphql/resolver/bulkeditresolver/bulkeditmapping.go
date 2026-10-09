package bulkeditresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/domain/bulkedit"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

func selectionFromInput(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	input *gqlmodel.BulkEditSelectionInput,
) (bulkedit.Selection, error) {
	if input == nil {
		return bulkedit.Selection{}, errortypes.NewValidationError(
			"selection",
			errortypes.ErrRequired,
			"Choose the rows to change",
		)
	}
	if len(input.Ids) > 0 {
		return bulkedit.Selection{IDs: input.Ids}, nil
	}
	if input.Filter == nil {
		return bulkedit.Selection{}, nil
	}

	connection, err := base.DataTableConnectionFromGraphQL(
		ctx,
		input.Filter,
		base.TenantInfo(authCtx),
	)
	if err != nil {
		return bulkedit.Selection{}, err
	}
	return bulkedit.Selection{
		Filter: &bulkedit.SelectionFilter{
			Query:        connection.Filter.Query,
			FieldFilters: connection.Filter.FieldFilters,
			FilterGroups: connection.Filter.FilterGroups,
			Options:      input.Options,
		},
	}, nil
}

func fieldsToModel(fields []services.BulkEditField) []*gqlmodel.BulkEditField {
	models := make([]*gqlmodel.BulkEditField, 0, len(fields))
	for _, field := range fields {
		options := make([]*gqlmodel.BulkEditOption, 0, len(field.Options))
		for _, option := range field.Options {
			options = append(
				options,
				&gqlmodel.BulkEditOption{Value: option.Value, Label: option.Label},
			)
		}
		model := &gqlmodel.BulkEditField{
			Name:    field.Name,
			Label:   field.Label,
			Kind:    string(field.Kind),
			Options: options,
		}
		if field.Record != "" {
			record := field.Record
			model.Record = &record
		}
		models = append(models, model)
	}
	return models
}

func intPointer(value *int64) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}

func jobToModel(entity *bulkedit.BulkEdit) *gqlmodel.BulkEditJob {
	if entity == nil {
		return nil
	}

	failures := make(
		[]*gqlmodel.BulkEditFailure,
		0,
		min(entity.FailedCount, bulkedit.MaxStoredFailures),
	)
	for _, target := range entity.Targets {
		if target.Error == "" {
			continue
		}
		failures = append(failures, &gqlmodel.BulkEditFailure{ID: target.ID, Message: target.Error})
		if len(failures) >= bulkedit.MaxStoredFailures {
			break
		}
	}

	return &gqlmodel.BulkEditJob{
		ID:             entity.ID.String(),
		Resource:       entity.Resource,
		Field:          entity.Field,
		Value:          entity.Value,
		Status:         string(entity.Status),
		TotalCount:     entity.TotalCount,
		ProcessedCount: entity.ProcessedCount,
		ChangedCount:   entity.ChangedCount,
		FailedCount:    entity.FailedCount,
		FailureMessage: entity.FailureMessage,
		Failures:       failures,
		CanUndo:        entity.CanUndo(timeutils.NowUnix()),
		CompletedAt:    intPointer(entity.CompletedAt),
		UndoneAt:       intPointer(entity.UndoneAt),
		CreatedAt:      int(entity.CreatedAt),
	}
}
