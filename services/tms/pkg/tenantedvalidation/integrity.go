package tenantedvalidation

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
)

type integrityFinding struct {
	FieldName string
	Message   string
	Args      []any
}

type integrityPlan struct {
	request    validationframework.IntegrityRequest
	duplicates []integrityFinding
	missing    []integrityFinding
}

func (v *Validator[T]) hasIntegrityChecks() bool {
	return len(v.uniqueFields) > 0 || len(v.compositeUniqueFields) > 0 ||
		len(v.references) > 0 || len(v.referenceLists) > 0
}

func (v *Validator[T]) checkIntegrity(
	ctx context.Context,
	entity T,
	valCtx *validationframework.Context,
	multiErr *errortypes.MultiError,
) {
	if !v.hasIntegrityChecks() {
		return
	}

	plan := integrityPlan{request: validationframework.IntegrityRequest{
		Table:          entity.GetTableName(),
		OrganizationID: valCtx.OrganizationID,
		BusinessUnitID: valCtx.BusinessUnitID,
		ExcludeID:      valCtx.EntityID,
	}}
	v.planUniqueness(entity, multiErr, &plan)
	v.planReferences(entity, multiErr, &plan)
	if plan.request.Empty() {
		return
	}

	checkCtx := ctx
	if v.integrityScope != nil {
		checkCtx = v.integrityScope(ctx)
	}

	result, err := v.checker.CheckIntegrity(checkCtx, &plan.request)
	if err != nil {
		multiErr.AddInternal(fmt.Errorf("check %s integrity: %w", v.modelName, err))
		return
	}
	if len(result.Duplicate) != len(plan.duplicates) || len(result.Exists) != len(plan.missing) {
		multiErr.AddInternal(fmt.Errorf(
			"check %s integrity: checker answered %d uniqueness and %d reference checks of %d and %d",
			v.modelName,
			len(result.Duplicate),
			len(result.Exists),
			len(plan.duplicates),
			len(plan.missing),
		))
		return
	}

	for i, duplicate := range result.Duplicate {
		if duplicate {
			finding := plan.duplicates[i]
			multiErr.Add(
				finding.FieldName,
				errortypes.ErrDuplicate,
				finding.Message,
				finding.Args...)
		}
	}
	for i, exists := range result.Exists {
		if !exists {
			finding := plan.missing[i]
			multiErr.Add(
				finding.FieldName,
				errortypes.ErrInvalidReference,
				finding.Message,
				finding.Args...)
		}
	}
}

func (v *Validator[T]) planUniqueness(
	entity T,
	multiErr *errortypes.MultiError,
	plan *integrityPlan,
) {
	for _, field := range v.uniqueFields {
		value := field.GetValue(entity)
		if isBlank(value) || multiErr.HasErrorAt(field.FieldName) {
			continue
		}

		plan.request.Unique = append(plan.request.Unique, validationframework.UniqueCheck{
			Fields: []validationframework.FieldCheck{{
				Column:        field.Column,
				Value:         value,
				CaseSensitive: field.CaseSensitive,
			}},
		})
		plan.duplicates = append(
			plan.duplicates,
			uniqueFinding(field.FieldName, field.Message, v.modelName),
		)
	}

	for _, composite := range v.compositeUniqueFields {
		if multiErr.HasErrorAt(composite.Name) {
			continue
		}

		checks := make([]validationframework.FieldCheck, 0, len(composite.Fields))
		for _, field := range composite.Fields {
			value := field.GetValue(entity)
			if isBlank(value) || multiErr.HasErrorAt(field.FieldName) {
				checks = nil
				break
			}
			checks = append(checks, validationframework.FieldCheck{
				Column:        field.Column,
				Value:         value,
				CaseSensitive: field.CaseSensitive,
			})
		}
		if len(checks) == 0 {
			continue
		}

		plan.request.Unique = append(
			plan.request.Unique,
			validationframework.UniqueCheck{Fields: checks},
		)
		finding := integrityFinding{FieldName: composite.Name, Message: composite.Message}
		if finding.Message == "" {
			finding = integrityFinding{
				FieldName: composite.Name,
				Message:   "{0} with this combination already exists",
				Args:      []any{v.modelName},
			}
		}
		plan.duplicates = append(plan.duplicates, finding)
	}
}

func uniqueFinding(fieldName, message, modelName string) integrityFinding {
	if message != "" {
		return integrityFinding{FieldName: fieldName, Message: message}
	}

	return integrityFinding{
		FieldName: fieldName,
		Message:   "{0} with this value already exists",
		Args:      []any{modelName},
	}
}

func (v *Validator[T]) planReferences(
	entity T,
	multiErr *errortypes.MultiError,
	plan *integrityPlan,
) {
	for _, reference := range v.references {
		if multiErr.HasErrorAt(reference.FieldName) {
			continue
		}

		id := reference.GetID(entity)
		if id.IsNil() {
			if !reference.Optional {
				multiErr.Add(reference.FieldName, errortypes.ErrRequired, "This field is required")
			}
			continue
		}

		plan.request.References = append(
			plan.request.References,
			validationframework.ReferenceCheck{
				Table:      reference.Table,
				Column:     reference.Column,
				ID:         id,
				Conditions: reference.Conditions,
			},
		)
		plan.missing = append(
			plan.missing,
			referenceFinding(reference.FieldName, reference.Message),
		)
	}

	for _, list := range v.referenceLists {
		for _, target := range list.Targets(entity) {
			if target.ID.IsNil() || multiErr.HasErrorAt(target.FieldName) {
				continue
			}

			plan.request.References = append(
				plan.request.References,
				validationframework.ReferenceCheck{
					Table:      list.Table,
					Column:     list.Column,
					ID:         target.ID,
					Conditions: list.Conditions,
				},
			)
			plan.missing = append(plan.missing, referenceFinding(target.FieldName, list.Message))
		}
	}
}

func referenceFinding(fieldName, message string) integrityFinding {
	if message == "" {
		return integrityFinding{
			FieldName: fieldName,
			Message:   "The selected record does not exist in your organization",
		}
	}

	return integrityFinding{FieldName: fieldName, Message: message}
}
