package errors

import (
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/wasmmodule"
)

const (
	CodeInputValue = "input-value"
	FieldSource    = "source"
)

type CheckFields struct {
	Source string
	Value  func(path string) string
}

func (f CheckFields) source() string {
	if f.Source == "" {
		return FieldSource
	}
	return f.Source
}

func (f CheckFields) value(path string) string {
	if f.Value == nil {
		return "inputs." + path
	}
	return f.Value(path)
}

func CheckRefusal(check *rules.TariffCheck, fields CheckFields) error {
	if check == nil {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	for i := range check.Diagnostics {
		diagnostic := &check.Diagnostics[i]
		if diagnostic.Severity != rules.SeverityError {
			continue
		}
		multiErr.AddError(diagnosticError(diagnostic, fields))
	}
	if !check.OK && !multiErr.HasErrors() {
		multiErr.Add(fields.source(), errortypes.ErrInvalid, "The formula does not check")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func diagnosticError(diagnostic *rules.TariffDiagnostic, fields CheckFields) *errortypes.Error {
	if diagnostic.Path != "" {
		return diagnostic.Message.
			ValidationError(fields.value(diagnostic.Path), errortypes.ErrInvalid).
			WithInternal(&DiagnosticError{Diagnostic: *diagnostic})
	}

	if diagnostic.Module != "" {
		return errortypes.NewValidationError(
			fields.source(),
			errortypes.ErrInvalid,
			"In {0}: {1}",
			diagnostic.Module,
			diagnostic.Message,
		).WithInternal(&DiagnosticError{Diagnostic: *diagnostic})
	}

	validation := diagnostic.Message.
		ValidationError(fields.source(), errortypes.ErrInvalid).
		WithInternal(&DiagnosticError{Diagnostic: *diagnostic})
	if diagnostic.Range != nil {
		validation = validation.WithRange(diagnostic.Range.From, diagnostic.Range.To)
	}
	return validation
}

type DiagnosticError struct {
	Diagnostic rules.TariffDiagnostic
}

func (e *DiagnosticError) Error() string {
	return e.Diagnostic.Message.String()
}

func (e *DiagnosticError) LocalizedMessage() (message string, args []any) {
	return e.Diagnostic.Message.LocalizedMessage()
}

type EvaluationError struct {
	Code    string
	Message wasmmodule.Message
	Range   *rules.TariffRange
}

func (e *EvaluationError) Error() string {
	return e.Message.String()
}

func (e *EvaluationError) LocalizedMessage() (message string, args []any) {
	return e.Message.LocalizedMessage()
}

func Evaluation(field string, failure *rules.TariffEvalError) *errortypes.Error {
	if failure == nil {
		return errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"The formula could not be evaluated",
		)
	}

	validation := failure.Message.
		ValidationError(field, errortypes.ErrInvalid).
		WithInternal(&EvaluationError{
			Code:    failure.Code,
			Message: failure.Message,
			Range:   failure.Range,
		})
	if failure.Range != nil {
		validation = validation.WithRange(failure.Range.From, failure.Range.To)
	}
	return validation
}
