package tenantpurgerepository

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/cloudsignup"
	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/jackc/pgerrcode"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	membersScopeReason      = "list an expired cloud organization's members before its purge"
	orgNameScopeReason      = "read a cloud organization's name and timezone for its lifecycle email"
	purgeRowsScopeReason    = "delete every row an expired cloud organization owns across all tenant tables"
	purgeUserScopeReason    = "remove or deactivate a purged cloud organization's user who belongs to no other organization"
	deleteTenantScopeReason = "delete an expired cloud organization, its business unit and its signup record"

	defaultBatchSize  = 500
	maxBatchSize      = 5_000
	defaultMaxBatches = 40
	maxPasses         = 4
	maxReferenceFixes = 3

	catalogTablesQuery = `SELECT c.relname AS table_name,
       bool_or(a.attname = 'organization_id') AS has_org,
       bool_or(a.attname = 'business_unit_id') AS has_bu
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
JOIN pg_catalog.pg_attribute a ON a.attrelid = c.oid
WHERE n.nspname = 'public'
  AND c.relkind IN ('r', 'p')
  AND NOT c.relispartition
  AND a.attnum > 0
  AND NOT a.attisdropped
  AND a.attname IN ('organization_id', 'business_unit_id')
GROUP BY c.relname`

	catalogForeignKeysQuery = `SELECT con.conname AS constraint_name,
       child.relname AS child_table,
       parent.relname AS parent_table,
       con.confdeltype::text AS on_delete,
       ARRAY(
           SELECT att.attname::text
           FROM unnest(con.conkey) WITH ORDINALITY AS k(attnum, ord)
           JOIN pg_catalog.pg_attribute att ON att.attrelid = con.conrelid AND att.attnum = k.attnum
           ORDER BY k.ord
       ) AS child_columns,
       ARRAY(
           SELECT att.attname::text
           FROM unnest(con.confkey) WITH ORDINALITY AS k(attnum, ord)
           JOIN pg_catalog.pg_attribute att ON att.attrelid = con.confrelid AND att.attnum = k.attnum
           ORDER BY k.ord
       ) AS parent_columns
FROM pg_catalog.pg_constraint con
JOIN pg_catalog.pg_class child ON child.oid = con.conrelid
JOIN pg_catalog.pg_class parent ON parent.oid = con.confrelid
JOIN pg_catalog.pg_namespace n ON n.oid = child.relnamespace
WHERE con.contype = 'f'
  AND n.nspname = 'public'
  AND NOT child.relispartition`

	batchDeleteStatement = "DELETE FROM ? WHERE (tableoid, ctid) IN " +
		"(SELECT tableoid, ctid FROM ? WHERE ? = ? LIMIT ?)"
	allDeleteStatement        = "DELETE FROM ? WHERE ? = ?"
	referencingDeleteTemplate = "DELETE FROM ? WHERE ? IN (SELECT ? FROM ? WHERE ? = ?)"
)

var ErrTenantRequired = errors.New("a tenant purge requires an organization and business unit")

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.TenantPurgeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.tenant-purge-repository"),
	}
}

type memberRow struct {
	UserID       pulid.ID `bun:"user_id"`
	Name         string   `bun:"name"`
	EmailAddress string   `bun:"email_address"`
	Username     string   `bun:"username"`
}

type otherMembershipRow struct {
	UserID pulid.ID `bun:"user_id"`
	Total  int      `bun:"total"`
}

