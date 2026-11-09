package errortypes

import (
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/emoss08/trenova/shared/i18n"
)

type Errorable interface {
	error
	GetCode() ErrorCode
	Unwrap() error
}

type BaseError struct {
	Code     ErrorCode     `json:"code"`
	Message  string        `json:"message"`
	Args     []any         `json:"-"`
	Context  *ErrorContext `json:"context,omitempty"`
	Internal error         `json:"-"`
}

func (e *BaseError) Error() string {
	return i18n.Format(e.Message, e.Args...)
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
		"error_message": e.Error(),
	}
	if e.Context != nil {
		maps.Copy(fields, e.Context.LogFields())
	}
	if e.Internal != nil {
		fields["internal_error"] = e.Internal.Error()
	}
	return fields
}

type BusinessError struct {
	BaseError
	Details string            `json:"details,omitempty"`
	Params  map[string]string `json:"params,omitempty"`
}

func NewBusinessError(message string, args ...any) *BusinessError {
	return &BusinessError{
		BaseError: BaseError{
			Code:    ErrBusinessLogic,
			Message: message,
			Args:    args,
		},
	}
}

func (e *BusinessError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s", e.BaseError.Error(), e.Details)
	}

	return e.BaseError.Error()
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

func NewDatabaseError(message string, args ...any) *DatabaseError {
	return &DatabaseError{
		BaseError: BaseError{
			Code:    ErrSystemError,
			Message: message,
			Args:    args,
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

func NewAuthenticationError(message string, args ...any) *AuthenticationError {
	return &AuthenticationError{
		BaseError: BaseError{
			Code:    ErrUnauthorized,
			Message: message,
			Args:    args,
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

func NewAuthorizationError(message string, args ...any) *AuthorizationError {
	return &AuthorizationError{
		BaseError: BaseError{
			Code:    ErrForbidden,
			Message: message,
			Args:    args,
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

func NewNotFoundError(message string, args ...any) *NotFoundError {
	return &NotFoundError{
		BaseError: BaseError{
			Code:    ErrNotFound,
			Message: message,
			Args:    args,
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
	Field      string        `json:"field,omitempty"`
	RetryAfter time.Duration `json:"-"`
}

func NewRateLimitError(field, message string, args ...any) *RateLimitError {
	return &RateLimitError{
		BaseError: BaseError{
			Code:    ErrTooManyRequests,
			Message: message,
			Args:    args,
		},
		Field: field,
	}
}

func IsRateLimitError(err error) bool {
	_, ok := errors.AsType[*RateLimitError](err)
	return ok
}

func (e *RateLimitError) WithRetryAfter(retryAfter time.Duration) *RateLimitError {
	e.RetryAfter = max(retryAfter, 0)
	return e
}

func RetryAfterOf(err error) (time.Duration, bool) {
	rateLimitErr, ok := errors.AsType[*RateLimitError](err)
	if !ok || rateLimitErr.RetryAfter <= 0 {
		return 0, false
	}

	return rateLimitErr.RetryAfter, true
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

type ConflictError struct {
	BaseError
	UsageStats any `json:"usageStats,omitempty"`
}

func NewConflictError(message string, args ...any) *ConflictError {
	return &ConflictError{
		BaseError: BaseError{
			Code:    ErrResourceInUse,
			Message: message,
			Args:    args,
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

type Localizable interface {
	LocalizedMessage() (message string, args []any)
}

type Parameterized interface {
	error
	Params() map[string]string
}

func (e *BaseError) LocalizedMessage() (message string, args []any) {
	return e.Message, e.Args
}
