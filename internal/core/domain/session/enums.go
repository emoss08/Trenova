/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
