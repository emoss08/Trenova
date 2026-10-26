package userrepository

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	membershipScopeReason     = "read the memberships row-level security shows a person: their own in every organization and everyone's in the organization they are working in"
	emailScopeReason          = "resolve a sign-in email before any tenant is known"
	loginScopeReason          = "load the user an identity provider subject is linked to before a session exists"
	systemUserScopeReason     = "resolve the instance system user, which belongs to the default organization, as the actor for any tenant"
	listMembershipsReason     = "list a user's memberships across the organizations of one business unit"
	replaceMembershipsReason  = "replace a user's memberships across the organizations of one business unit in one transaction"
	membershipLockTimeout     = 250 * time.Millisecond
	membershipsBusyMessage    = "The user's organization memberships are busy. Retry the request."
	organizationsOutsideScope = "organizations must belong to the current business unit"
)

var userErrors = dberror.Mapping{Entity: "User"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
	Quota  platformplanports.QuotaGuard `optional:"true"`
}

type repository struct {
	db    *postgres.Connection
	l     *zap.Logger
	quota platformplanports.QuotaGuard
}

func New(p Params) tenantports.UserRepository {
	return &repository{
		db:    p.DB,
		l:     p.Logger.Named("postgres.user-repository"),
		quota: quotaservice.OrUnlimited(p.Quota),
	}
}

func tenantUsers(orgID pulid.ID) *keel.Query[*tenant.User] {
	u := tenantdb.Users
	return u.Where(
		u.CurrentOrganizationID.Eq(orgID),
		u.Username.Ne(tenant.SystemUsername),
	)
}

func keepVersion() keel.Assignment {
	u := tenantdb.Users
	return u.Version.ToExpr(u.Version)
}

func allTenantsWhenSystem[T any](ctx context.Context, q *keel.Query[*T]) *keel.Query[*T] {
	if dbscope.IsSystem(ctx) {
		return q.AllTenants()
	}

	return q
}

func membershipReadScope(ctx context.Context) (context.Context, keel.Pred) {
	current, ok := dbscope.TenantFrom(ctx)
	if !ok {
		return ctx, nil
	}

	m := tenantdb.OrganizationMemberships
	visible := keel.Or(
		keel.And(
			m.OrganizationID.Eq(current.OrganizationID),
			m.BusinessUnitID.Eq(current.BusinessUnitID),
		),
		keel.When(current.UserID.IsNotNil(), m.UserID.Eq(current.UserID)),
	)

	return dbscope.WithSystem(ctx, membershipScopeReason), visible
}

func activeMembership(now int64) keel.Pred {
	m := tenantdb.OrganizationMemberships
	return keel.Or(m.ExpiresAt.IsNull(), m.ExpiresAt.Gt(now))
}

func (ur *repository) attachMemberships(
	ctx context.Context,
	users []*tenant.User,
	withState bool,
) error {
	if len(users) == 0 {
		return nil
	}

	userIDs := make([]pulid.ID, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}

	readCtx, visible := membershipReadScope(ctx)
	memberships, err := dbtx.Read(
		readCtx,
		ur.db,
		func(ctx context.Context) ([]*tenant.OrganizationMembership, error) {
			m := tenantdb.OrganizationMemberships
			org := m.Organization()
			load := keel.Loader(org)
			if withState {
				load = org.State()
			}

			return allTenantsWhenSystem(ctx, m.Where(m.UserID.In(userIDs...), visible)).
				Load(load).
				All(ctx, ur.db.Keel(ctx))
		},
	)
	if err != nil {
		return err
	}

	byUser := make(map[pulid.ID][]*tenant.OrganizationMembership, len(users))
	for _, membership := range memberships {
		byUser[membership.UserID] = append(byUser[membership.UserID], membership)
	}
	for _, user := range users {
		user.Memberships = byUser[user.ID]
	}

	return nil
}

func (ur *repository) List(
	ctx context.Context,
	req *tenantports.ListUsersRequest,
) (*pagination.ListResult[*tenant.User], error) {
	log := ur.l.With(zap.String("operation", "List"), zap.Any("request", req))

	result, err := dbtx.Read(
		ctx,
		ur.db,
		func(ctx context.Context) (*pagination.ListResult[*tenant.User], error) {
			u := tenantdb.Users
			return listquery.List(
				ctx,
				ur.db.Keel(ctx),
				tenantUsers(req.Filter.TenantInfo.OrgID),
				&listquery.Request[tenant.User]{
					Filter: req.Filter,
					Shape: func(q *keel.Query[*tenant.User]) *keel.Query[*tenant.User] {
						return q.Load(u.Assignments().Role())
					},
				},
			)
		},
	)
	if err != nil {
		log.Error("failed to list users", zap.Error(err))
		return nil, err
	}

	if req.IncludeMemberships {
		if err = ur.attachMemberships(ctx, result.Items, false); err != nil {
			log.Error("failed to load user memberships", zap.Error(err))
			return nil, err
		}
	}

	return result, nil
}

