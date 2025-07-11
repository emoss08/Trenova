package errors

import (
	"fmt"
)

// * ResolveError provides context about where a resolution error occurred
type ResolveError struct {
	Path       string
	EntityType string
	Cause      error
}

// * Error implements the error interface
func (e *ResolveError) Error() string {
	return fmt.Sprintf("failed to resolve %s on %s: %v",
		e.Path, e.EntityType, e.Cause)
}

// * Unwrap allows error unwrapping
func (e *ResolveError) Unwrap() error {
	return e.Cause
}

// * NewResolveError creates a new ResolveError
func NewResolveError(path, entityType string, cause error) *ResolveError {
	return &ResolveError{
		Path:       path,
		EntityType: entityType,
		Cause:      cause,
	}
}

// * ValidationError provides context about validation failures
type ValidationError struct {
	Field   string
	Value   any
	Message string
}

// * Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field %s: %s", e.Field, e.Message)
}

// * TransformError provides context about transformation failures
type TransformError struct {
	SourceType string
	TargetType string
	Value      any
	Cause      error
}

// * Error implements the error interface
func (e *TransformError) Error() string {
	return fmt.Sprintf("failed to transform %s to %s: %v",
		e.SourceType, e.TargetType, e.Cause)
}

// * Unwrap allows error unwrapping
func (e *TransformError) Unwrap() error {
	return e.Cause
}

// * NewTransformError creates a new TransformError
func NewTransformError(sourceType, targetType string, value any, cause error) *TransformError {
	return &TransformError{
		SourceType: sourceType,
		TargetType: targetType,
		Value:      value,
		Cause:      cause,
	}
}

// * ComputeError provides context about computation failures
type ComputeError struct {
	Function   string
	EntityType string
	Cause      error
}

// * Error implements the error interface
func (e *ComputeError) Error() string {
	return fmt.Sprintf("failed to compute %s for %s: %v",
		e.Function, e.EntityType, e.Cause)
}

// * Unwrap allows error unwrapping
func (e *ComputeError) Unwrap() error {
	return e.Cause
}

// * NewComputeError creates a new ComputeError
func NewComputeError(function, entityType string, cause error) *ComputeError {
	return &ComputeError{
		Function:   function,
		EntityType: entityType,
		Cause:      cause,
	}
}

// * VariableError provides context about variable resolution failures
type VariableError struct {
	VariableName string
	Context      string
	Cause        error
}

// * Error implements the error interface
func (e *VariableError) Error() string {
	return fmt.Sprintf("failed to resolve variable '%s' in context %s: %v",
		e.VariableName, e.Context, e.Cause)
}

// * Unwrap allows error unwrapping
func (e *VariableError) Unwrap() error {
	return e.Cause
}

// * NewVariableError creates a new VariableError
func NewVariableError(variableName, context string, cause error) *VariableError {
	return &VariableError{
		VariableName: variableName,
		Context:      context,
		Cause:        cause,
	}
}

// * SchemaError provides context about schema-related failures
type SchemaError struct {
	SchemaID string
	Action   string
	Cause    error
}

// * Error implements the error interface
func (e *SchemaError) Error() string {
	return fmt.Sprintf("schema error for '%s' during %s: %v",
		e.SchemaID, e.Action, e.Cause)
}

// * Unwrap allows error unwrapping
func (e *SchemaError) Unwrap() error {
	return e.Cause
}

// * NewSchemaError creates a new SchemaError
func NewSchemaError(schemaID, action string, cause error) *SchemaError {
	return &SchemaError{
		SchemaID: schemaID,
		Action:   action,
		Cause:    cause,
	}
}
