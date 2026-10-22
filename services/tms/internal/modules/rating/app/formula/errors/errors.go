package errors

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/pkg/rules"
)

type ResolveError struct {
	Path       string
	EntityType string
	Cause      error
}

func (e *ResolveError) Error() string {
	return fmt.Sprintf("failed to resolve %s: %v", e.Path, e.Cause)
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
	return fmt.Sprintf("failed to compute %s: %v", e.Function, e.Cause)
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

type UnresolvedFieldError struct {
	Fields []string
	Causes map[string]error
}

func (e *UnresolvedFieldError) Error() string {
	return fmt.Sprintf(
		"the formula reads %s, which could not be read from this record; "+
			"it cannot be priced until the field has a value",
		strings.Join(e.Fields, ", "),
	)
}

func (e *UnresolvedFieldError) LocalizedMessage() (message string, args []any) {
	return e.Text().LocalizedMessage()
}

func (e *UnresolvedFieldError) Text() rules.Message {
	return rules.Message{
		Template: "the formula reads {0}, which could not be read from this record; " +
			"it cannot be priced until the field has a value",
		Args: []any{strings.Join(e.Fields, ", ")},
	}
}

func NewUnresolvedFieldError(fields []string, causes map[string]error) *UnresolvedFieldError {
	return &UnresolvedFieldError{
		Fields: fields,
		Causes: causes,
	}
}

type EvaluationError struct {
	Scope string
	Err   *rules.FormulaError
}

func (e *EvaluationError) Error() string {
	return e.Err.Error()
}

func (e *EvaluationError) Unwrap() error {
	return e.Err
}

func (e *EvaluationError) LocalizedMessage() (message string, args []any) {
	return e.Err.LocalizedMessage()
}

func (e *EvaluationError) MarshalJSON() ([]byte, error) {
	return e.Err.AppendJSON(make([]byte, 0, 256), e.Scope)
}

func NewEvaluationError(scope string, err *rules.FormulaError) *EvaluationError {
	return &EvaluationError{Scope: scope, Err: err}
}