func (ur *repository) ListConnection(
	ctx context.Context,
	req *tenantports.ListUserConnectionRequest,
) (*pagination.CursorListResult[*tenant.User], error) {
	return dbtx.Read(
		ctx,
		ur.db,
		func(ctx context.Context) (*pagination.CursorListResult[*tenant.User], error) {
			result, err := listquery.Connection(
				ctx,
				ur.db.Keel(ctx),
				tenantUsers(req.Filter.TenantInfo.OrgID),
				&listquery.Request[tenant.User]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				ur.l.Error("failed to list users", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (ur *repository) GetByID(
	ctx context.Context,
	req tenantports.GetUserByIDRequest,
) (*tenant.User, error) {
	targetUserID := req.LookupUserID
	if targetUserID.IsNil() {
		targetUserID = req.TenantInfo.UserID
	}

	entity, err := dbtx.Read(ctx, ur.db, func(ctx context.Context) (*tenant.User, error) {
		u := tenantdb.Users
		user, err := u.Where(
			u.CurrentOrganizationID.Eq(req.TenantInfo.OrgID),
			u.ID.Eq(targetUserID),
		).One(ctx, ur.db.Keel(ctx))
		if err != nil {
			return nil, userErrors.Map(err, targetUserID.String())
		}

		return user, nil
	})
	if err != nil {
		return nil, err
	}

	if req.IncludeMemberships {
		if err = ur.attachMemberships(ctx, []*tenant.User{entity}, true); err != nil {
			return nil, err
		}
	}

	return entity, nil
}

func (ur *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*tenant.User], error) {
	return dbtx.Read(
		ctx,
		ur.db,
		func(ctx context.Context) (*pagination.ListResult[*tenant.User], error) {
			u := tenantdb.Users
			q := tenantUsers(req.TenantInfo.OrgID).
				Pick(
					u.ID,
					u.Name,
					u.Username,
					u.EmailAddress,
					u.Status,
					u.IsLocked,
					u.ProfilePicURL,
					u.ThumbnailURL,
				).
				Where(u.Status.Eq(domaintypes.StatusActive))

			return listquery.SelectOptions(
				ctx,
				ur.db.Keel(ctx),
				q,
				req,
				func(search string) keel.Pred {
					return keel.Or(
						u.Username.ContainsFold(search),
						u.Name.ContainsFold(search),
						u.EmailAddress.ContainsFold(search),
					)
				},
			)
		},
	)
}

func (ur *repository) FindByEmail(ctx context.Context, emailAddress string) (*tenant.User, error) {
	ctx = dbscope.WithSystem(ctx, emailScopeReason)
	return dbtx.Read(ctx, ur.db, func(ctx context.Context) (*tenant.User, error) {
		u := tenantdb.Users
		user, err := u.AllTenants().
			Where(u.EmailAddress.Lower().Eq(strings.ToLower(emailAddress))).
			One(ctx, ur.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, errortypes.NewValidationError(
					"emailAddress",
					errortypes.ErrNotFound,
					"User not found with the given email address",
				)
			}

			return nil, err
		}

		return user, nil
	})
}

func (ur *repository) FindByIDForLogin(ctx context.Context, userID pulid.ID) (*tenant.User, error) {
	ctx = dbscope.WithSystem(ctx, loginScopeReason)
	return dbtx.Read(ctx, ur.db, func(ctx context.Context) (*tenant.User, error) {
		u := tenantdb.Users
		user, err := u.AllTenants().Where(u.ID.Eq(userID)).One(ctx, ur.db.Keel(ctx))
		if err != nil {
			return nil, userErrors.Map(err, userID.String())
		}

		return user, nil
	})
}

func (ur *repository) UpdateLastLoginAt(ctx context.Context, userID pulid.ID) error {
	return dbtx.WriteErr(ctx, ur.db, func(ctx context.Context) error {
		u := tenantdb.Users
		if _, err := u.Where(u.ID.Eq(userID)).Update(
			ctx,
			ur.db.Keel(ctx),
			u.LastLoginAt.To(timeutils.NowUnix()),
			u.UpdatedAt.ToExpr(u.UpdatedAt),
			keepVersion(),
		); err != nil {
			ur.l.Error(
				"failed to update last login at in database",
				zap.String("userID", userID.String()),
				zap.Error(err),
			)
			return err
		}

		return nil
	})
}

func (ur *repository) GetOrganizations(
	ctx context.Context,
	userID pulid.ID,
) ([]*tenant.OrganizationMembership, error) {
	readCtx, visible := membershipReadScope(ctx)
	return dbtx.Read(
		readCtx,
		ur.db,
		func(ctx context.Context) ([]*tenant.OrganizationMembership, error) {
			m := tenantdb.OrganizationMemberships
			entities, err := allTenantsWhenSystem(ctx, m.Where(
				m.UserID.Eq(userID),
				activeMembership(timeutils.NowUnix()),
				visible,
			)).
				Load(m.Organization().State(), m.User()).
				Order(m.IsDefault.Desc()).
				All(ctx, ur.db.Keel(ctx))
			if err != nil {
				ur.l.Error(
					"failed to get user organizations",
					zap.String("userID", userID.String()),
					zap.Error(err),
				)
				return nil, err
			}

			return entities, nil
		},
	)
}

func (ur *repository) ListOrganizationMemberships(
	ctx context.Context,
	userID, businessUnitID pulid.ID,
) ([]*tenant.OrganizationMembership, error) {
	ctx = dbscope.WithSystem(ctx, listMembershipsReason)
	return dbtx.Read(
		ctx,
		ur.db,
		func(ctx context.Context) ([]*tenant.OrganizationMembership, error) {
			m := tenantdb.OrganizationMemberships
			return m.AllTenants().
				Where(m.UserID.Eq(userID), m.BusinessUnitID.Eq(businessUnitID)).
				Load(m.Organization().State()).
				Order(m.IsDefault.Desc(), m.JoinedAt.Asc()).
				All(ctx, ur.db.Keel(ctx))
		},
	)
}

func (ur *repository) ReplaceOrganizationMemberships(
	ctx context.Context,
	req tenantports.ReplaceOrganizationMembershipsRequest,
) ([]*tenant.OrganizationMembership, error) {
	if len(req.OrganizationIDs) == 0 {
		return nil, tenant.NewNoOrganizationMembershipsError()
	}

	ctx = dbscope.WithSystem(ctx, replaceMembershipsReason)
	return dbtx.Write(
		ctx,
		ur.db,
		func(ctx context.Context) ([]*tenant.OrganizationMembership, error) {
			if err := ur.checkMembershipTargets(ctx, req); err != nil {
				return nil, err
			}

			err := ur.db.Transact(
				ctx,
				kernelports.TxOptions{LockTimeout: membershipLockTimeout},
				func(ctx context.Context) error {
					return ur.replaceMemberships(ctx, req)
				},
			)
			if err != nil {
				return nil, dberror.MapRetryableTransactionError(err, membershipsBusyMessage)
			}

			return ur.ListOrganizationMemberships(ctx, req.UserID, req.BusinessUnitID)
		},
	)
}

func (ur *repository) checkMembershipTargets(
	ctx context.Context,
	req tenantports.ReplaceOrganizationMembershipsRequest,
) error {
	db := ur.db.Keel(ctx)
	u := tenantdb.Users
	o := tenantdb.Organizations

	var inBusinessUnit bool
	var orgCount int64
	err := keel.Batch(
		ctx,
		db,
		keel.ExistsInto(
			u.AllTenants().Where(u.ID.Eq(req.UserID), u.BusinessUnitID.Eq(req.BusinessUnitID)),
			&inBusinessUnit,
		),
		keel.CountInto(
			o.AllTenants().Where(
				o.ID.In(req.OrganizationIDs...),
				o.BusinessUnitID.Eq(req.BusinessUnitID),
			),
			&orgCount,
		),
	)
	if err != nil {
		return err
	}
	if !inBusinessUnit {
		return errortypes.NewNotFoundError("User not found")
	}
	if int(orgCount) != len(req.OrganizationIDs) {
		return errortypes.NewAuthorizationError(organizationsOutsideScope)
	}

	return nil
}

func (ur *repository) replaceMemberships(
	ctx context.Context,
	req tenantports.ReplaceOrganizationMembershipsRequest,
) error {
	db := ur.db.Keel(ctx)
	m := tenantdb.OrganizationMemberships

	locked, err := m.AllTenants().
		Where(m.UserID.Eq(req.UserID), m.BusinessUnitID.Eq(req.BusinessUnitID)).
		Order(m.OrganizationID.Asc()).
		Lock(keel.ForUpdate, keel.NoWait).
		All(ctx, db)
	if err != nil {
		return err
	}

	desired := make(map[pulid.ID]struct{}, len(req.OrganizationIDs))
	for _, orgID := range req.OrganizationIDs {
		desired[orgID] = struct{}{}
	}

	current := make(map[pulid.ID]struct{}, len(locked))
	var currentDefaultOrgID pulid.ID
	removed := make([]pulid.ID, 0, len(locked))
	for _, membership := range locked {
		current[membership.OrganizationID] = struct{}{}
		if membership.IsDefault {
			currentDefaultOrgID = membership.OrganizationID
		}
		if _, keep := desired[membership.OrganizationID]; !keep {
			removed = append(removed, membership.ID)
		}
	}

	if len(removed) > 0 {
		if _, err = m.AllTenants().Where(m.ID.In(removed...)).Delete(ctx, db); err != nil {
			return err
		}
	}

	added, err := ur.newMemberships(ctx, req, current)
	if err != nil {
		return err
	}
	if err = m.CreateMany(ctx, db, added); err != nil {
		return err
	}

	newDefaultOrgID := currentDefaultOrgID
	if _, ok := desired[newDefaultOrgID]; !ok {
		newDefaultOrgID = req.OrganizationIDs[0]
	}

	isDefault := m.OrganizationID.Eq(newDefaultOrgID)
	if _, err = m.AllTenants().
		Where(m.UserID.Eq(req.UserID), m.BusinessUnitID.Eq(req.BusinessUnitID)).
		Update(ctx, db, m.IsDefault.ToExpr(isDefault)); err != nil {
		return err
	}

	u := tenantdb.Users
	_, err = u.AllTenants().
		Where(
			u.ID.Eq(req.UserID),
			u.BusinessUnitID.Eq(req.BusinessUnitID),
			u.CurrentOrganizationID.NotIn(req.OrganizationIDs...),
		).
		Update(ctx, db, u.CurrentOrganizationID.To(newDefaultOrgID), keepVersion())

	return err
}

func (ur *repository) newMemberships(
	ctx context.Context,
	req tenantports.ReplaceOrganizationMembershipsRequest,
	current map[pulid.ID]struct{},
) ([]*tenant.OrganizationMembership, error) {
	added := make([]*tenant.OrganizationMembership, 0, len(req.OrganizationIDs))
	for _, orgID := range req.OrganizationIDs {
		if _, ok := current[orgID]; ok {
			continue
		}
		if err := quotaservice.EnforceAll(ctx, ur.quota, platformplanports.QuotaRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: req.BusinessUnitID},
			Meter:      platformcatalog.MeterUserSeats,
			Quantity:   1,
		}); err != nil {
			return nil, err
		}
		added = append(added, &tenant.OrganizationMembership{
			UserID:         req.UserID,
			BusinessUnitID: req.BusinessUnitID,
			OrganizationID: orgID,
			GrantedByID:    req.ActorID,
		})
	}

	return added, nil
}

func (ur *repository) UpdateCurrentOrganization(
	ctx context.Context,
	userID, orgID, buID pulid.ID,
) error {
	return dbtx.WriteErr(ctx, ur.db, func(ctx context.Context) error {
		u := tenantdb.Users
		if _, err := u.Where(u.ID.Eq(userID)).Update(
			ctx,
			ur.db.Keel(ctx),
			u.CurrentOrganizationID.To(orgID),
			u.BusinessUnitID.To(buID),
			keepVersion(),
		); err != nil {
			ur.l.Error(
				"failed to update current organization",
				zap.String("userID", userID.String()),
				zap.String("orgID", orgID.String()),
				zap.Error(err),
			)
			return err
		}

		return nil
	})
}

func (ur *repository) GetUserOrganizationSummaries(
	ctx context.Context,
	userID pulid.ID,
) ([]tenantports.OrgSummary, error) {
	readCtx, visible := membershipReadScope(ctx)
	return dbtx.Read(readCtx, ur.db, func(ctx context.Context) ([]tenantports.OrgSummary, error) {
		m := tenantdb.OrganizationMemberships
		org := m.Organization()
		summaries, err := keel.Into[tenantports.OrgSummary](
			ctx,
			ur.db.Keel(ctx),
			allTenantsWhenSystem(ctx, m.Where(
				m.UserID.Eq(userID),
				org.ID.IsNotNull(),
				activeMembership(timeutils.NowUnix()),
				visible,
			)).
				Pick(org.ID, org.Name).
				Order(org.Name.Asc()),
		)
		if err != nil {
			ur.l.Error(
				"failed to get user organization summaries",
				zap.String("userID", userID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		return summaries, nil
	})
}

func (ur *repository) UpdatePassword(
	ctx context.Context,
	req tenantports.UpdateUserPasswordRequest,
) error {
	return dbtx.WriteErr(ctx, ur.db, func(ctx context.Context) error {
		u := tenantdb.Users
		_, err := u.Where(
			u.ID.Eq(req.UserID),
			u.CurrentOrganizationID.Eq(req.OrganizationID),
		).Update(
			ctx,
			ur.db.Keel(ctx),
			u.Password.To(req.Password),
			u.MustChangePassword.To(req.MustChangePassword),
			u.PasswordChangedAt.To(timeutils.NowUnix()),
		)
		if err != nil {
			ur.l.Error(
				"failed to update user password",
				zap.String("userID", req.UserID.String()),
				zap.Error(err),
			)
			return err
		}

		return nil
	})
}

func (ur *repository) GetByIDs(
	ctx context.Context,
	req tenantports.GetUsersByIDsRequest,
) ([]*tenant.User, error) {
	return dbtx.Read(ctx, ur.db, func(ctx context.Context) ([]*tenant.User, error) {
		if len(req.UserIDs) == 0 {
			return []*tenant.User{}, nil
		}

		u := tenantdb.Users
		entities, err := u.Where(
			u.CurrentOrganizationID.Eq(req.TenantInfo.OrgID),
			u.ID.In(req.UserIDs...),
		).Limit(len(req.UserIDs)).List(ctx, ur.db.Keel(ctx))
		if err != nil {
			ur.l.Error("failed to get users", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (ur *repository) BulkUpdateStatus(
	ctx context.Context,
	req *tenantports.BulkUpdateUserStatusRequest,
) ([]*tenant.User, error) {
	return dbtx.Write(ctx, ur.db, func(ctx context.Context) ([]*tenant.User, error) {
		u := tenantdb.Users
		entities, err := u.Where(
			u.CurrentOrganizationID.Eq(req.TenantInfo.OrgID),
			u.ID.In(req.UserIDs...),
		).UpdateReturning(ctx, ur.db.Keel(ctx), u.Status.To(req.Status))
		if err != nil {
			ur.l.Error("failed to bulk update user status", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckBulkAffected(int64(len(entities)), "User", req.UserIDs); err != nil {
			return nil, err
		}

		return entities, nil
	})
}

func (ur *repository) GetSystemUser(ctx context.Context, columns ...string) (*tenant.User, error) {
	ctx = dbscope.WithSystem(ctx, systemUserScopeReason)
	return dbtx.Read(ctx, ur.db, func(ctx context.Context) (*tenant.User, error) {
		u := tenantdb.Users
		q, err := listquery.Pick(u.AllTenants(), columns)
		if err != nil {
			return nil, err
		}

		user, err := q.Where(u.Username.Lower().Eq(strings.ToLower(tenant.SystemUsername))).
			One(ctx, ur.db.Keel(ctx))
		if err != nil {
			return nil, userErrors.Map(err, tenant.SystemUsername)
		}

		return user, nil
	})
}

func (ur *repository) GetTenantMember(
	ctx context.Context,
	req tenantports.GetTenantMemberRequest,
) (*tenant.User, error) {
	return dbtx.Read(ctx, ur.db, func(ctx context.Context) (*tenant.User, error) {
		u := tenantdb.Users
		m := tenantdb.OrganizationMemberships

		member := m.Where(m.UserID.EqCol(u.ID), activeMembership(req.Now))
		entity, err := u.Where(
			u.ID.Eq(req.UserID),
			keel.Or(
				keel.Exists(member),
				u.CurrentOrganizationID.Eq(req.TenantInfo.OrgID),
			),
		).One(ctx, ur.db.Keel(ctx))
		if err != nil {
			return nil, userErrors.Map(err, req.UserID.String())
		}

		return entity, nil
	})
}
