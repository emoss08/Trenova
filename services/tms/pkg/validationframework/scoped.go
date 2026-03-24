package validationframework

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type ScopedEntity interface {
	GetID() pulid.ID
	GetTableName() string
	Validate(multiErr *errortypes.MultiError)
}

type ScopeFieldConfig[T ScopedEntity] struct {
	FieldName     string
	Column        string
	CaseSensitive bool
	GetValue      func(T) any
}

type ScopedUniqueFieldConfig[T ScopedEntity] struct {
	FieldName     string
	Column        string
	Message       string
	CaseSensitive bool
	GetValue      func(T) any
}

type ScopedValidationContext struct {
	Mode     ValidationMode
	EntityID pulid.ID
}

func (c *ScopedValidationContext) IsCreate() bool {
	return c.Mode == ModeCreate
}

func (c *ScopedValidationContext) IsUpdate() bool {
	return c.Mode == ModeUpdate
}

type ScopedValidateFn[T ScopedEntity] func(
	ctx context.Context,
	entity T,
	valCtx *ScopedValidationContext,
	multiErr *errortypes.MultiError,
) error

type ScopedRule[T ScopedEntity] interface {
	Name() string
	Stage() ValidationStage
	Priority() ValidationPriority
	ShouldRun(valCtx *ScopedValidationContext) bool
	Validate(
		ctx context.Context,
		entity T,
		valCtx *ScopedValidationContext,
		multiErr *errortypes.MultiError,
	) error
}

type BaseScopedRule[T ScopedEntity] struct {
	name       string
	stage      ValidationStage
	priority   ValidationPriority
	onCreate   bool
	onUpdate   bool
	validateFn ScopedValidateFn[T]
}

func NewScopedRule[T ScopedEntity](name string) *BaseScopedRule[T] {
	return &BaseScopedRule[T]{
		name:     name,
		stage:    ValidationStageBusinessRules,
		priority: ValidationPriorityMedium,
		onCreate: true,
		onUpdate: true,
	}
}

func (r *BaseScopedRule[T]) Name() string {
	return r.name
}

func (r *BaseScopedRule[T]) Stage() ValidationStage {
	return r.stage
}

func (r *BaseScopedRule[T]) Priority() ValidationPriority {
	return r.priority
}

func (r *BaseScopedRule[T]) OnCreate() *BaseScopedRule[T] {
	r.onCreate = true
	r.onUpdate = false
	return r
}

func (r *BaseScopedRule[T]) OnUpdate() *BaseScopedRule[T] {
	r.onCreate = false
	r.onUpdate = true
	return r
}

func (r *BaseScopedRule[T]) OnBoth() *BaseScopedRule[T] {
	r.onCreate = true
	r.onUpdate = true
	return r
}

func (r *BaseScopedRule[T]) WithStage(stage ValidationStage) *BaseScopedRule[T] {
	r.stage = stage
	return r
}

func (r *BaseScopedRule[T]) WithPriority(priority ValidationPriority) *BaseScopedRule[T] {
	r.priority = priority
	return r
}

func (r *BaseScopedRule[T]) WithValidation(
	fn ScopedValidateFn[T],
) *BaseScopedRule[T] {
	r.validateFn = fn
	return r
}

func (r *BaseScopedRule[T]) ShouldRun(valCtx *ScopedValidationContext) bool {
	if valCtx.IsCreate() && r.onCreate {
		return true
	}
	if valCtx.IsUpdate() && r.onUpdate {
		return true
	}
	return false
}

func (r *BaseScopedRule[T]) Validate(
	ctx context.Context,
	entity T,
	valCtx *ScopedValidationContext,
	multiErr *errortypes.MultiError,
) error {
	if r.validateFn == nil {
		return nil
	}
	return r.validateFn(ctx, entity, valCtx, multiErr)
}

type ScopedValidator[T ScopedEntity] struct {
	modelName         string
	uniquenessChecker UniquenessChecker
	scopeFields       []ScopeFieldConfig[T]
	uniqueFields      []ScopedUniqueFieldConfig[T]
	customRules       []ScopedRule[T]
	engineConfig      *EngineConfig
}

type ScopedValidatorBuilder[T ScopedEntity] struct {
	validator *ScopedValidator[T]
}

func NewScopedValidatorBuilder[T ScopedEntity]() *ScopedValidatorBuilder[T] {
	return &ScopedValidatorBuilder[T]{
		validator: &ScopedValidator[T]{
			modelName:    "Entity",
			scopeFields:  make([]ScopeFieldConfig[T], 0),
			uniqueFields: make([]ScopedUniqueFieldConfig[T], 0),
			customRules:  make([]ScopedRule[T], 0),
			engineConfig: DefaultEngineConfig(),
		},
	}
}

func (b *ScopedValidatorBuilder[T]) WithModelName(name string) *ScopedValidatorBuilder[T] {
	b.validator.modelName = name
	return b
}

func (b *ScopedValidatorBuilder[T]) WithUniquenessChecker(
	checker UniquenessChecker,
) *ScopedValidatorBuilder[T] {
	b.validator.uniquenessChecker = checker
	return b
}

func (b *ScopedValidatorBuilder[T]) WithScopeField(
	fieldName, column string,
	getter func(T) any,
) *ScopedValidatorBuilder[T] {
	b.validator.scopeFields = append(b.validator.scopeFields, ScopeFieldConfig[T]{
		FieldName:     fieldName,
		Column:        column,
		CaseSensitive: false,
		GetValue:      getter,
	})
	return b
}

