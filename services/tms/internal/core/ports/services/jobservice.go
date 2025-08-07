package services

import (
	"context"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/pulid"
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
	JobTypeDelayShipment        JobType = "shipment:delay"
	JobTypeShipmentNotification JobType = "shipment:notification"

	// Compliance Jobs
	JobTypeComplianceCheck       JobType = "compliance:check"
	JobTypeHazmatExpirationCheck JobType = "compliance:hazmat_expiration"

	// System Jobs
	JobTypeCleanupTempFiles JobType = "system:cleanup_temp_files"
	JobTypeGenerateReports  JobType = "system:generate_reports"
	JobTypeDataBackup       JobType = "system:data_backup"

	// Email Jobs
	JobTypeSendEmail         JobType = "email:send"
	JobTypeProcessEmailQueue JobType = "email:process_queue"
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
	QueueShipment   = "shipment"
	QueueSystem     = "system"
	QueueCritical   = "critical"
	QueueEmail      = "email"
)

// JobHandler defines the interface for handling background jobs
type JobHandler interface {
	ProcessTask(ctx context.Context, task *asynq.Task) error
	JobType() JobType
}

// BasePayload contains common fields for all job payloads
type JobBasePayload struct {
	JobID          string         `json:"jobId"`
	OrganizationID pulid.ID       `json:"organizationId"`
	BusinessUnitID pulid.ID       `json:"businessUnitId"`
	UserID         pulid.ID       `json:"userId,omitempty"`
	Timestamp      int64          `json:"timestamp"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// PatternAnalysisPayload for dedicated lane pattern analysis jobs
type PatternAnalysisPayload struct {
	JobBasePayload
	MinFrequency  int64  `json:"minFrequency"`
	TriggerReason string `json:"triggerReason"` // "shipment_created", "scheduled", "manual"
}

// ExpireSuggestionsPayload for expiring old suggestions
type ExpireSuggestionsPayload struct {
	JobBasePayload
	BatchSize int `json:"batchSize"`
}

// ShipmentStatusUpdatePayload for shipment status change notifications
type ShipmentStatusUpdatePayload struct {
	JobBasePayload
	ShipmentID pulid.ID `json:"shipmentId"`
	OldStatus  string   `json:"oldStatus"`
	NewStatus  string   `json:"newStatus"`
}

type DuplicateShipmentPayload struct {
	JobBasePayload
	ShipmentID               pulid.ID `json:"shipmentId"`
	Count                    int      `json:"count"`
	OverrideDates            bool     `json:"overrideDates"`
	IncludeCommodities       bool     `json:"includeCommodities"`
	IncludeAdditionalCharges bool     `json:"includeAdditionalCharges"`
}

type DelayShipmentPayload struct {
	JobBasePayload
}

// ComplianceCheckPayload for compliance verification jobs
type ComplianceCheckPayload struct {
	JobBasePayload
	WorkerID   *pulid.ID `json:"workerId,omitempty"`
	ShipmentID *pulid.ID `json:"shipmentId,omitempty"`
	CheckType  string    `json:"checkType"` // "license", "medical", "hazmat", "all"
}

// SystemMaintenancePayload for system maintenance jobs
type SystemMaintenancePayload struct {
	JobBasePayload
	TaskType   string            `json:"taskType"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// SendEmailPayload for email sending jobs
type SendEmailPayload struct {
	JobBasePayload
	EmailType        string                     `json:"emailType"` // "regular" or "templated"
	Request          *SendEmailRequest          `json:"request,omitempty"`
	TemplatedRequest *SendTemplatedEmailRequest `json:"templatedRequest,omitempty"`
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

func DelayShipmentOptions() *JobOptions {
	return &JobOptions{
		Queue:    QueueShipment,
		Priority: PriorityHigh,
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

// JobServiceInterface defines the contract for the job service
type JobService interface {
	// Job Scheduling
	Enqueue(
		jobType JobType,
		payload any,
		opts *JobOptions,
	) (*asynq.TaskInfo, error)
	EnqueueIn(
		jobType JobType,
		payload any,
		delay time.Duration,
		opts *JobOptions,
	) (*asynq.TaskInfo, error)
	EnqueueAt(
		jobType JobType,
		payload any,
		processAt time.Time,
		opts *JobOptions,
	) (*asynq.TaskInfo, error)

	// Dedicated Lane Pattern Analysis Jobs
	SchedulePatternAnalysis(
		payload *PatternAnalysisPayload,
		opts *JobOptions,
	) (*asynq.TaskInfo, error)
	ScheduleDelayShipmentJobs(
		payload *DelayShipmentPayload,
		opts *JobOptions,
	) (*asynq.TaskInfo, error)
	ScheduleExpireSuggestions(
		payload *ExpireSuggestionsPayload,
		opts *JobOptions,
	) (*asynq.TaskInfo, error)

	// System Jobs
	ScheduleComplianceCheck(
		payload *ComplianceCheckPayload,
		opts *JobOptions,
	) (*asynq.TaskInfo, error)
	ScheduleShipmentStatusUpdate(
		payload *ShipmentStatusUpdatePayload,
		opts *JobOptions,
	) (*asynq.TaskInfo, error)

	// Email Jobs
	ScheduleSendEmail(
		payload *SendEmailPayload,
		opts *JobOptions,
	) (*asynq.TaskInfo, error)

	// Job Management
	CancelJob(jobID string) error
	GetJobInfo(queue string, jobID string) (*asynq.TaskInfo, error)

	// Worker Management
	Start() error
	Shutdown() error
	RegisterHandler(handler JobHandler)

	// Health Monitoring
	IsHealthy() bool
	GetStats() JobServiceStats
}
