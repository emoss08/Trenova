package temporaltype

type TaskQueue string

const (
	TaskQueueAudit       TaskQueue = "audit-queue"
	TaskQueueDispatch    TaskQueue = "dispatch-queue"
	TaskQueueBilling     TaskQueue = "billing-queue"
	TaskQueueDocumentIntelligence TaskQueue = "document-intelligence-queue"
	TaskQueueIntegration TaskQueue = "integration-queue"
	TaskQueueSystem      TaskQueue = "system-queue"
	TaskQueueThumbnail   TaskQueue = "thumbnail-queue"
	TaskQueueUpload      TaskQueue = "upload-queue"
	TaskQueueSMS         TaskQueue = "sms-queue"
	TaskQueueFiscal      TaskQueue = "fiscal-queue"
)

func (t TaskQueue) String() string {
	return string(t)
}

const AuditTaskQueue = string(TaskQueueAudit)
const DocumentIntelligenceTaskQueue = string(TaskQueueDocumentIntelligence)
const ThumbnailTaskQueue = string(TaskQueueThumbnail)
const UploadTaskQueue = string(TaskQueueUpload)
const SMSTaskQueue = string(TaskQueueSMS)
const FiscalTaskQueue = string(TaskQueueFiscal)
const IntegrationTaskQueue = string(TaskQueueIntegration)
