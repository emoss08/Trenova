package billing

// ExceptionHandling defines how to handle exceptions in the billing process
type ExceptionHandling string

const (
	// * Queue the shipment for billing
	BillingExceptionQueue = ExceptionHandling("Queue")

	// * Notify the user that the shipment is in exception
	BillingExceptionNotify = ExceptionHandling("Notify")

	// * Automatically resolve the exception
	BillingExceptionAutoResolve = ExceptionHandling("AutoResolve")

	// * Reject the shipment
	BillingExceptionReject = ExceptionHandling("Reject")
)

type PaymentTerm string

const (
	PaymentTermNet15        = PaymentTerm("Net15")
	PaymentTermNet30        = PaymentTerm("Net30")
	PaymentTermNet45        = PaymentTerm("Net45")
	PaymentTermNet60        = PaymentTerm("Net60")
	PaymentTermNet90        = PaymentTerm("Net90")
	PaymentTermDueOnReceipt = PaymentTerm("DueOnReceipt")
)

type TransferSchedule string

const (
	TransferScheduleContinuous = TransferSchedule("Continuous")
	TransferScheduleHourly     = TransferSchedule("Hourly")
	TransferScheduleDaily      = TransferSchedule("Daily")
	TransferScheduleWeekly     = TransferSchedule("Weekly")
)

type DocumentClassification string

const (
	// ClassificationPublic indicates the document is publicly available in the storage bucket
	ClassificationPublic = DocumentClassification("Public")

	// ClassificationPrivate indicates the document is private and must be shared by the owner in the storage bucket
	ClassificationPrivate = DocumentClassification("Private")

	// ClassificationSensitive indicates the document contains sensitive information and must be shared by the owner in the storage bucket.
	ClassificationSensitive = DocumentClassification("Sensitive")

	// ClassificationRegulatory indicates the document contains regulatory information and must be shared by the owner in the storage bucket.
	ClassificationRegulatory = DocumentClassification("Regulatory")
)

func (dc DocumentClassification) String() string {
	return string(dc)
}

type DocumentCategory string

const (
	CategoryShipment   = DocumentCategory("Shipment")   // BOL, POD, etc...
	CategoryWorker     = DocumentCategory("Worker")     // Worker docs, licenses
	CategoryRegulatory = DocumentCategory("Regulatory") // Regulatory docs, certificates, etc...
	CategoryProfile    = DocumentCategory("Profile")    // Profile photos, etc...
	CategoryBranding   = DocumentCategory("Branding")   // Branding files, etc...
	CategoryInvoice    = DocumentCategory("Invoice")    // Invoice files, etc...
	CategoryContract   = DocumentCategory("Contract")   // Contract files, etc...
	CategoryOther      = DocumentCategory("Other")      // Other files, etc...
)

func (dc DocumentCategory) String() string {
	return string(dc)
}
