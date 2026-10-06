package seedaccountrepository

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/cloud/seedaccounts/seedaccountport"
	"github.com/emoss08/trenova/internal/core/domain/apikey"
	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgerrcode"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	transactionScopeReason = "retire the legacy seeded administrator accounts, whose rows span every tenant they joined"
	findUsersScopeReason   = "find the legacy seeded administrator accounts by username and email across every tenant"
	referencesScopeReason  = "count the rows of every tenant that reference a legacy seeded administrator account"
	systemUserScopeReason  = "find the instance system user, which no operator command holds a tenant scope for"
	inspectOrgsScopeReason = "inspect the seed-created demo organizations and every catalog table that references them"
	stripUserScopeReason   = "remove a legacy seeded account's memberships, roles, factors, reset tokens and API keys in every organization"
	removeUserScopeReason  = "delete or disable a legacy seeded account after counting every row that references it"
	removeOrgScopeReason   = "delete a seed-created demo organization that holds nothing but seed defaults"

	lockTimeout = 15 * time.Second

	seedEntitySeedName  = "sce.seed_name = ?"
	seedEntityTableName = "sce.table_name = ?"
	seedEntityIDIn      = "sce.entity_id IN (?)"
)

var ErrUserNotFound = errors.New("the seeded user no longer exists")

type Params struct {
	fx.In

	DB     ports.DBConnection
	Logger *zap.Logger
}

type repository struct {
	db ports.DBConnection
	l  *zap.Logger
}

func New(p Params) seedaccountport.Repository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.seed-account-repository"),
	}
}

func (r *repository) InTransaction(
	ctx context.Context,
	opts seedaccountport.TransactionOptions,
	fn func(ctx context.Context) error,
) error {
	ctx = dbscope.WithSystem(ctx, transactionScopeReason)

	return r.db.WithTx(
		ctx,
		ports.TxOptions{ReadOnly: opts.ReadOnly, LockTimeout: lockTimeout},
		func(ctx context.Context, _ bun.Tx) error {
			return fn(ctx)
		},
	)
}

func (r *repository) FindUsers(
	ctx context.Context,
	req *seedaccountport.FindUsersRequest,
) ([]*seedaccountport.UserFootprint, error) {
	if len(req.Accounts) == 0 {
		return []*seedaccountport.UserFootprint{}, nil
	}
	ctx = dbscope.WithSystem(ctx, findUsersScopeReason)

	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) ([]*seedaccountport.UserFootprint, error) {
			users, err := r.selectUsers(ctx, req)
			if err != nil {
				return nil, err
			}
			if len(users) == 0 {
				return []*seedaccountport.UserFootprint{}, nil
			}

			ids := make([]pulid.ID, 0, len(users))
			for _, user := range users {
				ids = append(ids, user.ID)
			}
			tracked, err := r.trackedEntities(ctx, req.SeedName, buncolgen.UserTable.Name, ids)
			if err != nil {
				return nil, err
			}

			out := make([]*seedaccountport.UserFootprint, 0, len(users))
			for _, user := range users {
				footprint, footErr := r.footprint(ctx, user)
				if footErr != nil {
					return nil, footErr
				}
				_, footprint.SeedTracked = tracked[user.ID]
				out = append(out, footprint)
			}

			return out, nil
		},
	)
}

func (r *repository) selectUsers(
	ctx context.Context,
	req *seedaccountport.FindUsersRequest,
) ([]*tenant.User, error) {
	cols := buncolgen.UserColumns
	matches := cols.Username.Eq() + " AND " + cols.EmailAddress.Expr("lower({})") + " = ?"

	users := make([]*tenant.User, 0, len(req.Accounts))
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&users).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			for _, account := range req.Accounts {
				q = q.WhereOr(matches, account.Username, strings.ToLower(account.EmailAddress))
			}
			return q
		}).
		Order(cols.Username.OrderAsc())
	if req.ForUpdate {
		q = q.For("UPDATE")
	}
	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("find seeded users: %w", err)
	}

	return users, nil
}

