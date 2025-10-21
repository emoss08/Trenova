package queryutils

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/uptrace/bun"
)

type UniquenessError string

const (
	ErrCheckFailed   UniquenessError = "check_failed"
	ErrAlreadyExists UniquenessError = "already_exists"
)

type OperationType string

const (
	OperationCreate OperationType = "create"
	OperationUpdate OperationType = "update"
)

const (
	DefaultCheckFailedTemplate   = "Failed to check if :modelName :fieldName ':value' already exists"
	DefaultAlreadyExistsTemplate = ":modelName with :fieldName ':value' already exists. Please try again with a different :fieldName"
)

type UniqueField struct {
	Name           string
	Value          any
	CaseSensitive  bool
	ErrorFieldName string
	ErrorTemplate  *ErrorTemplate
}

type UniquenessCriteria struct {
	TableName            string
	FieldName            string
	FieldValue           any
	ErrorFieldName       string
	Fields               []UniqueField
	PrimaryKeyField      string // defaults to "id"
	PrimaryKeyValue      string
	OrganizationID       pulid.ID
	BusinessUnitID       pulid.ID
	Operation            OperationType
	ErrorModelName       string
	ErrorMessages        map[UniquenessError]*ErrorTemplate
	AdditionalConditions []WhereCondition
	ErrorHandler         func(field string, err error) error
}

type WhereCondition struct {
	Query string
	Args  []any
}

type ErrorTemplate struct {
	Template string
	Vars     map[string]string
}

func CheckFieldUniqueness(
	ctx context.Context,
	tx bun.IDB,
	criteria *UniquenessCriteria,
	multiErr *errortypes.MultiError,
) {
	if err := validateCriteriaFields(criteria); err != nil {
		multiErr.Add("", errortypes.ErrInvalid, "Invalid validation criteria")
		return
	}

	if criteria.FieldName != "" {
		validateSingleField(ctx, tx, criteria, multiErr)
		return
	}

	for _, field := range criteria.Fields {
		validateFieldUniqueness(ctx, tx, criteria, field, multiErr)
	}
}

func validateFieldUniqueness(
	ctx context.Context,
	tx bun.IDB,
	criteria *UniquenessCriteria,
	field UniqueField,
	multiErr *errortypes.MultiError,
) {
	query := constructUniquenessQuery(tx, criteria, field)
	exists, err := query.Exists(ctx)
	if err != nil {
		handleValidationError(criteria, field, ErrCheckFailed, err, multiErr)
		return
	}

	if exists {
		handleValidationError(criteria, field, ErrAlreadyExists, nil, multiErr)
	}
}

func constructUniquenessQuery(
	tx bun.IDB,
	criteria *UniquenessCriteria,
	field UniqueField,
) *bun.SelectQuery {
	query := tx.NewSelect().TableExpr(criteria.TableName)

	if field.CaseSensitive {
		query = query.Where(fmt.Sprintf("%s.%s = ?", criteria.TableName, field.Name), field.Value)
	} else {
		query = query.Where(fmt.Sprintf("LOWER(%s.%s) = LOWER(?)", criteria.TableName, field.Name), field.Value)
	}

	if criteria.OrganizationID != "" {
		query = query.Where(
			fmt.Sprintf("%s.organization_id = ?", criteria.TableName),
			criteria.OrganizationID,
		)
	}
	if criteria.BusinessUnitID != "" {
		query = query.Where(
			fmt.Sprintf("%s.business_unit_id = ?", criteria.TableName),
			criteria.BusinessUnitID,
		)
	}

	if criteria.Operation == OperationUpdate && criteria.PrimaryKeyValue != "" {
		pkField := criteria.PrimaryKeyField
		if pkField == "" {
			pkField = "id"
		}
		query = query.Where(
			fmt.Sprintf("%s.%s != ?", criteria.TableName, pkField),
			criteria.PrimaryKeyValue,
		)
	}

	for _, cond := range criteria.AdditionalConditions {
		query = query.Where(cond.Query, cond.Args...)
	}

	return query
}