func (r *repository) ListMembers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*repositories.TenantMember, error) {
	if err := requireTenant(tenantInfo); err != nil {
		return nil, err
	}
	ctx = dbscope.WithSystem(ctx, membersScopeReason)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*repositories.TenantMember, error) {
		db := r.db.DBForContext(ctx)
		memberships := buncolgen.OrganizationMembershipColumns
		users := buncolgen.UserColumns

		rows := make([]memberRow, 0)
		err := db.NewSelect().
			Model((*tenant.OrganizationMembership)(nil)).
			Join(
				"JOIN "+buncolgen.UserTable.As(buncolgen.UserTable.Alias)+" ON "+
					users.ID.EqColumn(memberships.UserID),
			).
			ColumnExpr(memberships.UserID.As("user_id")).
			ColumnExpr(users.Name.As("name")).
			ColumnExpr(users.EmailAddress.As("email_address")).
			ColumnExpr(users.Username.As("username")).
			Apply(buncolgen.OrganizationMembershipApplyTenant(tenantInfo)).
			Order(memberships.UserID.OrderAsc()).
			Scan(ctx, &rows)
		if err != nil {
			return nil, fmt.Errorf("list organization members: %w", err)
		}
		if len(rows) == 0 {
			return []*repositories.TenantMember{}, nil
		}

		userIDs := make([]pulid.ID, 0, len(rows))
		for i := range rows {
			userIDs = append(userIDs, rows[i].UserID)
		}

		others := make([]otherMembershipRow, 0, len(rows))
		err = db.NewSelect().
			Model((*tenant.OrganizationMembership)(nil)).
			ColumnExpr(memberships.UserID.As("user_id")).
			ColumnExpr(buncolgen.Count("total")).
			Where(memberships.UserID.In(), bun.List(userIDs)).
			Where(memberships.OrganizationID.Ne(), tenantInfo.OrgID).
			GroupExpr(memberships.UserID.Qualified()).
			Scan(ctx, &others)
		if err != nil {
			return nil, fmt.Errorf("count other memberships: %w", err)
		}

		otherByUser := make(map[pulid.ID]int, len(others))
		for i := range others {
			otherByUser[others[i].UserID] = others[i].Total
		}

		members := make([]*repositories.TenantMember, 0, len(rows))
		for i := range rows {
			members = append(members, &repositories.TenantMember{
				UserID:           rows[i].UserID,
				Name:             rows[i].Name,
				EmailAddress:     rows[i].EmailAddress,
				Username:         rows[i].Username,
				OtherMemberships: otherByUser[rows[i].UserID],
			})
		}

		return members, nil
	})
}

func (r *repository) OrganizationProfile(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*repositories.TenantProfile, error) {
	if err := requireTenant(tenantInfo); err != nil {
		return nil, err
	}
	ctx = dbscope.WithSystem(ctx, orgNameScopeReason)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*repositories.TenantProfile, error) {
		cols := buncolgen.OrganizationColumns
		org := new(tenant.Organization)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(org).
			Column(cols.Name.Bare(), cols.Timezone.Bare()).
			Where(cols.ID.Eq(), tenantInfo.OrgID).
			Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID).
			Limit(1).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "Organization")
		}

		return &repositories.TenantProfile{Name: org.Name, Timezone: org.Timezone}, nil
	})
}

func (r *repository) PurgeRows(
	ctx context.Context,
	req *repositories.PurgeTenantRowsRequest,
) (*repositories.PurgeTenantRowsResult, error) {
	if err := requireTenant(req.TenantInfo); err != nil {
		return nil, err
	}
	ctx = dbscope.WithSystem(ctx, purgeRowsScopeReason)

	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*repositories.PurgeTenantRowsResult, error) {
		if err := r.requirePurgeable(ctx, req.TenantInfo); err != nil {
			return nil, err
		}

		return r.purgeRows(ctx, req)
	})
}

