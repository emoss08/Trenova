package notification

import "fmt"

type EventType string

const (
	EventJobUnknown = EventType("job.unknown")

	// EventJobShipmentDuplicate is fired when a shipment duplication job completes
	EventJobShipmentDuplicate = EventType("job.shipment.duplicate_complete")

	// EventJobPatternAnalysis is fired when a pattern analysis job completes
	EventJobPatternAnalysis = EventType("job.analysis.pattern_complete")

	// EventJobComplianceCheck is fired when a compliance check job completes
	EventJobComplianceCheck = EventType("job.compliance.check_complete")

	// EventJobBillingProcess is fired when a billing process job completes
	EventJobBillingProcess = EventType("job.billing.process_complete")

	// EventSystemMaintenance is fired when system maintenance is scheduled
	EventSystemMaintenance = EventType("system.maintenance.scheduled")

	// EventSystemAlert is fired when a critical system alert occurs
	EventSystemAlert = EventType("system.alert.critical")

	// EventShipmentStatusChange is fired when a shipment status changes
	EventShipmentStatusChange = EventType("business.shipment.status_change")

	// EventWorkerComplianceExpired is fired when a worker's compliance expires
	EventWorkerComplianceExpired = EventType("business.worker.compliance_expired")

	// EventCustomerDocumentReview is fired when a customer document needs review
	EventCustomerDocumentReview = EventType("business.customer.document_review")

	// Entity Update Events

	// EventEntityUpdated is fired when an entity owned by a user is updated by someone else
	EventEntityUpdated = EventType("entity.updated")

	// EventShipmentUpdated is fired when a shipment is updated
	EventShipmentUpdated = EventType("entity.shipment.updated")

	// EventWorkerUpdated is fired when a worker is updated
	EventWorkerUpdated = EventType("entity.worker.updated")

	// EventCustomerUpdated is fired when a customer is updated
	EventCustomerUpdated = EventType("entity.customer.updated")

	// EventTractorUpdated is fired when a tractor is updated
	EventTractorUpdated = EventType("entity.tractor.updated")

	// EventTrailerUpdated is fired when a trailer is updated
	EventTrailerUpdated = EventType("entity.trailer.updated")

	// EventLocationUpdated is fired when a location is updated
	EventLocationUpdated = EventType("entity.location.updated")

	// EventBatchSummary is fired for batched notifications
	EventBatchSummary = EventType("batch.summary")
)

// RateLimitPeriod represents the time period for rate limiting
type RateLimitPeriod string

const (
	RateLimitPeriodMinute = RateLimitPeriod("minute")
	RateLimitPeriodHour   = RateLimitPeriod("hour")
	RateLimitPeriodDay    = RateLimitPeriod("day")
)

// UpdateType represents the type of update that can trigger notifications
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
	// PriorityCritical is for system alerts and compliance violations
	PriorityCritical = Priority("critical")

	// PriorityHigh is for job failures and urgent approvals
	PriorityHigh = Priority("high")

	// PriorityMedium is for job completions and status updates
	PriorityMedium = Priority("medium")

	// PriorityLow is for info updates and suggestions
	PriorityLow = Priority("low")
)

// ShouldBypassQuietHours returns true if the priority level should bypass quiet hours
func (p Priority) ShouldBypassQuietHours() bool {
	return p == PriorityCritical || p == PriorityHigh
}

// ShouldBypassBatching returns true if the priority level should bypass batching
func (p Priority) ShouldBypassBatching() bool {
	return p == PriorityCritical || p == PriorityHigh
}

// CanBeBatched returns true if the priority level allows batching
func (p Priority) CanBeBatched() bool {
	return p == PriorityMedium || p == PriorityLow
}

// GetLevel returns the numeric level of the priority (higher is more important)
func (p Priority) GetLevel() int {
	switch p {
	case PriorityCritical:
		return 4
	case PriorityHigh:
		return 3
	case PriorityMedium:
		return 2
	case PriorityLow:
		return 1
	default:
		return 0
	}
}

type Channel string

const (
	// ChannelGlobal sends to all users in organization
	ChannelGlobal = Channel("global")

	// ChannelUser sends to a specific user
	ChannelUser = Channel("user")

	// ChannelRole sends to users with specific role in business unit/org
	ChannelRole = Channel("role")
)

type DeliveryStatus string

const (
	// DeliveryStatusPending indicates the notification is pending delivery
	DeliveryStatusPending = DeliveryStatus("pending")

	// DeliveryStatusDelivered indicates the notification has been delivered
	DeliveryStatusDelivered = DeliveryStatus("delivered")

	// DeliveryStatusFailed indicates the notification delivery failed
	DeliveryStatusFailed = DeliveryStatus("failed")

	// DeliveryStatusExpired indicates the notification has expired
	DeliveryStatusExpired = DeliveryStatus("expired")
)

// GenerateRoomName generates the WebSocket room name based on targeting
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
	case ChannelRole:
		return fmt.Sprint(
			"role_",
			targeting.OrganizationID.String(),
			"_",
			targeting.BusinessUnitID.String(),
			"_",
			targeting.TargetRoleID.String(),
		)
	default:
		return ""
	}
}
