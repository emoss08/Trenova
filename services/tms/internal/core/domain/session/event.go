/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package session

import (
	"github.com/emoss08/trenova/pkg/types/pulid"
)

// Event represents a session event
type Event struct {
	ID        pulid.ID       `json:"id"`
	SessionID pulid.ID       `json:"sessionId"`
	Type      EventType      `json:"type"`
	IP        string         `json:"ip"`
	UserAgent string         `json:"userAgent"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt int64          `json:"createdAt"`
}