func (r *repository) trackedEntities(
	ctx context.Context,
	seedName, table string,
	ids []pulid.ID,
) (map[pulid.ID]struct{}, error) {
	out := make(map[pulid.ID]struct{}, len(ids))
	if len(ids) == 0 || seedName == "" {
		return out, nil
	}

	rows := make([]seedhelpers.SeedCreatedEntity, 0, len(ids))
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&rows).
		Where(seedEntitySeedName, seedName).
		Where(seedEntityTableName, table).
		Where(seedEntityIDIn, bun.List(ids)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read seed tracking for %s: %w", table, err)
	}
	for i := range rows {
		out[rows[i].EntityID] = struct{}{}
	}

	return out, nil
}

func (r *repository) footprint(
	ctx context.Context,
	user *tenant.User,
) (*seedaccountport.UserFootprint, error) {
	out := &seedaccountport.UserFootprint{
		ID:                    user.ID,
		BusinessUnitID:        user.BusinessUnitID,
		CurrentOrganizationID: user.CurrentOrganizationID,
		Username:              user.Username,
		EmailAddress:          user.EmailAddress,
		Name:                  user.Name,
		Status:                user.Status,
		IsLocked:              user.IsLocked,
		PasswordHash:          user.Password,
	}

	var err error
	if out.Memberships, err = r.memberships(ctx, user.ID); err != nil {
		return nil, err
	}
	if out.RoleAssignments, err = r.roleAssignments(ctx, user.ID); err != nil {
		return nil, err
	}
	if out.ActiveAPIKeys, err = r.activeAPIKeys(ctx, user.ID); err != nil {
		return nil, err
	}
	if out.MFAAuthenticators, err = r.mfaAuthenticators(ctx, user.ID); err != nil {
		return nil, err
	}
	if out.OpenResetTokens, err = r.openResetTokens(ctx, user.ID); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *repository) memberships(
	ctx context.Context,
	userID pulid.ID,
) ([]seedaccountport.Membership, error) {
	cols := buncolgen.OrganizationMembershipColumns
	rows := make([]tenant.OrganizationMembership, 0)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&rows).
		Column(cols.ID.Bare(), cols.OrganizationID.Bare(), cols.BusinessUnitID.Bare()).
		Where(cols.UserID.Eq(), userID).
		Order(cols.JoinedAt.OrderAsc(), cols.ID.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read memberships: %w", err)
	}

	out := make([]seedaccountport.Membership, 0, len(rows))
	for i := range rows {
		out = append(out, seedaccountport.Membership{
			ID:             rows[i].ID,
			OrganizationID: rows[i].OrganizationID,
			BusinessUnitID: rows[i].BusinessUnitID,
		})
	}

	return out, nil
}

func (r *repository) roleAssignments(
	ctx context.Context,
	userID pulid.ID,
) ([]seedaccountport.RoleAssignment, error) {
	cols := buncolgen.UserRoleAssignmentColumns
	rows := make([]permission.UserRoleAssignment, 0)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&rows).
		Column(cols.ID.Bare(), cols.OrganizationID.Bare(), cols.RoleID.Bare()).
		Where(cols.UserID.Eq(), userID).
		Order(cols.AssignedAt.OrderAsc(), cols.ID.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read role assignments: %w", err)
	}

	out := make([]seedaccountport.RoleAssignment, 0, len(rows))
	for i := range rows {
		out = append(out, seedaccountport.RoleAssignment{
			ID:             rows[i].ID,
			OrganizationID: rows[i].OrganizationID,
			RoleID:         rows[i].RoleID,
		})
	}

	return out, nil
}

