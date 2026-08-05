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

func (v Provider) IsValid() bool {
	switch v {
	case ProviderResend,
		ProviderPostmark:
		return true
	default:
		return false
	}
}

func (v AuthType) IsValid() bool {
	switch v {
	case AuthTypeAPIKey:
		return true
	default:
		return false
	}
}

func (v Encryption) IsValid() bool {
	switch v {
	case EncryptionNone:
		return true
	default:
		return false
	}
}

func (v ProfileStatus) IsValid() bool {
	switch v {
	case ProfileStatusActive,
		ProfileStatusInactive:
		return true
	default:
		return false
	}
}

func (v Purpose) IsValid() bool {
	switch v {
	case PurposeGeneral,
		PurposeBilling,
		PurposeReporting,
		PurposeOperations,
		PurposeAuthentication,
		PurposeNotifications:
		return true
	default:
		return false
	}
}

func (v MessageStatus) IsValid() bool {
	switch v {
	case MessageStatusQueued,
		MessageStatusSending,
		MessageStatusSent,
		MessageStatusDelivered,
		MessageStatusFailed,
		MessageStatusBounced,
		MessageStatusComplained,
		MessageStatusOpened,
		MessageStatusClicked,
		MessageStatusSuppressed:
		return true
	default:
		return false
	}
}

func (v EventType) IsValid() bool {
	switch v {
	case EventTypeSent,
		EventTypeDelivered,
		EventTypeOpened,
		EventTypeClicked,
		EventTypeBounced,
		EventTypeComplained,
		EventTypeFailed:
		return true
	default:
		return false
	}
}

func (v SuppressionReason) IsValid() bool {
	switch v {
	case SuppressionReasonHardBounce,
		SuppressionReasonComplaint,
		SuppressionReasonSoftBounceLimit,
		SuppressionReasonManual:
		return true
	default:
		return false
	}
}
