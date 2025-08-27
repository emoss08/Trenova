/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package temporaljobs

import (
	"time"

	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
)

// Task Queue names - replacing Asynq queue names
const (
	TaskQueueCritical   = "critical-tasks"
	TaskQueueEmail      = "email-tasks"
	TaskQueueShipment   = "shipment-tasks"
	TaskQueuePattern    = "pattern-analysis-tasks"
	TaskQueueCompliance = "compliance-tasks"
	TaskQueueDefault    = "default-tasks"
)

// Workflow names - matching Asynq job types
const (
	// Pattern Analysis Workflows
	WorkflowAnalyzePatterns      = "pattern-analyze"
	WorkflowExpireOldSuggestions = "pattern-expire-suggestions"

	// Shipment Workflows
	WorkflowShipmentStatusUpdate = "shipment-status-update"
	WorkflowDuplicateShipment    = "shipment-duplicate"
	WorkflowDelayShipment        = "shipment-delay"
	WorkflowShipmentNotification = "shipment-notification"

	// Compliance Workflows
	WorkflowComplianceCheck       = "compliance-check"
	WorkflowHazmatExpirationCheck = "compliance-hazmat-expiration"

	// System Workflows
	WorkflowCleanupTempFiles = "system-cleanup-temp-files"
	WorkflowGenerateReports  = "system-generate-reports"
	WorkflowDataBackup       = "system-data-backup"

	// Email Workflows
	WorkflowSendEmail         = "email-send"
	WorkflowProcessEmailQueue = "email-process-queue"
)

// Activity names
const (
	ActivityAnalyzeShipmentPatterns = "analyze-shipment-patterns"
	ActivityCreateSuggestions       = "create-suggestions"
	ActivityExpireSuggestions       = "expire-suggestions"
	ActivitySendEmail               = "send-email"
	ActivityUpdateShipmentStatus    = "update-shipment-status"
	ActivityDuplicateShipment       = "duplicate-shipment"
	ActivityPerformComplianceCheck  = "perform-compliance-check"
	ActivityFetchEmailsFromQueue    = "fetch-emails-from-queue"
)

// WorkflowPriority maps to Asynq priority levels
type WorkflowPriority int

const (
	PriorityLow      WorkflowPriority = 1
	PriorityNormal   WorkflowPriority = 5
	PriorityHigh     WorkflowPriority = 10
	PriorityCritical WorkflowPriority = 20
)

// BasePayload contains common fields for all workflow payloads
type BasePayload struct {
	JobID          string         `json:"jobId"`
	OrganizationID pulid.ID       `json:"organizationId"`
	BusinessUnitID pulid.ID       `json:"businessUnitId"`
	UserID         pulid.ID       `json:"userId,omitempty"`
	Timestamp      int64          `json:"timestamp"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// PatternAnalysisPayload for dedicated lane pattern analysis workflows
type PatternAnalysisPayload struct {
	BasePayload
	MinFrequency  int64  `json:"minFrequency"`
	TriggerReason string `json:"triggerReason"`
}

// ExpireSuggestionsPayload for expiring old suggestions
type ExpireSuggestionsPayload struct {
	BasePayload
	BatchSize int `json:"batchSize"`
}

// ShipmentStatusUpdatePayload for shipment status change notifications
type ShipmentStatusUpdatePayload struct {
	BasePayload
	ShipmentID pulid.ID `json:"shipmentId"`
	OldStatus  string   `json:"oldStatus"`
	NewStatus  string   `json:"newStatus"`
}

// DuplicateShipmentPayload for duplicating shipments
type DuplicateShipmentPayload struct {
	BasePayload
	ShipmentID               pulid.ID `json:"shipmentId"`
	Count                    int      `json:"count"`
	OverrideDates            bool     `json:"overrideDates"`
	IncludeCommodities       bool     `json:"includeCommodities"`
	IncludeAdditionalCharges bool     `json:"includeAdditionalCharges"`
}

// DelayShipmentPayload for delaying shipments
type DelayShipmentPayload struct {
	BasePayload
}

// ComplianceCheckPayload for compliance verification workflows
type ComplianceCheckPayload struct {
	BasePayload
	WorkerID   *pulid.ID `json:"workerId,omitempty"`
	ShipmentID *pulid.ID `json:"shipmentId,omitempty"`
	CheckType  string    `json:"checkType"`
}

// EmailPayload for email sending workflows
type EmailPayload struct {
	BasePayload
	To        []string          `json:"to"`
	Subject   string            `json:"subject"`
	Body      string            `json:"body"`
	BodyHTML  string            `json:"bodyHtml,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
}

// WorkflowOptions provides options for workflow execution
type WorkflowOptions struct {
	TaskQueue             string
	Priority              WorkflowPriority
	WorkflowID            string
	WorkflowIDReusePolicy enums.WorkflowIdReusePolicy
	ExecutionTimeout      time.Duration
	RunTimeout            time.Duration
	TaskTimeout           time.Duration
	RetryPolicy           *temporal.RetryPolicy
	SearchAttributes      map[string]any
	Memo                  map[string]any
}

// DefaultWorkflowOptions returns sensible defaults for workflow options
func DefaultWorkflowOptions() *WorkflowOptions {
	return &WorkflowOptions{
		TaskQueue:             TaskQueueDefault,
		Priority:              PriorityNormal,
		ExecutionTimeout:      24 * time.Hour,
		RunTimeout:            1 * time.Hour,
		TaskTimeout:           10 * time.Second,
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
}

// CriticalWorkflowOptions returns options for critical workflows
func CriticalWorkflowOptions() *WorkflowOptions {
	opts := DefaultWorkflowOptions()
	opts.TaskQueue = TaskQueueCritical
	opts.Priority = PriorityCritical
	opts.RetryPolicy.MaximumAttempts = 5
	return opts
}

// EmailWorkflowOptions returns options for email workflows
func EmailWorkflowOptions() *WorkflowOptions {
	opts := DefaultWorkflowOptions()
	opts.TaskQueue = TaskQueueEmail
	opts.Priority = PriorityNormal
	opts.RetryPolicy.MaximumAttempts = 3
	return opts
}

// PatternAnalysisOptions returns options for pattern analysis workflows
func PatternAnalysisOptions() *WorkflowOptions {
	opts := DefaultWorkflowOptions()
	opts.TaskQueue = TaskQueuePattern
	opts.Priority = PriorityNormal
	opts.ExecutionTimeout = 1 * time.Hour
	opts.RetryPolicy.MaximumAttempts = 2
	return opts
}

// ShipmentWorkflowOptions returns options for shipment workflows
func ShipmentWorkflowOptions() *WorkflowOptions {
	opts := DefaultWorkflowOptions()
	opts.TaskQueue = TaskQueueShipment
	opts.Priority = PriorityHigh
	opts.RetryPolicy.MaximumAttempts = 2
	return opts
}

// WorkflowStats provides basic statistics about workflow execution
type WorkflowStats struct {
	IsRunning        bool       `json:"isRunning"`
	StartTime        time.Time  `json:"startTime"`
	Uptime           string     `json:"uptime"`
	WorkflowsStarted int64      `json:"workflowsStarted"`
	WorkflowsFailed  int64      `json:"workflowsFailed"`
	PanicCount       int        `json:"panicCount"`
	LastPanic        *time.Time `json:"lastPanic,omitempty"`
}
