package errortypes

type ErrorCode string

const (
	ErrRequired            = ErrorCode("REQUIRED")
	ErrInvalid             = ErrorCode("INVALID")
	ErrDuplicate           = ErrorCode("DUPLICATE")
	ErrNotFound            = ErrorCode("NOT_FOUND")
	ErrBusinessLogic       = ErrorCode("BUSINESS_LOGIC")
	ErrUnauthorized        = ErrorCode("UNAUTHORIZED")
	ErrForbidden           = ErrorCode("FORBIDDEN")
	ErrInvalidFormat       = ErrorCode("INVALID_FORMAT")
	ErrInvalidLength       = ErrorCode("INVALID_LENGTH")
	ErrInvalidReference    = ErrorCode("INVALID_REFERENCE")
	ErrInvalidOperation    = ErrorCode("INVALID_OPERATION")
	ErrSystemError         = ErrorCode("SYSTEM_ERROR")
	ErrAlreadyExists       = ErrorCode("ALREADY_EXISTS")
	ErrAlreadyCleared      = ErrorCode("ALREADY_CLEARED")
	ErrVersionMismatch     = ErrorCode("VERSION_MISMATCH")
	ErrTooManyRequests     = ErrorCode("TOO_MANY_REQUESTS")
	ErrComplianceViolation = ErrorCode("COMPLIANCE_VIOLATION")
	ErrResourceInUse       = ErrorCode("RESOURCE_IN_USE")
	ErrBreakingChange      = ErrorCode("BREAKING_CHANGE")
)
