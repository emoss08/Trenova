package validationframework

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type UniqueFieldConfig[T TenantedEntity] struct {
	FieldName     string
	Column        string
	Message       string
	CaseSensitive bool
	GetValue      func(T) any
}

type CompositeUniqueFieldConfig[T TenantedEntity] struct {
	Name    string
	Message string
	Fields  []CompositeField[T]
}

type CompositeField[T TenantedEntity] struct {
	FieldName     string
	Column        string
	CaseSensitive bool
	GetValue      func(T) any
}

type ImmutableFieldConfig[T TenantedEntity] struct {
	FieldName string
	Message   string
	GetValue  func(T) any
}

type DateComparisonConfig[T TenantedEntity] struct {
	FieldName    string
	Message      string
	GetStartDate func(T) *int64
	GetEndDate   func(T) *int64
	AllowEqual   bool
}

type NumericRangeConfig[T TenantedEntity] struct {
	FieldName string
	Message   string
	GetValue  func(T) *float64
	Min       *float64
	Max       *float64
}

type TenantedValidator[T TenantedEntity] struct {
	modelName             string
	uniquenessChecker     UniquenessChecker
	referenceChecker      ReferenceChecker
	uniqueFields          []UniqueFieldConfig[T]
	compositeUniqueFields []CompositeUniqueFieldConfig[T]
	referenceFields       []ReferenceFieldConfig[T]
	immutableFields       []ImmutableFieldConfig[T]
	dateComparisons       []DateComparisonConfig[T]
	numericRanges         []NumericRangeConfig[T]
	customRules           []TenantedRule[T]
	engineConfig          *EngineConfig
}

type TenantedValidatorBuilder[T TenantedEntity] struct {
	validator *TenantedValidator[T]
}

func NewTenantedValidatorBuilder[T TenantedEntity]() *TenantedValidatorBuilder[T] {
	return &TenantedValidatorBuilder[T]{
		validator: &TenantedValidator[T]{
			modelName:             "Entity",
			uniqueFields:          make([]UniqueFieldConfig[T], 0),
			compositeUniqueFields: make([]CompositeUniqueFieldConfig[T], 0),
			referenceFields:       make([]ReferenceFieldConfig[T], 0),
			immutableFields:       make([]ImmutableFieldConfig[T], 0),
			dateComparisons:       make([]DateComparisonConfig[T], 0),
			numericRanges:         make([]NumericRangeConfig[T], 0),
			customRules:           make([]TenantedRule[T], 0),
			engineConfig:          DefaultEngineConfig(),
		},
	}
}

func (b *TenantedValidatorBuilder[T]) WithModelName(name string) *TenantedValidatorBuilder[T] {
	b.validator.modelName = name
	return b
}

func (b *TenantedValidatorBuilder[T]) WithUniquenessChecker(
	checker UniquenessChecker,
) *TenantedValidatorBuilder[T] {
	b.validator.uniquenessChecker = checker
	return b
}

func (b *TenantedValidatorBuilder[T]) WithReferenceChecker(
	checker ReferenceChecker,
) *TenantedValidatorBuilder[T] {
	b.validator.referenceChecker = checker
	return b
}

func (b *TenantedValidatorBuilder[T]) WithUniqueField(
	fieldName, column, message string,
	getter func(T) any,
) *TenantedValidatorBuilder[T] {
	b.validator.uniqueFields = append(b.validator.uniqueFields, UniqueFieldConfig[T]{
		FieldName:     fieldName,
		Column:        column,
		Message:       message,
		CaseSensitive: false,
		GetValue:      getter,
	})
	return b
}

func (b *TenantedValidatorBuilder[T]) WithCaseSensitiveUniqueField(
	fieldName, column, message string,
	getter func(T) any,
) *TenantedValidatorBuilder[T] {
	b.validator.uniqueFields = append(b.validator.uniqueFields, UniqueFieldConfig[T]{
		FieldName:     fieldName,
		Column:        column,
		Message:       message,
		CaseSensitive: true,
		GetValue:      getter,
	})
	return b
}

func (b *TenantedValidatorBuilder[T]) WithCompositeUniqueFields(
	name, message string,
	fields ...CompositeField[T],
) *TenantedValidatorBuilder[T] {
	b.validator.compositeUniqueFields = append(
		b.validator.compositeUniqueFields,
		CompositeUniqueFieldConfig[T]{
			Name:    name,
			Message: message,
			Fields:  fields,
		},
	)
	return b
}

func (b *TenantedValidatorBuilder[T]) WithImmutableField(
	fieldName, message string,
	getter func(T) any,
) *TenantedValidatorBuilder[T] {
	b.validator.immutableFields = append(b.validator.immutableFields, ImmutableFieldConfig[T]{
		FieldName: fieldName,
		Message:   message,
		GetValue:  getter,
	})
	return b
}

