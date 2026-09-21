package temporaltype

type TaskQueue string

const (
	TaskQueueAudit                TaskQueue = "audit-queue"
	TaskQueueDispatch             TaskQueue = "dispatch-queue"
	TaskQueueBilling              TaskQueue = "billing-queue"
	TaskQueueDocumentIntelligence TaskQueue = "document-intelligence-queue"
	TaskQueueIntegration          TaskQueue = "integration-queue"
	TaskQueueDistanceMileage      TaskQueue = "distance-mileage-queue"
	TaskQueueWeatherAlert         TaskQueue = "weather-alert-queue"
	TaskQueueSystem               TaskQueue = "system-queue"
	TaskQueueThumbnail            TaskQueue = "thumbnail-queue"
	TaskQueueUpload               TaskQueue = "upload-queue"
	TaskQueueSMS                  TaskQueue = "sms-queue"
	TaskQueueEmail                TaskQueue = "email-queue"
	TaskQueueFiscal               TaskQueue = "fiscal-queue"
	TaskQueueEDI                  TaskQueue = "edi-queue"
	TaskQueueReport               TaskQueue = "report-queue"
	// TaskQueueAgent carries the cheap periodic work: the sweep that starts
	// due agents, proposal expiry, ask-thread retention.
	//
	// It is also where every run started before the split still dispatches. A
	// workflow's task queue is fixed when it starts, so retiring this queue
	// while runs are in flight would strand them until their timeouts fire.
	TaskQueueAgent TaskQueue = "agent-queue"
	// TaskQueueAgentChat carries interactive turns, where somebody is watching
	// a reply arrive.
	TaskQueueAgentChat TaskQueue = "agent-chat-queue"
	// TaskQueueAgentBackground carries scheduled and event-driven runs, which
	// are long and which nobody is waiting on. Keeping them off the chat queue
	// is the point: a research run must not hold the slot a person's question
	// needs.
	TaskQueueAgentBackground TaskQueue = "agent-background-queue"
	// TaskQueueAgentHeavy carries replays of recorded runs, which cost as much
	// as a run and are never urgent.
	TaskQueueAgentHeavy TaskQueue = "agent-heavy-queue"
)

func (t TaskQueue) String() string {
	return string(t)
}

const AuditTaskQueue = string(TaskQueueAudit)
const DocumentIntelligenceTaskQueue = string(TaskQueueDocumentIntelligence)
const ThumbnailTaskQueue = string(TaskQueueThumbnail)
const UploadTaskQueue = string(TaskQueueUpload)
const SMSTaskQueue = string(TaskQueueSMS)
const EmailTaskQueue = string(TaskQueueEmail)
const FiscalTaskQueue = string(TaskQueueFiscal)
const IntegrationTaskQueue = string(TaskQueueIntegration)
const DistanceMileageTaskQueue = string(TaskQueueDistanceMileage)
const WeatherAlertTaskQueue = string(TaskQueueWeatherAlert)
const EDITaskQueue = string(TaskQueueEDI)
const ReportTaskQueue = string(TaskQueueReport)
