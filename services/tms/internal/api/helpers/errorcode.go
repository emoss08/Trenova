package helpers

import (
	"errors"

	"github.com/emoss08/trenova/pkg/errortypes"
)

func ErrorCodeOf(err error, problemType ProblemType) errortypes.ErrorCode {
	if errorable, ok := errors.AsType[errortypes.Errorable](err); ok {
		return errorable.GetCode()
	}

	switch problemType {
	case ProblemTypeAuthentication:
		return errortypes.ErrUnauthorized
	case ProblemTypeAuthorization:
		return errortypes.ErrForbidden
	case ProblemTypeNotFound:
		return errortypes.ErrNotFound
	case ProblemTypeRateLimit:
		return errortypes.ErrTooManyRequests
	case ProblemTypeQuotaExceeded:
		return errortypes.ErrQuotaExceeded
	case ProblemTypePlanRestricted:
		return errortypes.ErrPlanRestricted
	case ProblemTypeConflict:
		return errortypes.ErrResourceInUse
	case ProblemTypeValidation, ProblemTypeBusiness:
		return errortypes.ErrInvalid
	default:
		return errortypes.ErrSystemError
	}
}
