package tablechangealert

import "errors"

type SubscriptionStatus string

const (
	SubscriptionStatusActive = SubscriptionStatus("Active")
	SubscriptionStatusPaused = SubscriptionStatus("Paused")
)

func (s SubscriptionStatus) String() string {
	return string(s)
}

func SubscriptionStatusFromString(v string) (SubscriptionStatus, error) {
	switch v {
	case "Active":
		return SubscriptionStatusActive, nil
	case "Paused":
		return SubscriptionStatusPaused, nil
	default:
		return "", errors.New("invalid subscription status")
	}
}

type EventType string

const (
	EventTypeInsert = EventType("INSERT")
	EventTypeUpdate = EventType("UPDATE")
	EventTypeDelete = EventType("DELETE")
)

func (e EventType) String() string {
	return string(e)
}

func ValidEventType(v string) bool {
	switch v {
	case "INSERT", "UPDATE", "DELETE":
		return true
	default:
		return false
	}
}
