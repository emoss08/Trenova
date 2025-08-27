/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package queryutils

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog/log"
	"github.com/uptrace/bun"
)

// UniquenessError represents possible error types in uniqueness validation
type UniquenessError string

const (
	ErrCheckFailed   UniquenessError = "check_failed"
	ErrAlreadyExists UniquenessError = "already_exists"
)

// OperationType represents the database operation being performed
type OperationType string

const (
	OperationCreate OperationType = "create"
	OperationUpdate OperationType = "update"
)

// Default error message templates
const (
	DefaultCheckFailedTemplate   = "Failed to check if :modelName :fieldName ':value' already exists"
	DefaultAlreadyExistsTemplate = ":modelName with :fieldName ':value' already exists. Please try again with a different :fieldName"
)

// UniqueField defines a field to be checked for uniqueness in the database.
// It supports both case-sensitive and case-insensitive comparison modes and
// allows custom error messaging per field.
type UniqueField struct {
	// Name is the database column name to check
	Name string

	// Value is the value to check for uniqueness
	Value any

	// CaseSensitive determines if the uniqueness check should be case-sensitive
	// When false, the check uses LOWER() for comparison
	CaseSensitive bool

	// ErrorFieldName is the field name to use in error messages
	// If empty, Name will be used
	ErrorFieldName string

	// ErrorTemplate provides custom error messaging for this field
	// If nil, falls back to criteria-level templates
	ErrorTemplate *ErrorTemplate
}

// UniquenessCriteria defines the criteria for uniqueness validation
type UniquenessCriteria struct {
	// The name of the table to check uniqueness on
	TableName string

	// Single field validation (backward compatibility)
	FieldName      string
	FieldValue     any
	ErrorFieldName string

	// Multiple field validation
	Fields []UniqueField

	// Primary key configuration
	PrimaryKeyField string // defaults to "id"
	PrimaryKeyValue string

	// Tenant configuration
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID

	// Operation being performed (create or update)
	Operation OperationType

	// Error configuration
	ErrorModelName string
	ErrorMessages  map[UniquenessError]*ErrorTemplate

	// Additional WHERE conditions for the uniqueness check
	AdditionalConditions []WhereCondition

	// Custom error handler
	ErrorHandler func(field string, err error) error
}

// WhereCondition represents an additional WHERE clause for uniqueness checking.
// Use this to add custom filtering beyond the standard tenant and field comparisons.
type WhereCondition struct {
	// Query is the SQL WHERE clause, may contain placeholders
	Query string

	// Args are the values for the Query placeholders
	Args []any
}

// ErrorTemplate represents a template for error messages
type ErrorTemplate struct {
	Template string
	Vars     map[string]string
}

// CheckFieldUniqueness performs uniqueness validation for database fields with enhanced error handling.
// It supports both legacy single-field validation and modern multiple-field validation through the
// UniquenessCriteria struct. Validation results are collected in the provided multiErr.
//
// For single field validation (legacy):
//   - Uses FieldName, FieldValue, and ErrorFieldName from criteria
//
// For multiple field validation:
//   - Iterates through Fields slice in criteria
//   - Supports case-sensitive and case-insensitive comparison
//   - Allows custom error templates per field
func CheckFieldUniqueness(
	ctx context.Context,
	tx bun.IDB,
	criteria *UniquenessCriteria,
	multiErr *errors.MultiError,
) {
	logger := log.With().
		Str("operation", "ValidateUniqueness").
		Str("table", criteria.TableName).
		Logger()

	if err := validateCriteriaFields(criteria); err != nil {
		logger.Error().Err(err).Msg("invalid uniqueness criteria")
		multiErr.Add("", errors.ErrInvalid, "Invalid validation criteria")
		return
	}

	// Handle legacy single field validation
	if criteria.FieldName != "" {
		validateSingleField(ctx, tx, criteria, multiErr)
		return
	}

	// Handle multiple field validation
	for _, field := range criteria.Fields {
		validateFieldUniqueness(ctx, tx, criteria, field, multiErr)
	}
}

