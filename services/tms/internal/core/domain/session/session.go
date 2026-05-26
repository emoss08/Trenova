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
	ID                    pulid.ID     `json:"id"`
	UserID                pulid.ID     `json:"userId"`
	BusinessUnitID        pulid.ID     `json:"businessUnitId"`
	OrganizationID        pulid.ID     `json:"organizationId"`
	ActiveRoleIDs         []pulid.ID   `json:"activeRoleIds"`
	AuthProvider          string       `json:"authProvider,omitempty"`
	ExternalIdentityID    string       `json:"externalIdentityId,omitempty"`
	ExternalSubject       string       `json:"externalSubject,omitempty"`
	AuthenticatorAAL      int          `json:"authenticatorAal"`
	FederationFAL         int          `json:"federationFal"`
	MFAAuthenticatedAt    int64        `json:"mfaAuthenticatedAt,omitempty"`
	LastReauthenticatedAt int64        `json:"lastReauthenticatedAt,omitempty"`
	RiskDecision          string       `json:"riskDecision,omitempty"`
	RiskDecisionID        pulid.ID     `json:"riskDecisionId,omitempty"`
	LastAccessedAt        int64        `json:"lastAccessedAt"`
	ExpiresAt             int64        `json:"expiresAt"`
	CreatedAt             int64        `json:"createdAt"`
	UpdatedAt             int64        `json:"updatedAt"`
	User                  *tenant.User `json:"user"`
}

type NewSessionRequest struct {
	TenantInfo            pagination.TenantInfo
	ExpiresAt             int64
	AuthProvider          string
	ExternalIdentityID    string
	ExternalSubject       string
	AuthenticatorAAL      int
	FederationFAL         int
	MFAAuthenticatedAt    int64
	LastReauthenticatedAt int64
	RiskDecision          string
	RiskDecisionID        pulid.ID
}

func NewSession(req *NewSessionRequest) *Session {
	now := timeutils.NowUnix()
	aal := req.AuthenticatorAAL
	if aal == 0 {
		aal = 1
	}
	fal := req.FederationFAL
	if fal == 0 {
		fal = 1
	}
	reauthAt := req.LastReauthenticatedAt
	if reauthAt == 0 {
		reauthAt = now
	}

	return &Session{
		ID:                    pulid.MustNew("ses_"),
		BusinessUnitID:        req.TenantInfo.BuID,
		UserID:                req.TenantInfo.UserID,
		OrganizationID:        req.TenantInfo.OrgID,
		ActiveRoleIDs:         []pulid.ID{},
		AuthProvider:          req.AuthProvider,
		ExternalIdentityID:    req.ExternalIdentityID,
		ExternalSubject:       req.ExternalSubject,
		AuthenticatorAAL:      aal,
		FederationFAL:         fal,
		MFAAuthenticatedAt:    req.MFAAuthenticatedAt,
		LastReauthenticatedAt: reauthAt,
		RiskDecision:          req.RiskDecision,
		RiskDecisionID:        req.RiskDecisionID,
		LastAccessedAt:        now,
		ExpiresAt:             req.ExpiresAt,
		CreatedAt:             now,
		UpdatedAt:             now,
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
