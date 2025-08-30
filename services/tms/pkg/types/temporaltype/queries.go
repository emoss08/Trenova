package temporaltype

import (
	"time"

	"github.com/emoss08/trenova/shared/pulid"
)

// Query names for workflow queries
const (
	QueryWorkflowStatus       = "workflow-status"
	QueryShipmentProgress     = "shipment-progress"
	QuerySystemJobStatus      = "system-job-status"
	QueryNotificationStatus   = "notification-status"
	QueryDuplicationProgress  = "duplication-progress"
	QueryCancellationProgress = "cancellation-progress"
	QueryAuditDeletionStatus  = "audit-deletion-status"
)

// WorkflowStatus represents the overall status of a workflow
type WorkflowStatus string

const (
	WorkflowStatusInitialized WorkflowStatus = "initialized"
	WorkflowStatusRunning     WorkflowStatus = "running"
	WorkflowStatusPaused      WorkflowStatus = "paused"
	WorkflowStatusCompleted   WorkflowStatus = "completed"
	WorkflowStatusFailed      WorkflowStatus = "failed"
	WorkflowStatusCancelled   WorkflowStatus = "cancelled"
)

// WorkflowStatusQuery is the standard query for workflow status
type WorkflowStatusQuery struct {
	Status           WorkflowStatus `json:"status"`
	StartedAt        time.Time      `json:"startedAt"`
	LastUpdatedAt    time.Time      `json:"lastUpdatedAt"`
	Progress         float64        `json:"progress"` // 0.0 to 1.0
	CurrentOperation string         `json:"currentOperation"`
	ErrorMessage     string         `json:"errorMessage,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

// ShipmentProgressQuery provides detailed progress for shipment operations
type ShipmentProgressQuery struct {
	WorkflowStatusQuery
	ShipmentID       pulid.ID `json:"shipmentId"`
	TotalCount       int      `json:"totalCount"`
	ProcessedCount   int      `json:"processedCount"`
	SuccessCount     int      `json:"successCount"`
	FailedCount      int      `json:"failedCount"`
	CurrentShipment  string   `json:"currentShipment,omitempty"`
	ProcessedProNums []string `json:"processedProNums,omitempty"`
}

// DuplicationProgressQuery tracks shipment duplication progress
type DuplicationProgressQuery struct {
	ShipmentProgressQuery
	OriginalShipmentID pulid.ID `json:"originalShipmentId"`
	DuplicatedProNums  []string `json:"duplicatedProNums"`
	IncludedOptions    struct {
		Commodities       bool `json:"commodities"`
		AdditionalCharges bool `json:"additionalCharges"`
		OverrideDates     bool `json:"overrideDates"`
	} `json:"includedOptions"`
}

// CancellationProgressQuery tracks bulk cancellation progress
type CancellationProgressQuery struct {
	WorkflowStatusQuery
	TotalOrganizations      int                      `json:"totalOrganizations"`
	ProcessedOrganizations  int                      `json:"processedOrganizations"`
	TotalShipmentsCancelled int                      `json:"totalShipmentsCancelled"`
	CurrentOrganization     pulid.ID                 `json:"currentOrganization,omitempty"`
	OrganizationResults     []OrganizationCancelInfo `json:"organizationResults"`
}

// OrganizationCancelInfo contains cancellation info for an organization
type OrganizationCancelInfo struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	CancelledCount int      `json:"cancelledCount"`
	SkippedReason  string   `json:"skippedReason,omitempty"`
}

// SystemJobStatusQuery tracks system job execution
type SystemJobStatusQuery struct {
	WorkflowStatusQuery
	JobType             string    `json:"jobType"`
	LastExecutionAt     time.Time `json:"lastExecutionAt"`
	NextScheduledAt     time.Time `json:"nextScheduledAt"`
	ExecutionCount      int       `json:"executionCount"`
	ConsecutiveFailures int       `json:"consecutiveFailures"`
}

// AuditDeletionStatusQuery tracks audit deletion progress
type AuditDeletionStatusQuery struct {
	SystemJobStatusQuery
	TotalOrganizations     int            `json:"totalOrganizations"`
	ProcessedOrganizations int            `json:"processedOrganizations"`
	TotalDeleted           int            `json:"totalDeleted"`
	DeletedByOrg           map[string]int `json:"deletedByOrg"`
}

// NotificationStatusQuery tracks notification delivery
type NotificationStatusQuery struct {
	WorkflowStatusQuery
	NotificationType string    `json:"notificationType"`
	RecipientUserID  pulid.ID  `json:"recipientUserId"`
	DeliveryStatus   string    `json:"deliveryStatus"`
	DeliveryAttempts int       `json:"deliveryAttempts"`
	LastAttemptAt    time.Time `json:"lastAttemptAt,omitempty"`
	DeliveredAt      time.Time `json:"deliveredAt,omitempty"`
	FailureReason    string    `json:"failureReason,omitempty"`
}

// QueryResponse is a generic wrapper for query responses
type QueryResponse struct {
	QueryType string    `json:"queryType"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

// NewQueryResponse creates a new query response
func NewQueryResponse(queryType string, data any) QueryResponse {
	return QueryResponse{
		QueryType: queryType,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// Helper functions for creating query responses

// NewWorkflowStatusResponse creates a workflow status response
func NewWorkflowStatusResponse(
	status WorkflowStatus,
	progress float64,
	operation string,
) WorkflowStatusQuery {
	return WorkflowStatusQuery{
		Status:           status,
		StartedAt:        time.Now(), // Should be set when workflow starts
		LastUpdatedAt:    time.Now(),
		Progress:         progress,
		CurrentOperation: operation,
	}
}

// UpdateProgress updates the progress of a query status
func (q *WorkflowStatusQuery) UpdateProgress(progress float64, operation string) {
	q.Progress = progress
	q.CurrentOperation = operation
	q.LastUpdatedAt = time.Now()
}

// SetError sets error information in the query status
func (q *WorkflowStatusQuery) SetError(err error) {
	q.Status = WorkflowStatusFailed
	if err != nil {
		q.ErrorMessage = err.Error()
	}
	q.LastUpdatedAt = time.Now()
}

// Complete marks the workflow as completed
func (q *WorkflowStatusQuery) Complete() {
	q.Status = WorkflowStatusCompleted
	q.Progress = 1.0
	q.LastUpdatedAt = time.Now()
}

// CalculateProgress calculates progress percentage
func CalculateProgress(processed, total int) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(processed) / float64(total)
}