func (r *repository) purgeRows(
	ctx context.Context,
	req *repositories.PurgeTenantRowsRequest,
) (*repositories.PurgeTenantRowsResult, error) {
	plan, err := r.loadPlan(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	run := &purgeRun{
		repo:       r,
		plan:       plan,
		batchSize:  clampBatchSize(req.BatchSize),
		maxBatches: req.MaxBatches,
		deleted:    make(map[string]int64),
		retained:   make(map[string]struct{}),
		cleared:    make(map[string]struct{}),
	}
	if run.maxBatches <= 0 {
		run.maxBatches = defaultMaxBatches
	}

	return run.execute(ctx)
}

func (r *repository) loadPlan(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*purgePlan, error) {
	db := r.db.DBForContext(ctx)

	tables := make([]tenantTable, 0, 512)
	if err := db.NewRaw(catalogTablesQuery).Scan(ctx, &tables); err != nil {
		return nil, fmt.Errorf("read tenant tables: %w", err)
	}

	fks := make([]*foreignKey, 0, 1024)
	if err := db.NewRaw(catalogForeignKeysQuery).Scan(ctx, &fks); err != nil {
		return nil, fmt.Errorf("read foreign keys: %w", err)
	}

	exclusive, err := r.exclusiveBusinessUnit(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	return buildPlan(&planInput{
		Tables:         tables,
		ForeignKeys:    fks,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ExclusiveBU:    exclusive,
	}), nil
}

func (r *repository) exclusiveBusinessUnit(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	cols := buncolgen.OrganizationColumns
	others, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*tenant.Organization)(nil)).
		Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID).
		Where(cols.ID.Ne(), tenantInfo.OrgID).
		Count(ctx)
	if err != nil {
		return false, fmt.Errorf("count business unit organizations: %w", err)
	}

	return others == 0, nil
}

type purgeRun struct {
	repo       *repository
	plan       *purgePlan
	batchSize  int
	maxBatches int
	batches    int
	total      int64
	deleted    map[string]int64
	retained   map[string]struct{}
	cleared    map[string]struct{}
}

func (p *purgeRun) execute(ctx context.Context) (*repositories.PurgeTenantRowsResult, error) {
	var blocked map[string]struct{}
	for range maxPasses {
		blocked = make(map[string]struct{})
		before := p.total

		for _, target := range p.plan.targets {
			if _, retained := p.retained[target.Table]; retained {
				continue
			}

			done, err := p.drain(ctx, target)
			if err != nil {
				return nil, err
			}
			if p.batches >= p.maxBatches && !done {
				return p.result(nil, false), nil
			}
			if !done {
				blocked[target.Table] = struct{}{}
			}
		}

		if len(blocked) == 0 {
			return p.result(nil, true), nil
		}
		if p.total == before {
			break
		}
	}

	return p.result(blocked, false), nil
}

func (p *purgeRun) clearDependents(ctx context.Context, target purgeTarget) error {
	if _, done := p.cleared[target.Table]; done {
		return nil
	}
	p.cleared[target.Table] = struct{}{}

	for _, fk := range p.plan.dependents[target.Table] {
		if Retained(fk.ChildTable) {
			continue
		}

		affected, err := p.repo.deleteReferencing(ctx, fk, target)
		if err != nil {
			if retainedByRule(err) {
				continue
			}
			return fmt.Errorf("purge %s rows referencing %s: %w", fk.ChildTable, target.Table, err)
		}
		p.record(fk.ChildTable, affected)
	}

	return nil
}

func (p *purgeRun) drain(ctx context.Context, target purgeTarget) (bool, error) {
	if err := p.clearDependents(ctx, target); err != nil {
		return false, err
	}

	fixes := 0
	for p.batches < p.maxBatches {
		p.batches++
		affected, err := p.repo.deleteBatch(ctx, target, p.batchSize)
		if err == nil {
			p.record(target.Table, affected)
			if affected < int64(p.batchSize) {
				return true, nil
			}
			continue
		}

		switch dberror.ExtractCode(err) {
		case pgerrcode.ForeignKeyViolation:
			if fixes >= maxReferenceFixes {
				return false, nil
			}
			fixes++
			resolved, fixErr := p.resolveReference(ctx, target, dberror.ExtractConstraintName(err))
			if fixErr != nil {
				return false, fixErr
			}
			if !resolved {
				return false, nil
			}
		case pgerrcode.InsufficientPrivilege, pgerrcode.RaiseException:
			p.repo.l.Warn("tenant purge left a table its retention rules protect",
				zap.String("table", target.Table),
				zap.Error(err),
			)
			p.retained[target.Table] = struct{}{}
			return true, nil
		default:
			return false, fmt.Errorf("purge %s: %w", target.Table, err)
		}
	}

	return false, nil
}

