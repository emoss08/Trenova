package jobs

import (
	"context"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/hibiken/asynq"
)

// JobType defines the different types of background jobs
type JobType string

const (
	// Pattern Analysis Jobs
	JobTypeAnalyzePatterns      JobType = "pattern:analyze"
	JobTypeExpireOldSuggestions JobType = "pattern:expire_suggestions"

	// Shipment Jobs
	JobTypeShipmentStatusUpdate JobType = "shipment:status_update"
	JobTypeDuplicateShipment    JobType = "shipment:duplicate"
	JobTypeShipmentNotification JobType = "shipment:notification"

	// Compliance Jobs
	JobTypeComplianceCheck       JobType = "compliance:check"
	JobTypeHazmatExpirationCheck JobType = "compliance:hazmat_expiration"

	// System Jobs
	JobTypeCleanupTempFiles JobType = "system:cleanup_temp_files"
	JobTypeGenerateReports  JobType = "system:generate_reports"
	JobTypeDataBackup       JobType = "system:data_backup"
)

// Priority levels for job processing
const (
	PriorityLow      = 1
	PriorityNormal   = 5
	PriorityHigh     = 10
	PriorityCritical = 20
)

// Queue names for different job categories
const (
	QueueDefault    = "default"
	QueuePattern    = "pattern_analysis"
	QueueCompliance = "compliance"
	QueueSystem     = "system"
	QueueCritical   = "critical"
)

// JobHandler defines the interface for handling background jobs
type JobHandler interface {
	ProcessTask(ctx context.Context, task *asynq.Task) error
	JobType() JobType
}

// BasePayload contains common fields for all job payloads
type BasePayload struct {
	JobID          string         `json:"jobId"`
	OrganizationID pulid.ID       `json:"organizationId"`
	BusinessUnitID pulid.ID       `json:"businessUnitId"`
	UserID         pulid.ID       `json:"userId,omitempty"`
	Timestamp      int64          `json:"timestamp"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// PatternAnalysisPayload for dedicated lane pattern analysis jobs
type PatternAnalysisPayload struct {
	BasePayload
	CustomerID    *pulid.ID `json:"customerId,omitempty"`
	StartDate     int64     `json:"startDate"`
	EndDate       int64     `json:"endDate"`
	MinFrequency  int64     `json:"minFrequency"`
	TriggerReason string    `json:"triggerReason"` // "shipment_created", "scheduled", "manual"
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

type DuplicateShipmentPayload struct {
	BasePayload
	ShipmentID               pulid.ID `json:"shipmentId"`
	Count                    int      `json:"count"`
	OverrideDates            bool     `json:"overrideDates"`
	IncludeCommodities       bool     `json:"includeCommodities"`
	IncludeAdditionalCharges bool     `json:"includeAdditionalCharges"`
}

// ComplianceCheckPayload for compliance verification jobs
type ComplianceCheckPayload struct {
	BasePayload
	WorkerID   *pulid.ID `json:"workerId,omitempty"`
	ShipmentID *pulid.ID `json:"shipmentId,omitempty"`
	CheckType  string    `json:"checkType"` // "license", "medical", "hazmat", "all"
}

// SystemMaintenancePayload for system maintenance jobs
type SystemMaintenancePayload struct {
	BasePayload
	TaskType   string            `json:"taskType"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// JobOptions defines options for job scheduling
type JobOptions struct {
	Queue     string
	Priority  int
	MaxRetry  int
	Delay     int64  // seconds
	Deadline  int64  // unix timestamp
	UniqueKey string // for job deduplication
	ProcessIn int64  // seconds from now
}

// DefaultJobOptions returns sensible defaults for job options
func DefaultJobOptions() *JobOptions {
	return &JobOptions{
		Queue:    QueueDefault,
		Priority: PriorityNormal,
		MaxRetry: 3,
	}
}

// PatternAnalysisOptions returns optimized options for pattern analysis jobs
func PatternAnalysisOptions() *JobOptions {
	return &JobOptions{
		Queue:    QueuePattern,
		Priority: PriorityNormal,
		MaxRetry: 2,
	}
}

// CriticalJobOptions returns options for critical system jobs
func CriticalJobOptions() *JobOptions {
	return &JobOptions{
		Queue:    QueueCritical,
		Priority: PriorityCritical,
		MaxRetry: 5,
	}
}

// JobServiceStats provides health and performance metrics
type JobServiceStats struct {
	IsRunning    bool       `json:"isRunning"`
	StartTime    time.Time  `json:"startTime"`
	Uptime       string     `json:"uptime"`
	PanicCount   int        `json:"panicCount"`
	LastPanic    *time.Time `json:"lastPanic,omitempty"`
	HandlerCount int        `json:"handlerCount"`
}

// Helper functions for payload marshaling
func MarshalPayload(payload any) ([]byte, error) {
	return sonic.Marshal(payload)
}

func UnmarshalPayload(data []byte, payload any) error {
	return sonic.Unmarshal(data, payload)
}
