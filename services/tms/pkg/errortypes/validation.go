package errortypes

import (
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/fieldpath"
	"github.com/emoss08/trenova/pkg/ozzoerrors"
	"github.com/emoss08/trenova/shared/i18n"
)

const (
	MaxCollectedErrors     = 500
	validationFailedHeader = "validation failed:"
	validationRunFailed    = "validation could not complete"
)

type SourceRange struct {
	From int `json:"from"`
	To   int `json:"to"`
}

type Error struct {
	Path     fieldpath.Path `json:"path"`
	Code     ErrorCode      `json:"code"`
	Message  string         `json:"message"`
	Args     []any          `json:"-"`
	Range    *SourceRange   `json:"range,omitempty"`
	Internal error          `json:"-"`
}

func NewValidationError(field string, code ErrorCode, message string, args ...any) *Error {
	return &Error{
		Path:    fieldpath.Parse(field),
		Code:    code,
		Message: message,
		Args:    args,
	}
}

func (e *Error) Field() string {
	return e.Path.String()
}

func (e *Error) Error() string {
	return i18n.Format(e.Message, e.Args...)
}

func (e *Error) WithRange(from, to int) *Error {
	e.Range = &SourceRange{From: from, To: to}
	return e
}

func (e *Error) WithInternal(err error) *Error {
	e.Internal = err
	return e
}

func (e *Error) Unwrap() error {
	return e.Internal
}

func (e *Error) GetCode() ErrorCode {
	return e.Code
}

func (e *Error) LocalizedMessage() (message string, args []any) {
	return e.Message, e.Args
}

func IsError(err error) bool {
	if _, ok := errors.AsType[*Error](err); ok {
		return true
	}

	return IsMultiError(err)
}

type MultiError struct {
	root       *MultiError
	prefix     fieldpath.Path
	internal   error
	dropped    int
	Context    *ErrorContext    `json:"context,omitempty"`
	Errors     []*Error         `json:"errors"`
	Advisories []*AdvisoryError `json:"advisories,omitempty"`
}

func NewMultiError() *MultiError {
	return &MultiError{}
}

func (m *MultiError) top() *MultiError {
	if m.root != nil {
		return m.root
	}

	return m
}

func (m *MultiError) WithContext(ctx *ErrorContext) *MultiError {
	m.top().Context = ctx
	return m
}

func (m *MultiError) WithPrefix(prefix string) *MultiError {
	return m.WithPath(fieldpath.Parse(prefix))
}

func (m *MultiError) WithIndex(prefix string, idx int) *MultiError {
	return m.WithPath(fieldpath.Parse(prefix).Append(fieldpath.Index(idx)))
}

func (m *MultiError) WithPath(path fieldpath.Path) *MultiError {
	return &MultiError{root: m.top(), prefix: m.prefix.Join(path)}
}

func (m *MultiError) Add(field string, code ErrorCode, message string, args ...any) {
	m.AddAt(fieldpath.Parse(field), code, message, args...)
}

func (m *MultiError) AddAt(path fieldpath.Path, code ErrorCode, message string, args ...any) {
	m.collect(&Error{
		Path:    m.prefix.Join(path),
		Code:    code,
		Message: message,
		Args:    args,
	})
}

func (m *MultiError) AddError(err *Error) {
	if err == nil {
		return
	}

	scoped := *err
	scoped.Path = m.prefix.Join(err.Path)
	m.collect(&scoped)
}

func (m *MultiError) collect(err *Error) {
	top := m.top()
	if len(top.Errors) >= MaxCollectedErrors {
		top.dropped++
		return
	}

	top.Errors = append(top.Errors, err)
}

func (m *MultiError) AddOzzoError(err error) {
	leaves, internal := ozzoerrors.Flatten(err)
	m.AddInternal(internal)
	for _, leaf := range leaves {
		m.AddAt(leaf.Path, codeOfKind(leaf.Kind), leaf.Message, leaf.Args...)
	}
}

func codeOfKind(kind ozzoerrors.Kind) ErrorCode {
	switch kind {
	case ozzoerrors.KindRequired:
		return ErrRequired
	case ozzoerrors.KindLength:
		return ErrInvalidLength
	case ozzoerrors.KindFormat:
		return ErrInvalidFormat
	case ozzoerrors.KindInvalid:
		return ErrInvalid
	}

	return ErrInvalid
}