func (b *TenantedValidatorBuilder[T]) WithDateAfter(
	fieldName, message string,
	getStart, getEnd func(T) *int64,
) *TenantedValidatorBuilder[T] {
	b.validator.dateComparisons = append(b.validator.dateComparisons, DateComparisonConfig[T]{
		FieldName:    fieldName,
		Message:      message,
		GetStartDate: getStart,
		GetEndDate:   getEnd,
		AllowEqual:   false,
	})
	return b
}

func (b *TenantedValidatorBuilder[T]) WithDateAfterOrEqual(
	fieldName, message string,
	getStart, getEnd func(T) *int64,
) *TenantedValidatorBuilder[T] {
	b.validator.dateComparisons = append(b.validator.dateComparisons, DateComparisonConfig[T]{
		FieldName:    fieldName,
		Message:      message,
		GetStartDate: getStart,
		GetEndDate:   getEnd,
		AllowEqual:   true,
	})
	return b
}

func (b *TenantedValidatorBuilder[T]) WithNumericRange(
	fieldName, message string,
	getValue func(T) *float64,
	minValue, maxValue *float64,
) *TenantedValidatorBuilder[T] {
	b.validator.numericRanges = append(b.validator.numericRanges, NumericRangeConfig[T]{
		FieldName: fieldName,
		Message:   message,
		GetValue:  getValue,
		Min:       minValue,
		Max:       maxValue,
	})
	return b
}

func (b *TenantedValidatorBuilder[T]) WithReferenceCheck(
	fieldName, tableName, message string,
	getter func(T) pulid.ID,
) *TenantedValidatorBuilder[T] {
	b.validator.referenceFields = append(b.validator.referenceFields, ReferenceFieldConfig[T]{
		FieldName: fieldName,
		TableName: tableName,
		Message:   message,
		Optional:  false,
		GetID:     getter,
	})
	return b
}

func (b *TenantedValidatorBuilder[T]) WithOptionalReferenceCheck(
	fieldName, tableName, message string,
	getter func(T) pulid.ID,
) *TenantedValidatorBuilder[T] {
	b.validator.referenceFields = append(b.validator.referenceFields, ReferenceFieldConfig[T]{
		FieldName: fieldName,
		TableName: tableName,
		Message:   message,
		Optional:  true,
		GetID:     getter,
	})
	return b
}

func (b *TenantedValidatorBuilder[T]) WithCustomReferenceCheck(
	fieldName, message string,
	getter func(T) pulid.ID,
	customCheck CustomReferenceCheckFunc,
) *TenantedValidatorBuilder[T] {
	b.validator.referenceFields = append(b.validator.referenceFields, ReferenceFieldConfig[T]{
		FieldName:   fieldName,
		Message:     message,
		Optional:    false,
		GetID:       getter,
		CustomCheck: customCheck,
	})
	return b
}

func (b *TenantedValidatorBuilder[T]) WithOptionalCustomReferenceCheck(
	fieldName, message string,
	getter func(T) pulid.ID,
	customCheck CustomReferenceCheckFunc,
) *TenantedValidatorBuilder[T] {
	b.validator.referenceFields = append(b.validator.referenceFields, ReferenceFieldConfig[T]{
		FieldName:   fieldName,
		Message:     message,
		Optional:    true,
		GetID:       getter,
		CustomCheck: customCheck,
	})
	return b
}

func (b *TenantedValidatorBuilder[T]) WithCustomRule(
	rule TenantedRule[T],
) *TenantedValidatorBuilder[T] {
	b.validator.customRules = append(b.validator.customRules, rule)
	return b
}

func (b *TenantedValidatorBuilder[T]) WithEngineConfig(
	config *EngineConfig,
) *TenantedValidatorBuilder[T] {
	b.validator.engineConfig = config
	return b
}

func (b *TenantedValidatorBuilder[T]) Build() *TenantedValidator[T] {
	return b.validator
}

func (v *TenantedValidator[T]) ValidateCreate(
	ctx context.Context,
	entity T,
) *errortypes.MultiError {
	valCtx := &TenantedValidationContext{
		Mode:           ModeCreate,
		OrganizationID: entity.GetOrganizationID(),
		BusinessUnitID: entity.GetBusinessUnitID(),
	}
	return v.validate(ctx, entity, nil, valCtx)
}

func (v *TenantedValidator[T]) ValidateUpdate(
	ctx context.Context,
	entity T,
) *errortypes.MultiError {
	valCtx := &TenantedValidationContext{
		Mode:           ModeUpdate,
		OrganizationID: entity.GetOrganizationID(),
		BusinessUnitID: entity.GetBusinessUnitID(),
		EntityID:       entity.GetID(),
	}
	return v.validate(ctx, entity, nil, valCtx)
}