func (p *purgeRun) resolveReference(
	ctx context.Context,
	target purgeTarget,
	constraint string,
) (bool, error) {
	fk, ok := p.plan.constraints[constraint]
	if !ok {
		return false, nil
	}

	if fk.selfReferencing() {
		affected, err := p.repo.deleteAll(ctx, target)
		if err != nil {
			if dberror.IsForeignKeyConstraintViolation(err) {
				return false, nil
			}
			return false, fmt.Errorf("purge %s: %w", target.Table, err)
		}
		p.record(target.Table, affected)
		return true, nil
	}

	if _, childIsTarget := p.plan.byTable[fk.ChildTable]; childIsTarget || Retained(fk.ChildTable) {
		return false, nil
	}

	affected, err := p.repo.deleteReferencing(ctx, fk, target)
	if err != nil {
		code := dberror.ExtractCode(err)
		if code == pgerrcode.ForeignKeyViolation || code == pgerrcode.InsufficientPrivilege ||
			code == pgerrcode.RaiseException {
			return false, nil
		}
		return false, fmt.Errorf("purge %s rows referencing %s: %w", fk.ChildTable, target.Table, err)
	}
	p.record(fk.ChildTable, affected)

	return true, nil
}

func (p *purgeRun) record(table string, affected int64) {
	if affected <= 0 {
		return
	}
	p.deleted[table] += affected
	p.total += affected
}

func (p *purgeRun) result(
	blocked map[string]struct{},
	complete bool,
) *repositories.PurgeTenantRowsResult {
	result := &repositories.PurgeTenantRowsResult{
		Deleted:  p.total,
		Tables:   make([]repositories.PurgeTableOutcome, 0, len(p.deleted)),
		Retained: sortedKeys(p.retained),
		Blocked:  sortedKeys(blocked),
		Complete: complete,
	}
	for table, deleted := range p.deleted {
		result.Tables = append(result.Tables, repositories.PurgeTableOutcome{
			Table:   table,
			Deleted: deleted,
		})
	}
	slices.SortFunc(result.Tables, func(a, b repositories.PurgeTableOutcome) int {
		switch {
		case a.Table < b.Table:
			return -1
		case a.Table > b.Table:
			return 1
		default:
			return 0
		}
	})

	return result
}

func (r *repository) deleteBatch(ctx context.Context, target purgeTarget, limit int) (int64, error) {
	var affected int64
	err := dbtx.Savepoint(ctx, r.db, func(ctx context.Context) error {
		result, err := r.db.DBForContext(ctx).
			NewRaw(
				batchDeleteStatement,
				bun.Ident(target.Table),
				bun.Ident(target.Table),
				bun.Ident(target.Column),
				target.Value,
				limit,
			).
			Exec(ctx)
		if err != nil {
			return err
		}
		affected, err = result.RowsAffected()
		return err
	})

	return affected, err
}

func (r *repository) deleteAll(ctx context.Context, target purgeTarget) (int64, error) {
	var affected int64
	err := dbtx.Savepoint(ctx, r.db, func(ctx context.Context) error {
		result, err := r.db.DBForContext(ctx).
			NewRaw(
				allDeleteStatement,
				bun.Ident(target.Table),
				bun.Ident(target.Column),
				target.Value,
			).
			Exec(ctx)
		if err != nil {
			return err
		}
		affected, err = result.RowsAffected()
		return err
	})

	return affected, err
}

func (r *repository) deleteReferencing(
	ctx context.Context,
	fk *foreignKey,
	target purgeTarget,
) (int64, error) {
	if len(fk.ChildColumns) == 0 || len(fk.ChildColumns) != len(fk.ParentColumns) {
		return 0, nil
	}

	var affected int64
	err := dbtx.Savepoint(ctx, r.db, func(ctx context.Context) error {
		result, err := r.db.DBForContext(ctx).
			NewRaw(
				referencingDeleteTemplate,
				bun.Ident(fk.ChildTable),
				bun.Safe(columnList(fk.ChildColumns)),
				bun.Safe(joinedColumns(fk.ParentColumns)),
				bun.Ident(target.Table),
				bun.Ident(target.Column),
				target.Value,
			).
			Exec(ctx)
		if err != nil {
			return err
		}
		affected, err = result.RowsAffected()
		return err
	})

	return affected, err
}