func (r *repository) activeAPIKeys(
	ctx context.Context,
	userID pulid.ID,
) ([]seedaccountport.APIKey, error) {
	cols := buncolgen.KeyColumns
	rows := make([]apikey.Key, 0)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&rows).
		Column(
			cols.ID.Bare(),
			cols.OrganizationID.Bare(),
			cols.BusinessUnitID.Bare(),
			cols.Name.Bare(),
			cols.KeyPrefix.Bare(),
		).
		Where(cols.CreatedByID.Eq(), userID).
		Where(cols.Status.Eq(), apikey.StatusActive).
		Order(cols.CreatedAt.OrderAsc(), cols.ID.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read api keys: %w", err)
	}

	out := make([]seedaccountport.APIKey, 0, len(rows))
	for i := range rows {
		out = append(out, seedaccountport.APIKey{
			ID:             rows[i].ID,
			OrganizationID: rows[i].OrganizationID,
			BusinessUnitID: rows[i].BusinessUnitID,
			Name:           rows[i].Name,
			KeyPrefix:      rows[i].KeyPrefix,
		})
	}

	return out, nil
}

func (r *repository) mfaAuthenticators(
	ctx context.Context,
	userID pulid.ID,
) ([]seedaccountport.MFAAuthenticator, error) {
	cols := buncolgen.MFAAuthenticatorColumns
	rows := make([]iam.MFAAuthenticator, 0)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&rows).
		Column(cols.ID.Bare(), cols.OrganizationID.Bare()).
		Where(cols.UserID.Eq(), userID).
		Order(cols.ID.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read mfa authenticators: %w", err)
	}

	out := make([]seedaccountport.MFAAuthenticator, 0, len(rows))
	for i := range rows {
		out = append(out, seedaccountport.MFAAuthenticator{
			ID:             rows[i].ID,
			OrganizationID: rows[i].OrganizationID,
		})
	}

	return out, nil
}

func (r *repository) openResetTokens(ctx context.Context, userID pulid.ID) (int64, error) {
	cols := buncolgen.PasswordResetTokenColumns
	count, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*tenant.PasswordResetToken)(nil)).
		Where(cols.UserID.Eq(), userID).
		Where(cols.UsedAt.IsNull()).
		Where(cols.InvalidatedAt.IsNull()).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count password reset tokens: %w", err)
	}

	return int64(count), nil
}

func (r *repository) CountReferences(
	ctx context.Context,
	req *seedaccountport.CountReferencesRequest,
) ([]seedaccountport.Reference, error) {
	ctx = dbscope.WithSystem(ctx, referencesScopeReason)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]seedaccountport.Reference, error) {
		return r.countReferences(ctx, req.UserID, req.ExcludeOwners)
	})
}

func (r *repository) countReferences(
	ctx context.Context,
	userID pulid.ID,
	excludeOwners []pulid.ID,
) ([]seedaccountport.Reference, error) {
	columns, err := r.referencing(ctx, buncolgen.UserTable.Name)
	if err != nil {
		return nil, err
	}

	owned := ownedByUserTables()
	out := make([]seedaccountport.Reference, 0)
	for _, column := range columns {
		var rows int64
		switch _, isOwned := owned[column.Table]; {
		case column.Table == buncolgen.UserTable.Name:
			rows, err = r.cappedCountExcluding(ctx, column, userID, idColumn, userID)
		case isOwned && len(excludeOwners) > 0:
			rows, err = r.cappedCountExcludingOwners(ctx, column, userID, excludeOwners)
		default:
			rows, err = r.cappedCount(ctx, column, userID)
		}
		if err != nil {
			return nil, err
		}
		if rows > 0 {
			out = append(out, seedaccountport.Reference{
				Table:  column.Table,
				Column: column.Column,
				Rows:   rows,
			})
		}
	}

	return out, nil
}

func (r *repository) referencing(ctx context.Context, parent string) ([]referenceColumn, error) {
	refs := make([]catalogReference, 0)
	if err := r.db.DBForContext(ctx).
		NewRaw(catalogReferencesQuery, parent).
		Scan(ctx, &refs); err != nil {
		return nil, fmt.Errorf("read foreign keys referencing %s: %w", parent, err)
	}

	return referenceColumns(refs), nil
}