func (m *MultiError) AddInternal(err error) {
	if err == nil {
		return
	}

	top := m.top()
	if top.internal == nil {
		top.internal = err
		return
	}
	top.internal = errors.Join(top.internal, err)
}

func (m *MultiError) Internal() error {
	if m == nil {
		return nil
	}

	return m.top().internal
}

func (m *MultiError) HasErrors() bool {
	if m == nil {
		return false
	}

	top := m.top()
	if top.internal != nil {
		return true
	}
	if len(m.prefix) == 0 {
		return len(top.Errors) > 0
	}
	for _, err := range top.Errors {
		if err.Path.HasPrefix(m.prefix) {
			return true
		}
	}

	return false
}

func (m *MultiError) HasErrorAt(field string) bool {
	if m == nil {
		return false
	}

	path := m.prefix.Join(fieldpath.Parse(field))
	for _, err := range m.top().Errors {
		if err.Path.HasPrefix(path) {
			return true
		}
	}

	return false
}

func (m *MultiError) Err() error {
	if m == nil {
		return nil
	}

	top := m.top()
	if top.internal != nil {
		return fmt.Errorf("%s: %w", validationRunFailed, top.internal)
	}
	if len(top.Errors) > 0 {
		return top
	}

	return nil
}

func (m *MultiError) Unwrap() error {
	return m.top().internal
}

func (m *MultiError) Error() string {
	top := m.top()
	if len(top.Errors) == 0 {
		if top.internal != nil {
			return validationRunFailed + ": " + top.internal.Error()
		}
		return ""
	}

	messages := make([]string, 0, len(top.Errors)+1)
	for _, err := range top.Errors {
		if err.Path.IsZero() {
			messages = append(messages, err.Error())
			continue
		}
		messages = append(messages, err.Field()+": "+err.Error())
	}
	if top.internal != nil {
		messages = append(messages, validationRunFailed+": "+top.internal.Error())
	}

	return validationFailedHeader + "\n- " + strings.Join(messages, "\n- ")
}

func (m *MultiError) LogFields() LogFields {
	top := m.top()
	fields := LogFields{
		"error_count": len(top.Errors),
	}
	if top.Context != nil {
		maps.Copy(fields, top.Context.LogFields())
	}
	if len(top.Errors) > 0 {
		fieldNames := make([]string, 0, len(top.Errors))
		for _, err := range top.Errors {
			fieldNames = append(fieldNames, err.Field())
		}
		fields["error_fields"] = fieldNames
	}
	if top.dropped > 0 {
		fields["errors_dropped"] = top.dropped
	}
	if top.internal != nil {
		fields["internal_error"] = top.internal.Error()
	}

	return fields
}

func (m *MultiError) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}

	top := m.top()
	if len(top.Errors) == 0 && len(top.Advisories) == 0 {
		return []byte("null"), nil
	}

	return sonic.Marshal(struct {
		Errors     []*Error         `json:"errors"`
		Advisories []*AdvisoryError `json:"advisories,omitempty"`
	}{
		Errors:     top.Errors,
		Advisories: top.Advisories,
	})
}

func IsMultiError(err error) bool {
	_, ok := RefusalOf(err)
	return ok
}

func RefusalOf(err error) (*MultiError, bool) {
	multiErr, ok := errors.AsType[*MultiError](err)
	if !ok || multiErr.Internal() != nil {
		return nil, false
	}

	return multiErr, true
}

func MergeMultiErrors(multiErrs ...*MultiError) *MultiError {
	var merged *MultiError
	for _, multiErr := range multiErrs {
		if multiErr == nil {
			continue
		}
		source := multiErr.top()
		if len(source.Errors) == 0 && len(source.Advisories) == 0 && source.internal == nil {
			continue
		}
		if merged == nil {
			merged = NewMultiError()
		}
		for _, err := range source.Errors {
			merged.AddError(err)
		}
		merged.Advisories = append(merged.Advisories, source.Advisories...)
		merged.AddInternal(source.internal)
		merged.dropped += source.dropped
	}

	return merged
}
