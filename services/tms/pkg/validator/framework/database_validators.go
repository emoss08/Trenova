package framework

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/queryutils"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/uptrace/bun"
)

type UniquenessRule struct {
	*ConcreteRule
	getDB         func(context.Context) (*bun.DB, error)
	tableName     string
	modelName     string
	fields        []UniquenessField
	getTenantIDs  func() (organizationID, businessUnitID pulid.ID)
	getPrimaryKey func() string
	isCreate      bool
}

type UniquenessField struct {
	Name     string
	GetValue func() string
	Message  string
}

func NewUniquenessRule(
	name string,
	getDB func(context.Context) (*bun.DB, error),
) *UniquenessRule {
	rule := &UniquenessRule{
		ConcreteRule: NewConcreteRule(name),
		getDB:        getDB,
		fields:       make([]UniquenessField, 0),
	}

	rule.stage = ValidationStageDataIntegrity
	rule.priority = ValidationPriorityHigh

	// Set the validation function
	rule.WithValidation(func(ctx context.Context, multiErr *errortypes.MultiError) error {
		return rule.validateUniqueness(ctx, multiErr)
	})

	return rule
}

func (ur *UniquenessRule) ForTable(tableName string) *UniquenessRule {
	ur.tableName = tableName
	return ur
}

func (ur *UniquenessRule) ForModel(modelName string) *UniquenessRule {
	ur.modelName = modelName
	return ur
}

func (ur *UniquenessRule) WithTenant(
	getTenantIDs func() (organizationID, businessUnitID pulid.ID),
) *UniquenessRule {
	ur.getTenantIDs = getTenantIDs
	return ur
}

func (ur *UniquenessRule) WithPrimaryKey(getPrimaryKey func() string) *UniquenessRule {
	ur.getPrimaryKey = getPrimaryKey
	return ur
}

func (ur *UniquenessRule) ForOperation(isCreate bool) *UniquenessRule {
	ur.isCreate = isCreate
	return ur
}

func (ur *UniquenessRule) CheckField(
	name string,
	getValue func() string,
	messageTemplate string,
) *UniquenessRule {
	ur.fields = append(ur.fields, UniquenessField{
		Name:     name,
		GetValue: getValue,
		Message:  messageTemplate,
	})
	return ur
}

func (ur *UniquenessRule) CheckFieldWithDefault(
	name string,
	getValue func() string,
) *UniquenessRule {
	message := fmt.Sprintf(
		"%s with %s ':value' already exists in the organization.",
		ur.modelName,
		name,
	)
	return ur.CheckField(name, getValue, message)
}

func (ur *UniquenessRule) validateUniqueness(
	ctx context.Context,
	multiErr *errortypes.MultiError,
) error {
	dba, err := ur.getDB(ctx)
	if err != nil {
		return err
	}

	vb := queryutils.NewUniquenessValidator(ur.tableName).
		WithModelName(ur.modelName)

	if ur.getTenantIDs != nil {
		orgID, buID := ur.getTenantIDs()
		vb.WithTenant(orgID, buID)
	}

	for _, field := range ur.fields {
		value := field.GetValue()
		vb.WithFieldAndTemplate(field.Name, value, field.Message, map[string]string{
			"value": value,
		})
	}

	if ur.isCreate {
		vb.WithOperation(queryutils.OperationCreate)
	} else {
		vb.WithOperation(queryutils.OperationUpdate)
		if ur.getPrimaryKey != nil {
			vb.WithPrimaryKey("id", ur.getPrimaryKey())
		}
	}

	queryutils.CheckFieldUniqueness(ctx, dba, vb.Build(), multiErr)

	return nil
}

type UniquenessBuilder struct {
	getDB  func(context.Context) (*bun.DB, error)
	valCtx *validator.ValidationContext
}

func NewUniquenessBuilder(getDB func(context.Context) (*bun.DB, error)) *UniquenessBuilder {
	return &UniquenessBuilder{
		getDB: getDB,
	}
}

func (ub *UniquenessBuilder) WithValidationContext(
	valCtx *validator.ValidationContext,
) *UniquenessBuilder {
	ub.valCtx = valCtx
	return ub
}