func (r *repository) cappedCount(
	ctx context.Context,
	column referenceColumn,
	value pulid.ID,
) (int64, error) {
	var rows int64
	if err := r.db.DBForContext(ctx).
		NewRaw(
			cappedCountQuery,
			bun.Ident(column.Table),
			bun.Ident(column.Column),
			value,
			seedaccountport.ReferenceCap,
		).
		Scan(ctx, &rows); err != nil {
		return 0, fmt.Errorf("count %s.%s: %w", column.Table, column.Column, err)
	}

	return rows, nil
}

func (r *repository) cappedCountExcluding(
	ctx context.Context,
	column referenceColumn,
	value pulid.ID,
	excludeColumn string,
	excluded pulid.ID,
) (int64, error) {
	var rows int64
	if err := r.db.DBForContext(ctx).
		NewRaw(
			cappedCountExcludingSelf,
			bun.Ident(column.Table),
			bun.Ident(column.Column),
			value,
			bun.Ident(excludeColumn),
			excluded,
			seedaccountport.ReferenceCap,
		).
		Scan(ctx, &rows); err != nil {
		return 0, fmt.Errorf("count %s.%s: %w", column.Table, column.Column, err)
	}

	return rows, nil
}

func (r *repository) cappedCountExcludingOwners(
	ctx context.Context,
	column referenceColumn,
	value pulid.ID,
	owners []pulid.ID,
) (int64, error) {
	var rows int64
	if err := r.db.DBForContext(ctx).
		NewRaw(
			cappedCountExcludingOwns,
			bun.Ident(column.Table),
			bun.Ident(column.Column),
			value,
			bun.Ident(ownerColumn),
			bun.List(owners),
			seedaccountport.ReferenceCap,
		).
		Scan(ctx, &rows); err != nil {
		return 0, fmt.Errorf("count %s.%s: %w", column.Table, column.Column, err)
	}

	return rows, nil
}

func (r *repository) FindSystemUser(ctx context.Context) (*seedaccountport.SystemUser, error) {
	ctx = dbscope.WithSystem(ctx, systemUserScopeReason)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*seedaccountport.SystemUser, error) {
		cols := buncolgen.UserColumns
		user := new(tenant.User)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(user).
			Column(cols.ID.Bare(), cols.CurrentOrganizationID.Bare(), cols.BusinessUnitID.Bare()).
			Where(cols.Username.Expr("lower({})")+" = ?", tenant.SystemUsername).
			Order(cols.CreatedAt.OrderAsc()).
			Limit(1).
			Scan(ctx)
		switch {
		case err == nil:
			return &seedaccountport.SystemUser{
				ID:             user.ID,
				OrganizationID: user.CurrentOrganizationID,
				BusinessUnitID: user.BusinessUnitID,
			}, nil
		case dberror.IsNotFoundError(err):
			return nil, nil
		default:
			return nil, fmt.Errorf("find system user: %w", err)
		}
	})
}

func (r *repository) InspectOrganizations(
	ctx context.Context,
	req *seedaccountport.InspectOrganizationsRequest,
) ([]*seedaccountport.OrganizationFootprint, error) {
	if len(req.Organizations) == 0 {
		return []*seedaccountport.OrganizationFootprint{}, nil
	}
	ctx = dbscope.WithSystem(ctx, inspectOrgsScopeReason)

	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) ([]*seedaccountport.OrganizationFootprint, error) {
			orgs, err := r.selectOrganizations(ctx, req)
			if err != nil {
				return nil, err
			}
			if len(orgs) == 0 {
				return []*seedaccountport.OrganizationFootprint{}, nil
			}

			ids := make([]pulid.ID, 0, len(orgs))
			for _, org := range orgs {
				ids = append(ids, org.ID)
			}
			tracked, err := r.trackedEntities(
				ctx,
				req.SeedName,
				buncolgen.OrganizationTable.Name,
				ids,
			)
			if err != nil {
				return nil, err
			}

			columns, err := r.referencing(ctx, buncolgen.OrganizationTable.Name)
			if err != nil {
				return nil, err
			}

			out := make([]*seedaccountport.OrganizationFootprint, 0, len(orgs))
			for _, org := range orgs {
				footprint, inspectErr := r.inspectOrganization(ctx, org, columns, req)
				if inspectErr != nil {
					return nil, inspectErr
				}
				_, footprint.SeedTracked = tracked[org.ID]
				out = append(out, footprint)
			}

			return out, nil
		},
	)
}

