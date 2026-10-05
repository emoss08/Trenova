package supportaccessrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/cloud/cloudcolgen"
	"github.com/emoss08/trenova/internal/cloud/domain/supportaccess"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

const (
	defaultListLimit = 50
	maxListLimit     = 200

	reasonListGrantedOrganizations = "list the organizations whose administrators granted Trenova support access, for platform staff choosing where to help"
	reasonListStaffSessions        = "list one platform staff member's open support sessions, which span the organizations that granted access"
	reasonEndStaffSessions         = "end every open support session of a platform staff member removed from the roster, across organizations"
	reasonStaffRoster              = "maintain the Trenova Cloud platform staff roster, which belongs to no organization"
)

var (
	grantCols     = cloudcolgen.GrantColumns
	sessionCols   = cloudcolgen.SessionColumns
	staffCols     = cloudcolgen.StaffMemberColumns
	principalCols = cloudcolgen.PrincipalColumns
	userCols      = buncolgen.UserColumns
	orgCols       = buncolgen.OrganizationColumns
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Repository struct {
	db *postgres.Connection
}

func New(p Params) *Repository {
	return &Repository{db: p.DB}
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return defaultListLimit
	}
	return min(limit, maxListLimit)
}

func (r *Repository) GetActiveStaffMember(
	ctx context.Context,
	userID pulid.ID,
) (*supportaccess.StaffMember, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*supportaccess.StaffMember, error) {
		member := new(supportaccess.StaffMember)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(member).
			Where(staffCols.UserID.Eq(), userID).
			Where(staffCols.Active.IsTrue()).
			Limit(1).
			Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // not being staff is a valid answer
		}
		if err != nil {
			return nil, fmt.Errorf("get platform staff member: %w", err)
		}

		return member, nil
	})
}

func (r *Repository) UpsertStaffMember(
	ctx context.Context,
	member *supportaccess.StaffMember,
) (*supportaccess.StaffMember, error) {
	ctx = dbscope.WithSystem(ctx, reasonStaffRoster)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*supportaccess.StaffMember, error) {
		_, err := r.db.DBForContext(ctx).NewInsert().
			Model(member).
			On("CONFLICT (user_id) DO UPDATE").
			Set(staffCols.Role.SetExcluded()).
			Set(staffCols.Active.Set(), true).
			Set(staffCols.AddedBy.SetExcluded()).
			Set(staffCols.DeactivatedBy.SetNull()).
			Set(staffCols.DeactivatedAt.SetNull()).
			Set(staffCols.UpdatedAt.SetExcluded()).
			Returning("*").
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("upsert platform staff member: %w", err)
		}

		return member, nil
	})
}

type DeactivateStaffRequest struct {
	UserID        pulid.ID
	DeactivatedBy string
	Now           int64
}

func (r *Repository) DeactivateStaffMember(
	ctx context.Context,
	req DeactivateStaffRequest,
) (bool, error) {
	ctx = dbscope.WithSystem(ctx, reasonStaffRoster)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		result, err := r.db.DBForContext(ctx).NewUpdate().
			Model((*supportaccess.StaffMember)(nil)).
			Set(staffCols.Active.Set(), false).
			Set(staffCols.DeactivatedBy.Set(), req.DeactivatedBy).
			Set(staffCols.DeactivatedAt.Set(), req.Now).
			Set(staffCols.UpdatedAt.Set(), req.Now).
			Where(staffCols.UserID.Eq(), req.UserID).
			Where(staffCols.Active.IsTrue()).
			Exec(ctx)
		if err != nil {
			return false, fmt.Errorf("deactivate platform staff member: %w", err)
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("deactivate platform staff member: %w", err)
		}

		return affected == 1, nil
	})
}

