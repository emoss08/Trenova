package supportaccessservice

import (
	"context"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/cloud/domain/supportaccess"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/supportaccessrepository"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type memoryStore struct {
	mu         sync.Mutex
	staff      map[pulid.ID]*supportaccess.StaffMember
	grants     []*supportaccess.Grant
	sessions   []*supportaccess.Session
	principals map[string]*tenant.User
	orgNames   map[pulid.ID]string
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		staff:      map[pulid.ID]*supportaccess.StaffMember{},
		principals: map[string]*tenant.User{},
		orgNames:   map[pulid.ID]string{},
	}
}

func (m *memoryStore) GetActiveStaffMember(
	_ context.Context,
	userID pulid.ID,
) (*supportaccess.StaffMember, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	member, ok := m.staff[userID]
	if !ok || !member.Active {
		return nil, nil //nolint:nilnil // mirrors the repository contract
	}
	return member, nil
}

func (m *memoryStore) UpsertStaffMember(
	_ context.Context,
	member *supportaccess.StaffMember,
) (*supportaccess.StaffMember, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	member.Active = true
	m.staff[member.UserID] = member
	return member, nil
}

func (m *memoryStore) DeactivateStaffMember(
	_ context.Context,
	req supportaccessrepository.DeactivateStaffRequest,
) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	member, ok := m.staff[req.UserID]
	if !ok || !member.Active {
		return false, nil
	}
	member.Active = false
	return true, nil
}

func (m *memoryStore) ListStaffMembers(
	context.Context,
	bool,
) ([]*supportaccess.StaffMember, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	members := make([]*supportaccess.StaffMember, 0, len(m.staff))
	for _, member := range m.staff {
		members = append(members, member)
	}
	return members, nil
}

func (m *memoryStore) GetOpenGrant(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	now int64,
) (*supportaccess.Grant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, grant := range m.grants {
		if grant.OrganizationID == tenantInfo.OrgID && grant.BusinessUnitID == tenantInfo.BuID &&
			grant.IsActive(now) {
			copied := *grant
			return &copied, nil
		}
	}
	return nil, nil //nolint:nilnil // mirrors the repository contract
}

func (m *memoryStore) ListGrants(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	_ int,
) ([]*supportaccess.Grant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*supportaccess.Grant, 0)
	for _, grant := range m.grants {
		if grant.OrganizationID == tenantInfo.OrgID {
			out = append(out, grant)
		}
	}
	return out, nil
}

func (m *memoryStore) closeOpen(
	tenantInfo pagination.TenantInfo,
	by pulid.ID,
	now int64,
	reason supportaccess.EndReason,
) []pulid.ID {
	for _, grant := range m.grants {
		if grant.OrganizationID == tenantInfo.OrgID && grant.RevokedAt == nil {
			revokedAt := now
			grant.RevokedAt = &revokedAt
			grant.RevokedByID = by
		}
	}
	ended := make([]pulid.ID, 0)
	for _, session := range m.sessions {
		if session.OrganizationID == tenantInfo.OrgID && session.EndedAt == nil {
			endedAt := now
			session.EndedAt = &endedAt
			session.EndReason = reason
			ended = append(ended, session.ID)
		}
	}
	return ended
}

func (m *memoryStore) ReplaceGrant(
	_ context.Context,
	grant *supportaccess.Grant,
) (*supportaccessrepository.ReplaceGrantResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ended := m.closeOpen(
		pagination.TenantInfo{OrgID: grant.OrganizationID, BuID: grant.BusinessUnitID},
		grant.GrantedByID,
		grant.StartsAt,
		supportaccess.EndReasonReplaced,
	)
	grant.ID = pulid.MustNew("sag_")
	m.grants = append(m.grants, grant)
	return &supportaccessrepository.ReplaceGrantResult{Grant: grant, EndedSessionIDs: ended}, nil
}

func (m *memoryStore) RevokeGrants(
	_ context.Context,
	req supportaccessrepository.RevokeGrantsRequest,
) ([]pulid.ID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closeOpen(req.TenantInfo, req.RevokedBy, req.Now, supportaccess.EndReasonGrantRevoked), nil
}

func (m *memoryStore) CreateSession(_ context.Context, session *supportaccess.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions = append(m.sessions, session)
	return nil
}

func (m *memoryStore) GetSession(
	_ context.Context,
	key supportaccessrepository.SessionKey,
) (*supportaccess.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, session := range m.sessions {
		if session.ID != key.SessionID || session.OrganizationID != key.OrganizationID ||
			session.BusinessUnitID != key.BusinessUnitID {
			continue
		}
		copied := *session
		for _, grant := range m.grants {
			if grant.ID == session.GrantID {
				copied.GrantAccessMode = grant.AccessMode.String()
				copied.GrantExpiresAt = grant.ExpiresAt
				copied.GrantRevokedAt = grant.RevokedAt
				copied.GrantStartsAt = grant.StartsAt
			}
		}
		copied.OrganizationName = m.orgNames[session.OrganizationID]
		return &copied, nil
	}
	return nil, errortypes.NewNotFoundError("Support session not found")
}

