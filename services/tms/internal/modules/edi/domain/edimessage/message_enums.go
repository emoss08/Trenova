package edimessage

type MessageStatus string

const (
	MessageStatusGenerated = MessageStatus("Generated")
	MessageStatusFailed    = MessageStatus("Failed")
)

type MessageDeliveryStatus string

const (
	MessageDeliveryStatusQueued       = MessageDeliveryStatus("Queued")
	MessageDeliveryStatusSending      = MessageDeliveryStatus("Sending")
	MessageDeliveryStatusSent         = MessageDeliveryStatus("Sent")
	MessageDeliveryStatusFailed       = MessageDeliveryStatus("Failed")
	MessageDeliveryStatusDeadLettered = MessageDeliveryStatus("DeadLettered")
)

func (s MessageDeliveryStatus) IsRetryable() bool {
	return s == MessageDeliveryStatusFailed || s == MessageDeliveryStatusDeadLettered
}

func (s MessageDeliveryStatus) IsDeliverable() bool {
	return s == MessageDeliveryStatusQueued ||
		s == MessageDeliveryStatusSending ||
		s == MessageDeliveryStatusFailed
}

type MessageAcknowledgmentStatus string

const (
	MessageAcknowledgmentStatusNotExpected        = MessageAcknowledgmentStatus("NotExpected")
	MessageAcknowledgmentStatusPending            = MessageAcknowledgmentStatus("Pending")
	MessageAcknowledgmentStatusAccepted           = MessageAcknowledgmentStatus("Accepted")
	MessageAcknowledgmentStatusAcceptedWithErrors = MessageAcknowledgmentStatus(
		"AcceptedWithErrors",
	)
	MessageAcknowledgmentStatusRejected = MessageAcknowledgmentStatus("Rejected")
	MessageAcknowledgmentStatusFailed   = MessageAcknowledgmentStatus("Failed")
)

func (s MessageStatus) IsValid() bool {
	switch s {
	case MessageStatusGenerated,
		MessageStatusFailed:
		return true
	default:
		return false
	}
}

func (s MessageDeliveryStatus) IsValid() bool {
	switch s {
	case MessageDeliveryStatusQueued,
		MessageDeliveryStatusSending,
		MessageDeliveryStatusSent,
		MessageDeliveryStatusFailed,
		MessageDeliveryStatusDeadLettered:
		return true
	default:
		return false
	}
}

func (s MessageAcknowledgmentStatus) IsReceived() bool {
	switch s {
	case MessageAcknowledgmentStatusAccepted,
		MessageAcknowledgmentStatusAcceptedWithErrors,
		MessageAcknowledgmentStatusRejected:
		return true
	case MessageAcknowledgmentStatusNotExpected,
		MessageAcknowledgmentStatusPending,
		MessageAcknowledgmentStatusFailed:
		return false
	default:
		return false
	}
}

func (s MessageAcknowledgmentStatus) IsValid() bool {
	switch s {
	case MessageAcknowledgmentStatusNotExpected,
		MessageAcknowledgmentStatusPending,
		MessageAcknowledgmentStatusAccepted,
		MessageAcknowledgmentStatusAcceptedWithErrors,
		MessageAcknowledgmentStatusRejected,
		MessageAcknowledgmentStatusFailed:
		return true
	default:
		return false
	}
}

const (
	maxSegmentIDLength = 10
)