func (r *Repository) ListStaffMembers(
	ctx context.Context,
	includeInactive bool,
) ([]*supportaccess.StaffMember, error) {
	ctx = dbscope.WithSystem(ctx, reasonStaffRoster)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*supportaccess.StaffMember, error) {
		members := make([]*supportaccess.StaffMember, 0)
		query := r.db.DBForContext(ctx).NewSelect().
			Model(&members).
			ColumnExpr(cloudcolgen.StaffMemberTable.All()).
			ColumnExpr(userCols.Name.As("user_name")).
			ColumnExpr(userCols.EmailAddress.As("user_email")).
			Join(
				"JOIN " + buncolgen.UserTable.As(buncolgen.UserTable.Alias) + " ON " +
					userCols.ID.EqColumn(staffCols.UserID),
			).
			Order(staffCols.CreatedAt.OrderAsc())
		if !includeInactive {
			query = query.Where(staffCols.Active.IsTrue())
		}

		if err := query.Scan(ctx); err != nil {
			return nil, fmt.Errorf("list platform staff members: %w", err)
		}

		return members, nil
	})
}

func (r *Repository) GetOpenGrant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	now int64,
) (*supportaccess.Grant, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*supportaccess.Grant, error) {
		grant := new(supportaccess.Grant)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(grant).
			Where(grantCols.OrganizationID.Eq(), tenantInfo.OrgID).
			Where(grantCols.BusinessUnitID.Eq(), tenantInfo.BuID).
			Where(grantCols.RevokedAt.IsNull()).
			Where(grantCols.StartsAt.Lte(), now).
			Where(grantCols.ExpiresAt.Gt(), now).
			Order(grantCols.ExpiresAt.OrderDesc()).
			Limit(1).
			Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // no open grant is a valid answer
		}
		if err != nil {
			return nil, fmt.Errorf("get open support access grant: %w", err)
		}

		return grant, nil
	})
}

func (r *Repository) ListGrants(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	limit int,
) ([]*supportaccess.Grant, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*supportaccess.Grant, error) {
		grants := make([]*supportaccess.Grant, 0)
		if err := r.db.DBForContext(ctx).NewSelect().
			Model(&grants).
			Where(grantCols.OrganizationID.Eq(), tenantInfo.OrgID).
			Where(grantCols.BusinessUnitID.Eq(), tenantInfo.BuID).
			Order(grantCols.CreatedAt.OrderDesc()).
			Limit(clampLimit(limit)).
			Scan(ctx); err != nil {
			return nil, fmt.Errorf("list support access grants: %w", err)
		}

		return grants, nil
	})
}

type ReplaceGrantResult struct {
	Grant           *supportaccess.Grant
	EndedSessionIDs []pulid.ID
}

func (r *Repository) ReplaceGrant(
	ctx context.Context,
	grant *supportaccess.Grant,
) (*ReplaceGrantResult, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ReplaceGrantResult, error) {
		tenantInfo := pagination.TenantInfo{OrgID: grant.OrganizationID, BuID: grant.BusinessUnitID}
		ended, err := r.closeOpenAccess(ctx, &closeAccessRequest{
			tenantInfo: tenantInfo,
			by:         grant.GrantedByID,
			now:        grant.StartsAt,
			reason:     supportaccess.EndReasonReplaced,
		})
		if err != nil {
			return nil, err
		}

		if _, err = r.db.DBForContext(ctx).NewInsert().Model(grant).Exec(ctx); err != nil {
			return nil, fmt.Errorf("insert support access grant: %w", err)
		}

		return &ReplaceGrantResult{Grant: grant, EndedSessionIDs: ended}, nil
	})
}

type RevokeGrantsRequest struct {
	TenantInfo pagination.TenantInfo
	RevokedBy  pulid.ID
	Now        int64
}

func (r *Repository) RevokeGrants(
	ctx context.Context,
	req RevokeGrantsRequest,
) ([]pulid.ID, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		return r.closeOpenAccess(ctx, &closeAccessRequest{
			tenantInfo: req.TenantInfo,
			by:         req.RevokedBy,
			now:        req.Now,
			reason:     supportaccess.EndReasonGrantRevoked,
		})
	})
}

