package errors

import (
	"fmt"
)

type ResolveError struct {
	Path       string
	EntityType string
	Cause      error
}

func (e *ResolveError) Error() string {
	return fmt.Sprintf("failed to resolve %s on %s: %v",
		e.Path, e.EntityType, e.Cause)
}

func (e *ResolveError) Unwrap() error {
	return e.Cause
}

func NewResolveError(path, entityType string, cause error) *ResolveError {
	return &ResolveError{
		Path:       path,
		EntityType: entityType,
		Cause:      cause,
	}
}

type ValidationError struct {
	Field   string
	Value   any
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field %s: %s", e.Field, e.Message)
}

type TransformError struct {
	SourceType string
	TargetType string
	Value      any
	Cause      error
}

func (e *TransformError) Error() string {
	return fmt.Sprintf("failed to transform %s to %s: %v",
		e.SourceType, e.TargetType, e.Cause)
}

func (e *TransformError) Unwrap() error {
	return e.Cause
}

func NewTransformError(sourceType, targetType string, value any, cause error) *TransformError {
	return &TransformError{
		SourceType: sourceType,
		TargetType: targetType,
		Value:      value,
		Cause:      cause,
	}
}

type ComputeError struct {
	Function   string
	EntityType string
	Cause      error
}

func (e *ComputeError) Error() string {
	return fmt.Sprintf("failed to compute %s for %s: %v",
		e.Function, e.EntityType, e.Cause)
}

func (e *ComputeError) Unwrap() error {
	return e.Cause
}

func NewComputeError(function, entityType string, cause error) *ComputeError {
	return &ComputeError{
		Function:   function,
		EntityType: entityType,
		Cause:      cause,
	}
}

type VariableError struct {
	VariableName string
	Context      string
	Cause        error
}

func (e *VariableError) Error() string {
	return fmt.Sprintf("failed to resolve variable '%s' in context %s: %v",
		e.VariableName, e.Context, e.Cause)
}

func (e *VariableError) Unwrap() error {
	return e.Cause
}

func NewVariableError(variableName, context string, cause error) *VariableError {
	return &VariableError{
		VariableName: variableName,
		Context:      context,
		Cause:        cause,
	}
}

type SchemaError struct {
	SchemaID string
	Action   string
	Cause    error
}

func (e *SchemaError) Error() string {
	return fmt.Sprintf("schema error for '%s' during %s: %v",
		e.SchemaID, e.Action, e.Cause)
}

func (e *SchemaError) Unwrap() error {
	return e.Cause
}

func NewSchemaError(schemaID, action string, cause error) *SchemaError {
	return &SchemaError{
		SchemaID: schemaID,
		Action:   action,
		Cause:    cause,
	}
}
