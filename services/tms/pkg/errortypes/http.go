package errortypes

import "net/http"

func HTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	switch {
	case IsMultiError(err):
		return http.StatusUnprocessableEntity
	case IsError(err):
		return http.StatusUnprocessableEntity
	case IsNotFoundError(err):
		return http.StatusNotFound
	case IsAuthenticationError(err):
		return http.StatusUnauthorized
	case IsAuthorizationError(err):
		return http.StatusForbidden
	case IsRateLimitError(err):
		return http.StatusTooManyRequests
	case IsBusinessError(err):
		return http.StatusUnprocessableEntity
	case IsDatabaseError(err):
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

func HTTPStatusWithCode(code ErrorCode) int {
	switch code {
	case ErrRequired,
		ErrInvalid,
		ErrInvalidFormat,
		ErrInvalidLength,
		ErrInvalidReference,
		ErrResourceInUse,
		ErrInvalidOperation:
		return http.StatusUnprocessableEntity
	case ErrDuplicate, ErrAlreadyExists, ErrBreakingChange:
		return http.StatusConflict
	case ErrNotFound:
		return http.StatusNotFound
	case ErrUnauthorized:
		return http.StatusUnauthorized
	case ErrForbidden:
		return http.StatusForbidden
	case ErrTooManyRequests:
		return http.StatusTooManyRequests
	case ErrBusinessLogic, ErrComplianceViolation, ErrAlreadyCleared:
		return http.StatusUnprocessableEntity
	case ErrVersionMismatch:
		return http.StatusConflict
	case ErrSystemError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