func validateCriteriaFields(criteria *UniquenessCriteria) error {
	if criteria.TableName == "" {
		return ErrTableRequired
	}

	if criteria.ErrorModelName == "" {
		return ErrModelNameRequired
	}

	if criteria.FieldName != "" {
		if criteria.FieldValue == nil {
			return ErrFieldValueRequired
		}
		if criteria.ErrorFieldName == "" {
			return ErrFieldErrorNameRequired
		}
		return nil
	}

	if len(criteria.Fields) == 0 {
		return ErrAtLeastOneFieldRequired
	}

	for _, field := range criteria.Fields {
		if field.Name == "" {
			return ErrFieldNameRequired
		}
		if field.Value == nil {
			return ErrFieldValueRequired
		}
	}

	return nil
}

func validateSingleField(
	ctx context.Context,
	tx bun.IDB,
	criteria *UniquenessCriteria,
	multiErr *errortypes.MultiError,
) {
	field := UniqueField{
		Name:           criteria.FieldName,
		Value:          criteria.FieldValue,
		ErrorFieldName: criteria.ErrorFieldName,
		ErrorTemplate:  criteria.ErrorMessages[ErrAlreadyExists],
	}

	validateFieldUniqueness(ctx, tx, criteria, field, multiErr)
}

func handleValidationError(
	criteria *UniquenessCriteria,
	field UniqueField,
	errType UniquenessError,
	err error,
	multiErr *errortypes.MultiError,
) {
	if criteria.ErrorHandler != nil {
		if handlerErr := criteria.ErrorHandler(field.Name, err); handlerErr != nil {
			multiErr.Add(field.ErrorFieldName, errortypes.ErrInvalid, handlerErr.Error())
			return
		}
	}

	var template *ErrorTemplate
	if field.ErrorTemplate != nil {
		template = field.ErrorTemplate
	} else if tmpl, exists := criteria.ErrorMessages[errType]; exists {
		template = tmpl
	} else {
		switch errType {
		case ErrCheckFailed:
			template = &ErrorTemplate{
				Template: DefaultCheckFailedTemplate,
				Vars: map[string]string{
					"modelName": criteria.ErrorModelName,
					"fieldName": field.Name,
					"value":     fmt.Sprintf("%v", field.Value),
				},
			}
		case ErrAlreadyExists:
			template = &ErrorTemplate{
				Template: DefaultAlreadyExistsTemplate,
				Vars: map[string]string{
					"modelName": criteria.ErrorModelName,
					"fieldName": field.Name,
					"value":     fmt.Sprintf("%v", field.Value),
				},
			}
		}
	}

	message := formatErrorMessage(template)

	switch errType {
	case ErrCheckFailed:
		multiErr.Add(
			field.ErrorFieldName,
			errortypes.ErrInvalid,
			message,
		)
	case ErrAlreadyExists:
		multiErr.Add(
			field.ErrorFieldName,
			errortypes.ErrDuplicate,
			message,
		)
	}
}

func formatErrorMessage(template *ErrorTemplate) string {
	if template == nil {
		return "Validation error occurred"
	}

	message := template.Template
	for key, value := range template.Vars {
		message = strings.ReplaceAll(message, fmt.Sprintf(":%s", key), value)
	}
	return message
}

func CheckCompositeUniqueness(
	ctx context.Context,
	tx bun.IDB,
	tableName string,
	fields map[string]any,
	opts *CompositeUniquenessOptions,
	multiErr *errortypes.MultiError,
) {
	if tableName == "" {
		multiErr.Add(
			"",
			errortypes.ErrInvalid,
			"Invalid validation criteria: table name is required",
		)
		return
	}

	if len(fields) == 0 {
		multiErr.Add(
			"",
			errortypes.ErrInvalid,
			"Invalid validation criteria: at least one field is required",
		)
		return
	}

	query := tx.NewSelect().TableExpr(tableName)

	for name, value := range fields {
		if value == nil {
			continue
		}

		if opts.CaseSensitive {
			query = query.Where(fmt.Sprintf("%s.%s = ?", tableName, name), value)
		} else {
			query = query.Where(fmt.Sprintf("LOWER(%s.%s) = LOWER(?)", tableName, name), value)
		}
	}

	if opts.OrganizationID != "" {
		query = query.Where(fmt.Sprintf("%s.organization_id = ?", tableName), opts.OrganizationID)
	}
	if opts.BusinessUnitID != "" {
		query = query.Where(fmt.Sprintf("%s.business_unit_id = ?", tableName), opts.BusinessUnitID)
	}

	if opts.Operation == OperationUpdate && opts.PrimaryKeyValue != "" {
		pkField := opts.PrimaryKeyField
		if pkField == "" {
			pkField = "id"
		}
		query = query.Where(fmt.Sprintf("%s.%s != ?", tableName, pkField), opts.PrimaryKeyValue)
	}

	for _, cond := range opts.AdditionalConditions {
		query = query.Where(cond.Query, cond.Args...)
	}

	exists, err := query.Exists(ctx)
	if err != nil {
		multiErr.Add(
			opts.ErrorFieldName,
			errortypes.ErrInvalid,
			"Failed to validate uniqueness constraint",
		)
		return
	}

	if exists {
		message := opts.ErrorTemplate
		if message == "" {
			fieldNames := make([]string, 0, len(fields))
			for name := range fields {
				fieldNames = append(fieldNames, name)
			}
			message = fmt.Sprintf(
				"A record with this combination of %s already exists",
				strings.Join(fieldNames, ", "),
			)
		} else {
			for key, value := range opts.ErrorVars {
				message = strings.ReplaceAll(message, fmt.Sprintf(":%s", key), value)
			}
		}

		multiErr.Add(opts.ErrorFieldName, errortypes.ErrDuplicate, message)
	}
}

