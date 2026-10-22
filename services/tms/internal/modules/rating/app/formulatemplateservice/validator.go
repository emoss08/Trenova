package formulatemplateservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *Service) validateTemplate(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) error {
	entity.NormalizeRounding()

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if err := s.validateVariableNames(ctx, entity, multiErr); err != nil {
		return err
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return s.validateExpression(ctx, entity)
}

func (s *Service) validateVariableNames(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
	multiErr *errortypes.MultiError,
) error {
	if len(entity.VariableDefinitions) == 0 {
		return nil
	}

	roots := s.formulaService.FieldRoots(entity.SchemaID)
	for i, def := range entity.VariableDefinitions {
		if def == nil || def.Name == "" {
			continue
		}
		field := fmt.Sprintf("variableDefinitions[%d].name", i)
		reserved, err := s.formulaService.IsReservedName(ctx, def.Name)
		if err != nil {
			return err
		}
		if reserved {
			multiErr.Add(
				field,
				errortypes.ErrInvalid,
				"{0} is the name of a formula function; choose another name for the input",
				def.Name,
			)
			continue
		}
		if _, shadows := roots[def.Name]; shadows {
			multiErr.Add(
				field,
				errortypes.ErrInvalid,
				"{0} is already a shipment value; an input cannot replace it, so choose another name",
				def.Name,
			)
		}
	}

	return nil
}

func (s *Service) validateExpression(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) error {
	checked, err := s.formulaService.CheckTemplate(ctx, entity)
	if err != nil {
		return err
	}
	if err = formula.Refusal(checked); err != nil {
		return err
	}

	s.logUnguardedNullableFields(entity, checked)
	return nil
}

func (s *Service) logUnguardedNullableFields(
	entity *formulatemplate.FormulaTemplate,
	checked []formula.Checked,
) {
	fields := make([]string, 0, 2)
	for _, item := range checked {
		for _, name := range item.Check.Unguarded() {
			fields = appendUnique(fields, name)
		}
	}
	if len(fields) == 0 {
		return
	}

	s.l.Info("formula template uses nullable fields without a guard",
		zap.String("templateID", entity.ID.String()),
		zap.String("name", entity.Name),
		zap.Strings("fields", fields),
	)
}

func validateBulkTemplateIDs(ids []pulid.ID) error {
	switch {
	case len(ids) == 0:
		return errortypes.NewValidationError(
			"templateIds",
			errortypes.ErrRequired,
			"At least one template is required",
		)
	case len(ids) > maxBulkTemplateIDs:
		return errortypes.NewValidationError(
			"templateIds",
			errortypes.ErrInvalid,
			"Cannot act on more than {0} templates at once", maxBulkTemplateIDs,
		)
	default:
		return nil
	}
}
