package notification

import "fmt"

type EventType string

const (
	EventJobUnknown                   = EventType("job.unknown")
	EventJobShipmentDuplicate         = EventType("job.shipment.duplicate_complete")
	EventJobPatternAnalysis           = EventType("job.analysis.pattern_complete")
	EventJobShipmentDelay             = EventType("job.shipment.delay_complete")
	EventJobComplianceCheck           = EventType("job.compliance.check_complete")
	EventJobBillingProcess            = EventType("job.billing.process_complete")
	EventSystemMaintenance            = EventType("system.maintenance.scheduled")
	EventConfigurationCopied          = EventType("configuration.copied")
	EventShipmentHoldRelease          = EventType("shipment.hold.released")
	EventShipmentOwnershipTransferred = EventType("shipment.ownership.transferred")
	EventShipmentComment              = EventType("shipment.comment.created")
	EventSystemAlert                  = EventType("system.alert.critical")
	EventShipmentStatusChange         = EventType("business.shipment.status_change")
	EventWorkerComplianceExpired      = EventType("business.worker.compliance_expired")
	EventCustomerDocumentReview       = EventType("business.customer.document_review")
	EventEntityUpdated                = EventType("entity.updated")
	EventShipmentUpdated              = EventType("entity.shipment.updated")
	EventWorkerUpdated                = EventType("entity.worker.updated")
	EventCustomerUpdated              = EventType("entity.customer.updated")
	EventTractorUpdated               = EventType("entity.tractor.updated")
	EventTrailerUpdated               = EventType("entity.trailer.updated")
	EventLocationUpdated              = EventType("entity.location.updated")
	EventBatchSummary                 = EventType("batch.summary")
)

type RateLimitPeriod string

const (
	RateLimitPeriodMinute = RateLimitPeriod("minute")
	RateLimitPeriodHour   = RateLimitPeriod("hour")
	RateLimitPeriodDay    = RateLimitPeriod("day")
)

type UpdateType string

const (
	UpdateTypeAny              = UpdateType("any")
	UpdateTypeStatusChange     = UpdateType("status_change")
	UpdateTypeAssignment       = UpdateType("assignment")
	UpdateTypeDateChange       = UpdateType("date_change")
	UpdateTypeLocationChange   = UpdateType("location_change")
	UpdateTypeDocumentUpload   = UpdateType("document_upload")
	UpdateTypePriceChange      = UpdateType("price_change")
	UpdateTypeComplianceChange = UpdateType("compliance_change")
	UpdateTypeFieldChange      = UpdateType("field_change")
)

type Priority string

const (
	PriorityCritical = Priority("critical")
	PriorityHigh     = Priority("high")
	PriorityMedium   = Priority("medium")
	PriorityLow      = Priority("low")
)

type Channel string

const (
	ChannelGlobal = Channel("global")
	ChannelUser   = Channel("user")
)

type DeliveryStatus string

const (
	DeliveryStatusPending   = DeliveryStatus("pending")
	DeliveryStatusDelivered = DeliveryStatus("delivered")
	DeliveryStatusFailed    = DeliveryStatus("failed")
	DeliveryStatusExpired   = DeliveryStatus("expired")
)

func GenerateRoomName(targeting Targeting) string {
	switch targeting.Channel {
	case ChannelGlobal:
		return fmt.Sprint("org_", targeting.OrganizationID.String())

	case ChannelUser:
		return fmt.Sprint(
			"user_",
			targeting.OrganizationID.String(),
			"_",
			targeting.TargetUserID.String(),
		)

	default:
		return ""
	}
}