func (r *repository) selectOrganizations(
	ctx context.Context,
	req *seedaccountport.InspectOrganizationsRequest,
) ([]*tenant.Organization, error) {
	cols := buncolgen.OrganizationColumns
	matches := cols.Name.Eq() + " AND " + cols.ScacCode.Eq()

	orgs := make([]*tenant.Organization, 0, len(req.Organizations))
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&orgs).
		Column(cols.ID.Bare(), cols.BusinessUnitID.Bare(), cols.Name.Bare(), cols.ScacCode.Bare()).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			for _, org := range req.Organizations {
				q = q.WhereOr(matches, org.Name, org.ScacCode)
			}
			return q
		}).
		Order(cols.CreatedAt.OrderAsc(), cols.ID.OrderAsc())
	if req.ForUpdate {
		q = q.For("UPDATE")
	}
	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("find seeded organizations: %w", err)
	}

	return orgs, nil
}

func (r *repository) inspectOrganization(
	ctx context.Context,
	org *tenant.Organization,
	columns []referenceColumn,
	req *seedaccountport.InspectOrganizationsRequest,
) (*seedaccountport.OrganizationFootprint, error) {
	out := &seedaccountport.OrganizationFootprint{
		ID:             org.ID,
		BusinessUnitID: org.BusinessUnitID,
		Name:           org.Name,
		ScacCode:       org.ScacCode,
		History:        make([]seedaccountport.Reference, 0),
		Data:           make([]seedaccountport.Reference, 0),
	}

	users, err := r.organizationUsers(ctx, org.ID, req.ExcludeUsers)
	if err != nil {
		return nil, err
	}
	out.Users = users

	history := historyTables()
	defaults := seedDefaultTables()
	members := membershipTables()
	for _, column := range columns {
		if column.Table == buncolgen.UserTable.Name {
			continue
		}
		if _, isDefault := defaults[column.Table]; isDefault {
			continue
		}

		var rows int64
		_, isMember := members[column.Table]
		switch {
		case column.Table == buncolgen.OrganizationTable.Name:
			rows, err = r.cappedCountExcluding(ctx, column, org.ID, idColumn, org.ID)
		case isMember && len(req.ExcludeOwners) > 0:
			rows, err = r.cappedCountExcludingOwners(ctx, column, org.ID, req.ExcludeOwners)
		default:
			rows, err = r.cappedCount(ctx, column, org.ID)
		}
		if err != nil {
			return nil, err
		}
		if rows == 0 {
			continue
		}

		ref := seedaccountport.Reference{Table: column.Table, Column: column.Column, Rows: rows}
		if _, isHistory := history[column.Table]; isHistory {
			out.History = append(out.History, ref)
			continue
		}
		out.Data = append(out.Data, ref)
	}

	return out, nil
}

func (r *repository) organizationUsers(
	ctx context.Context,
	orgID pulid.ID,
	excludeUsers []pulid.ID,
) ([]string, error) {
	users := buncolgen.UserColumns
	memberships := buncolgen.OrganizationMembershipColumns
	db := r.db.DBForContext(ctx)

	memberOf := db.NewSelect().
		Model((*tenant.OrganizationMembership)(nil)).
		Column(memberships.UserID.Bare()).
		Where(memberships.OrganizationID.Eq(), orgID)

	rows := make([]tenant.User, 0)
	q := db.NewSelect().
		Model(&rows).
		Column(users.Username.Bare()).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.
				Where(users.CurrentOrganizationID.Eq(), orgID).
				WhereOr(users.ID.In(), memberOf)
		}).
		Order(users.Username.OrderAsc())
	if len(excludeUsers) > 0 {
		q = q.Where(users.ID.NotIn(), bun.List(excludeUsers))
	}
	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("read organization users: %w", err)
	}

	out := make([]string, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].Username)
	}

	return out, nil
}

