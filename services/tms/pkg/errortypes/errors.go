package errortypes

import (
	"errors"
	"fmt"
	"maps"
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

type Errorable interface {
	error
	GetCode() ErrorCode
	Unwrap() error
}

type BaseError struct {
	Code     ErrorCode     `json:"code"`
	Message  string        `json:"message"`
	Context  *ErrorContext `json:"context,omitempty"`
	Internal error         `json:"-"`
}

func (e *BaseError) Error() string {
	return e.Message
}

func (e *BaseError) Unwrap() error {
	return e.Internal
}

func (e *BaseError) GetCode() ErrorCode {
	return e.Code
}

func (e *BaseError) WithContext(ctx *ErrorContext) {
	e.Context = ctx
}

func (e *BaseError) LogFields() LogFields {
	fields := LogFields{
		"error_code":    e.Code,
		"error_message": e.Message,
	}
	if e.Context != nil {
		maps.Copy(fields, e.Context.LogFields())
	}
	if e.Internal != nil {
		fields["internal_error"] = e.Internal.Error()
	}
	return fields
}

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

func (e *Error) Unwrap() error {
	return e.Internal
}

func (e *Error) GetCode() ErrorCode {
	return e.Code
}

func IsError(err error) bool {
	var validationErr *Error
	var multiErr *MultiError
	return errors.As(err, &validationErr) || errors.As(err, &multiErr)
}

type MultiError struct {
	prefix          string
	parent          *MultiError
	currentPriority ValidationPriority
	maxErrors       int
	Context         *ErrorContext `json:"context,omitempty"`
	Errors          []*Error      `json:"errors"`
}

func NewMultiError() *MultiError {
	return &MultiError{
		Errors:          make([]*Error, 0),
		currentPriority: PriorityHigh,
	}
}

func (m *MultiError) WithContext(ctx *ErrorContext) *MultiError {
	m.Context = ctx
	return m
}

func (m *MultiError) LogFields() LogFields {
	fields := LogFields{
		"error_count": len(m.Errors),
	}

	if m.Context != nil {
		maps.Copy(fields, m.Context.LogFields())
	}

	if len(m.Errors) > 0 {
		fieldNames := make([]string, 0, len(m.Errors))
		for _, err := range m.Errors {
			fieldNames = append(fieldNames, err.Field)
		}
		fields["error_fields"] = fieldNames
	}

	return fields
}

func NewMultiErrorWithLimit(maxErrors int) *MultiError {
	return &MultiError{
		Errors:          make([]*Error, 0, maxErrors),
		currentPriority: PriorityHigh,
		maxErrors:       maxErrors,
	}
}

func (m *MultiError) root() *MultiError {
	root := m
	for root.parent != nil {
		root = root.parent
	}
	return root
}

func (m *MultiError) IsFull() bool {
	root := m.root()
	return root.maxErrors > 0 && len(root.Errors) >= root.maxErrors
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

	root := m.root()
	if root.maxErrors > 0 && len(root.Errors) >= root.maxErrors {
		return
	}

	errCopy := &Error{
		Field:    err.Field,
		Code:     err.Code,
		Message:  err.Message,
		Priority: err.Priority,
		Internal: err.Internal,
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
	root := m.root()
	if root.maxErrors > 0 && len(root.Errors) >= root.maxErrors {
		return
	}

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
	_, ok := errors.AsType[*MultiError](err)
	return ok
}

type BusinessError struct {
	BaseError
	Details string            `json:"details,omitempty"`
	Params  map[string]string `json:"params,omitempty"`
}

func NewBusinessError(message string) *BusinessError {
	return &BusinessError{
		BaseError: BaseError{
			Code:    ErrBusinessLogic,
			Message: message,
		},
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

func (e *BusinessError) WithContext(ctx *ErrorContext) *BusinessError {
	e.Context = ctx
	return e
}

func (e *BusinessError) LogFields() LogFields {
	fields := e.BaseError.LogFields()
	if e.Details != "" {
		fields["error_details"] = e.Details
	}
	for k, v := range e.Params {
		fields["param_"+k] = v
	}
	return fields
}

func IsBusinessError(err error) bool {
	_, ok := errors.AsType[*BusinessError](err)
	return ok
}

type DatabaseError struct {
	BaseError
}

func NewDatabaseError(message string) *DatabaseError {
	return &DatabaseError{
		BaseError: BaseError{
			Code:    ErrSystemError,
			Message: message,
		},
	}
}

func IsDatabaseError(err error) bool {
	_, ok := errors.AsType[*DatabaseError](err)
	return ok
}

func (e *DatabaseError) WithInternal(err error) *DatabaseError {
	e.Internal = err
	return e
}

func (e *DatabaseError) WithContext(ctx *ErrorContext) *DatabaseError {
	e.Context = ctx
	return e
}

type AuthenticationError struct {
	BaseError
}

func NewAuthenticationError(message string) *AuthenticationError {
	return &AuthenticationError{
		BaseError: BaseError{
			Code:    ErrUnauthorized,
			Message: message,
		},
	}
}

func IsAuthenticationError(err error) bool {
	_, ok := errors.AsType[*AuthenticationError](err)
	return ok
}

func (e *AuthenticationError) WithInternal(err error) *AuthenticationError {
	e.Internal = err
	return e
}

func (e *AuthenticationError) WithContext(ctx *ErrorContext) *AuthenticationError {
	e.Context = ctx
	return e
}

type AuthorizationError struct {
	BaseError
}

func NewAuthorizationError(message string) *AuthorizationError {
	return &AuthorizationError{
		BaseError: BaseError{
			Code:    ErrForbidden,
			Message: message,
		},
	}
}

func IsAuthorizationError(err error) bool {
	_, ok := errors.AsType[*AuthorizationError](err)
	return ok
}

func (e *AuthorizationError) WithInternal(err error) *AuthorizationError {
	e.Internal = err
	return e
}

func (e *AuthorizationError) WithContext(ctx *ErrorContext) *AuthorizationError {
	e.Context = ctx
	return e
}

type NotFoundError struct {
	BaseError
}

func NewNotFoundError(message string) *NotFoundError {
	return &NotFoundError{
		BaseError: BaseError{
			Code:    ErrNotFound,
			Message: message,
		},
	}
}

func IsNotFoundError(err error) bool {
	_, ok := errors.AsType[*NotFoundError](err)
	return ok
}

func (e *NotFoundError) WithInternal(err error) *NotFoundError {
	e.Internal = err
	return e
}

func (e *NotFoundError) WithContext(ctx *ErrorContext) *NotFoundError {
	e.Context = ctx
	return e
}

type RateLimitError struct {
	BaseError
	Field string `json:"field,omitempty"`
}

func NewRateLimitError(field, message string) *RateLimitError {
	return &RateLimitError{
		BaseError: BaseError{
			Code:    ErrTooManyRequests,
			Message: message,
		},
		Field: field,
	}
}

func IsRateLimitError(err error) bool {
	_, ok := errors.AsType[*RateLimitError](err)
	return ok
}

func (e *RateLimitError) WithInternal(err error) *RateLimitError {
	e.Internal = err
	return e
}

func (e *RateLimitError) WithContext(ctx *ErrorContext) *RateLimitError {
	e.Context = ctx
	return e
}

func (e *RateLimitError) LogFields() LogFields {
	fields := e.BaseError.LogFields()
	if e.Field != "" {
		fields["rate_limit_field"] = e.Field
	}
	return fields
}

func InferErrorCode(err error) ErrorCode {
	if errors.Is(err, val.ErrRequired) {
		return ErrRequired
	}
	if errors.Is(err, val.ErrNilOrNotEmpty) {
		return ErrRequired
	}
	if errors.Is(err, val.ErrLengthOutOfRange) {
		return ErrInvalidLength
	}
	if errors.Is(err, val.ErrMinGreaterEqualThanRequired) ||
		errors.Is(err, val.ErrMaxLessEqualThanRequired) {
		return ErrInvalidLength
	}

	msg := err.Error()
	switch {
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

func (m *MultiError) AddOzzoError(err error) bool {
	if err == nil {
		return false
	}

	if valErrs, ok := errors.AsType[val.Errors](err); ok {
		FromOzzoErrors(valErrs, m)
		return true
	}

	return false
}

type ConflictError struct {
	BaseError
	UsageStats any `json:"usageStats,omitempty"`
}

func NewConflictError(message string) *ConflictError {
	return &ConflictError{
		BaseError: BaseError{
			Code:    ErrResourceInUse,
			Message: message,
		},
	}
}

func (e *ConflictError) WithUsageStats(stats any) *ConflictError {
	e.UsageStats = stats
	return e
}

func (e *ConflictError) WithInternal(err error) *ConflictError {
	e.Internal = err
	return e
}

func (e *ConflictError) WithContext(ctx *ErrorContext) *ConflictError {
	e.Context = ctx
	return e
}

func (e *ConflictError) LogFields() LogFields {
	fields := e.BaseError.LogFields()
	if e.UsageStats != nil {
		fields["usage_stats"] = e.UsageStats
	}
	return fields
}

func IsConflictError(err error) bool {
	_, ok := errors.AsType[*ConflictError](err)
	return ok
}

func MergeMultiErrors(multiErrs ...*MultiError) *MultiError {
	var merged *MultiError
	for _, multiErr := range multiErrs {
		if multiErr == nil || !multiErr.HasErrors() {
			continue
		}
		if merged == nil {
			merged = NewMultiError()
		}
		for _, err := range multiErr.Errors {
			merged.AddError(err)
		}
	}

	return merged
}
