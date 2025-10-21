package framework

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/uptrace/bun"
)

type TenantedValidator[T TenantedEntity] struct {
	entity       T
	valCtx       *validator.ValidationContext
	getDB        func(context.Context) (*bun.DB, error)
	modelName    string
	uniqueFields []UniqueField
	customRules  []ValidationRule
}

func NewTenantedValidator[T TenantedEntity](
	entity T,
	valCtx *validator.ValidationContext,
	getDB func(context.Context) (*bun.DB, error),
) *TenantedValidator[T] {
	return &TenantedValidator[T]{
		entity:    entity,
		valCtx:    valCtx,
		getDB:     getDB,
		modelName: "Entity", // Default, should be overridden
	}
}

func (tv *TenantedValidator[T]) WithModelName(name string) *TenantedValidator[T] {
	tv.modelName = name
	return tv
}

func (tv *TenantedValidator[T]) WithUniqueFields(fields ...UniqueField) *TenantedValidator[T] {
	tv.uniqueFields = fields
	return tv
}

func (tv *TenantedValidator[T]) WithCustomRules(rules ...ValidationRule) *TenantedValidator[T] {
	tv.customRules = rules
	return tv
}

func (tv *TenantedValidator[T]) Validate(ctx context.Context) *errortypes.MultiError {
	engine := NewValidationEngine(&EngineConfig{
		MaxParallel: 5,
		FailFast:    false,
	})

	engine.AddRule(
		NewConcreteRule("domain_validation").
			WithStage(ValidationStageBasic).
			WithPriority(ValidationPriorityHigh).
			WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
				tv.entity.Validate(multiErr)
				return nil
			}),
	)

	if tv.valCtx.IsCreate {
		engine.AddRule(
			NewBusinessRule("id_validation").
				WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
					if id := tv.entity.GetID(); id != "" &&
						id != "00000000-0000-0000-0000-000000000000" {
						multiErr.Add("id", errortypes.ErrInvalid, "ID cannot be set on create")
					}
					return nil
				}),
		)
	}

	if len(tv.uniqueFields) > 0 {
		engine.AddRule(tv.createUniquenessRule())
	}

	for _, rule := range tv.customRules {
		engine.AddRule(rule)
	}

	return engine.Validate(ctx)
}

func (tv *TenantedValidator[T]) createUniquenessRule() *UniquenessRule {
	rule := NewUniquenessRule("uniqueness_check", tv.getDB).
		ForTable(tv.entity.GetTableName()).
		ForModel(tv.modelName).
		ForOperation(tv.valCtx.IsCreate).
		WithTenant(func() (pulid.ID, pulid.ID) {
			return tv.entity.GetOrganizationID(), tv.entity.GetBusinessUnitID()
		})

	if !tv.valCtx.IsCreate {
		rule.WithPrimaryKey(func() string {
			return tv.entity.GetID()
		})
	}

	for _, field := range tv.uniqueFields {
		message := field.Message
		if message == "" {
			message = fmt.Sprintf(
				"%s with %s ':value' already exists in the organization.",
				tv.modelName,
				field.Name,
			)
		}
		rule.CheckField(field.Name, field.GetValue, message)
	}

	return rule
}

type TenantedValidatorFactory[T TenantedEntity] struct {
	getDB        func(context.Context) (*bun.DB, error)
	modelName    string
	uniqueFields func(T) []UniqueField
	customRules  func(T, *validator.ValidationContext) []ValidationRule
}

func NewTenantedValidatorFactory[T TenantedEntity](
	getDB func(context.Context) (*bun.DB, error),
) *TenantedValidatorFactory[T] {
	return &TenantedValidatorFactory[T]{
		getDB:     getDB,
		modelName: "Entity", // Default
	}
}

func (tvf *TenantedValidatorFactory[T]) WithModelName(name string) *TenantedValidatorFactory[T] {
	tvf.modelName = name
	return tvf
}

func (tvf *TenantedValidatorFactory[T]) WithUniqueFields(
	fn func(T) []UniqueField,
) *TenantedValidatorFactory[T] {
	tvf.uniqueFields = fn
	return tvf
}

func (tvf *TenantedValidatorFactory[T]) WithCustomRules(
	fn func(T, *validator.ValidationContext) []ValidationRule,
) *TenantedValidatorFactory[T] {
	tvf.customRules = fn
	return tvf
}

func (tvf *TenantedValidatorFactory[T]) CreateValidator(
	entity T,
	valCtx *validator.ValidationContext,
) *TenantedValidator[T] {
	val := NewTenantedValidator(entity, valCtx, tvf.getDB).
		WithModelName(tvf.modelName)

	if tvf.uniqueFields != nil {
		val.WithUniqueFields(tvf.uniqueFields(entity)...)
	}

	if tvf.customRules != nil {
		val.WithCustomRules(tvf.customRules(entity, valCtx)...)
	}

	return val
}

func (tvf *TenantedValidatorFactory[T]) Validate(
	ctx context.Context,
	entity T,
	valCtx *validator.ValidationContext,
) *errortypes.MultiError {
	val := tvf.CreateValidator(entity, valCtx)
	return val.Validate(ctx)
}