func (r *repository) StripUser(
	ctx context.Context,
	req *seedaccountport.StripUserRequest,
) (*seedaccountport.StripUserResult, error) {
	ctx = dbscope.WithSystem(ctx, stripUserScopeReason)

	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*seedaccountport.StripUserResult, error) {
			return r.stripUser(ctx, req)
		},
	)
}

func (r *repository) stripUser(
	ctx context.Context,
	req *seedaccountport.StripUserRequest,
) (*seedaccountport.StripUserResult, error) {
	out := new(seedaccountport.StripUserResult)

	var err error
	if out.Memberships, err = r.memberships(ctx, req.UserID); err != nil {
		return nil, err
	}
	if out.RoleAssignments, err = r.roleAssignments(ctx, req.UserID); err != nil {
		return nil, err
	}
	if out.APIKeys, err = r.activeAPIKeys(ctx, req.UserID); err != nil {
		return nil, err
	}
	if out.MFAAuthenticators, err = r.mfaAuthenticators(ctx, req.UserID); err != nil {
		return nil, err
	}

	db := r.db.DBForContext(ctx)

	if _, err = db.NewDelete().
		Model((*permission.UserRoleAssignment)(nil)).
		Where(buncolgen.UserRoleAssignmentColumns.UserID.Eq(), req.UserID).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("delete role assignments: %w", err)
	}

	if _, err = db.NewDelete().
		Model((*tenant.OrganizationMembership)(nil)).
		Where(buncolgen.OrganizationMembershipColumns.UserID.Eq(), req.UserID).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("delete memberships: %w", err)
	}

	if _, err = db.NewDelete().
		Model((*iam.MFAAuthenticator)(nil)).
		Where(buncolgen.MFAAuthenticatorColumns.UserID.Eq(), req.UserID).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("delete mfa authenticators: %w", err)
	}

	resetCols := buncolgen.PasswordResetTokenColumns
	reset, err := db.NewUpdate().
		Model((*tenant.PasswordResetToken)(nil)).
		Set(resetCols.InvalidatedAt.Set(), req.Now).
		Where(resetCols.UserID.Eq(), req.UserID).
		Where(resetCols.UsedAt.IsNull()).
		Where(resetCols.InvalidatedAt.IsNull()).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("invalidate password reset tokens: %w", err)
	}
	if out.ResetTokens, err = reset.RowsAffected(); err != nil {
		return nil, fmt.Errorf("invalidate password reset tokens: %w", err)
	}

	if len(out.APIKeys) > 0 {
		if err = r.revokeAPIKeys(ctx, req, out.APIKeys); err != nil {
			return nil, err
		}
	}

	return out, nil
}

func (r *repository) revokeAPIKeys(
	ctx context.Context,
	req *seedaccountport.StripUserRequest,
	keys []seedaccountport.APIKey,
) error {
	cols := buncolgen.KeyColumns
	ids := make([]pulid.ID, 0, len(keys))
	for i := range keys {
		ids = append(ids, keys[i].ID)
	}

	q := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*apikey.Key)(nil)).
		Set(cols.Status.Set(), apikey.StatusRevoked).
		Set(cols.RevokedAt.Set(), req.Now).
		Set(cols.UpdatedAt.Set(), req.Now).
		Where(cols.ID.In(), bun.List(ids)).
		Where(cols.Status.Eq(), apikey.StatusActive)
	if req.RevokedByID.IsNotNil() {
		q = q.Set(cols.RevokedByID.Set(), req.RevokedByID)
	}
	if _, err := q.Exec(ctx); err != nil {
		return fmt.Errorf("revoke api keys: %w", err)
	}

	return nil
}