type closeAccessRequest struct {
	tenantInfo pagination.TenantInfo
	by         pulid.ID
	now        int64
	reason     supportaccess.EndReason
}

func (r *Repository) closeOpenAccess(
	ctx context.Context,
	req *closeAccessRequest,
) ([]pulid.ID, error) {
	db := r.db.DBForContext(ctx)

	if _, err := db.NewUpdate().
		Model((*supportaccess.Grant)(nil)).
		Set(grantCols.RevokedAt.Set(), req.now).
		Set(grantCols.RevokedByID.Set(), req.by).
		Set(grantCols.UpdatedAt.Set(), req.now).
		Where(grantCols.OrganizationID.Eq(), req.tenantInfo.OrgID).
		Where(grantCols.BusinessUnitID.Eq(), req.tenantInfo.BuID).
		Where(grantCols.RevokedAt.IsNull()).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("revoke support access grants: %w", err)
	}

	ended := make([]pulid.ID, 0)
	if err := db.NewUpdate().
		Model((*supportaccess.Session)(nil)).
		Set(sessionCols.EndedAt.Set(), req.now).
		Set(sessionCols.EndReason.Set(), req.reason).
		Set(sessionCols.UpdatedAt.Set(), req.now).
		Where(sessionCols.OrganizationID.Eq(), req.tenantInfo.OrgID).
		Where(sessionCols.BusinessUnitID.Eq(), req.tenantInfo.BuID).
		Where(sessionCols.EndedAt.IsNull()).
		Returning(sessionCols.ID.Bare()).
		Scan(ctx, &ended); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("end support sessions: %w", err)
	}

	return ended, nil
}

func (r *Repository) CreateSession(
	ctx context.Context,
	session *supportaccess.Session,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.DBForContext(ctx)

		if _, err := db.NewUpdate().
			Model((*supportaccess.Session)(nil)).
			Set(sessionCols.EndedAt.Set(), session.StartedAt).
			Set(sessionCols.EndReason.Set(), supportaccess.EndReasonReplaced).
			Set(sessionCols.UpdatedAt.Set(), session.StartedAt).
			Where(sessionCols.OrganizationID.Eq(), session.OrganizationID).
			Where(sessionCols.BusinessUnitID.Eq(), session.BusinessUnitID).
			Where(sessionCols.StaffUserID.Eq(), session.StaffUserID).
			Where(sessionCols.EndedAt.IsNull()).
			Exec(ctx); err != nil {
			return fmt.Errorf("end earlier support sessions: %w", err)
		}

		if _, err := db.NewInsert().Model(session).Exec(ctx); err != nil {
			return fmt.Errorf("insert support session: %w", err)
		}

		return nil
	})
}

type SessionKey struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	SessionID      pulid.ID
}

func (r *Repository) GetSession(
	ctx context.Context,
	key SessionKey,
) (*supportaccess.Session, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*supportaccess.Session, error) {
		session := new(supportaccess.Session)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(session).
			ColumnExpr(cloudcolgen.SessionTable.All()).
			ColumnExpr(grantCols.AccessMode.As("grant_access_mode")).
			ColumnExpr(grantCols.ExpiresAt.As("grant_expires_at")).
			ColumnExpr(grantCols.RevokedAt.As("grant_revoked_at")).
			ColumnExpr(grantCols.StartsAt.As("grant_starts_at")).
			ColumnExpr(orgCols.Name.As("organization_name")).
			Join(
				"JOIN "+cloudcolgen.GrantTable.As(cloudcolgen.GrantTable.Alias)+" ON "+
					grantCols.ID.EqColumn(sessionCols.GrantID),
			).
			Join(
				"JOIN "+buncolgen.OrganizationTable.As(buncolgen.OrganizationTable.Alias)+" ON "+
					orgCols.ID.EqColumn(sessionCols.OrganizationID),
			).
			Where(sessionCols.ID.Eq(), key.SessionID).
			Where(sessionCols.OrganizationID.Eq(), key.OrganizationID).
			Where(sessionCols.BusinessUnitID.Eq(), key.BusinessUnitID).
			Limit(1).
			Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errortypes.NewNotFoundError("Support session not found")
		}
		if err != nil {
			return nil, fmt.Errorf("get support session: %w", err)
		}

		return session, nil
	})
}

