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

type Lens string

const (
	LensForYou = Lens("forYou")
	LensTeam   = Lens("team")
	LensLater  = Lens("later")
	LensDone   = Lens("done")
	LensOpen   = Lens("open")
)

func (l Lens) String() string {
	return string(l)
}

func LensFromString(v string) (Lens, error) {
	lens := Lens(v)
	if !lens.IsValid() {
		return "", errors.New("invalid lens")
	}

	return lens, nil
}

type Reason string

const (
	ReasonDirect = Reason("direct")
	ReasonTeam   = Reason("team")
)

func (r Reason) String() string {
	return string(r)
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

func (v Priority) IsValid() bool {
	switch v {
	case PriorityCritical,
		PriorityHigh,
		PriorityMedium,
		PriorityLow:
		return true
	default:
		return false
	}
}

func (v Channel) IsValid() bool {
	switch v {
	case ChannelGlobal,
		ChannelUser,
		ChannelRole:
		return true
	default:
		return false
	}
}

func (v Lens) IsValid() bool {
	switch v {
	case LensForYou,
		LensTeam,
		LensLater,
		LensDone,
		LensOpen:
		return true
	default:
		return false
	}
}

func (v Reason) IsValid() bool {
	switch v {
	case ReasonDirect,
		ReasonTeam:
		return true
	default:
		return false
	}
}

func (v DeliveryStatus) IsValid() bool {
	switch v {
	case DeliveryStatusPending,
		DeliveryStatusDelivered,
		DeliveryStatusFailed,
		DeliveryStatusExpired:
		return true
	default:
		return false
	}
}