func (r *repository) RemoveUser(
	ctx context.Context,
	req *seedaccountport.RemoveUserRequest,
) (*seedaccountport.RemoveUserResult, error) {
	ctx = dbscope.WithSystem(ctx, removeUserScopeReason)

	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*seedaccountport.RemoveUserResult, error) {
			return r.removeUser(ctx, req)
		},
	)
}

func (r *repository) removeUser(
	ctx context.Context,
	req *seedaccountport.RemoveUserRequest,
) (*seedaccountport.RemoveUserResult, error) {
	refs, err := r.countReferences(ctx, req.UserID, nil)
	if err != nil {
		return nil, err
	}

	out := &seedaccountport.RemoveUserResult{References: refs}
	if len(refs) == 0 {
		deleted, deleteErr := r.deleteUser(ctx, req.UserID)
		switch {
		case deleteErr == nil && deleted:
			out.Deleted = true
			return out, nil
		case deleteErr != nil && !retainedByRule(deleteErr):
			return nil, deleteErr
		case deleteErr != nil:
			out.RetainedReason = deleteErr.Error()
		}
	}

	if err = r.disableUser(ctx, req); err != nil {
		return nil, err
	}
	out.Disabled = true

	return out, nil
}

func (r *repository) deleteUser(ctx context.Context, userID pulid.ID) (bool, error) {
	var affected int64
	err := dbtx.Savepoint(ctx, r.db, func(ctx context.Context) error {
		res, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*tenant.User)(nil)).
			Where(buncolgen.UserColumns.ID.Eq(), userID).
			Exec(ctx)
		if err != nil {
			return err
		}
		affected, err = res.RowsAffected()
		return err
	})
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

func (r *repository) disableUser(
	ctx context.Context,
	req *seedaccountport.RemoveUserRequest,
) error {
	cols := buncolgen.UserColumns
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*tenant.User)(nil)).
		Set(cols.Status.Set(), domaintypes.StatusInactive).
		Set(cols.IsLocked.Set(), true).
		Set(cols.MustChangePassword.Set(), true).
		Set(cols.Password.Set(), req.PasswordHash).
		Set(cols.UpdatedAt.Set(), req.Now).
		Set(cols.Version.Inc(1)).
		Where(cols.ID.Eq(), req.UserID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("disable user: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("disable user: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("disable user %s: %w", req.UserID, ErrUserNotFound)
	}

	return nil
}

func (r *repository) RemoveOrganization(
	ctx context.Context,
	org *seedaccountport.OrganizationFootprint,
) (*seedaccountport.RemoveOrganizationResult, error) {
	ctx = dbscope.WithSystem(ctx, removeOrgScopeReason)

	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*seedaccountport.RemoveOrganizationResult, error) {
			cols := buncolgen.OrganizationColumns
			out := new(seedaccountport.RemoveOrganizationResult)

			var affected int64
			err := dbtx.Savepoint(ctx, r.db, func(ctx context.Context) error {
				res, err := r.db.DBForContext(ctx).
					NewDelete().
					Model((*tenant.Organization)(nil)).
					Where(cols.ID.Eq(), org.ID).
					Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
					Exec(ctx)
				if err != nil {
					return err
				}
				affected, err = res.RowsAffected()
				return err
			})
			switch {
			case err == nil:
				out.Deleted = affected > 0
				if !out.Deleted {
					out.RetainedReason = "the organization no longer exists"
				}
				return out, nil
			case retainedByRule(err):
				out.RetainedReason = err.Error()
				return out, nil
			default:
				return nil, fmt.Errorf("delete organization: %w", err)
			}
		},
	)
}

func retainedByRule(err error) bool {
	return slices.Contains([]string{
		pgerrcode.ForeignKeyViolation,
		pgerrcode.RestrictViolation,
		pgerrcode.InsufficientPrivilege,
		pgerrcode.RaiseException,
	}, dberror.ExtractCode(err))
}
