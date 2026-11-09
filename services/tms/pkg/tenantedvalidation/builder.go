package tenantedvalidation

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
)

type UniqueFieldConfig[T validationframework.Entity] struct {
	FieldName     string
	Column        string
	Message       string
	CaseSensitive bool
	GetValue      func(T) any
}

type CompositeUniqueFieldConfig[T validationframework.Entity] struct {
	Name    string
	Message string
	Fields  []CompositeField[T]
}

type CompositeField[T validationframework.Entity] struct {
	FieldName     string
	Column        string
	CaseSensitive bool
	GetValue      func(T) any
}

type ReferenceConfig[T validationframework.Entity] struct {
	FieldName  string
	Table      string
	Column     string
	Message    string
	Optional   bool
	Conditions []validationframework.Condition
	GetID      func(T) pulid.ID
}

type ReferenceTarget struct {
	FieldName string
	ID        pulid.ID
}

type ReferenceListConfig[T validationframework.Entity] struct {
	Table      string
	Column     string
	Message    string
	Conditions []validationframework.Condition
	Targets    func(T) []ReferenceTarget
}

type ImmutableFieldConfig[T validationframework.Entity] struct {
	FieldName string
	Message   string
	GetValue  func(T) any
}

type DateComparisonConfig[T validationframework.Entity] struct {
	FieldName    string
	Message      string
	GetStartDate func(T) *int64
	GetEndDate   func(T) *int64
	AllowEqual   bool
}

type NumericRangeConfig[T validationframework.Entity] struct {
	FieldName string
	Message   string
	GetValue  func(T) *float64
	Min       *float64
	Max       *float64
}

type Builder[T validationframework.Entity] struct {
	validator *Validator[T]
}

func NewBuilder[T validationframework.Entity]() *Builder[T] {
	return &Builder[T]{validator: &Validator[T]{modelName: "Entity"}}
}

func (b *Builder[T]) WithModelName(name string) *Builder[T] {
	b.validator.modelName = name
	return b
}

func (b *Builder[T]) WithIntegrityChecker(
	checker validationframework.IntegrityChecker,
) *Builder[T] {
	b.validator.checker = checker
	return b
}

func (b *Builder[T]) WithIntegrityScope(scope func(context.Context) context.Context) *Builder[T] {
	b.validator.integrityScope = scope
	return b
}

func (b *Builder[T]) WithUniqueField(
	fieldName, column, message string,
	getter func(T) any,
) *Builder[T] {
	b.validator.uniqueFields = append(b.validator.uniqueFields, UniqueFieldConfig[T]{
		FieldName: fieldName,
		Column:    column,
		Message:   message,
		GetValue:  getter,
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

func (b *Builder[T]) WithCompositeUniqueFields(
	name, message string,
	fields ...CompositeField[T],
) *Builder[T] {
	b.validator.compositeUniqueFields = append(
		b.validator.compositeUniqueFields,
		CompositeUniqueFieldConfig[T]{Name: name, Message: message, Fields: fields},
	)
	return b
}

func (b *Builder[T]) WithImmutableField(
	fieldName, message string,
	getter func(T) any,
) *Builder[T] {
	b.validator.immutableFields = append(b.validator.immutableFields, ImmutableFieldConfig[T]{
		FieldName: fieldName,
		Message:   message,
		GetValue:  getter,
	})
	return b
}

func (b *Builder[T]) WithDateAfter(
	fieldName, message string,
	getStart, getEnd func(T) *int64,
) *Builder[T] {
	b.validator.dateComparisons = append(b.validator.dateComparisons, DateComparisonConfig[T]{
		FieldName:    fieldName,
		Message:      message,
		GetStartDate: getStart,
		GetEndDate:   getEnd,
	})
	return b
}

func (b *Builder[T]) WithDateAfterOrEqual(
	fieldName, message string,
	getStart, getEnd func(T) *int64,
) *Builder[T] {
	b.validator.dateComparisons = append(b.validator.dateComparisons, DateComparisonConfig[T]{
		FieldName:    fieldName,
		Message:      message,
		GetStartDate: getStart,
		GetEndDate:   getEnd,
		AllowEqual:   true,
	})
	return b
}

func (b *Builder[T]) WithNumericRange(
	fieldName, message string,
	getValue func(T) *float64,
	minValue, maxValue *float64,
) *Builder[T] {
	b.validator.numericRanges = append(b.validator.numericRanges, NumericRangeConfig[T]{
		FieldName: fieldName,
		Message:   message,
		GetValue:  getValue,
		Min:       minValue,
		Max:       maxValue,
	})
	return b
}

func (b *Builder[T]) WithReferenceCheck(
	fieldName, table, message string,
	getter func(T) pulid.ID,
) *Builder[T] {
	return b.WithReference(ReferenceConfig[T]{
		FieldName: fieldName,
		Table:     table,
		Message:   message,
		GetID:     getter,
	})
}

func (b *Builder[T]) WithOptionalReferenceCheck(
	fieldName, table, message string,
	getter func(T) pulid.ID,
) *Builder[T] {
	return b.WithReference(ReferenceConfig[T]{
		FieldName: fieldName,
		Table:     table,
		Message:   message,
		Optional:  true,
		GetID:     getter,
	})
}

func (b *Builder[T]) WithReference(config ReferenceConfig[T]) *Builder[T] {
	b.validator.references = append(b.validator.references, config)
	return b
}

func (b *Builder[T]) WithReferenceList(config ReferenceListConfig[T]) *Builder[T] {
	b.validator.referenceLists = append(b.validator.referenceLists, config)
	return b
}

func (b *Builder[T]) WithCustomRule(rule validationframework.Rule[T]) *Builder[T] {
	stage := rule.Stage()
	if stage < validationframework.StageBasic || stage > validationframework.StageCompliance {
		panic(fmt.Sprintf("tenantedvalidation: rule %q has unknown stage %d", rule.Name(), stage))
	}
	b.validator.rules[stage] = append(b.validator.rules[stage], rule)
	return b
}

func (b *Builder[T]) Build() *Validator[T] {
	v := b.validator
	if v.checker == nil && v.hasIntegrityChecks() {
		panic(fmt.Sprintf(
			"tenantedvalidation: %s declares uniqueness or reference checks without an integrity checker",
			v.modelName,
		))
	}

	return v
}
