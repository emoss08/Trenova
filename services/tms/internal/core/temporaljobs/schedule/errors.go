package schedule

import "errors"

var (
	ErrScheduleNotFound    = errors.New("schedule not found")
	ErrDuplicateScheduleID = errors.New("duplicate schedule ID")
	ErrInvalidScheduleSpec = errors.New("invalid schedule spec: must have either cron or interval")
	ErrScheduleIDRequired  = errors.New("schedule ID is required")
	ErrWorkflowRequired    = errors.New("workflow is required")
	ErrTaskQueueRequired   = errors.New("task queue is required")
	ErrTemporalUnavailable = errors.New("temporal server unavailable")
	ErrNoProviders         = errors.New("no schedule providers registered")
)