type EndSessionRequest struct {
	Key    SessionKey
	Reason supportaccess.EndReason
	Now    int64
}

func (r *Repository) EndSession(ctx context.Context, req EndSessionRequest) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		result, err := r.db.DBForContext(ctx).NewUpdate().
			Model((*supportaccess.Session)(nil)).
			Set(sessionCols.EndedAt.Set(), req.Now).
			Set(sessionCols.EndReason.Set(), req.Reason).
			Set(sessionCols.UpdatedAt.Set(), req.Now).
			Where(sessionCols.ID.Eq(), req.Key.SessionID).
			Where(sessionCols.OrganizationID.Eq(), req.Key.OrganizationID).
			Where(sessionCols.BusinessUnitID.Eq(), req.Key.BusinessUnitID).
			Where(sessionCols.EndedAt.IsNull()).
			Exec(ctx)
		if err != nil {
			return false, fmt.Errorf("end support session: %w", err)
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("end support session: %w", err)
		}

		return affected == 1, nil
	})
}

type ElevateSessionRequest struct {
	Key           SessionKey
	ElevatedAt    int64
	ElevatedUntil int64
	Reason        string
	Ticket        string
}

func (r *Repository) ElevateSession(ctx context.Context, req *ElevateSessionRequest) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		result, err := r.db.DBForContext(ctx).NewUpdate().
			Model((*supportaccess.Session)(nil)).
			Set(sessionCols.ElevatedAt.Set(), req.ElevatedAt).
			Set(sessionCols.ElevatedUntil.Set(), req.ElevatedUntil).
			Set(sessionCols.ElevationReason.Set(), req.Reason).
			Set(sessionCols.ElevationTicket.Set(), nullIfEmpty(req.Ticket)).
			Set(sessionCols.ElevationCount.Inc(1)).
			Set(sessionCols.UpdatedAt.Set(), req.ElevatedAt).
			Where(sessionCols.ID.Eq(), req.Key.SessionID).
			Where(sessionCols.OrganizationID.Eq(), req.Key.OrganizationID).
			Where(sessionCols.BusinessUnitID.Eq(), req.Key.BusinessUnitID).
			Where(sessionCols.EndedAt.IsNull()).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("elevate support session: %w", err)
		}

		if affected, _ := result.RowsAffected(); affected == 0 {
			return errortypes.NewBusinessError("The support session has ended")
		}

		return nil
	})
}

func (r *Repository) DropElevation(ctx context.Context, key SessionKey, now int64) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if _, err := r.db.DBForContext(ctx).NewUpdate().
			Model((*supportaccess.Session)(nil)).
			Set(sessionCols.ElevatedUntil.SetNull()).
			Set(sessionCols.UpdatedAt.Set(), now).
			Where(sessionCols.ID.Eq(), key.SessionID).
			Where(sessionCols.OrganizationID.Eq(), key.OrganizationID).
			Where(sessionCols.BusinessUnitID.Eq(), key.BusinessUnitID).
			Exec(ctx); err != nil {
			return fmt.Errorf("drop support session elevation: %w", err)
		}

		return nil
	})
}

func (r *Repository) TouchSession(ctx context.Context, key SessionKey, now int64) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if _, err := r.db.DBForContext(ctx).NewUpdate().
			Model((*supportaccess.Session)(nil)).
			Set(sessionCols.LastSeenAt.Set(), now).
			Where(sessionCols.ID.Eq(), key.SessionID).
			Where(sessionCols.OrganizationID.Eq(), key.OrganizationID).
			Where(sessionCols.BusinessUnitID.Eq(), key.BusinessUnitID).
			Where(sessionCols.EndedAt.IsNull()).
			Exec(ctx); err != nil {
			return fmt.Errorf("touch support session: %w", err)
		}

		return nil
	})
}