// validateFieldUniqueness performs the actual uniqueness check for a single field.
// It constructs and executes the uniqueness query, then handles any errors that occur.
func validateFieldUniqueness(
	ctx context.Context,
	tx bun.IDB,
	criteria *UniquenessCriteria,
	field UniqueField,
	multiErr *errors.MultiError,
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

// constructUniquenessQuery builds the SQL query for checking field uniqueness.
// The query includes:
//   - Field comparison (case-sensitive or case-insensitive)
//   - Tenant filtering (organization and business unit)
//   - Primary key exclusion for updates
//   - Additional custom conditions
func constructUniquenessQuery(
	tx bun.IDB,
	criteria *UniquenessCriteria,
	field UniqueField,
) *bun.SelectQuery {
	query := tx.NewSelect().TableExpr(criteria.TableName)

	// Build the field comparison
	if field.CaseSensitive {
		query = query.Where(fmt.Sprintf("%s.%s = ?", criteria.TableName, field.Name), field.Value)
	} else {
		query = query.Where(fmt.Sprintf("LOWER(%s.%s) = LOWER(?)", criteria.TableName, field.Name), field.Value)
	}

	// Add tenant conditions
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

	// Add primary key exclusion for updates
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

// validateCriteriaFields ensures all required fields are present in the UniquenessCriteria.
// It validates:
//   - Required table name and model name
//   - For legacy validation: field name, value, and error field name
//   - For multiple field validation: at least one field with name and value
func validateCriteriaFields(criteria *UniquenessCriteria) error {
	if criteria.TableName == "" {
		return eris.New("table name is required")
	}

	if criteria.ErrorModelName == "" {
		return eris.New("error model name is required")
	}

	// For legacy single field validation
	if criteria.FieldName != "" {
		if criteria.FieldValue == nil {
			return eris.New("field value is required")
		}
		if criteria.ErrorFieldName == "" {
			return eris.New("error field name is required")
		}
		return nil
	}

	// For multiple field validation
	if len(criteria.Fields) == 0 {
		return eris.New("at least one field is required for validation")
	}

	for _, field := range criteria.Fields {
		if field.Name == "" {
			return eris.New("field name is required")
		}
		if field.Value == nil {
			return eris.New("field value is required")
		}
	}

	return nil
}

// validateSingleField handles legacy single field validation
func validateSingleField(
	ctx context.Context,
	tx bun.IDB,
	criteria *UniquenessCriteria,
	multiErr *errors.MultiError,
) {
	field := UniqueField{
		Name:           criteria.FieldName,
		Value:          criteria.FieldValue,
		ErrorFieldName: criteria.ErrorFieldName,
		ErrorTemplate:  criteria.ErrorMessages[ErrAlreadyExists],
	}

	validateFieldUniqueness(ctx, tx, criteria, field, multiErr)
}

// handleValidationError processes uniqueness validation errors and adds them to multiErr.
// Error handling priority:
//  1. Custom error handler if provided
//  2. Field-specific error template
//  3. Criteria-level error template
//  4. Default error template
func handleValidationError(
	criteria *UniquenessCriteria,
	field UniqueField,
	errType UniquenessError,
	err error,
	multiErr *errors.MultiError,
) {
	logger := log.With().
		Str("operation", "handleError").
		Str("field", field.Name).
		Str("errorType", string(errType)).
		Logger()

	// Use custom error handler if provided
	if criteria.ErrorHandler != nil {
		if handlerErr := criteria.ErrorHandler(field.Name, err); handlerErr != nil {
			multiErr.Add(field.ErrorFieldName, errors.ErrInvalid, handlerErr.Error())
			return
		}
	}

	// Get error message template
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

	// Format error message
	message := formatErrorMessage(template)

	// Add error to multiErr
	switch errType {
	case ErrCheckFailed:
		logger.Error().Err(err).Msg("failed to check uniqueness")
		multiErr.Add(
			field.ErrorFieldName,
			errors.ErrInvalid,
			message,
		)
	case ErrAlreadyExists:
		multiErr.Add(
			field.ErrorFieldName,
			errors.ErrDuplicate,
			message,
		)
	}
}

// formatErrorMessage formats an error message by replacing variables in the template
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

// CheckCompositeUniqueness performs uniqueness validation for multiple fields as a composite constraint.
// Unlike CheckFieldUniqueness which validates each field separately, this checks the combination
// of all fields together as a single unique constraint.
//
// Example usage for a constraint like:
// UNIQUE(shipment_id, organization_id, business_unit_id, type) WHERE released_at IS NULL
func CheckCompositeUniqueness(
	ctx context.Context,
	tx bun.IDB,
	tableName string,
	fields map[string]any,
	opts *CompositeUniquenessOptions,
	multiErr *errors.MultiError,
) {
	logger := log.With().
		Str("operation", "CheckCompositeUniqueness").
		Str("table", tableName).
		Logger()

	if tableName == "" {
		logger.Error().Msg("table name is required")
		multiErr.Add("", errors.ErrInvalid, "Invalid validation criteria: table name is required")
		return
	}

	if len(fields) == 0 {
		logger.Error().Msg("at least one field is required for composite uniqueness")
		multiErr.Add(
			"",
			errors.ErrInvalid,
			"Invalid validation criteria: at least one field is required",
		)
		return
	}

	// Build the query
	query := tx.NewSelect().TableExpr(tableName)

	// Add all composite fields to the WHERE clause
	for name, value := range fields {
		if value == nil {
			continue // Skip nil values
		}

		if opts.CaseSensitive {
			query = query.Where(fmt.Sprintf("%s.%s = ?", tableName, name), value)
		} else {
			query = query.Where(fmt.Sprintf("LOWER(%s.%s) = LOWER(?)", tableName, name), value)
		}
	}

	// Add tenant conditions
	if opts.OrganizationID != "" {
		query = query.Where(fmt.Sprintf("%s.organization_id = ?", tableName), opts.OrganizationID)
	}
	if opts.BusinessUnitID != "" {
		query = query.Where(fmt.Sprintf("%s.business_unit_id = ?", tableName), opts.BusinessUnitID)
	}

	// Add primary key exclusion for updates
	if opts.Operation == OperationUpdate && opts.PrimaryKeyValue != "" {
		pkField := opts.PrimaryKeyField
		if pkField == "" {
			pkField = "id"
		}
		query = query.Where(fmt.Sprintf("%s.%s != ?", tableName, pkField), opts.PrimaryKeyValue)
	}

	// Add any additional conditions
	for _, cond := range opts.AdditionalConditions {
		query = query.Where(cond.Query, cond.Args...)
	}

	// Execute the query
	exists, err := query.Exists(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("failed to check composite uniqueness")
		multiErr.Add(
			opts.ErrorFieldName,
			errors.ErrInvalid,
			"Failed to validate uniqueness constraint",
		)
		return
	}

	if exists {
		// Format the error message
		message := opts.ErrorTemplate
		if message == "" {
			// Default message - list the field names
			fieldNames := make([]string, 0, len(fields))
			for name := range fields {
				fieldNames = append(fieldNames, name)
			}
			message = fmt.Sprintf(
				"A record with this combination of %s already exists",
				strings.Join(fieldNames, ", "),
			)
		} else {
			// Replace variables in the template
			for key, value := range opts.ErrorVars {
				message = strings.ReplaceAll(message, fmt.Sprintf(":%s", key), value)
			}
		}

		multiErr.Add(opts.ErrorFieldName, errors.ErrDuplicate, message)
	}
}

// CompositeUniquenessOptions contains options for composite uniqueness validation
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

// UniquenessValidatorBuilder provides a fluent interface for constructing uniqueness validation criteria.
// It supports configuration of:
//   - Multiple fields with case sensitivity options
//   - Custom error templates
//   - Tenant context
//   - Additional WHERE conditions
type UniquenessValidatorBuilder struct {
	criteria UniquenessCriteria
}

// NewUniquenessValidator creates a new builder for configuring field uniqueness validation.
// It initializes the validation criteria with the specified table name and empty slices
// for fields and error messages.
//
// Parameters:
//   - tableName: The database table name to check for uniqueness
//
// Returns:
//   - A new UniquenessValidatorBuilder instance ready for configuration
func NewUniquenessValidator(tableName string) *UniquenessValidatorBuilder {
	return &UniquenessValidatorBuilder{
		criteria: UniquenessCriteria{
			TableName:     tableName,
			Fields:        make([]UniqueField, 0),
			ErrorMessages: make(map[UniquenessError]*ErrorTemplate),
		},
	}
}

// WithField adds a case-insensitive field to the uniqueness validation.
// The field's value will be compared using LOWER() in the database query.
//
// Parameters:
//   - name: The database column name to check
//   - value: The value to check for uniqueness
//
// Returns:
//   - The builder instance for method chaining
func (b *UniquenessValidatorBuilder) WithField(name string, value any) *UniquenessValidatorBuilder {
	b.criteria.Fields = append(b.criteria.Fields, UniqueField{
		Name:           name,
		Value:          value,
		ErrorFieldName: name,
		CaseSensitive:  false,
	})
	return b
}

// WithCaseSensitiveField adds a case-sensitive field to the uniqueness validation.
// The field's value will be compared exactly as provided, preserving case.
//
// Parameters:
//   - name: The database column name to check
//   - value: The value to check for uniqueness
//
// Returns:
//   - The builder instance for method chaining
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

// WithPrimaryKey sets the primary key field and value for update scenarios.
// When specified, records with this primary key value will be excluded from
// the uniqueness check to allow updating other fields on the same record.
//
// Parameters:
//   - field: The primary key column name (defaults to "id" if not specified)
//   - value: The primary key value to exclude
//
// Returns:
//   - The builder instance for method chaining
func (b *UniquenessValidatorBuilder) WithPrimaryKey(
	field, value string,
) *UniquenessValidatorBuilder {
	b.criteria.PrimaryKeyField = field
	b.criteria.PrimaryKeyValue = value
	return b
}

// WithBusinessUnit configures multi-tenant validation by setting business unit ID.
// When specified, uniqueness checks will be scoped to records within the same business unit.
//
// Parameters:
//   - buID: The business unit ID for tenant filtering
//
// Note:
//   - This method is mutually exclusive with WithTenant.
//   - If both are used, the business unit ID will be used instead of the organization ID.
//
// Returns:
//   - The builder instance for method chaining
func (b *UniquenessValidatorBuilder) WithBusinessUnit(buID pulid.ID) *UniquenessValidatorBuilder {
	b.criteria.BusinessUnitID = buID
	return b
}

// WithTenant configures multi-tenant validation by setting organization and business unit IDs.
// When specified, uniqueness checks will be scoped to records within the same tenant.
//
// Parameters:
//   - orgID: The organization ID for tenant filtering
//   - buID: The business unit ID for tenant filtering
//
// Returns:
//   - The builder instance for method chaining
func (b *UniquenessValidatorBuilder) WithTenant(orgID, buID pulid.ID) *UniquenessValidatorBuilder {
	b.criteria.OrganizationID = orgID
	b.criteria.BusinessUnitID = buID
	return b
}

// WithModelName sets the model name used in error messages.
// This name appears in the default error templates to identify the type of record.
//
// Parameters:
//   - name: The model name to use in error messages (e.g., "User", "Organization")
//
// Returns:
//   - The builder instance for method chaining
func (b *UniquenessValidatorBuilder) WithModelName(name string) *UniquenessValidatorBuilder {
	b.criteria.ErrorModelName = name
	return b
}

// WithCondition adds a custom WHERE condition to the uniqueness check query.
// Use this to add additional filtering beyond the standard tenant and field comparisons.
//
// Parameters:
//   - query: The SQL WHERE clause, may contain placeholders
//   - args: Values for the query placeholders
//
// Example:
//
//	WithCondition("status != ?", "DELETED")
//
// Returns:
//   - The builder instance for method chaining
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

// WithErrorTemplate sets a global error template for a specific error type.
// This template will be used for all fields unless overridden by field-specific templates.
//
// Parameters:
//   - errType: The type of error (ErrCheckFailed or ErrAlreadyExists)
//   - template: The error message template with :placeholder syntax
//   - vars: Map of placeholder names to values
//
// Example:
//
//	WithErrorTemplate(ErrAlreadyExists, ":model with :field ':value' exists",
//	  map[string]string{"model": "User", "field": "email"})
//
// Returns:
//   - The builder instance for method chaining
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

// WithFieldAndTemplate adds a case-insensitive field with a custom error template.
// The field's value will be compared using LOWER() in the database query.
//
// Parameters:
//   - name: The database column name to check
//   - value: The value to check for uniqueness
//   - template: The error message template with :placeholder syntax
//   - vars: Map of placeholder names to values
//
// Returns:
//   - The builder instance for method chaining
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

// WithCaseSensitiveFieldAndTemplate adds a case-sensitive field with a custom error template.
// The field's value will be compared exactly as provided, preserving case.
//
// Parameters:
//   - name: The database column name to check
//   - value: The value to check for uniqueness
//   - template: The error message template with :placeholder syntax
//   - vars: Map of placeholder names to values
//
// Returns:
//   - The builder instance for method chaining
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

// WithOperation sets the operation being performed (create or update)
// This is used to exclude the current record from the uniqueness check
// when updating a record.
//
// Parameters:
//   - op: The operation type (OperationCreate or OperationUpdate)
//
// Returns:
//   - The builder instance for method chaining
func (b *UniquenessValidatorBuilder) WithOperation(op OperationType) *UniquenessValidatorBuilder {
	b.criteria.Operation = op
	return b
}

// Build creates and returns the UniquenessCriteria based on the builder's configuration.
// This method should be called after all desired options have been set.
//
// Returns:
//   - A pointer to the configured UniquenessCriteria
func (b *UniquenessValidatorBuilder) Build() *UniquenessCriteria {
	return &b.criteria
}

// CompositeUniquenessValidatorBuilder provides a fluent interface for building composite uniqueness validations
type CompositeUniquenessValidatorBuilder struct {
	tableName string
	fields    map[string]any
	options   CompositeUniquenessOptions
}

// NewCompositeUniquenessValidator creates a new builder for composite field uniqueness validation
func NewCompositeUniquenessValidator(tableName string) *CompositeUniquenessValidatorBuilder {
	return &CompositeUniquenessValidatorBuilder{
		tableName: tableName,
		fields:    make(map[string]any),
		options:   CompositeUniquenessOptions{},
	}
}

// WithField adds a field to the composite uniqueness check
func (b *CompositeUniquenessValidatorBuilder) WithField(
	name string,
	value any,
) *CompositeUniquenessValidatorBuilder {
	b.fields[name] = value
	return b
}

// WithFields adds multiple fields at once to the composite uniqueness check
func (b *CompositeUniquenessValidatorBuilder) WithFields(
	fields map[string]any,
) *CompositeUniquenessValidatorBuilder {
	for name, value := range fields {
		b.fields[name] = value
	}
	return b
}

// WithTenant sets the tenant context for the validation
func (b *CompositeUniquenessValidatorBuilder) WithTenant(
	orgID, buID pulid.ID,
) *CompositeUniquenessValidatorBuilder {
	b.options.OrganizationID = orgID
	b.options.BusinessUnitID = buID
	return b
}

// WithErrorField sets which field should receive the error if validation fails
func (b *CompositeUniquenessValidatorBuilder) WithErrorField(
	fieldName string,
) *CompositeUniquenessValidatorBuilder {
	b.options.ErrorFieldName = fieldName
	return b
}

// WithErrorTemplate sets a custom error message template
func (b *CompositeUniquenessValidatorBuilder) WithErrorTemplate(
	template string,
	vars map[string]string,
) *CompositeUniquenessValidatorBuilder {
	b.options.ErrorTemplate = template
	b.options.ErrorVars = vars
	return b
}

// WithCaseSensitive sets whether the comparison should be case-sensitive
func (b *CompositeUniquenessValidatorBuilder) WithCaseSensitive(
	caseSensitive bool,
) *CompositeUniquenessValidatorBuilder {
	b.options.CaseSensitive = caseSensitive
	return b
}

// WithCondition adds an additional WHERE condition to the uniqueness check
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

// ForCreate sets the validation for a create operation
func (b *CompositeUniquenessValidatorBuilder) ForCreate() *CompositeUniquenessValidatorBuilder {
	b.options.Operation = OperationCreate
	return b
}

// ForUpdate sets the validation for an update operation with the given primary key
func (b *CompositeUniquenessValidatorBuilder) ForUpdate(
	primaryKeyValue string,
) *CompositeUniquenessValidatorBuilder {
	b.options.Operation = OperationUpdate
	b.options.PrimaryKeyValue = primaryKeyValue
	return b
}

// ForUpdateWithPK sets the validation for an update with a custom primary key field
func (b *CompositeUniquenessValidatorBuilder) ForUpdateWithPK(
	pkField, pkValue string,
) *CompositeUniquenessValidatorBuilder {
	b.options.Operation = OperationUpdate
	b.options.PrimaryKeyField = pkField
	b.options.PrimaryKeyValue = pkValue
	return b
}

// Validate executes the composite uniqueness validation
func (b *CompositeUniquenessValidatorBuilder) Validate(
	ctx context.Context,
	tx bun.IDB,
	multiErr *errors.MultiError,
) {
	CheckCompositeUniqueness(ctx, tx, b.tableName, b.fields, &b.options, multiErr)
}
