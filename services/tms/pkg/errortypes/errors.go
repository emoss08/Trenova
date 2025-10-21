package errortypes

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	val "github.com/go-ozzo/ozzo-validation/v4"
)

type ValidationPriority string

const (
	PriorityHigh   ValidationPriority = "HIGH"
	PriorityMedium ValidationPriority = "MEDIUM"
	PriorityLow    ValidationPriority = "LOW"
)

type Error struct {
	Field    string             `json:"field"`
	Code     ErrorCode          `json:"code"`
	Message  string             `json:"message"`
	Priority ValidationPriority `json:"priority,omitempty"`
	Internal error              `json:"-"`
}

func NewValidationError(field string, code ErrorCode, message string) *Error {
	return &Error{
		Field:   field,
		Code:    code,
		Message: message,
	}
}

func NewValidationErrorWithPriority(
	field string,
	code ErrorCode,
	message string,
	priority ValidationPriority,
) *Error {
	return &Error{
		Field:    field,
		Code:     code,
		Message:  message,
		Priority: priority,
	}
}

func (e *Error) Error() string {
	return e.Message
}

func IsError(err error) bool {
	var validationErr *Error
	var multiErr *MultiError
	return errors.As(err, &validationErr) || errors.As(err, &multiErr)
}

type MultiError struct {
	prefix          string
	parent          *MultiError
	currentPriority ValidationPriority // Track current validation priority context
	Errors          []*Error           `json:"errors"`
}

func NewMultiError() *MultiError {
	return &MultiError{
		Errors:          make([]*Error, 0),
		currentPriority: PriorityHigh, // Default to high priority
	}
}

func (m *MultiError) getFullPrefix() string {
	var prefixes []string
	current := m

	for current != nil && current.prefix != "" {
		prefixes = append([]string{current.prefix}, prefixes...)
		current = current.parent
	}

	if len(prefixes) == 0 {
		return ""
	}

	return strings.Join(prefixes, ".")
}

func (m *MultiError) WithPrefix(prefix string) *MultiError {
	child := &MultiError{
		prefix:          prefix,
		parent:          m,
		currentPriority: m.currentPriority,
		Errors:          make([]*Error, 0),
	}
	return child
}

func (m *MultiError) AddError(err *Error) {
	if err == nil {
		return
	}

	errCopy := &Error{
		Field:    err.Field,
		Code:     err.Code,
		Message:  err.Message,
		Priority: err.Priority,
		Internal: err.Internal,
	}

	root := m
	for root.parent != nil {
		root = root.parent
	}

	if prefix := m.getFullPrefix(); prefix != "" && errCopy.Field != "" {
		errCopy.Field = prefix + "." + errCopy.Field
	}

	root.Errors = append(root.Errors, errCopy)
}

func (m *MultiError) WithIndex(prefix string, idx int) *MultiError {
	return m.WithPrefix(fmt.Sprintf("%s[%d]", prefix, idx))
}

func (m *MultiError) SetPriority(priority ValidationPriority) {
	m.currentPriority = priority
	if m.parent != nil {
		m.parent.SetPriority(priority)
	}
}

func (m *MultiError) Add(field string, code ErrorCode, message string) {
	m.AddWithPriority(field, code, message, "")
}

func (m *MultiError) AddWithPriority(
	field string,
	code ErrorCode,
	message string,
	priority ValidationPriority,
) {
	fieldPath := field
	fullPrefix := m.getFullPrefix()

	if fullPrefix != "" {
		if field != "" {
			fieldPath = fmt.Sprintf("%s.%s", fullPrefix, field)
		} else {
			fieldPath = fullPrefix
		}
	}

	if priority == "" {
		priority = m.currentPriority
	}

	err := &Error{
		Field:    fieldPath,
		Code:     code,
		Message:  message,
		Priority: priority,
	}

	root := m
	for root.parent != nil {
		root = root.parent
	}
	root.Errors = append(root.Errors, err)
}

func (m *MultiError) HasErrors() bool {
	return len(m.Errors) > 0
}

func (m *MultiError) Error() string {
	if len(m.Errors) == 0 {
		return ""
	}

	var messages []string
	for _, err := range m.Errors {
		messages = append(messages, err.Error())
	}

	return fmt.Sprintf("validation failed:\n- %s", strings.Join(messages, "\n- "))
}

func (m *MultiError) MarshalJSON() ([]byte, error) {
	if m == nil || len(m.Errors) == 0 {
		return []byte("null"), nil
	}
	return sonic.Marshal(struct {
		Errors []*Error `json:"errors"`
	}{
		Errors: m.Errors,
	})
}

func (m *MultiError) ToJSON() string {
	output, err := sonic.Marshal(m)
	if err != nil {
		return ""
	}
	return string(output)
}

