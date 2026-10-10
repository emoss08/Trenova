package sim

import "errors"

var (
	ErrFixturePathRequired      = errors.New("fixture path is required")
	ErrFixturePayloadEmpty      = errors.New("fixture payload is empty")
	ErrRouteDatasetPathRequired = errors.New("route dataset path is required")
	ErrRouteDatasetInvalid      = errors.New("route dataset is invalid")
	ErrUnsupportedResource      = errors.New("unsupported resource")
	ErrRecordIDRequired         = errors.New("record id is required")
	ErrRecordNotFound           = errors.New("object not found")
	ErrInvalidBody              = errors.New("invalid request body")
	ErrProfileNotFound          = errors.New("scenario profile not found")
	ErrUnauthorized             = errors.New("invalid token")
	ErrForbidden                = errors.New(
		"this API token is read-only and cannot make write requests",
	)
	ErrInvalidAuthorization = errors.New(
		"missing or malformed Authorization header; expected \"Bearer <token>\"",
	)
	ErrRateLimitExceeded        = errors.New("exceeded rate limit")
	ErrWebhookURLRequired       = errors.New("webhook url is required")
	ErrWebhookEventTypeRequired = errors.New("eventType is required")
	ErrWebhookEventTypeUnknown  = errors.New("eventType is not a Samsara webhook event type")
	ErrWebhookQueueSaturated    = errors.New("webhook queue is saturated")
	ErrQueryIDRequired          = errors.New("id query parameter is required")
	ErrPathIDRequired           = errors.New("path id is required")
	ErrLimitInvalid             = errors.New("invalid value for parameter limit")
	ErrCursorInvalid            = errors.New(
		"invalid value for parameter after: it must be an endCursor returned by this endpoint",
	)
	ErrStatTypesRequired = errors.New("types query parameter is required")
	ErrStatTypeInvalid   = errors.New("unsupported stat type")
	ErrAssetTypeInvalid  = errors.New(
		"type must be one of uncategorized, trailer, equipment, unpowered, vehicle",
	)
	ErrTimeRangeRequired        = errors.New("startTime and endTime query parameters are required")
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
	ErrRecordConflict     = errors.New("record already exists")
	ErrRouteNotFound      = errors.New("not found")
	ErrMethodNotAllowed   = errors.New("method not allowed")
	ErrStatePathRequired  = errors.New("state path is required")
	ErrStateCorrupt       = errors.New("persisted simulator state is corrupt")
	ErrStateVersion       = errors.New("persisted simulator state version is unsupported")
	ErrPersistenceEnabled = errors.New("state persistence is already enabled")
	ErrInvalidParameter   = errors.New("invalid value for parameter")
	ErrUniqueConflict     = errors.New("invalid value")
	ErrReferenceNotFound  = errors.New("referenced object not found")
	ErrTagCycle           = errors.New("invalid value for parentTagId")
)
