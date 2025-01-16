package errors

type ErrorCode string

const (
	// ErrRequired indicates a required field is missing
	ErrRequired = ErrorCode("REQUIRED")

	// ErrInvalid indicates an invalid value for a field
	ErrInvalid = ErrorCode("INVALID")

	// ErrDuplicate indicates a duplicate value for a field
	ErrDuplicate = ErrorCode("DUPLICATE")

	// ErrNotFound indicates a resource that was not found
	ErrNotFound = ErrorCode("NOT_FOUND")

	// ErrBusinessLogic indicates a business logic error
	ErrBusinessLogic = ErrorCode("BUSINESS_LOGIC")

	// ErrUnauthorized indicates an unauthorized request
	ErrUnauthorized = ErrorCode("UNAUTHORIZED")

	// ErrForbidden indicates a forbidden request
	ErrForbidden = ErrorCode("FORBIDDEN")

	// ErrInvalidFormat indicates an invalid format for a field
	ErrInvalidFormat = ErrorCode("INVALID_FORMAT")

	// ErrInvalidLength indicates an invalid length for a field
	ErrInvalidLength = ErrorCode("INVALID_LENGTH")

	// ErrInvalidReference indicates an invalid reference for a field
	ErrInvalidReference = ErrorCode("INVALID_REFERENCE")

	// ErrInvalidOperation indicates an invalid operation for a field
	ErrInvalidOperation = ErrorCode("INVALID_OPERATION")

	// ErrSystemError indicates a system error
	ErrSystemError = ErrorCode("SYSTEM_ERROR")

	// ErrAlreadyExists indicates a resource that already exists
	ErrAlreadyExists = ErrorCode("ALREADY_EXISTS")

	// ErrAlreadyCleared indicates a resource that has already been cleared
	// This is primarily used when a resource is being deleted and the user is trying to delete it again
	ErrAlreadyCleared = ErrorCode("ALREADY_CLEARED")

	// ErrVersionMismatch indicates a version mismatch between requested and current version
	ErrVersionMismatch = ErrorCode("VERSION_MISMATCH")

	// ErrTooManyRequests indicates too many requests are being sent within a short period of time
	ErrTooManyRequests = ErrorCode("TOO_MANY_REQUESTS")

	// ErrComplianceViolation indicates a violation of a CFR
	ErrComplianceViolation = ErrorCode("COMPLIANCE_VIOLATION")
)
