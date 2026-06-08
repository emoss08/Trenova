package email

import "slices"

type Provider string

const (
	ProviderResend   Provider = "Resend"
	ProviderPostmark Provider = "Postmark"
)

type AuthType string

const (
	AuthTypeAPIKey AuthType = "APIKey"
)

type Encryption string

const (
	EncryptionNone Encryption = "None"
)

type ProfileStatus string

const (
	ProfileStatusActive   ProfileStatus = "Active"
	ProfileStatusInactive ProfileStatus = "Inactive"
)

type Purpose string

const (
	PurposeGeneral        Purpose = "General"
	PurposeBilling        Purpose = "Billing"
	PurposeReporting      Purpose = "Reporting"
	PurposeOperations     Purpose = "Operations"
	PurposeAuthentication Purpose = "Authentication"
	PurposeNotifications  Purpose = "Notifications"
)

type MessageStatus string

const (
	MessageStatusQueued     MessageStatus = "Queued"
	MessageStatusSending    MessageStatus = "Sending"
	MessageStatusSent       MessageStatus = "Sent"
	MessageStatusDelivered  MessageStatus = "Delivered"
	MessageStatusFailed     MessageStatus = "Failed"
	MessageStatusBounced    MessageStatus = "Bounced"
	MessageStatusComplained MessageStatus = "Complained"
	MessageStatusOpened     MessageStatus = "Opened"
	MessageStatusClicked    MessageStatus = "Clicked"
	MessageStatusSuppressed MessageStatus = "Suppressed"
)

type EventType string

const (
	EventTypeSent       EventType = "Sent"
	EventTypeDelivered  EventType = "Delivered"
	EventTypeOpened     EventType = "Opened"
	EventTypeClicked    EventType = "Clicked"
	EventTypeBounced    EventType = "Bounced"
	EventTypeComplained EventType = "Complained"
	EventTypeFailed     EventType = "Failed"
)

type SuppressionReason string

const (
	SuppressionReasonHardBounce      SuppressionReason = "HardBounce"
	SuppressionReasonComplaint       SuppressionReason = "Complaint"
	SuppressionReasonSoftBounceLimit SuppressionReason = "SoftBounceLimit"
	SuppressionReasonManual          SuppressionReason = "Manual"
)

func Purposes() []Purpose {
	return []Purpose{
		PurposeGeneral,
		PurposeBilling,
		PurposeReporting,
		PurposeOperations,
		PurposeAuthentication,
		PurposeNotifications,
	}
}

func IsValidPurpose(p Purpose) bool {
	return slices.Contains(Purposes(), p)
}
