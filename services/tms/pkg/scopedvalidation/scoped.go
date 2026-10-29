package scopedvalidation

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
)

type Entity interface {
	GetID() pulid.ID
	GetTableName() string
	Validate(multiErr *errortypes.MultiError)
}

type FieldConfig[T Entity] struct {
	FieldName     string
	Column        string
	CaseSensitive bool
	GetValue      func(T) any
}

type UniqueFieldConfig[T Entity] struct {
	FieldName     string
	Column        string
	Message       string
	CaseSensitive bool
	GetValue      func(T) any
}

type ValidationContext struct {
	Mode     validationframework.ValidationMode
	EntityID pulid.ID
}

func (c *ValidationContext) IsCreate() bool {
	return c.Mode == validationframework.ModeCreate
}

func (c *ValidationContext) IsUpdate() bool {
	return c.Mode == validationframework.ModeUpdate
}

type ValidateFn[T Entity] func(
	ctx context.Context,
	entity T,
	valCtx *ValidationContext,
	multiErr *errortypes.MultiError,
) error

type Rule[T Entity] interface {
	Name() string
	Stage() validationframework.ValidationStage
	Priority() validationframework.ValidationPriority
	ExecutionMode() validationframework.ValidationExecutionMode
	ShouldRun(valCtx *ValidationContext) bool
	Validate(
		ctx context.Context,
		entity T,
		valCtx *ValidationContext,
		multiErr *errortypes.MultiError,
	) error
}

type BaseRule[T Entity] struct {
	name          string
	stage         validationframework.ValidationStage
	priority      validationframework.ValidationPriority
	executionMode validationframework.ValidationExecutionMode
	onCreate      bool
	onUpdate      bool
	validateFn    ValidateFn[T]
}

func NewRule[T Entity](name string) *BaseRule[T] {
	return &BaseRule[T]{
		name:          name,
		stage:         validationframework.ValidationStageBusinessRules,
		priority:      validationframework.ValidationPriorityMedium,
		executionMode: validationframework.ValidationExecutionModeSerial,
		onCreate:      true,
		onUpdate:      true,
	}
}

func (r *BaseRule[T]) Name() string {
	return r.name
}

func (r *BaseRule[T]) Stage() validationframework.ValidationStage {
	return r.stage
}

func (r *BaseRule[T]) Priority() validationframework.ValidationPriority {
	return r.priority
}

func (r *BaseRule[T]) OnCreate() *BaseRule[T] {
	r.onCreate = true
	r.onUpdate = false
	return r
}

func (r *BaseRule[T]) OnUpdate() *BaseRule[T] {
	r.onCreate = false
	r.onUpdate = true
	return r
}

func (r *BaseRule[T]) OnBoth() *BaseRule[T] {
	r.onCreate = true
	r.onUpdate = true
	return r
}

func (r *BaseRule[T]) WithStage(stage validationframework.ValidationStage) *BaseRule[T] {
	r.stage = stage
	return r
}

func (r *BaseRule[T]) WithPriority(priority validationframework.ValidationPriority) *BaseRule[T] {
	r.priority = priority
	return r
}

func (r *BaseRule[T]) WithExecutionMode(
	mode validationframework.ValidationExecutionMode,
) *BaseRule[T] {
	r.executionMode = mode
	return r
}

func (r *BaseRule[T]) WithParallelSafeExecution() *BaseRule[T] {
	r.executionMode = validationframework.ValidationExecutionModeParallelSafe
	return r
}

func (r *BaseRule[T]) WithValidation(
	fn ValidateFn[T],
) *BaseRule[T] {
	r.validateFn = fn
	return r
}

func (r *BaseRule[T]) ExecutionMode() validationframework.ValidationExecutionMode {
	return r.executionMode
}

func (r *BaseRule[T]) ShouldRun(valCtx *ValidationContext) bool {
	if valCtx.IsCreate() && r.onCreate {
		return true
	}
	if valCtx.IsUpdate() && r.onUpdate {
		return true
	}
	return false
}

func (r *BaseRule[T]) Validate(
	ctx context.Context,
	entity T,
	valCtx *ValidationContext,
	multiErr *errortypes.MultiError,
) error {
	if r.validateFn == nil {
		return nil
	}
	return r.validateFn(ctx, entity, valCtx, multiErr)
}

type EntityValidator[T Entity] interface {
	ValidateCreate(ctx context.Context, entity T) *errortypes.MultiError
	ValidateUpdate(ctx context.Context, entity T) *errortypes.MultiError
}

type Validator[T Entity] struct {
	modelName         string
	uniquenessChecker validationframework.UniquenessChecker
	scopeFields       []FieldConfig[T]
	uniqueFields      []UniqueFieldConfig[T]
	customRules       []Rule[T]
	engineConfig      *validationframework.EngineConfig
}

type Builder[T Entity] struct {
	validator *Validator[T]
}

func NewBuilder[T Entity]() *Builder[T] {
	return &Builder[T]{
		validator: &Validator[T]{
			modelName:    "Entity",
			scopeFields:  make([]FieldConfig[T], 0),
			uniqueFields: make([]UniqueFieldConfig[T], 0),
			customRules:  make([]Rule[T], 0),
			engineConfig: validationframework.DefaultEngineConfig(),
		},
	}
}