func (b *ScopedValidatorBuilder[T]) WithCaseSensitiveScopeField(
	fieldName, column string,
	getter func(T) any,
) *ScopedValidatorBuilder[T] {
	b.validator.scopeFields = append(b.validator.scopeFields, ScopeFieldConfig[T]{
		FieldName:     fieldName,
		Column:        column,
		CaseSensitive: true,
		GetValue:      getter,
	})
	return b
}

func (b *ScopedValidatorBuilder[T]) WithUniqueField(
	fieldName, column, message string,
	getter func(T) any,
) *ScopedValidatorBuilder[T] {
	b.validator.uniqueFields = append(b.validator.uniqueFields, ScopedUniqueFieldConfig[T]{
		FieldName:     fieldName,
		Column:        column,
		Message:       message,
		CaseSensitive: false,
		GetValue:      getter,
	})
	return b
}

func (b *ScopedValidatorBuilder[T]) WithCaseSensitiveUniqueField(
	fieldName, column, message string,
	getter func(T) any,
) *ScopedValidatorBuilder[T] {
	b.validator.uniqueFields = append(b.validator.uniqueFields, ScopedUniqueFieldConfig[T]{
		FieldName:     fieldName,
		Column:        column,
		Message:       message,
		CaseSensitive: true,
		GetValue:      getter,
	})
	return b
}

func (b *ScopedValidatorBuilder[T]) WithCustomRule(
	rule ScopedRule[T],
) *ScopedValidatorBuilder[T] {
	b.validator.customRules = append(b.validator.customRules, rule)
	return b
}

func (b *ScopedValidatorBuilder[T]) WithEngineConfig(
	config *EngineConfig,
) *ScopedValidatorBuilder[T] {
	b.validator.engineConfig = config
	return b
}

func (b *ScopedValidatorBuilder[T]) Build() *ScopedValidator[T] {
	return b.validator
}

func (v *ScopedValidator[T]) ValidateCreate(
	ctx context.Context,
	entity T,
) *errortypes.MultiError {
	valCtx := &ScopedValidationContext{Mode: ModeCreate}
	return v.validate(ctx, entity, valCtx)
}

func (v *ScopedValidator[T]) ValidateUpdate(
	ctx context.Context,
	entity T,
) *errortypes.MultiError {
	valCtx := &ScopedValidationContext{Mode: ModeUpdate, EntityID: entity.GetID()}
	return v.validate(ctx, entity, valCtx)
}

func (v *ScopedValidator[T]) validate(
	ctx context.Context,
	entity T,
	valCtx *ScopedValidationContext,
) *errortypes.MultiError {
	engine := NewEngine(v.engineConfig)

	engine.AddRule(v.createDomainValidationRule(entity))

	if valCtx.IsCreate() {
		engine.AddRule(v.createIDValidationRule(entity))
	}

	if v.uniquenessChecker != nil && len(v.uniqueFields) > 0 {
		engine.AddRule(v.createUniquenessRule(entity, valCtx))
	}

	for _, rule := range v.customRules {
		if rule.ShouldRun(valCtx) {
			engine.AddRule(v.wrapScopedRule(rule, entity, valCtx))
		}
	}

	return engine.Validate(ctx)
}

func (v *ScopedValidator[T]) createDomainValidationRule(entity T) ValidationRule {
	return NewConcreteRule("domain_validation").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			entity.Validate(multiErr)
			return nil
		})
}

func (v *ScopedValidator[T]) createIDValidationRule(entity T) ValidationRule {
	return NewConcreteRule("id_validation").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			if entity.GetID().IsNotNil() {
				multiErr.Add("id", errortypes.ErrInvalid, "ID must not be set on create")
			}
			return nil
		})
}

func (v *ScopedValidator[T]) createUniquenessRule(
	entity T,
	valCtx *ScopedValidationContext,
) ValidationRule {
	return NewConcreteRule("scoped_uniqueness_validation").
		WithStage(ValidationStageDataIntegrity).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(ctx context.Context, multiErr *errortypes.MultiError) error {
			scopeChecks := make([]FieldCheck, 0, len(v.scopeFields))
			for _, scopeField := range v.scopeFields {
				value := scopeField.GetValue(entity)
				if value == nil || value == "" {
					continue
				}
				scopeChecks = append(scopeChecks, FieldCheck{
					Column:        scopeField.Column,
					Value:         value,
					CaseSensitive: scopeField.CaseSensitive,
				})
			}

			for _, field := range v.uniqueFields {
				value := field.GetValue(entity)
				if value == nil || value == "" {
					continue
				}

				req := &UniquenessRequest{
					TableName:   entity.GetTableName(),
					ExcludeID:   valCtx.EntityID,
					ScopeFields: scopeChecks,
					Fields: []FieldCheck{{
						Column:        field.Column,
						Value:         value,
						CaseSensitive: field.CaseSensitive,
					}},
				}

				exists, err := v.uniquenessChecker.CheckUniqueness(ctx, req)
				if err != nil {
					return fmt.Errorf(
						"failed to check uniqueness for field %s: %w",
						field.FieldName,
						err,
					)
				}

				if exists {
					message := field.Message
					if message == "" {
						message = fmt.Sprintf(
							"%s with this %s already exists",
							v.modelName,
							field.FieldName,
						)
					}
					multiErr.Add(field.FieldName, errortypes.ErrDuplicate, message)
				}
			}

			return nil
		})
}

func (v *ScopedValidator[T]) wrapScopedRule(
	rule ScopedRule[T],
	entity T,
	valCtx *ScopedValidationContext,
) ValidationRule {
	return NewConcreteRule(rule.Name()).
		WithStage(rule.Stage()).
		WithPriority(rule.Priority()).
		WithValidation(func(ctx context.Context, multiErr *errortypes.MultiError) error {
			return rule.Validate(ctx, entity, valCtx, multiErr)
		})
}
