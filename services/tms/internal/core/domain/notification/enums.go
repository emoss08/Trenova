package notification

import "errors"

type Priority string

const (
	PriorityCritical = Priority("critical")
	PriorityHigh     = Priority("high")
	PriorityMedium   = Priority("medium")
	PriorityLow      = Priority("low")
)

func (p Priority) String() string {
	return string(p)
}

func PriorityFromString(v string) (Priority, error) {
	switch v {
	case "critical":
		return PriorityCritical, nil
	case "high":
		return PriorityHigh, nil
	case "medium":
		return PriorityMedium, nil
	case "low":
		return PriorityLow, nil
	default:
		return "", errors.New("invalid priority")
	}
}

type Channel string

const (
	ChannelGlobal = Channel("global")
	ChannelUser   = Channel("user")
	ChannelRole   = Channel("role")
)

func (c Channel) String() string {
	return string(c)
}

func ChannelFromString(v string) (Channel, error) {
	switch v {
	case "global":
		return ChannelGlobal, nil
	case "user":
		return ChannelUser, nil
	case "role":
		return ChannelRole, nil
	default:
		return "", errors.New("invalid channel")
	}
}

type DeliveryStatus string

const (
	DeliveryStatusPending   = DeliveryStatus("pending")
	DeliveryStatusDelivered = DeliveryStatus("delivered")
	DeliveryStatusFailed    = DeliveryStatus("failed")
	DeliveryStatusExpired   = DeliveryStatus("expired")
)

func (d DeliveryStatus) String() string {
	return string(d)
}

func DeliveryStatusFromString(v string) (DeliveryStatus, error) {
	switch v {
	case "pending":
		return DeliveryStatusPending, nil
	case "delivered":
		return DeliveryStatusDelivered, nil
	case "failed":
		return DeliveryStatusFailed, nil
	case "expired":
		return DeliveryStatusExpired, nil
	default:
		return "", errors.New("invalid delivery status")
	}
}
