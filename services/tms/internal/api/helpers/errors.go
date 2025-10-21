package helpers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ErrorHandler struct {
	logger *zap.Logger
}

func NewErrorHandler(logger *zap.Logger) *ErrorHandler {
	return &ErrorHandler{
		logger: logger.Named("error-handler"),
	}
}

type ProblemDetail struct {
	Type          string         `json:"type"`
	Title         string         `json:"title"`
	Status        int            `json:"status"`
	Detail        string         `json:"detail"`
	Instance      string         `json:"instance,omitempty"`
	InvalidParams []InvalidParam `json:"invalidParams,omitempty"`
	TraceID       string         `json:"traceId,omitempty"`
}

type InvalidParam struct {
	Name     string                        `json:"name"`
	Reason   string                        `json:"reason"`
	Code     string                        `json:"code,omitempty"`
	Location string                        `json:"location,omitempty"` // body, query, path, header
	Value    any                           `json:"value,omitempty"`    // The invalid value (sanitized)
	Priority errortypes.ValidationPriority `json:"priority,omitempty"` // HIGH, MEDIUM, LOW - for validation errors
}

type ErrorType int

const (
	ErrorTypeValidation ErrorType = iota
	ErrorTypeDatabase
	ErrorTypeBusiness
	ErrorTypeAuthentication
	ErrorTypeAuthorization
	ErrorTypeNotFound
	ErrorTypeInternal
	ErrorTypeTooManyRequests
	ErrorTypeConflict
	ErrorTypeBadGateway
	ErrorTypeServiceUnavailable
)

func (et ErrorType) String() string {
	switch et {
	case ErrorTypeValidation:
		return "validation-error"
	case ErrorTypeBusiness:
		return "business-error"
	case ErrorTypeDatabase:
		return "database-error"
	case ErrorTypeAuthentication:
		return "authentication-error"
	case ErrorTypeAuthorization:
		return "authorization-error"
	case ErrorTypeNotFound:
		return "not-found-error"
	case ErrorTypeTooManyRequests:
		return "rate-limit-error"
	case ErrorTypeConflict:
		return "conflict-error"
	case ErrorTypeBadGateway:
		return "bad-gateway-error"
	case ErrorTypeServiceUnavailable:
		return "service-unavailable-error"
	case ErrorTypeInternal:
		return "internal-server-error"
	default:
		return "internal-server-error"
	}
}

type errorDetails struct {
	originalError error
	stackTrace    string
	errorType     ErrorType
	statusCode    int
	invalidParams []InvalidParam
}

func (h *ErrorHandler) HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	details := h.getErrorDetails(err)
	h.logError(c, details)
	h.sendErrorResponse(c, details)
}

func (h *ErrorHandler) HandleErrorWithStatus(c *gin.Context, err error, statusCode int) {
	if err == nil {
		return
	}

	details := h.getErrorDetails(err)
	details.statusCode = statusCode
	h.logError(c, details)
	h.sendErrorResponse(c, details)
}

func (h *ErrorHandler) getErrorDetails(err error) errorDetails {
	details := errorDetails{
		originalError: err,
		stackTrace:    err.Error(),
		statusCode:    http.StatusInternalServerError,
		errorType:     ErrorTypeInternal,
		invalidParams: []InvalidParam{},
	}

	originalErr := err
	for {
		unwrapped := errors.Unwrap(originalErr)
		if unwrapped == nil {
			break
		}
		originalErr = unwrapped
	}

	switch {
	case errortypes.IsMultiError(err):
		details.errorType = ErrorTypeValidation
		details.statusCode = http.StatusBadRequest
		details.invalidParams = h.extractMultiErrorParams(err)

	case errortypes.IsError(err):
		details.errorType = ErrorTypeValidation
		details.statusCode = http.StatusBadRequest
		details.invalidParams = h.extractValidationErrorParams(err)

	case errortypes.IsBusinessError(err):
		details.errorType = ErrorTypeBusiness
		details.statusCode = http.StatusUnprocessableEntity
		details.invalidParams = h.extractBusinessErrorParams(err)

	case errortypes.IsDatabaseError(err):
		details.errorType = ErrorTypeDatabase
		details.statusCode = http.StatusInternalServerError

	case errortypes.IsAuthenticationError(err):
		details.errorType = ErrorTypeAuthentication
		details.statusCode = http.StatusUnauthorized

	case errortypes.IsAuthorizationError(err):
		details.errorType = ErrorTypeAuthorization
		details.statusCode = http.StatusForbidden

	case errortypes.IsNotFoundError(err):
		details.errorType = ErrorTypeNotFound
		details.statusCode = http.StatusNotFound

	case errortypes.IsRateLimitError(err):
		details.errorType = ErrorTypeTooManyRequests
		details.statusCode = http.StatusTooManyRequests
		details.invalidParams = h.extractRateLimitErrorParams(err)

	default:
		details.errorType = ErrorTypeInternal
		details.statusCode = http.StatusInternalServerError
	}

	return details
}