func (v *TenantedValidator[T]) ValidateUpdateWithOriginal(
	ctx context.Context,
	entity T,
	original T,
) *errortypes.MultiError {
	valCtx := &TenantedValidationContext{
		Mode:           ModeUpdate,
		OrganizationID: entity.GetOrganizationID(),
		BusinessUnitID: entity.GetBusinessUnitID(),
		EntityID:       entity.GetID(),
	}
	return v.validate(ctx, entity, &original, valCtx)
}

func (v *TenantedValidator[T]) validate(
	ctx context.Context,
	entity T,
	original *T,
	valCtx *TenantedValidationContext,
) *errortypes.MultiError {
	engine := NewEngine(v.engineConfig)

	engine.AddRule(v.createDomainValidationRule(entity))

	if valCtx.IsCreate() {
		engine.AddRule(v.createIDValidationRule(entity))
	}

	if v.uniquenessChecker != nil && len(v.uniqueFields) > 0 {
		engine.AddRule(v.createUniquenessRule(entity, valCtx))
	}

	if v.uniquenessChecker != nil && len(v.compositeUniqueFields) > 0 {
		engine.AddRule(v.createCompositeUniquenessRule(entity, valCtx))
	}

	if v.referenceChecker != nil && len(v.referenceFields) > 0 {
		engine.AddRule(v.createReferenceRule(entity, valCtx))
	}

	if valCtx.IsUpdate() && original != nil && len(v.immutableFields) > 0 {
		engine.AddRule(v.createImmutableFieldRule(entity, *original))
	}

	if len(v.dateComparisons) > 0 {
		engine.AddRule(v.createDateComparisonRule(entity))
	}

	if len(v.numericRanges) > 0 {
		engine.AddRule(v.createNumericRangeRule(entity))
	}

	for _, rule := range v.customRules {
		if rule.ShouldRun(valCtx) {
			engine.AddRule(v.wrapTenantedRule(rule, entity, valCtx))
		}
	}

	return engine.Validate(ctx)
}

func (v *TenantedValidator[T]) createDomainValidationRule(entity T) ValidationRule {
	return NewConcreteRule("domain_validation").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			entity.Validate(multiErr)
			return nil
		})
}