func (r *repository) PurgeUser(
	ctx context.Context,
	req *repositories.PurgeTenantUserRequest,
) (*repositories.PurgeTenantUserResult, error) {
	if err := requireTenant(req.TenantInfo); err != nil {
		return nil, err
	}
	if req.UserID.IsNil() {
		return nil, ErrTenantRequired
	}
	ctx = dbscope.WithSystem(ctx, purgeUserScopeReason)

	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*repositories.PurgeTenantUserResult, error) {
		if err := r.requirePurgeable(ctx, req.TenantInfo); err != nil {
			return nil, err
		}

		return r.purgeUser(ctx, req)
	})
}

func (r *repository) purgeUser(
	ctx context.Context,
	req *repositories.PurgeTenantUserRequest,
) (*repositories.PurgeTenantUserResult, error) {
	db := r.db.DBForContext(ctx)
	memberships := buncolgen.OrganizationMembershipColumns
	users := buncolgen.UserColumns
	result := new(repositories.PurgeTenantUserResult)

	other := new(tenant.OrganizationMembership)
	err := db.NewSelect().
		Model(other).
		Where(memberships.UserID.Eq(), req.UserID).
		Where(memberships.OrganizationID.Ne(), req.TenantInfo.OrgID).
		Order(memberships.IsDefault.OrderDesc(), memberships.JoinedAt.OrderAsc()).
		Limit(1).
		Scan(ctx)
	switch {
	case err == nil:
		if _, err = db.NewUpdate().
			Model((*tenant.User)(nil)).
			Set(users.CurrentOrganizationID.Set(), other.OrganizationID).
			Set(users.BusinessUnitID.Set(), other.BusinessUnitID).
			Set(users.UpdatedAt.Set(), timeutils.NowUnix()).
			Where(users.ID.Eq(), req.UserID).
			Where(users.CurrentOrganizationID.Eq(), req.TenantInfo.OrgID).
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("move user to another organization: %w", err)
		}
		result.Reassigned = true
		return result, nil
	case !dberror.IsNotFoundError(err):
		return nil, fmt.Errorf("read the user's other memberships: %w", err)
	}

	var deleted int64
	err = dbtx.Savepoint(ctx, r.db, func(ctx context.Context) error {
		res, deleteErr := r.db.DBForContext(ctx).
			NewDelete().
			Model((*tenant.User)(nil)).
			Where(users.ID.Eq(), req.UserID).
			Where(ownedByTenantClause(), req.TenantInfo.OrgID, req.TenantInfo.BuID).
			Exec(ctx)
		if deleteErr != nil {
			return deleteErr
		}
		deleted, deleteErr = res.RowsAffected()
		return deleteErr
	})
	if err == nil {
		result.Deleted = deleted > 0
		return result, nil
	}

	code := dberror.ExtractCode(err)
	if code != pgerrcode.ForeignKeyViolation && code != pgerrcode.InsufficientPrivilege &&
		code != pgerrcode.RaiseException {
		return nil, fmt.Errorf("delete user: %w", err)
	}

	deactivated, err := db.NewUpdate().
		Model((*tenant.User)(nil)).
		Set(users.Status.Set(), domaintypes.StatusInactive).
		Set(users.UpdatedAt.Set(), timeutils.NowUnix()).
		Where(users.ID.Eq(), req.UserID).
		Where(ownedByTenantClause(), req.TenantInfo.OrgID, req.TenantInfo.BuID).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("deactivate user: %w", err)
	}
	affected, err := deactivated.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("deactivate user: %w", err)
	}
	result.Deactivated = affected > 0

	return result, nil
}

func (r *repository) DeleteTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*repositories.DeleteTenantResult, error) {
	if err := requireTenant(tenantInfo); err != nil {
		return nil, err
	}
	ctx = dbscope.WithSystem(ctx, deleteTenantScopeReason)

	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*repositories.DeleteTenantResult, error) {
		return r.deleteTenant(ctx, tenantInfo)
	})
}