func (r *Repository) ListSessions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	limit int,
) ([]*supportaccess.Session, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*supportaccess.Session, error) {
		sessions := make([]*supportaccess.Session, 0)
		if err := r.db.DBForContext(ctx).NewSelect().
			Model(&sessions).
			Where(sessionCols.OrganizationID.Eq(), tenantInfo.OrgID).
			Where(sessionCols.BusinessUnitID.Eq(), tenantInfo.BuID).
			Order(sessionCols.StartedAt.OrderDesc()).
			Limit(clampLimit(limit)).
			Scan(ctx); err != nil {
			return nil, fmt.Errorf("list support sessions: %w", err)
		}

		return sessions, nil
	})
}

func (r *Repository) ListOpenSessionsForStaff(
	ctx context.Context,
	staffUserID pulid.ID,
	now int64,
) ([]*supportaccess.Session, error) {
	ctx = dbscope.WithSystem(ctx, reasonListStaffSessions)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*supportaccess.Session, error) {
		sessions := make([]*supportaccess.Session, 0)
		if err := r.db.DBForContext(ctx).NewSelect().
			Model(&sessions).
			ColumnExpr(cloudcolgen.SessionTable.All()).
			ColumnExpr(orgCols.Name.As("organization_name")).
			Join(
				"JOIN "+buncolgen.OrganizationTable.As(buncolgen.OrganizationTable.Alias)+" ON "+
					orgCols.ID.EqColumn(sessionCols.OrganizationID),
			).
			Where(sessionCols.StaffUserID.Eq(), staffUserID).
			Where(sessionCols.EndedAt.IsNull()).
			Where(sessionCols.ExpiresAt.Gt(), now).
			Order(sessionCols.StartedAt.OrderDesc()).
			Scan(ctx); err != nil {
			return nil, fmt.Errorf("list open support sessions: %w", err)
		}

		return sessions, nil
	})
}

type EndStaffSessionsRequest struct {
	StaffUserID pulid.ID
	Reason      supportaccess.EndReason
	Now         int64
}

func (r *Repository) EndSessionsForStaff(
	ctx context.Context,
	req EndStaffSessionsRequest,
) ([]*supportaccess.Session, error) {
	ctx = dbscope.WithSystem(ctx, reasonEndStaffSessions)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*supportaccess.Session, error) {
		ended := make([]*supportaccess.Session, 0)
		if err := r.db.DBForContext(ctx).NewUpdate().
			Model(&ended).
			Set(sessionCols.EndedAt.Set(), req.Now).
			Set(sessionCols.EndReason.Set(), req.Reason).
			Set(sessionCols.UpdatedAt.Set(), req.Now).
			Where(sessionCols.StaffUserID.Eq(), req.StaffUserID).
			Where(sessionCols.EndedAt.IsNull()).
			Returning("*").
			Scan(ctx); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("end support sessions of staff member: %w", err)
		}

		return ended, nil
	})
}

type GrantedOrganization struct {
	bun.BaseModel `bun:"table:support_access_grants,alias:sag"`

	GrantID          pulid.ID                 `bun:"grant_id"          json:"grantId"`
	OrganizationID   pulid.ID                 `bun:"organization_id"   json:"organizationId"`
	BusinessUnitID   pulid.ID                 `bun:"business_unit_id"  json:"businessUnitId"`
	OrganizationName string                   `bun:"organization_name" json:"organizationName"`
	AccessMode       supportaccess.AccessMode `bun:"access_mode"       json:"accessMode"`
	Note             string                   `bun:"note"              json:"note,omitempty"`
	StartsAt         int64                    `bun:"starts_at"         json:"startsAt"`
	ExpiresAt        int64                    `bun:"expires_at"        json:"expiresAt"`
}

