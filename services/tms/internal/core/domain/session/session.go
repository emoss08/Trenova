package session

import (
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const DefaultTTL = 30 * 24 * time.Hour

type Session struct {
	ID             pulid.ID     `json:"id"`
	UserID         pulid.ID     `json:"userId"`
	BusinessUnitID pulid.ID     `json:"businessUnitId"`
	OrganizationID pulid.ID     `json:"organizationId"`
	LastAccessedAt int64        `json:"lastAccessedAt"`
	ExpiresAt      int64        `json:"expiresAt"`
	CreatedAt      int64        `json:"createdAt"`
	UpdatedAt      int64        `json:"updatedAt"`
	User           *tenant.User `json:"user"`
}

type NewSessionRequest struct {
	TenantInfo pagination.TenantInfo
	ExpiresAt  int64
}

func NewSession(req *NewSessionRequest) *Session {
	now := timeutils.NowUnix()
	return &Session{
		ID:             pulid.MustNew("ses_"),
		BusinessUnitID: req.TenantInfo.BuID,
		UserID:         req.TenantInfo.UserID,
		OrganizationID: req.TenantInfo.OrgID,
		LastAccessedAt: now,
		ExpiresAt:      req.ExpiresAt,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func (s *Session) IsValid() bool {
	return !s.IsExpired()
}

func (s *Session) IsExpired() bool {
	return s.ExpiresAt < timeutils.NowUnix()
}

func (s *Session) UpdateLastAccessedAt() {
	s.LastAccessedAt = timeutils.NowUnix()
	s.UpdatedAt = s.LastAccessedAt
}

func (s *Session) Validate() error {
	if s.IsExpired() {
		return errortypes.NewBusinessError("Session has expired. Please login again.")
	}

	return nil
}