func (r *repository) deleteTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*repositories.DeleteTenantResult, error) {
	result := new(repositories.DeleteTenantResult)

	exists, err := r.organizationExists(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if !exists {
		result.OrganizationDeleted = true
		return result, nil
	}
	if err = r.requirePurgeable(ctx, tenantInfo); err != nil {
		return nil, err
	}

	signups, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*cloudsignup.CloudSignup)(nil)).
		Where(buncolgen.CloudSignupColumns.ProvisionedOrganizationID.Eq(), tenantInfo.OrgID).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("delete signup records: %w", err)
	}
	result.SignupsDeleted, _ = signups.RowsAffected()

	orgCols := buncolgen.OrganizationColumns
	err = dbtx.Savepoint(ctx, r.db, func(ctx context.Context) error {
		_, deleteErr := r.db.DBForContext(ctx).
			NewDelete().
			Model((*tenant.Organization)(nil)).
			Where(orgCols.ID.Eq(), tenantInfo.OrgID).
			Where(orgCols.BusinessUnitID.Eq(), tenantInfo.BuID).
			Exec(ctx)
		return deleteErr
	})
	if err != nil {
		if retainedByRule(err) {
			result.RetainedReason = err.Error()
			return result, nil
		}
		return nil, fmt.Errorf("delete organization: %w", err)
	}
	result.OrganizationDeleted = true

	exclusive, err := r.exclusiveBusinessUnit(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if !exclusive {
		return result, nil
	}

	err = dbtx.Savepoint(ctx, r.db, func(ctx context.Context) error {
		_, deleteErr := r.db.DBForContext(ctx).
			NewDelete().
			Model((*tenant.BusinessUnit)(nil)).
			Where(buncolgen.BusinessUnitColumns.ID.Eq(), tenantInfo.BuID).
			Exec(ctx)
		return deleteErr
	})
	if err != nil {
		if retainedByRule(err) {
			result.RetainedReason = err.Error()
			return result, nil
		}
		return nil, fmt.Errorf("delete business unit: %w", err)
	}
	result.BusinessUnitDeleted = true

	return result, nil
}

func (r *repository) requirePurgeable(ctx context.Context, tenantInfo pagination.TenantInfo) error {
	exists, err := r.organizationExists(ctx, tenantInfo)
	if err != nil {
		return err
	}
	if !exists {
		return repositories.ErrTenantNotPurgeable
	}

	cols := buncolgen.SubscriptionColumns
	sub := new(subscription.Subscription)
	err = r.db.DBForContext(ctx).
		NewSelect().
		Model(sub).
		Column(cols.ID.Bare()).
		Where(cols.OrganizationID.Eq(), tenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID).
		Where(cols.Status.Eq(), subscription.StatusExpired).
		For("UPDATE").
		Limit(1).
		Scan(ctx)
	if err == nil {
		return nil
	}
	if dberror.IsNotFoundError(err) {
		return repositories.ErrTenantNotPurgeable
	}

	return fmt.Errorf("read the organization's subscription: %w", err)
}

func (r *repository) organizationExists(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	cols := buncolgen.OrganizationColumns
	exists, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*tenant.Organization)(nil)).
		Where(cols.ID.Eq(), tenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("read organization: %w", err)
	}

	return exists, nil
}

func ownedByTenantClause() string {
	users := buncolgen.UserColumns
	return "(" + users.CurrentOrganizationID.Eq() + " OR " + users.BusinessUnitID.Eq() + ")"
}

func retainedByRule(err error) bool {
	switch dberror.ExtractCode(err) {
	case pgerrcode.ForeignKeyViolation, pgerrcode.InsufficientPrivilege, pgerrcode.RaiseException:
		return true
	default:
		return false
	}
}

func requireTenant(tenantInfo pagination.TenantInfo) error {
	if tenantInfo.OrgID.IsNil() || tenantInfo.BuID.IsNil() {
		return ErrTenantRequired
	}

	return nil
}

func clampBatchSize(size int) int {
	if size <= 0 {
		return defaultBatchSize
	}

	return min(size, maxBatchSize)
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	return keys
}
