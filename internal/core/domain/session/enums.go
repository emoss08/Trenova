package session

type Status string

const (
	StatusActive  = Status("Active")
	StatusExpired = Status("Expired")
	StatusRevoked = Status("Revoked")
	StatusInvalid = Status("Invalid")
)

type EventType string

const (
	EventTypeLogin          = EventType("Login")
	EventTypeLogout         = EventType("Logout")
	EventTypeRevoked        = EventType("Revoked")
	EventTypeExpired        = EventType("Expired")
	EventTypeInvalid        = EventType("InvalidAttempt")
	EventTypeLocationChange = EventType("LocationChange")
	EventTypeDeviceChange   = EventType("DeviceChange")
	EventTypeAccessed       = EventType("Accessed")
)