func (h *ErrorHandler) logError(c *gin.Context, details errorDetails) {
	if details.errorType == ErrorTypeValidation ||
		details.errorType == ErrorTypeTooManyRequests ||
		details.errorType == ErrorTypeNotFound {
		return
	}

	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = c.GetString("request_id")
	}

	fields := []zap.Field{
		zap.String("request_id", requestID),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("ip", c.ClientIP()),
		zap.Int("status", details.statusCode),
		zap.String("error_type", details.errorType.String()),
		zap.Error(details.originalError),
	}

	if c.Request.URL.RawQuery != "" {
		fields = append(fields, zap.String("query", c.Request.URL.RawQuery))
	}

	if len(details.invalidParams) > 0 {
		fields = append(fields, zap.Any("validation_errors", details.invalidParams))
	}

	switch {
	case details.statusCode >= 500:
		h.logger.Error("Internal server error", fields...)
	case details.statusCode >= 400:
		h.logger.Warn("Client error", fields...)
	default:
		h.logger.Info("Request error", fields...)
	}
}

func (h *ErrorHandler) sendErrorResponse(c *gin.Context, details errorDetails) {
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = c.GetString("request_id")
	}

	problem := &ProblemDetail{
		Type:     details.errorType.String(),
		Title:    h.getErrorTitle(details.errorType),
		Status:   details.statusCode,
		Detail:   h.sanitizeErrorMessage(details.originalError.Error()),
		Instance: c.Request.URL.Path,
		TraceID:  requestID,
	}

	if len(details.invalidParams) > 0 {
		problem.InvalidParams = details.invalidParams
	}

	c.JSON(details.statusCode, problem)
	c.Abort()
}

func (h *ErrorHandler) getErrorTitle(et ErrorType) string {
	switch et {
	case ErrorTypeValidation:
		return "Validation Failed"
	case ErrorTypeBusiness:
		return "Business Rule Violation"
	case ErrorTypeDatabase:
		return "Database Operation Failed"
	case ErrorTypeAuthentication:
		return "Authentication Failed"
	case ErrorTypeAuthorization:
		return "Authorization Failed"
	case ErrorTypeNotFound:
		return "Resource Not Found"
	case ErrorTypeTooManyRequests:
		return "Rate Limit Exceeded"
	case ErrorTypeConflict:
		return "Resource Conflict"
	case ErrorTypeBadGateway:
		return "Bad Gateway"
	case ErrorTypeServiceUnavailable:
		return "Service Unavailable"
	case ErrorTypeInternal:
		return "Internal Server Error"
	default:
		return "Internal Server Error"
	}
}

func (h *ErrorHandler) sanitizeErrorMessage(msg string) string {
	// ! TODO: Add sanitization logic here
	return msg
}

func (h *ErrorHandler) extractMultiErrorParams(err error) []InvalidParam {
	var multiErr *errortypes.MultiError
	if !errors.As(err, &multiErr) {
		return nil
	}

	params := make([]InvalidParam, 0, len(multiErr.Errors))
	for _, validErr := range multiErr.Errors {
		params = append(params, InvalidParam{
			Name:     validErr.Field,
			Reason:   validErr.Message,
			Code:     string(validErr.Code),
			Location: "body",
			Priority: validErr.Priority,
		})
	}
	return params
}

func (h *ErrorHandler) extractValidationErrorParams(err error) []InvalidParam {
	var validErr *errortypes.Error
	if !errors.As(err, &validErr) {
		return nil
	}

	return []InvalidParam{
		{
			Name:     validErr.Field,
			Reason:   validErr.Message,
			Code:     string(validErr.Code),
			Location: "body",
			Priority: validErr.Priority,
		},
	}
}

func (h *ErrorHandler) extractBusinessErrorParams(err error) []InvalidParam {
	var businessErr *errortypes.BusinessError
	if !errors.As(err, &businessErr) {
		return nil
	}

	param := InvalidParam{
		Name:     "business",
		Reason:   businessErr.Message,
		Code:     string(businessErr.Code),
		Location: "business",
	}

	if businessErr.Details != "" {
		param.Reason = fmt.Sprintf("%s: %s", businessErr.Message, businessErr.Details)
	}

	return []InvalidParam{param}
}

func (h *ErrorHandler) extractRateLimitErrorParams(err error) []InvalidParam {
	var rateLimitErr *errortypes.RateLimitError
	if !errors.As(err, &rateLimitErr) {
		return nil
	}

	return []InvalidParam{
		{
			Name:     rateLimitErr.Field,
			Reason:   rateLimitErr.Message,
			Code:     string(rateLimitErr.Code),
			Location: "rate-limit",
		},
	}
}

func (h *ErrorHandler) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				var err error
				switch x := r.(type) {
				case string:
					err = errors.New(x)
				case error:
					err = x
				default:
					err = fmt.Errorf("unknown panic: %v", r)
				}

				h.logger.Error("Panic recovered",
					zap.Any("panic", r),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)

				h.HandleErrorWithStatus(c, err, http.StatusInternalServerError)
			}
		}()

		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			h.HandleError(c, err.Err)
		}
	}
}