func (r *Repository) ListGrantedOrganizations(
	ctx context.Context,
	now int64,
) ([]*GrantedOrganization, error) {
	ctx = dbscope.WithSystem(ctx, reasonListGrantedOrganizations)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*GrantedOrganization, error) {
		rows := make([]*GrantedOrganization, 0)
		if err := r.db.DBForContext(ctx).NewSelect().
			Model(&rows).
			ColumnExpr(grantCols.ID.As("grant_id")).
			ColumnExpr(grantCols.OrganizationID.Qualified()).
			ColumnExpr(grantCols.BusinessUnitID.Qualified()).
			ColumnExpr(orgCols.Name.As("organization_name")).
			ColumnExpr(grantCols.AccessMode.Qualified()).
			ColumnExpr(grantCols.Note.Qualified()).
			ColumnExpr(grantCols.StartsAt.Qualified()).
			ColumnExpr(grantCols.ExpiresAt.Qualified()).
			Join(
				"JOIN "+buncolgen.OrganizationTable.As(buncolgen.OrganizationTable.Alias)+" ON "+
					orgCols.ID.EqColumn(grantCols.OrganizationID),
			).
			Where(grantCols.RevokedAt.IsNull()).
			Where(grantCols.StartsAt.Lte(), now).
			Where(grantCols.ExpiresAt.Gt(), now).
			Order(orgCols.Name.OrderAsc()).
			Scan(ctx); err != nil {
			return nil, fmt.Errorf("list granted organizations: %w", err)
		}

		return rows, nil
	})
}

type EnsurePrincipalRequest struct {
	TenantInfo  pagination.TenantInfo
	StaffUserID pulid.ID
	NewUser     *tenant.User
}

func (r *Repository) EnsurePrincipal(
	ctx context.Context,
	req *EnsurePrincipalRequest,
) (*tenant.User, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tenant.User, error) {
		db := r.db.DBForContext(ctx)

		existing := new(supportaccess.Principal)
		err := db.NewSelect().
			Model(existing).
			Where(principalCols.OrganizationID.Eq(), req.TenantInfo.OrgID).
			Where(principalCols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
			Where(principalCols.StaffUserID.Eq(), req.StaffUserID).
			Limit(1).
			Scan(ctx)
		switch {
		case err == nil:
			return r.refreshPrincipalUser(ctx, existing.UserID, req.NewUser)
		case !errors.Is(err, sql.ErrNoRows):
			return nil, fmt.Errorf("find support principal: %w", err)
		}

		if _, err = db.NewInsert().Model(req.NewUser).Exec(ctx); err != nil {
			return nil, fmt.Errorf("create support principal user: %w", err)
		}

		principal := &supportaccess.Principal{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			StaffUserID:    req.StaffUserID,
			UserID:         req.NewUser.ID,
		}
		if _, err = db.NewInsert().Model(principal).Exec(ctx); err != nil {
			return nil, fmt.Errorf("record support principal: %w", err)
		}

		return req.NewUser, nil
	})
}

func (r *Repository) refreshPrincipalUser(
	ctx context.Context,
	userID pulid.ID,
	desired *tenant.User,
) (*tenant.User, error) {
	usr := new(tenant.User)
	if err := r.db.DBForContext(ctx).NewUpdate().
		Model(usr).
		Set(userCols.Name.Set(), desired.Name).
		Set(userCols.Timezone.Set(), desired.Timezone).
		Set(userCols.Locale.Set(), desired.Locale).
		Set(userCols.UpdatedAt.Set(), desired.UpdatedAt).
		Where(userCols.ID.Eq(), userID).
		Returning("*").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("refresh support principal user: %w", err)
	}

	return usr, nil
}

func (r *Repository) IsPrincipal(ctx context.Context, userID pulid.ID) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		exists, err := r.db.DBForContext(ctx).NewSelect().
			Model((*supportaccess.Principal)(nil)).
			Where(principalCols.UserID.Eq(), userID).
			Exists(ctx)
		if err != nil {
			return false, fmt.Errorf("check support principal: %w", err)
		}

		return exists, nil
	})
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