func (v *TenantedValidator[T]) createIDValidationRule(entity T) ValidationRule {
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

func (v *TenantedValidator[T]) createUniquenessRule(
	entity T,
	valCtx *TenantedValidationContext,
) ValidationRule {
	return NewConcreteRule("uniqueness_validation").
		WithStage(ValidationStageDataIntegrity).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(ctx context.Context, multiErr *errortypes.MultiError) error {
			for _, field := range v.uniqueFields {
				value := field.GetValue(entity)
				if value == nil || value == "" {
					continue
				}

				req := &UniquenessRequest{
					TableName:      entity.GetTableName(),
					OrganizationID: valCtx.OrganizationID,
					BusinessUnitID: valCtx.BusinessUnitID,
					ExcludeID:      valCtx.EntityID,
					Fields: []FieldCheck{
						{
							Column:        field.Column,
							Value:         value,
							CaseSensitive: field.CaseSensitive,
						},
					},
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

func (v *TenantedValidator[T]) createReferenceRule(
	entity T,
	valCtx *TenantedValidationContext,
) ValidationRule {
	return NewConcreteRule("reference_validation").
		WithStage(ValidationStageDataIntegrity).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(ctx context.Context, multiErr *errortypes.MultiError) error {
			for _, field := range v.referenceFields {
				refID := field.GetID(entity)

				if refID.IsNil() {
					if field.Optional {
						continue
					}
					message := field.Message
					if message == "" {
						message = fmt.Sprintf("%s is required", field.FieldName)
					}
					multiErr.Add(field.FieldName, errortypes.ErrRequired, message)
					continue
				}

				var exists bool
				var err error

				if field.CustomCheck != nil {
					exists, err = field.CustomCheck(
						ctx,
						valCtx.OrganizationID,
						valCtx.BusinessUnitID,
						refID,
					)
				} else {
					req := &ReferenceRequest{
						TableName:      field.TableName,
						OrganizationID: valCtx.OrganizationID,
						BusinessUnitID: valCtx.BusinessUnitID,
						ID:             refID,
					}
					exists, err = v.referenceChecker.CheckReference(ctx, req)
				}

				if err != nil {
					return fmt.Errorf(
						"failed to check reference for field %s: %w",
						field.FieldName,
						err,
					)
				}

				if !exists {
					message := field.Message
					if message == "" {
						message = fmt.Sprintf(
							"Referenced %s does not exist or belongs to a different organization",
							field.FieldName,
						)
					}
					multiErr.Add(field.FieldName, errortypes.ErrInvalidReference, message)
				}
			}
			return nil
		})
}

func (v *TenantedValidator[T]) createCompositeUniquenessRule(
	entity T,
	valCtx *TenantedValidationContext,
) ValidationRule {
	return NewConcreteRule("composite_uniqueness_validation").
		WithStage(ValidationStageDataIntegrity).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(ctx context.Context, multiErr *errortypes.MultiError) error {
			for _, composite := range v.compositeUniqueFields {
				fieldChecks := make([]FieldCheck, 0, len(composite.Fields))
				allFieldsHaveValues := true

				for _, field := range composite.Fields {
					value := field.GetValue(entity)
					if value == nil || value == "" {
						allFieldsHaveValues = false
						break
					}
					fieldChecks = append(fieldChecks, FieldCheck{
						Column:        field.Column,
						Value:         value,
						CaseSensitive: field.CaseSensitive,
					})
				}

				if !allFieldsHaveValues {
					continue
				}

				req := &UniquenessRequest{
					TableName:      entity.GetTableName(),
					OrganizationID: valCtx.OrganizationID,
					BusinessUnitID: valCtx.BusinessUnitID,
					ExcludeID:      valCtx.EntityID,
					Fields:         fieldChecks,
				}

				exists, err := v.uniquenessChecker.CheckUniqueness(ctx, req)
				if err != nil {
					return fmt.Errorf(
						"failed to check composite uniqueness for %s: %w",
						composite.Name,
						err,
					)
				}

				if exists {
					message := composite.Message
					if message == "" {
						message = fmt.Sprintf(
							"%s with this combination already exists",
							v.modelName,
						)
					}
					multiErr.Add(composite.Name, errortypes.ErrDuplicate, message)
				}
			}
			return nil
		})
}

func (v *TenantedValidator[T]) createImmutableFieldRule(entity, original T) ValidationRule {
	return NewConcreteRule("immutable_field_validation").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			for _, field := range v.immutableFields {
				newValue := field.GetValue(entity)
				oldValue := field.GetValue(original)

				if !valuesEqual(newValue, oldValue) {
					message := field.Message
					if message == "" {
						message = fmt.Sprintf(
							"%s cannot be changed after creation",
							field.FieldName,
						)
					}
					multiErr.Add(field.FieldName, errortypes.ErrInvalidOperation, message)
				}
			}
			return nil
		})
}

func valuesEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func (v *TenantedValidator[T]) createDateComparisonRule(entity T) ValidationRule {
	return NewConcreteRule("date_comparison_validation").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			for _, config := range v.dateComparisons {
				startDate := config.GetStartDate(entity)
				endDate := config.GetEndDate(entity)

				if startDate == nil || endDate == nil {
					continue
				}

				invalid := (*endDate < *startDate) || (!config.AllowEqual && *endDate == *startDate)
				if !invalid {
					continue
				}

				message := config.Message
				if message == "" {
					if config.AllowEqual {
						message = fmt.Sprintf(
							"%s must be on or after the start date",
							config.FieldName,
						)
					} else {
						message = fmt.Sprintf("%s must be after the start date", config.FieldName)
					}
				}
				multiErr.Add(config.FieldName, errortypes.ErrInvalid, message)
			}
			return nil
		})
}

func (v *TenantedValidator[T]) createNumericRangeRule(entity T) ValidationRule {
	return NewConcreteRule("numeric_range_validation").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			for _, config := range v.numericRanges {
				value := config.GetValue(entity)
				if value == nil {
					continue
				}

				if config.Min != nil && *value < *config.Min {
					message := config.Message
					if message == "" {
						message = fmt.Sprintf(
							"%s must be at least %v",
							config.FieldName,
							*config.Min,
						)
					}
					multiErr.Add(config.FieldName, errortypes.ErrInvalid, message)
					continue
				}

				if config.Max != nil && *value > *config.Max {
					message := config.Message
					if message == "" {
						message = fmt.Sprintf(
							"%s must be at most %v",
							config.FieldName,
							*config.Max,
						)
					}
					multiErr.Add(config.FieldName, errortypes.ErrInvalid, message)
				}
			}
			return nil
		})
}

func (v *TenantedValidator[T]) wrapTenantedRule(
	rule TenantedRule[T],
	entity T,
	valCtx *TenantedValidationContext,
) ValidationRule {
	return NewConcreteRule(rule.Name()).
		WithStage(rule.Stage()).
		WithPriority(rule.Priority()).
		WithValidation(func(ctx context.Context, multiErr *errortypes.MultiError) error {
			return rule.Validate(ctx, entity, valCtx, multiErr)
		})
}
