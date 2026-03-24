package sim

import "errors"

var (
	ErrFixturePathRequired      = errors.New("fixture path is required")
	ErrFixturePayloadEmpty      = errors.New("fixture payload is empty")
	ErrRouteDatasetPathRequired = errors.New("route dataset path is required")
	ErrRouteDatasetInvalid      = errors.New("route dataset is invalid")
	ErrUnsupportedResource      = errors.New("unsupported resource")
	ErrRecordIDRequired         = errors.New("record id is required")
	ErrRecordNotFound           = errors.New("record not found")
	ErrInvalidBody              = errors.New("invalid request body")
	ErrProfileNotFound          = errors.New("scenario profile not found")
	ErrUnauthorized             = errors.New("unauthorized")
	ErrForbidden                = errors.New("forbidden")
	ErrInvalidAuthorization     = errors.New("invalid authorization header")
	ErrRateLimitExceeded        = errors.New("rate limit exceeded")
	ErrWebhookURLRequired       = errors.New("webhook url is required")
	ErrWebhookEventTypeRequired = errors.New("eventType is required")
	ErrWebhookQueueSaturated    = errors.New("webhook queue is saturated")
	ErrQueryIDRequired          = errors.New("id query parameter is required")
	ErrPathIDRequired           = errors.New("path id is required")
	ErrLimitInvalid             = errors.New(
		"limit must be an integer between 1 and the endpoint max",
	)
	ErrCursorInvalid            = errors.New("after cursor was not found for this result set")
	ErrSortByInvalid            = errors.New("unsupported sortBy value")
	ErrSortOrderInvalid         = errors.New("sortOrder must be asc or desc")
	ErrClockStepInvalid         = errors.New("step durationMs must be between 1 and 86400000")
	ErrClockSpeedInvalid        = errors.New("speed must be between 0.1 and 20")
	ErrScriptConfigInvalid      = errors.New("scenario script config is invalid")
	ErrScriptParseFailed        = errors.New("scenario script parse failed")
	ErrFaultRuleInvalid         = errors.New("fault rule is invalid")
	ErrFaultTargetKindInvalid   = errors.New("fault target kind must be endpoint or webhook")
	ErrFaultTargetPathRequired  = errors.New("fault target pathPattern is required")
	ErrFaultTargetEventRequired = errors.New("fault target webhookEventType is required")
	ErrFaultRateOutOfRange      = errors.New("fault rate must be between 0 and 1")
	ErrFaultStatusCodeInvalid   = errors.New("fault statusCode must be a valid HTTP status")
	ErrFaultLatencyInvalid      = errors.New("fault latencyMs must be greater than or equal to 0")
	ErrFaultTruncateInvalid     = errors.New(
		"fault truncateJsonBytes must be greater than or equal to 0",
	)
	ErrRecordConflict = errors.New("record already exists")
)
