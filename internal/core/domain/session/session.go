package session

import (
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

// Session represents a user session
type Session struct {
	// Primary identifiers
	ID             pulid.ID `json:"id"`
	UserID         pulid.ID `json:"userId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	OrganizationID pulid.ID `json:"organizationId"`

	// Core fields
	Status         Status `json:"status"`
	IP             string `json:"ip"`
	UserAgent      string `json:"userAgent"`
	LastAccessedAt int64  `json:"lastAccessedAt"`
	ExpiresAt      int64  `json:"expiresAt"`
	RevokedAt      *int64 `json:"revokedAt,omitempty"`
	CreatedAt      int64  `json:"createdAt"`
	UpdatedAt      int64  `json:"updatedAt"`

	// Related entities
	User   *user.User `json:"user,omitempty"`
	Events []Event    `json:"events,omitempty"`
}

// NewSession creates a new session
func NewSession(
	userID, businessUnitID, organizationID pulid.ID,
	ip, userAgent string,
	expiresAt int64,
) *Session {
	now := timeutils.NowUnix()
	return &Session{
		ID:             pulid.MustNew("ses_"),
		UserID:         userID,
		BusinessUnitID: businessUnitID,
		OrganizationID: organizationID,
		Status:         StatusActive,
		IP:             ip,
		UserAgent:      userAgent,
		LastAccessedAt: now,
		ExpiresAt:      expiresAt,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// IsValid checks if the session is valid
func (s *Session) IsValid() bool {
	if s.Status != StatusActive {
		return false
	}
	now := timeutils.NowUnix()
	return s.ExpiresAt > now && (s.RevokedAt == nil || *s.RevokedAt > now)
}

// IsExpired returns true if the session is expired
func (s *Session) IsExpired() bool {
	return s.ExpiresAt < timeutils.NowUnix()
}

// Validate validates the session
func (s *Session) Validate(clientIP string) error {
	if !s.IsValid() {
		return errors.NewBusinessError("session is not valid")
	}
	if s.IsExpired() {
		return errors.NewBusinessError("session is expired")
	}
	if s.IP != clientIP {
		return errors.NewBusinessError("session IP mismatch")
	}
	return nil
}

// UpdateLastAccessedAt updates the last accessed at timestamp
func (s *Session) UpdateLastAccessedAt() {
	s.LastAccessedAt = timeutils.NowUnix()
	s.UpdatedAt = s.LastAccessedAt
}

// Revoke revokes the session
func (s *Session) Revoke() {
	now := timeutils.NowUnix()
	s.Status = StatusRevoked
	s.RevokedAt = &now
	s.UpdatedAt = now
}

// AddEvent adds an event to the session
func (s *Session) AddEvent(
	eventType EventType,
	ip, userAgent string,
	metadata map[string]any,
) *Event {
	event := &Event{
		ID:        pulid.MustNew("sev_"),
		SessionID: s.ID,
		Type:      eventType,
		IP:        ip,
		UserAgent: userAgent,
		Metadata:  metadata,
		CreatedAt: timeutils.NowUnix(),
	}
	s.Events = append(s.Events, *event)
	return event
}
