package session

import (
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
)

type Event struct {
	ID        pulid.ID       `json:"id"`
	SessionID pulid.ID       `json:"sessionId"`
	Type      EventType      `json:"type"`
	IP        string         `json:"ip"`
	UserAgent string         `json:"userAgent"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt int64          `json:"createdAt"`
}

type Session struct {
	ID             pulid.ID     `json:"id"`
	UserID         pulid.ID     `json:"userId"`
	BusinessUnitID pulid.ID     `json:"businessUnitId"`
	OrganizationID pulid.ID     `json:"organizationId"`
	Status         Status       `json:"status"`
	IP             string       `json:"ip"`
	UserAgent      string       `json:"userAgent"`
	LastAccessedAt int64        `json:"lastAccessedAt"`
	ExpiresAt      int64        `json:"expiresAt"`
	RevokedAt      *int64       `json:"revokedAt,omitempty"`
	CreatedAt      int64        `json:"createdAt"`
	UpdatedAt      int64        `json:"updatedAt"`
	User           *tenant.User `json:"user,omitempty"`
	Events         []Event      `json:"events,omitempty"`
}

type NewSessionRequest struct {
	UserID                pulid.ID
	BusinessUnitID        pulid.ID
	CurrentOrganizationID pulid.ID
	IP                    string
	UserAgent             string
	ExpiresAt             int64
}

func NewSession(
	req NewSessionRequest,
) *Session {
	now := utils.NowUnix()
	return &Session{
		ID:             pulid.MustNew("ses_"),
		UserID:         req.UserID,
		BusinessUnitID: req.BusinessUnitID,
		OrganizationID: req.CurrentOrganizationID,
		Status:         StatusActive,
		IP:             req.IP,
		UserAgent:      req.UserAgent,
		LastAccessedAt: now,
		ExpiresAt:      req.ExpiresAt,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func (s *Session) IsValid() bool {
	if s.Status != StatusActive {
		return false
	}
	now := utils.NowUnix()
	return s.ExpiresAt > now && (s.RevokedAt == nil || *s.RevokedAt > now)
}

func (s *Session) IsExpired() bool {
	return s.ExpiresAt < utils.NowUnix()
}

func (s *Session) Validate(clientIP string) error {
	if !s.IsValid() {
		return errortypes.NewBusinessError("Session is no longer valid. Please login again.")
	}
	if s.IsExpired() {
		return errortypes.NewBusinessError("Session has expired. Please login again.")
	}
	if s.IP != clientIP {
		return errortypes.NewBusinessError("Session IP mismatch. Please login again.")
	}
	return nil
}

func (s *Session) UpdateLastAccessedAt() {
	s.LastAccessedAt = utils.NowUnix()
	s.UpdatedAt = s.LastAccessedAt
}

func (s *Session) Revoke() {
	now := utils.NowUnix()
	s.Status = StatusRevoked
	s.RevokedAt = &now
	s.UpdatedAt = now
}
