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
