package formulatemplateservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
	"go.uber.org/zap"
)

func (s *Service) validateTemplate(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) error {
	entity.NormalizeRounding()

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return s.validateExpression(ctx, entity)
}

func (s *Service) validateExpression(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) error {
	variables := make(map[string]any, len(entity.VariableDefinitions))
	for _, varDef := range entity.VariableDefinitions {
		if varDef.DefaultValue != nil {
			variables[varDef.Name] = varDef.DefaultValue
			continue
		}

		variables[varDef.Name] = typeutils.DefaultValueForType(string(varDef.Type))
	}

	env, err := s.formulaService.BuildValidationEnvironment(entity.SchemaID, variables)
	if err != nil {
		return errortypes.NewValidationError(
			"schemaId",
			errortypes.ErrInvalid,
			expressionErrorMessage(err),
		)
	}

	outcome := s.formulaService.ValidateExpressionDetailed(ctx, entity.Expression, env)
	if outcome.Err != nil {
		return errortypes.NewValidationError(
			"expression",
			errortypes.ErrInvalid,
			expressionErrorMessage(outcome.Err),
		)
	}
	if outcome.Warning != "" {
		s.l.Warn("expression produced a runtime error against the synthetic validation environment",
			zap.String("expression", entity.Expression),
			zap.String("warning", outcome.Warning),
		)
	}
	s.logUnguardedNullableFields(ctx, entity, variables)

	multiErr := errortypes.NewMultiError()
	for i, def := range entity.BreakdownDefinitions {
		if def == nil {
			continue
		}
		defOutcome := s.formulaService.ValidateExpressionDetailed(ctx, def.Expression, env)
		if defOutcome.Err != nil {
			multiErr.Add(
				fmt.Sprintf("breakdownDefinitions[%d].expression", i),
				errortypes.ErrInvalid,
				expressionErrorMessage(defOutcome.Err),
			)
		}
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	if err = s.formulaService.ValidateLookupTables(ctx, entity.Expression, tenantInfo); err != nil {
		return err
	}

	for _, def := range entity.BreakdownDefinitions {
		if def == nil {
			continue
		}
		if err = s.formulaService.ValidateLookupTables(
			ctx,
			def.Expression,
			tenantInfo,
		); err != nil {
			return err
		}
	}

	return nil
}

// logUnguardedNullableFields records, at save time, the nullable fields a
// template relies on without a guard. Saving is still allowed: the Studio has
// already shown the author the same warning with a one-click fix, and a draft
// is where an unfinished formula belongs.
func (s *Service) logUnguardedNullableFields(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
	variables map[string]any,
) {
	found, err := s.formulaService.UnguardedNullableFields(
		ctx,
		entity.Expression,
		entity.SchemaID,
		variables,
	)
	if err != nil || len(found) == 0 {
		return
	}

	fields := make([]string, 0, len(found))
	for _, item := range found {
		fields = append(fields, item.Field)
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
			fmt.Sprintf("Cannot act on more than %d templates at once", maxBulkTemplateIDs),
		)
	default:
		return nil
	}
}