type CompositeUniquenessOptions struct {
	ModelName            string
	ErrorFieldName       string
	ErrorTemplate        string
	ErrorVars            map[string]string
	CaseSensitive        bool
	PrimaryKeyField      string
	PrimaryKeyValue      string
	OrganizationID       pulid.ID
	BusinessUnitID       pulid.ID
	Operation            OperationType
	AdditionalConditions []WhereCondition
}

type UniquenessValidatorBuilder struct {
	criteria UniquenessCriteria
}

func NewUniquenessValidator(tableName string) *UniquenessValidatorBuilder {
	return &UniquenessValidatorBuilder{
		criteria: UniquenessCriteria{
			TableName:     tableName,
			Fields:        make([]UniqueField, 0),
			ErrorMessages: make(map[UniquenessError]*ErrorTemplate),
		},
	}
}

func (b *UniquenessValidatorBuilder) WithField(name string, value any) *UniquenessValidatorBuilder {
	b.criteria.Fields = append(b.criteria.Fields, UniqueField{
		Name:           name,
		Value:          value,
		ErrorFieldName: name,
		CaseSensitive:  false,
	})
	return b
}

func (b *UniquenessValidatorBuilder) WithCaseSensitiveField(
	name string,
	value any,
) *UniquenessValidatorBuilder {
	b.criteria.Fields = append(b.criteria.Fields, UniqueField{
		Name:           name,
		Value:          value,
		ErrorFieldName: name,
		CaseSensitive:  true,
	})
	return b
}

func (b *UniquenessValidatorBuilder) WithPrimaryKey(
	field, value string,
) *UniquenessValidatorBuilder {
	b.criteria.PrimaryKeyField = field
	b.criteria.PrimaryKeyValue = value
	return b
}

func (b *UniquenessValidatorBuilder) WithBusinessUnit(buID pulid.ID) *UniquenessValidatorBuilder {
	b.criteria.BusinessUnitID = buID
	return b
}

func (b *UniquenessValidatorBuilder) WithTenant(orgID, buID pulid.ID) *UniquenessValidatorBuilder {
	b.criteria.OrganizationID = orgID
	b.criteria.BusinessUnitID = buID
	return b
}

func (b *UniquenessValidatorBuilder) WithModelName(name string) *UniquenessValidatorBuilder {
	b.criteria.ErrorModelName = name
	return b
}

func (b *UniquenessValidatorBuilder) WithCondition(
	query string,
	args ...any,
) *UniquenessValidatorBuilder {
	b.criteria.AdditionalConditions = append(b.criteria.AdditionalConditions, WhereCondition{
		Query: query,
		Args:  args,
	})
	return b
}

func (b *UniquenessValidatorBuilder) WithErrorTemplate(
	errType UniquenessError,
	template string,
	vars map[string]string,
) *UniquenessValidatorBuilder {
	b.criteria.ErrorMessages[errType] = &ErrorTemplate{
		Template: template,
		Vars:     vars,
	}
	return b
}