func (b *Builder[T]) WithModelName(name string) *Builder[T] {
	b.validator.modelName = name
	return b
}

func (b *Builder[T]) WithUniquenessChecker(
	checker validationframework.UniquenessChecker,
) *Builder[T] {
	b.validator.uniquenessChecker = checker
	return b
}

func (b *Builder[T]) WithScopeField(
	fieldName, column string,
	getter func(T) any,
) *Builder[T] {
	b.validator.scopeFields = append(b.validator.scopeFields, FieldConfig[T]{
		FieldName:     fieldName,
		Column:        column,
		CaseSensitive: false,
		GetValue:      getter,
	})
	return b
}

func (b *Builder[T]) WithCaseSensitiveScopeField(
	fieldName, column string,
	getter func(T) any,
) *Builder[T] {
	b.validator.scopeFields = append(b.validator.scopeFields, FieldConfig[T]{
		FieldName:     fieldName,
		Column:        column,
		CaseSensitive: true,
		GetValue:      getter,
	})
	return b
}

func (b *Builder[T]) WithUniqueField(
	fieldName, column, message string,
	getter func(T) any,
) *Builder[T] {
	b.validator.uniqueFields = append(b.validator.uniqueFields, UniqueFieldConfig[T]{
		FieldName:     fieldName,
		Column:        column,
		Message:       message,
		CaseSensitive: false,
		GetValue:      getter,
	})
	return b
}

func (b *Builder[T]) WithCaseSensitiveUniqueField(
	fieldName, column, message string,
	getter func(T) any,
) *Builder[T] {
	b.validator.uniqueFields = append(b.validator.uniqueFields, UniqueFieldConfig[T]{
		FieldName:     fieldName,
		Column:        column,
		Message:       message,
		CaseSensitive: true,
		GetValue:      getter,
	})
	return b
}

func (b *Builder[T]) WithCustomRule(
	rule Rule[T],
) *Builder[T] {
	b.validator.customRules = append(b.validator.customRules, rule)
	return b
}

func (b *Builder[T]) WithEngineConfig(
	config *validationframework.EngineConfig,
) *Builder[T] {
	b.validator.engineConfig = config
	return b
}

func (b *Builder[T]) Build() *Validator[T] {
	return b.validator
}

func (v *Validator[T]) ValidateCreate(
	ctx context.Context,
	entity T,
) *errortypes.MultiError {
	valCtx := &ValidationContext{Mode: validationframework.ModeCreate}
	return v.validate(ctx, entity, valCtx)
}

func (v *Validator[T]) ValidateUpdate(
	ctx context.Context,
	entity T,
) *errortypes.MultiError {
	valCtx := &ValidationContext{Mode: validationframework.ModeUpdate, EntityID: entity.GetID()}
	return v.validate(ctx, entity, valCtx)
}

func (v *Validator[T]) validate(
	ctx context.Context,
	entity T,
	valCtx *ValidationContext,
) *errortypes.MultiError {
	engine := validationframework.NewEngine(v.engineConfig)

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

func (v *Validator[T]) createDomainValidationRule(entity T) validationframework.ValidationRule {
	return validationframework.NewConcreteRule("domain_validation").
		WithStage(validationframework.ValidationStageBasic).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithParallelSafeExecution().
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			entity.Validate(multiErr)
			return nil
		})
}

func (v *Validator[T]) createIDValidationRule(entity T) validationframework.ValidationRule {
	return validationframework.NewConcreteRule("id_validation").
		WithStage(validationframework.ValidationStageBasic).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithParallelSafeExecution().
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			if entity.GetID().IsNotNil() {
				multiErr.Add("id", errortypes.ErrInvalid, "ID must not be set on create")
			}
			return nil
		})
}

func (v *Validator[T]) createUniquenessRule(
	entity T,
	valCtx *ValidationContext,
) validationframework.ValidationRule {
	return validationframework.NewConcreteRule("scoped_uniqueness_validation").
		WithStage(validationframework.ValidationStageDataIntegrity).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(ctx context.Context, multiErr *errortypes.MultiError) error {
			scopeChecks := make([]validationframework.FieldCheck, 0, len(v.scopeFields))
			for _, scopeField := range v.scopeFields {
				value := scopeField.GetValue(entity)
				if value == nil || value == "" {
					continue
				}
				scopeChecks = append(scopeChecks, validationframework.FieldCheck{
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

				req := &validationframework.UniquenessRequest{
					TableName:   entity.GetTableName(),
					ExcludeID:   valCtx.EntityID,
					ScopeFields: scopeChecks,
					Fields: []validationframework.FieldCheck{{
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

func (v *Validator[T]) wrapScopedRule(
	rule Rule[T],
	entity T,
	valCtx *ValidationContext,
) validationframework.ValidationRule {
	return validationframework.NewConcreteRule(rule.Name()).
		WithStage(rule.Stage()).
		WithPriority(rule.Priority()).
		WithExecutionMode(rule.ExecutionMode()).
		WithValidation(func(ctx context.Context, multiErr *errortypes.MultiError) error {
			return rule.Validate(ctx, entity, valCtx, multiErr)
		})
}
