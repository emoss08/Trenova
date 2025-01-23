package validator

import (
	"fmt"
	"net/http"

	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/rotisserie/eris"
)

type ErrorHandler struct {
	logger *logger.Logger
}

func NewErrorHandler(l *logger.Logger) *ErrorHandler {
	return &ErrorHandler{
		logger: l,
	}
}

type ProblemDetail struct {
	Type          string         `json:"type"`
	Title         string         `json:"title"`
	Status        int            `json:"status"`
	Detail        string         `json:"detail"`
	Instance      string         `json:"instance,omitempty"`
	InvalidParams []InvalidParam `json:"invalid-params,omitempty"`
	TraceID       string         `json:"trace_id,omitempty"`
}

type InvalidParam struct {
	Name     string `json:"name"`
	Reason   string `json:"reason"`
	Code     string `json:"code,omitempty"`
	Location string `json:"location,omitempty"` // body, query, path, header
}

func (h *ErrorHandler) HandleError(c *fiber.Ctx, err error) error {
	errDetails := h.getErrorDetails(err)

	h.logError(c, errDetails)

	return h.createErrorResponse(c, errDetails)
}

type errorDetails struct {
	originalError error
	stackTrace    string
	errorType     ErrorType
	statusCode    int
	invalidParams []InvalidParam
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
)

func (h *ErrorHandler) getErrorDetails(err error) errorDetails {
	details := errorDetails{
		originalError: err,
		stackTrace:    err.Error(),
		statusCode:    http.StatusInternalServerError,
	}

	switch {
	case errors.IsError(err):
		details.errorType = ErrorTypeValidation
		details.statusCode = http.StatusBadRequest
		details.invalidParams = h.extractValidationParams(err)

	case errors.IsBusinessError(err):
		details.errorType = ErrorTypeBusiness
		details.statusCode = http.StatusUnprocessableEntity

	case errors.IsDatabaseError(err):
		details.errorType = ErrorTypeDatabase
		details.statusCode = http.StatusInternalServerError

	case errors.IsAuthenticationError(err):
		details.errorType = ErrorTypeAuthentication
		details.statusCode = http.StatusUnauthorized

	case errors.IsAuthorizationError(err):
		details.errorType = ErrorTypeAuthorization
		details.statusCode = http.StatusForbidden

	case errors.IsNotFoundError(err):
		details.errorType = ErrorTypeNotFound
		details.statusCode = http.StatusNotFound

	case errors.IsRateLimitError(err):
		details.errorType = ErrorTypeTooManyRequests
		details.statusCode = http.StatusTooManyRequests
		details.invalidParams = h.extractValidationParams(err)

	default:
		details.errorType = ErrorTypeInternal
		details.statusCode = http.StatusInternalServerError
	}

	return details
}

func (h *ErrorHandler) logError(c *fiber.Ctx, details errorDetails) {
	event := h.logger.Error().
		Str("request_id", c.GetRespHeader("X-Request-ID")).
		Str("method", c.Method()).
		Str("path", c.Path()).
		Str("ip", c.IP()).
		Int("status", details.statusCode).
		Str("error_type", details.errorType.String()).
		Err(details.originalError)

	if len(details.invalidParams) > 0 {
		event = event.Interface("validation_errors", details.invalidParams)
	}

	if details.stackTrace != "" {
		event = event.Str("stack_trace", details.stackTrace)
	}

	// ignore validation errors and rate limit errors
	if details.errorType == ErrorTypeValidation || details.errorType == ErrorTypeTooManyRequests {
		return
	}

	event.Msg("Request error occurred")
}

func (h *ErrorHandler) createErrorResponse(c *fiber.Ctx, details errorDetails) error {
	problem := &ProblemDetail{
		Type:     details.errorType.String(),
		Title:    h.getErrorTitle(details.errorType),
		Status:   details.statusCode,
		Detail:   details.originalError.Error(),
		Instance: fmt.Sprintf("%s/probs/%s", c.BaseURL(), details.errorType.String()),
		TraceID:  c.GetRespHeader("X-Request-ID"),
	}

	if len(details.invalidParams) > 0 {
		problem.InvalidParams = details.invalidParams
	}

	return c.Status(details.statusCode).JSON(problem)
}

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
	case ErrorTypeInternal:
		return "internal-server-error"
	default:
		return "internal-server-error"
	}
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
	case ErrorTypeInternal:
		return "Internal Server Error"
	default:
		return "Internal Server Error"
	}
}

func (h *ErrorHandler) extractValidationParams(err error) []InvalidParam {
	var params []InvalidParam

	// Handle MultiError
	var multiErr *errors.MultiError
	if eris.As(err, &multiErr) {
		for _, validErr := range multiErr.Errors {
			params = append(params, InvalidParam{
				Name:     validErr.Field,
				Reason:   validErr.Message,
				Code:     string(validErr.Code),
				Location: "body", // Default to body, can be enhanced
			})
		}
		return params
	}

	// Handle single validation error
	var validErr *errors.Error
	if eris.As(err, &validErr) {
		params = append(params, InvalidParam{
			Name:     validErr.Field,
			Reason:   validErr.Message,
			Code:     string(validErr.Code),
			Location: "body",
		})
		return params
	}

	// Handle business error
	var businessErr *errors.BusinessError
	if eris.As(err, &businessErr) {
		param := InvalidParam{
			Name:     "business",
			Reason:   businessErr.Message,
			Code:     string(businessErr.Code),
			Location: "business",
		}
		if businessErr.Details != "" {
			param.Reason = fmt.Sprintf("%s: %s", businessErr.Message, businessErr.Details)
		}
		params = append(params, param)
		return params
	}

	// Handle database error
	var dbErr *errors.DatabaseError
	if eris.As(err, &dbErr) {
		params = append(params, InvalidParam{
			Name:     "database",
			Reason:   dbErr.Message,
			Code:     string(dbErr.Code),
			Location: "database",
		})
		return params
	}

	// Handle authentication error
	var authErr *errors.AuthenticationError
	if eris.As(err, &authErr) {
		params = append(params, InvalidParam{
			Name:     "authentication",
			Reason:   authErr.Message,
			Code:     string(authErr.Code),
			Location: "authentication",
		})
		return params
	}

	// Handle authorization error
	var authzErr *errors.AuthorizationError
	if eris.As(err, &authzErr) {
		params = append(params, InvalidParam{
			Name:     "authorization",
			Reason:   authzErr.Message,
			Code:     string(authzErr.Code),
			Location: "authorization",
		})
		return params
	}

	// Handle not found error
	var notFoundErr *errors.NotFoundError
	if eris.As(err, &notFoundErr) {
		params = append(params, InvalidParam{
			Name:     "notFound",
			Reason:   notFoundErr.Message,
			Code:     string(notFoundErr.Code),
			Location: "resource",
		})
		return params
	}

	// Handle rate limit error
	var rateLimitErr *errors.RateLimitError
	if eris.As(err, &rateLimitErr) {
		params = append(params, InvalidParam{
			Name:     rateLimitErr.Field,
			Reason:   rateLimitErr.Message,
			Code:     string(rateLimitErr.Code),
			Location: "rate-limit",
		})
		return params
	}

	// Default case: generic error
	params = append(params, InvalidParam{
		Name:     "error",
		Reason:   err.Error(),
		Code:     string(errors.ErrSystemError),
		Location: "system",
	})

	return params
}