func (m *memoryStore) find(id pulid.ID) *supportaccess.Session {
	for _, session := range m.sessions {
		if session.ID == id {
			return session
		}
	}
	return nil
}

func (m *memoryStore) EndSession(
	_ context.Context,
	req supportaccessrepository.EndSessionRequest,
) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session := m.find(req.Key.SessionID)
	if session == nil || session.EndedAt != nil {
		return false, nil
	}
	endedAt := req.Now
	session.EndedAt = &endedAt
	session.EndReason = req.Reason
	return true, nil
}

func (m *memoryStore) ElevateSession(
	_ context.Context,
	req *supportaccessrepository.ElevateSessionRequest,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	session := m.find(req.Key.SessionID)
	if session == nil || session.EndedAt != nil {
		return errortypes.NewBusinessError("ended")
	}
	at, until := req.ElevatedAt, req.ElevatedUntil
	session.ElevatedAt = &at
	session.ElevatedUntil = &until
	session.ElevationReason = req.Reason
	session.ElevationTicket = req.Ticket
	session.ElevationCount++
	return nil
}

func (m *memoryStore) DropElevation(
	_ context.Context,
	key supportaccessrepository.SessionKey,
	_ int64,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if session := m.find(key.SessionID); session != nil {
		session.ElevatedUntil = nil
	}
	return nil
}

func (m *memoryStore) TouchSession(
	_ context.Context,
	key supportaccessrepository.SessionKey,
	now int64,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if session := m.find(key.SessionID); session != nil {
		session.LastSeenAt = now
	}
	return nil
}

func (m *memoryStore) ListSessions(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	_ int,
) ([]*supportaccess.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*supportaccess.Session, 0)
	for _, session := range m.sessions {
		if session.OrganizationID == tenantInfo.OrgID {
			out = append(out, session)
		}
	}
	return out, nil
}

func (m *memoryStore) ListOpenSessionsForStaff(
	_ context.Context,
	staffUserID pulid.ID,
	now int64,
) ([]*supportaccess.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*supportaccess.Session, 0)
	for _, session := range m.sessions {
		if session.StaffUserID == staffUserID && session.IsOpen(now) {
			out = append(out, session)
		}
	}
	return out, nil
}

func (m *memoryStore) EndSessionsForStaff(
	_ context.Context,
	req supportaccessrepository.EndStaffSessionsRequest,
) ([]*supportaccess.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*supportaccess.Session, 0)
	for _, session := range m.sessions {
		if session.StaffUserID == req.StaffUserID && session.EndedAt == nil {
			endedAt := req.Now
			session.EndedAt = &endedAt
			session.EndReason = req.Reason
			out = append(out, session)
		}
	}
	return out, nil
}

func (m *memoryStore) ListGrantedOrganizations(
	_ context.Context,
	now int64,
) ([]*supportaccessrepository.GrantedOrganization, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*supportaccessrepository.GrantedOrganization, 0)
	for _, grant := range m.grants {
		if grant.IsActive(now) {
			out = append(out, &supportaccessrepository.GrantedOrganization{
				GrantID:          grant.ID,
				OrganizationID:   grant.OrganizationID,
				BusinessUnitID:   grant.BusinessUnitID,
				OrganizationName: m.orgNames[grant.OrganizationID],
				AccessMode:       grant.AccessMode,
				ExpiresAt:        grant.ExpiresAt,
			})
		}
	}
	return out, nil
}

func (m *memoryStore) EnsurePrincipal(
	_ context.Context,
	req *supportaccessrepository.EnsurePrincipalRequest,
) (*tenant.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := req.TenantInfo.OrgID.String() + "/" + req.StaffUserID.String()
	if existing, ok := m.principals[key]; ok {
		existing.Name = req.NewUser.Name
		return existing, nil
	}
	m.principals[key] = req.NewUser
	return req.NewUser, nil
}

type fakeMFA struct {
	services.MFAService
	enrolled  bool
	password  string
	validCode string
	calls     int
}

func (f *fakeMFA) HasActiveFactor(context.Context, pulid.ID) (bool, error) {
	return f.enrolled, nil
}

func (f *fakeMFA) Reauthenticate(
	_ context.Context,
	req *services.ReauthenticateRequest,
) (string, error) {
	f.calls++
	if req.Password != f.password {
		return "", errortypes.NewValidationError("password", errortypes.ErrInvalid, "bad password")
	}
	if req.Code != f.validCode {
		return "", errortypes.NewValidationError("code", errortypes.ErrInvalid, "bad code")
	}
	return services.MFAMethodTOTP, nil
}

type recordingAuditor struct {
	mu      sync.Mutex
	changes []*services.SecurityChange
}

func (a *recordingAuditor) RecordChange(_ context.Context, change *services.SecurityChange) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.changes = append(a.changes, change)
}

func (a *recordingAuditor) comments() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]string, 0, len(a.changes))
	for _, change := range a.changes {
		out = append(out, change.Comment)
	}
	return out
}

type fakeLimiter struct {
	remaining int
}

func (l *fakeLimiter) Allow(context.Context, pulid.ID, int) (bool, time.Duration, error) {
	if l.remaining <= 0 {
		return false, time.Minute, nil
	}
	l.remaining--
	return true, 0, nil
}
