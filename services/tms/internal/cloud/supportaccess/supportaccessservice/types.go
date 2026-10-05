package supportaccessservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/cloud/domain/supportaccess"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/supportaccessrepository"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const mfaAssuranceLevel = 2

type StaffContext struct {
	UserID             pulid.ID
	OrganizationID     pulid.ID
	BusinessUnitID     pulid.ID
	SessionID          pulid.ID
	AuthenticatorAAL   int
	MFAAuthenticatedAt int64
}

func (s StaffContext) MFAVerified() bool {
	return s.AuthenticatorAAL >= mfaAssuranceLevel && s.MFAAuthenticatedAt > 0
}

func (s StaffContext) scope(ctx context.Context) context.Context {
	return dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: s.OrganizationID,
		BusinessUnitID: s.BusinessUnitID,
		UserID:         s.UserID,
	})
}

func tenantScope(ctx context.Context, orgID, buID, userID pulid.ID) context.Context {
	return dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		UserID:         userID,
	})
}

type SessionEndedError struct {
	Reason supportaccess.EndReason
}

func (e *SessionEndedError) Error() string {
	return fmt.Sprintf("support session has ended (%s)", e.Reason)
}

type SessionView struct {
	ID               pulid.ID                 `json:"id"`
	OrganizationID   pulid.ID                 `json:"organizationId"`
	BusinessUnitID   pulid.ID                 `json:"businessUnitId"`
	OrganizationName string                   `json:"organizationName"`
	StaffName        string                   `json:"staffName"`
	Mode             supportaccess.AccessMode `json:"mode"`
	GrantMode        supportaccess.AccessMode `json:"grantMode"`
	Reason           string                   `json:"reason"`
	TicketReference  string                   `json:"ticketReference,omitempty"`
	StartedAt        int64                    `json:"startedAt"`
	ExpiresAt        int64                    `json:"expiresAt"`
	ElevatedUntil    int64                    `json:"elevatedUntil,omitempty"`
	CanElevate       bool                     `json:"canElevate"`
	ServerTime       int64                    `json:"serverTime"`
}

type CurrentSessionResponse struct {
	Active  bool         `json:"active"`
	Session *SessionView `json:"session,omitempty"`
	Ended   string       `json:"endedReason,omitempty"`
}

type StaffProfile struct {
	IsStaff          bool           `json:"isStaff"`
	Role             string         `json:"role,omitempty"`
	MFAEnrolled      bool           `json:"mfaEnrolled"`
	SessionVerified  bool           `json:"sessionVerified"`
	OpenSessions     []*SessionView `json:"openSessions"`
	MaxSessionHours  float64        `json:"maxSessionHours"`
	ElevationMinutes float64        `json:"elevationMinutes"`
}

type StartSessionRequest struct {
	Staff           StaffContext `json:"-"`
	OrganizationID  pulid.ID     `json:"organizationId"`
	BusinessUnitID  pulid.ID     `json:"businessUnitId"`
	Reason          string       `json:"reason"`
	TicketReference string       `json:"ticketReference"`
	ClientIP        string       `json:"-"`
	UserAgent       string       `json:"-"`
}

type StartedSession struct {
	Session *SessionView
	Token   string
}

type ElevateRequest struct {
	Password        string `json:"password"`
	Code            string `json:"code"`
	RecoveryCode    string `json:"recoveryCode"`
	Reason          string `json:"reason"`
	TicketReference string `json:"ticketReference"`
}

type GrantState struct {
	Grant            *supportaccess.Grant       `json:"grant"`
	Sessions         []*CustomerSessionView     `json:"sessions"`
	History          []*supportaccess.Grant     `json:"history"`
	DurationHours    []int                      `json:"durationHours"`
	MaxDurationHours int                        `json:"maxDurationHours"`
	AccessModes      []supportaccess.AccessMode `json:"accessModes"`
	ServerTime       int64                      `json:"serverTime"`
}

type CustomerSessionView struct {
	ID              pulid.ID                 `json:"id"`
	StaffName       string                   `json:"staffName"`
	Status          string                   `json:"status"`
	Mode            supportaccess.AccessMode `json:"mode"`
	Reason          string                   `json:"reason"`
	TicketReference string                   `json:"ticketReference,omitempty"`
	StartedAt       int64                    `json:"startedAt"`
	ExpiresAt       int64                    `json:"expiresAt"`
	LastSeenAt      int64                    `json:"lastSeenAt"`
	ElevationCount  int                      `json:"elevationCount"`
	ElevatedUntil   int64                    `json:"elevatedUntil,omitempty"`
	ElevationReason string                   `json:"elevationReason,omitempty"`
	EndedAt         int64                    `json:"endedAt,omitempty"`
	EndReason       string                   `json:"endReason,omitempty"`
}

type CreateGrantRequest struct {
	TenantInfo    pagination.TenantInfo    `json:"-"`
	DurationHours int                      `json:"durationHours"`
	AccessMode    supportaccess.AccessMode `json:"accessMode"`
	Note          string                   `json:"note"`
}

type AddStaffRequest struct {
	EmailAddress string
	Role         supportaccess.StaffRole
	AddedBy      string
}

type RemoveStaffRequest struct {
	EmailAddress string
	RemovedBy    string
}

type RemoveStaffResult struct {
	Removed       bool
	EndedSessions int
}

type store interface {
	GetActiveStaffMember(ctx context.Context, userID pulid.ID) (*supportaccess.StaffMember, error)
	GetOpenGrant(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		now int64,
	) (*supportaccess.Grant, error)
	ListGrants(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		limit int,
	) ([]*supportaccess.Grant, error)
	ReplaceGrant(
		ctx context.Context,
		grant *supportaccess.Grant,
	) (*supportaccessrepository.ReplaceGrantResult, error)
	RevokeGrants(
		ctx context.Context,
		req supportaccessrepository.RevokeGrantsRequest,
	) ([]pulid.ID, error)
	CreateSession(ctx context.Context, session *supportaccess.Session) error
	GetSession(
		ctx context.Context,
		key supportaccessrepository.SessionKey,
	) (*supportaccess.Session, error)
	EndSession(ctx context.Context, req supportaccessrepository.EndSessionRequest) (bool, error)
	ElevateSession(ctx context.Context, req *supportaccessrepository.ElevateSessionRequest) error
	DropElevation(ctx context.Context, key supportaccessrepository.SessionKey, now int64) error
	TouchSession(ctx context.Context, key supportaccessrepository.SessionKey, now int64) error
	ListSessions(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		limit int,
	) ([]*supportaccess.Session, error)
	ListOpenSessionsForStaff(
		ctx context.Context,
		staffUserID pulid.ID,
		now int64,
	) ([]*supportaccess.Session, error)
	EndSessionsForStaff(
		ctx context.Context,
		req supportaccessrepository.EndStaffSessionsRequest,
	) ([]*supportaccess.Session, error)
	ListGrantedOrganizations(
		ctx context.Context,
		now int64,
	) ([]*supportaccessrepository.GrantedOrganization, error)
	EnsurePrincipal(
		ctx context.Context,
		req *supportaccessrepository.EnsurePrincipalRequest,
	) (*tenant.User, error)
}
