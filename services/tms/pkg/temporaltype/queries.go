package temporaltype

import (
	"time"

	"github.com/emoss08/trenova/pkg/pulid"
)

const (
	QueryWorkflowStatus       = "workflow-status"
	QueryShipmentProgress     = "shipment-progress"
	QuerySystemJobStatus      = "system-job-status"
	QueryNotificationStatus   = "notification-status"
	QueryDuplicationProgress  = "duplication-progress"
	QueryCancellationProgress = "cancellation-progress"
	QueryAuditDeletionStatus  = "audit-deletion-status"
)

type WorkflowStatus string

const (
	WorkflowStatusInitialized WorkflowStatus = "initialized"
	WorkflowStatusRunning     WorkflowStatus = "running"
	WorkflowStatusPaused      WorkflowStatus = "paused"
	WorkflowStatusCompleted   WorkflowStatus = "completed"
	WorkflowStatusFailed      WorkflowStatus = "failed"
	WorkflowStatusCancelled   WorkflowStatus = "cancelled"
)

type WorkflowStatusQuery struct {
	Status           WorkflowStatus `json:"status"`
	StartedAt        time.Time      `json:"startedAt"`
	LastUpdatedAt    time.Time      `json:"lastUpdatedAt"`
	Progress         float64        `json:"progress"` // 0.0 to 1.0
	CurrentOperation string         `json:"currentOperation"`
	ErrorMessage     string         `json:"errorMessage,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

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

type CancellationProgressQuery struct {
	WorkflowStatusQuery
	TotalOrganizations      int                      `json:"totalOrganizations"`
	ProcessedOrganizations  int                      `json:"processedOrganizations"`
	TotalShipmentsCancelled int                      `json:"totalShipmentsCancelled"`
	CurrentOrganization     pulid.ID                 `json:"currentOrganization,omitempty"`
	OrganizationResults     []OrganizationCancelInfo `json:"organizationResults"`
}

type OrganizationCancelInfo struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	CancelledCount int      `json:"cancelledCount"`
	SkippedReason  string   `json:"skippedReason,omitempty"`
}

type SystemJobStatusQuery struct {
	WorkflowStatusQuery
	JobType             string    `json:"jobType"`
	LastExecutionAt     time.Time `json:"lastExecutionAt"`
	NextScheduledAt     time.Time `json:"nextScheduledAt"`
	ExecutionCount      int       `json:"executionCount"`
	ConsecutiveFailures int       `json:"consecutiveFailures"`
}

type AuditDeletionStatusQuery struct {
	SystemJobStatusQuery
	TotalOrganizations     int            `json:"totalOrganizations"`
	ProcessedOrganizations int            `json:"processedOrganizations"`
	TotalDeleted           int            `json:"totalDeleted"`
	DeletedByOrg           map[string]int `json:"deletedByOrg"`
}

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

type QueryResponse struct {
	QueryType string    `json:"queryType"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

func NewQueryResponse(queryType string, data any) QueryResponse {
	return QueryResponse{
		QueryType: queryType,
		Timestamp: time.Now(),
		Data:      data,
	}
}

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

func (q *WorkflowStatusQuery) UpdateProgress(progress float64, operation string) {
	q.Progress = progress
	q.CurrentOperation = operation
	q.LastUpdatedAt = time.Now()
}

func (q *WorkflowStatusQuery) SetError(err error) {
	q.Status = WorkflowStatusFailed
	if err != nil {
		q.ErrorMessage = err.Error()
	}
	q.LastUpdatedAt = time.Now()
}

func (q *WorkflowStatusQuery) Complete() {
	q.Status = WorkflowStatusCompleted
	q.Progress = 1.0
	q.LastUpdatedAt = time.Now()
}

func CalculateProgress(processed, total int) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(processed) / float64(total)
}
