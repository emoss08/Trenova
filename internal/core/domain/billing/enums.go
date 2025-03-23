package billing

type TransferCriteria string

const (
	// * Shipment must be marked as ready to bill and completed before transfer is allowed.
	TransferCriteriaReadyAndCompleted = TransferCriteria("ReadyAndCompleted")

	// * Shipment must be marked as completed before transfer is allowed.
	TransferCriteriaCompleted = TransferCriteria("Completed")

	// * Shipment must be marked as ready to bill before transfer is allowed.
	TransferCriteriaReadyToBill = TransferCriteria("ReadyToBill")

	// * Shipment must have all required documents attached
	TransferCriteriaDocumentsAttached = TransferCriteria("DocumentsAttached")

	// * Shipment must have proof of delivery
	TransferCriteriaPODReceived = TransferCriteria("PODReceived")
)

type AutoBillCriteria string

const (
	// * Shipment must be delivered before billing can occur
	AutoBillCriteriaDelivered = AutoBillCriteria("Delivered")

	// * Shipment must be transferred before billing can occur
	AutoBillCriteriaTransferred = AutoBillCriteria("Transferred")

	// * Shipment must be marked as ready to bill before billing can occur
	AutoBillCriteriaMarkedReadyToBill = AutoBillCriteria("MarkedReadyToBill")

	// * Shipment must have a proof of delivery before billing can occur
	AutoBillCriteriaPODReceived = AutoBillCriteria("PODReceived")

	// * Shipment must have all required documents attached before billing can occur
	AutoBillCriteriaDocumentsVerified = AutoBillCriteria("DocumentsVerified")
)

// BillingExceptionHandling defines how to handle exceptions in the billing process
type BillingExceptionHandling string

const (
	// * Queue the shipment for billing
	BillingExceptionQueue = BillingExceptionHandling("Queue")

	// * Notify the user that the shipment is in exception
	BillingExceptionNotify = BillingExceptionHandling("Notify")

	// * Automatically resolve the exception
	BillingExceptionAutoResolve = BillingExceptionHandling("AutoResolve")

	// * Reject the shipment
	BillingExceptionReject = BillingExceptionHandling("Reject")
)