func (b *UniquenessValidatorBuilder) WithFieldAndTemplate(
	name string,
	value any,
	template string,
	vars map[string]string,
) *UniquenessValidatorBuilder {
	b.criteria.Fields = append(b.criteria.Fields, UniqueField{
		Name:           name,
		Value:          value,
		ErrorFieldName: name,
		CaseSensitive:  false,
		ErrorTemplate: &ErrorTemplate{
			Template: template,
			Vars:     vars,
		},
	})
	return b
}

func (b *UniquenessValidatorBuilder) WithCaseSensitiveFieldAndTemplate(
	name string,
	value any,
	template string,
	vars map[string]string,
) *UniquenessValidatorBuilder {
	b.criteria.Fields = append(b.criteria.Fields, UniqueField{
		Name:           name,
		Value:          value,
		ErrorFieldName: name,
		CaseSensitive:  true,
		ErrorTemplate: &ErrorTemplate{
			Template: template,
			Vars:     vars,
		},
	})
	return b
}

func (b *UniquenessValidatorBuilder) WithOperation(op OperationType) *UniquenessValidatorBuilder {
	b.criteria.Operation = op
	return b
}

func (b *UniquenessValidatorBuilder) Build() *UniquenessCriteria {
	return &b.criteria
}

type CompositeUniquenessValidatorBuilder struct {
	tableName string
	fields    map[string]any
	options   CompositeUniquenessOptions
}

func NewCompositeUniquenessValidator(tableName string) *CompositeUniquenessValidatorBuilder {
	return &CompositeUniquenessValidatorBuilder{
		tableName: tableName,
		fields:    make(map[string]any),
		options:   CompositeUniquenessOptions{},
	}
}

func (b *CompositeUniquenessValidatorBuilder) WithField(
	name string,
	value any,
) *CompositeUniquenessValidatorBuilder {
	b.fields[name] = value
	return b
}

func (b *CompositeUniquenessValidatorBuilder) WithFields(
	fields map[string]any,
) *CompositeUniquenessValidatorBuilder {
	for name, value := range fields {
		b.fields[name] = value
	}
	return b
}

func (b *CompositeUniquenessValidatorBuilder) WithTenant(
	orgID, buID pulid.ID,
) *CompositeUniquenessValidatorBuilder {
	b.options.OrganizationID = orgID
	b.options.BusinessUnitID = buID
	return b
}

func (b *CompositeUniquenessValidatorBuilder) WithErrorField(
	fieldName string,
) *CompositeUniquenessValidatorBuilder {
	b.options.ErrorFieldName = fieldName
	return b
}

func (b *CompositeUniquenessValidatorBuilder) WithErrorTemplate(
	template string,
	vars map[string]string,
) *CompositeUniquenessValidatorBuilder {
	b.options.ErrorTemplate = template
	b.options.ErrorVars = vars
	return b
}

func (b *CompositeUniquenessValidatorBuilder) WithCaseSensitive(
	caseSensitive bool,
) *CompositeUniquenessValidatorBuilder {
	b.options.CaseSensitive = caseSensitive
	return b
}

func (b *CompositeUniquenessValidatorBuilder) WithCondition(
	query string,
	args ...any,
) *CompositeUniquenessValidatorBuilder {
	b.options.AdditionalConditions = append(b.options.AdditionalConditions, WhereCondition{
		Query: query,
		Args:  args,
	})
	return b
}

func (b *CompositeUniquenessValidatorBuilder) ForCreate() *CompositeUniquenessValidatorBuilder {
	b.options.Operation = OperationCreate
	return b
}

func (b *CompositeUniquenessValidatorBuilder) ForUpdate(
	primaryKeyValue string,
) *CompositeUniquenessValidatorBuilder {
	b.options.Operation = OperationUpdate
	b.options.PrimaryKeyValue = primaryKeyValue
	return b
}

func (b *CompositeUniquenessValidatorBuilder) ForUpdateWithPK(
	pkField, pkValue string,
) *CompositeUniquenessValidatorBuilder {
	b.options.Operation = OperationUpdate
	b.options.PrimaryKeyField = pkField
	b.options.PrimaryKeyValue = pkValue
	return b
}

func (b *CompositeUniquenessValidatorBuilder) Validate(
	ctx context.Context,
	tx bun.IDB,
	multiErr *errortypes.MultiError,
) {
	CheckCompositeUniqueness(ctx, tx, b.tableName, b.fields, &b.options, multiErr)
}