func IsMultiError(err error) bool {
	var multiErr *MultiError
	return errors.As(err, &multiErr)
}

type BusinessError struct {
	Code     ErrorCode         `json:"code"`
	Message  string            `json:"message"`
	Details  string            `json:"details,omitempty"`
	Params   map[string]string `json:"params,omitempty"`
	Internal error             `json:"-"`
}

func NewBusinessError(message string) *BusinessError {
	return &BusinessError{
		Code:    ErrBusinessLogic,
		Message: message,
	}
}

func (e *BusinessError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Details)
	}
	return e.Message
}

func (e *BusinessError) WithParam(key, value string) *BusinessError {
	if e.Params == nil {
		e.Params = make(map[string]string)
	}
	e.Params[key] = value
	return e
}

func (e *BusinessError) WithInternal(err error) *BusinessError {
	e.Internal = err
	return e
}

func IsBusinessError(err error) bool {
	var businessErr *BusinessError
	return errors.As(err, &businessErr)
}

type DatabaseError struct {
	Code     ErrorCode `json:"code"`
	Message  string    `json:"message"`
	Internal error     `json:"-"`
}

func NewDatabaseError(message string) *DatabaseError {
	return &DatabaseError{
		Code:    ErrSystemError,
		Message: message,
	}
}

func (e *DatabaseError) Error() string {
	return e.Message
}

func IsDatabaseError(err error) bool {
	var databaseErr *DatabaseError
	return errors.As(err, &databaseErr)
}

func (e *DatabaseError) WithInternal(err error) *DatabaseError {
	e.Internal = err
	return e
}

type AuthenticationError struct {
	Code     ErrorCode `json:"code"`
	Message  string    `json:"message"`
	Internal error     `json:"-"`
}

func NewAuthenticationError(message string) *AuthenticationError {
	return &AuthenticationError{
		Code:    ErrUnauthorized,
		Message: message,
	}
}

func IsAuthenticationError(err error) bool {
	var authenticationErr *AuthenticationError
	return errors.As(err, &authenticationErr)
}

func (e *AuthenticationError) Error() string {
	return e.Message
}

func (e *AuthenticationError) WithInternal(err error) *AuthenticationError {
	e.Internal = err
	return e
}

type AuthorizationError struct {
	Code     ErrorCode `json:"code"`
	Message  string    `json:"message"`
	Internal error     `json:"-"`
}

func NewAuthorizationError(message string) *AuthorizationError {
	return &AuthorizationError{
		Code:    ErrForbidden,
		Message: message,
	}
}

func (e *AuthorizationError) Error() string {
	return e.Message
}

func IsAuthorizationError(err error) bool {
	var authorizationErr *AuthorizationError
	return errors.As(err, &authorizationErr)
}

func (e *AuthorizationError) WithInternal(err error) *AuthorizationError {
	e.Internal = err
	return e
}

type NotFoundError struct {
	Code     ErrorCode `json:"code"`
	Message  string    `json:"message"`
	Internal error     `json:"-"`
}

func NewNotFoundError(message string) *NotFoundError {
	return &NotFoundError{
		Code:    ErrNotFound,
		Message: message,
	}
}

func (e *NotFoundError) Error() string {
	return e.Message
}

func IsNotFoundError(err error) bool {
	var notFoundErr *NotFoundError
	return errors.As(err, &notFoundErr)
}

type RateLimitError struct {
	Field    string    `json:"field,omitempty"`
	Code     ErrorCode `json:"code"`
	Message  string    `json:"message"`
	Internal error     `json:"-"`
}

func NewRateLimitError(field, message string) *RateLimitError {
	return &RateLimitError{
		Field:   field,
		Code:    ErrTooManyRequests,
		Message: message,
	}
}

func IsRateLimitError(err error) bool {
	var rateLimitErr *RateLimitError
	return errors.As(err, &rateLimitErr)
}

func (e *RateLimitError) Error() string {
	return e.Message
}

func (e *RateLimitError) WithInternal(err error) *RateLimitError {
	e.Internal = err
	return e
}

func InferErrorCode(err error) ErrorCode {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "required"):
		return ErrRequired
	case strings.Contains(msg, "length"):
		return ErrInvalidLength
	case strings.Contains(msg, "format"):
		return ErrInvalidFormat
	case strings.Contains(msg, "match"):
		return ErrInvalidFormat
	case strings.Contains(msg, "rate limit"):
		return ErrTooManyRequests
	default:
		return ErrInvalid
	}
}

func FromOzzoErrors(valErrors val.Errors, multiErr *MultiError) {
	for field, err := range valErrors {
		validationErr := &Error{
			Field:   field,
			Code:    InferErrorCode(err),
			Message: err.Error(),
		}
		multiErr.AddError(validationErr)
	}
}
