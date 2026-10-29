package detentionnotice

import (
	"errors"
)

type NoticeChannel string

const (
	NoticeChannelEmail  = NoticeChannel("Email")
	NoticeChannelEDI    = NoticeChannel("EDI")
	NoticeChannelPortal = NoticeChannel("Portal")
	NoticeChannelManual = NoticeChannel("Manual")
)

func (c NoticeChannel) String() string { return string(c) }

func NoticeChannelFromString(v string) (NoticeChannel, error) {
	switch v {
	case "Email":
		return NoticeChannelEmail, nil
	case "EDI":
		return NoticeChannelEDI, nil
	case "Portal":
		return NoticeChannelPortal, nil
	case "Manual":
		return NoticeChannelManual, nil
	default:
		return "", errors.New("invalid detention notice channel")
	}
}

type NoticeKind string

const (
	NoticeKindWarning = NoticeKind("Warning")
	NoticeKindStarted = NoticeKind("Started")
	NoticeKindUpdate  = NoticeKind("Update")
	NoticeKindFinal   = NoticeKind("Final")
)

func (k NoticeKind) String() string { return string(k) }

func NoticeKindFromString(v string) (NoticeKind, error) {
	switch v {
	case "Warning":
		return NoticeKindWarning, nil
	case "Started":
		return NoticeKindStarted, nil
	case "Update":
		return NoticeKindUpdate, nil
	case "Final":
		return NoticeKindFinal, nil
	default:
		return "", errors.New("invalid detention notice kind")
	}
}

type NoticeDeliveryStatus string

const (
	NoticeDeliveryStatusQueued    = NoticeDeliveryStatus("Queued")
	NoticeDeliveryStatusSent      = NoticeDeliveryStatus("Sent")
	NoticeDeliveryStatusDelivered = NoticeDeliveryStatus("Delivered")
	NoticeDeliveryStatusOpened    = NoticeDeliveryStatus("Opened")
	NoticeDeliveryStatusBounced   = NoticeDeliveryStatus("Bounced")
	NoticeDeliveryStatusFailed    = NoticeDeliveryStatus("Failed")
)

func (s NoticeDeliveryStatus) String() string { return string(s) }

func (s NoticeDeliveryStatus) IsDelivered() bool {
	switch s {
	case NoticeDeliveryStatusDelivered, NoticeDeliveryStatusOpened:
		return true
	case NoticeDeliveryStatusQueued, NoticeDeliveryStatusSent,
		NoticeDeliveryStatusBounced, NoticeDeliveryStatusFailed:
		return false
	default:
		return false
	}
}

func (s NoticeDeliveryStatus) IsFailure() bool {
	switch s {
	case NoticeDeliveryStatusBounced, NoticeDeliveryStatusFailed:
		return true
	case NoticeDeliveryStatusQueued, NoticeDeliveryStatusSent,
		NoticeDeliveryStatusDelivered, NoticeDeliveryStatusOpened:
		return false
	default:
		return false
	}
}

func NoticeDeliveryStatusFromString(v string) (NoticeDeliveryStatus, error) {
	switch v {
	case "Queued":
		return NoticeDeliveryStatusQueued, nil
	case "Sent":
		return NoticeDeliveryStatusSent, nil
	case "Delivered":
		return NoticeDeliveryStatusDelivered, nil
	case "Opened":
		return NoticeDeliveryStatusOpened, nil
	case "Bounced":
		return NoticeDeliveryStatusBounced, nil
	case "Failed":
		return NoticeDeliveryStatusFailed, nil
	default:
		return "", errors.New("invalid detention notice delivery status")
	}
}