func (ub *UniquenessBuilder) Build(cfg *UniquenessConfig) *UniquenessRule {
	rule := NewUniquenessRule(cfg.Name, ub.getDB).
		ForTable(cfg.TableName).
		ForModel(cfg.ModelName).
		WithTenant(cfg.GetTenantIDs)

	if ub.valCtx != nil {
		rule.ForOperation(ub.valCtx.IsCreate)
		if !ub.valCtx.IsCreate && cfg.GetPrimaryKey != nil {
			rule.WithPrimaryKey(cfg.GetPrimaryKey)
		}
	}

	for _, field := range cfg.Fields {
		if field.Message != "" {
			rule.CheckField(field.Name, field.GetValue, field.Message)
		} else {
			rule.CheckFieldWithDefault(field.Name, field.GetValue)
		}
	}

	return rule
}

type UniquenessConfig struct {
	Name          string
	TableName     string
	ModelName     string
	GetTenantIDs  func() (organizationID, businessUnitID pulid.ID)
	GetPrimaryKey func() string
	Fields        []UniquenessFieldConfig
}

type UniquenessFieldConfig struct {
	Name     string
	GetValue func() string
	Message  string
}

type GenericUniquenessValidator[T any] struct {
	getDB         func(context.Context) (*bun.DB, error)
	getTableName  func(T) string
	getModelName  func(T) string
	getTenantIDs  func(T) (organizationID, businessUnitID pulid.ID)
	getPrimaryKey func(T) string
	fields        []GenericUniquenessField[T]
}

type GenericUniquenessField[T any] struct {
	Name     string
	GetValue func(T) string
	Message  string
}

func NewGenericUniquenessValidator[T any](
	getDB func(context.Context) (*bun.DB, error),
) *GenericUniquenessValidator[T] {
	return &GenericUniquenessValidator[T]{
		getDB:  getDB,
		fields: make([]GenericUniquenessField[T], 0),
	}
}

func (guv *GenericUniquenessValidator[T]) WithTableName(
	fn func(T) string,
) *GenericUniquenessValidator[T] {
	guv.getTableName = fn
	return guv
}

func (guv *GenericUniquenessValidator[T]) WithModelName(
	fn func(T) string,
) *GenericUniquenessValidator[T] {
	guv.getModelName = fn
	return guv
}

func (guv *GenericUniquenessValidator[T]) WithTenantIDs(
	fn func(T) (organizationID, businessUnitID pulid.ID),
) *GenericUniquenessValidator[T] {
	guv.getTenantIDs = fn
	return guv
}

func (guv *GenericUniquenessValidator[T]) WithPrimaryKey(
	fn func(T) string,
) *GenericUniquenessValidator[T] {
	guv.getPrimaryKey = fn
	return guv
}

func (guv *GenericUniquenessValidator[T]) CheckField(
	name string,
	getValue func(T) string,
	message string,
) *GenericUniquenessValidator[T] {
	guv.fields = append(guv.fields, GenericUniquenessField[T]{
		Name:     name,
		GetValue: getValue,
		Message:  message,
	})
	return guv
}

func (guv *GenericUniquenessValidator[T]) CreateRule(
	entity T,
	valCtx *validator.ValidationContext,
) *UniquenessRule {
	rule := NewUniquenessRule("uniqueness_check", guv.getDB)

	if guv.getTableName != nil {
		rule.ForTable(guv.getTableName(entity))
	}

	if guv.getModelName != nil {
		rule.ForModel(guv.getModelName(entity))
	}

	if guv.getTenantIDs != nil {
		rule.WithTenant(func() (organizationID, businessUnitID pulid.ID) {
			return guv.getTenantIDs(entity)
		})
	}

	rule.ForOperation(valCtx.IsCreate)

	if !valCtx.IsCreate && guv.getPrimaryKey != nil {
		rule.WithPrimaryKey(func() string {
			return guv.getPrimaryKey(entity)
		})
	}

	for _, field := range guv.fields {
		rule.CheckField(field.Name, func() string {
			return field.GetValue(entity)
		}, field.Message)
	}

	return rule
}
