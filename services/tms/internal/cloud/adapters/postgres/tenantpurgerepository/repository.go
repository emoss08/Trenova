package tenantpurgerepository

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures/managedfeaturesdb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage/managedusagedb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	planReason         = "read the catalog's tenant tables and foreign keys to delete one removed shared-server tenant's rows in dependency order"
	purgeReason        = "delete one removed shared-server tenant's rows, a batch at a time, across every table that carries its organization or business unit"
	organizationReason = "delete a removed shared-server tenant's organization and its managed scope, whose cascade removes the append-only audit rows no direct delete may touch"
	unitReason         = "delete a removed shared-server tenant's business unit once nothing of the tenant references it"
	countReason        = "count what is left of one removed shared-server tenant in every table that carries it, to prove the removal is complete"

	defaultBatchSize  = 500
	maxBatchSize      = 5_000
	defaultMaxBatches = 200
	maxPasses         = 4
	maxReferenceFixes = 3
	statementTimeout  = "30s"
	lockTimeout       = "5s"

	catalogTablesQuery = `SELECT c.relname AS table_name,
       bool_or(a.attname = 'organization_id') AS has_org,
       bool_or(a.attname = 'business_unit_id') AS has_bu,
       COALESCE(min(a.attname::text) FILTER (
           WHERE a.attname LIKE '%\_business\_unit\_id'
             AND EXISTS (
                 SELECT 1
                 FROM pg_catalog.pg_attribute o
                 WHERE o.attrelid = c.oid
                   AND NOT o.attisdropped
                   AND o.attname = left(a.attname, -length('business_unit_id')) || 'organization_id'
             )
             AND NOT EXISTS (
                 SELECT 1 FROM trenova_rls.global_tables g WHERE g.table_name = c.relname
             )
       ), '') AS party_bu
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
JOIN pg_catalog.pg_attribute a ON a.attrelid = c.oid
WHERE n.nspname = 'public'
  AND c.relkind IN ('r', 'p')
  AND NOT c.relispartition
  AND a.attnum > 0
  AND NOT a.attisdropped
  AND (a.attname IN ('organization_id', 'business_unit_id') OR a.attname LIKE '%\_business\_unit\_id')
GROUP BY c.oid, c.relname
ORDER BY c.relname`

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
  AND NOT child.relispartition
ORDER BY con.conname`
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

func New(p Params) cloudtenantports.Purge {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.cloud-tenant-purge")}
}

func bound(ctx context.Context, db *keel.DB) error {
	if _, err := keel.Exec(
		ctx,
		db,
		"SELECT set_config('statement_timeout', $1, true), set_config('lock_timeout', $2, true)",
		statementTimeout,
		lockTimeout,
	); err != nil {
		return fmt.Errorf("bound the purge statement: %w", err)
	}
	return nil
}

func (r *repository) PurgeRows(
	ctx context.Context,
	req *cloudtenantports.PurgeRowsRequest,
) (*cloudtenantports.PurgeRowsResult, error) {
	if !req.Tenant.Valid() {
		return nil, ErrTenantRequired
	}
	plan, err := r.loadPlan(ctx, req.Tenant)
	if err != nil {
		return nil, err
	}
	if err = r.refuseCrossTenantReferences(ctx, plan, req.Tenant); err != nil {
		return nil, err
	}
	run := &purgeRun{
		repo:       r,
		plan:       plan,
		tenant:     req.Tenant,
		batchSize:  clampBatchSize(req.BatchSize),
		maxBatches: req.MaxBatches,
		deleted:    make(map[string]int64),
		protected:  make(map[string]struct{}),
		cleared:    make(map[string]struct{}),
		beat:       req.Beat,
	}
	if run.maxBatches <= 0 {
		run.maxBatches = defaultMaxBatches
	}
	return run.execute(ctx)
}

func (r *repository) refuseInbound(ctx context.Context, tenant dbscope.Tenant) error {
	plan, err := r.loadPlan(ctx, tenant)
	if err != nil {
		return err
	}
	return r.refuseCrossTenantReferences(ctx, plan, tenant)
}

func (r *repository) loadPlan(ctx context.Context, tenant dbscope.Tenant) (*purgePlan, error) {
	ctx = dbscope.WithSystem(ctx, planReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*purgePlan, error) {
		db := r.db.Keel(ctx)
		tables, err := keel.QueryAll(
			ctx,
			db,
			pgx.RowToStructByName[tenantTable],
			catalogTablesQuery,
		)
		if err != nil {
			return nil, fmt.Errorf("read the tenant tables: %w", err)
		}
		fks, err := keel.QueryAll(
			ctx,
			db,
			pgx.RowToAddrOfStructByName[foreignKey],
			catalogForeignKeysQuery,
		)
		if err != nil {
			return nil, fmt.Errorf("read the foreign keys: %w", err)
		}
		o := tenantdb.Organizations
		others, err := o.Where(
			o.BusinessUnitID.Eq(tenant.BusinessUnitID),
			o.ID.Ne(tenant.OrganizationID),
		).AllTenants().Count(ctx, db)
		if err != nil {
			return nil, fmt.Errorf("count the business unit's other organizations: %w", err)
		}
		holding, err := keel.QueryAll(
			ctx,
			db,
			pgx.RowTo[string],
			HoldingQuery(tables, others == 0),
			tenant.OrganizationID.String(),
			tenant.BusinessUnitID.String(),
		)
		if err != nil {
			return nil, fmt.Errorf("find the tables that hold the tenant: %w", err)
		}
		return buildPlan(&planInput{
			Tables:         holdingTables(tables, holding),
			AllTables:      tables,
			ForeignKeys:    fks,
			OrganizationID: tenant.OrganizationID,
			BusinessUnitID: tenant.BusinessUnitID,
			ExclusiveBU:    others == 0,
		}), nil
	})
}

type purgeRun struct {
	repo       *repository
	plan       *purgePlan
	tenant     dbscope.Tenant
	batchSize  int
	maxBatches int
	batches    int
	total      int64
	deleted    map[string]int64
	protected  map[string]struct{}
	cleared    map[string]struct{}
	beat       func()
}

func (p *purgeRun) execute(ctx context.Context) (*cloudtenantports.PurgeRowsResult, error) {
	var left map[string]struct{}
	for range maxPasses {
		left = make(map[string]struct{})
		before := p.total

		for _, target := range p.plan.targets {
			if _, protected := p.protected[target.Table]; protected {
				left[target.Table] = struct{}{}
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
				left[target.Table] = struct{}{}
			}
		}

		if len(left) == 0 {
			return p.result(nil, true), nil
		}
		if p.total == before {
			break
		}
	}

	return p.result(left, false), nil
}

func (p *purgeRun) clearDependents(ctx context.Context, target purgeTarget) error {
	if _, done := p.cleared[target.Table]; done {
		return nil
	}
	for _, fk := range p.plan.dependents[target.Table] {
		for p.batches < p.maxBatches {
			p.batches++
			affected, err := p.repo.deleteReferencing(ctx, p.reference(fk, target), p.batchSize)
			if err != nil {
				if protectedByRule(err) || referenceViolation(err) {
					break
				}
				return fmt.Errorf(
					"purge %s rows referencing %s: %w",
					fk.ChildTable,
					target.Table,
					err,
				)
			}
			p.record(fk.ChildTable, affected)
			if affected < int64(p.batchSize) {
				break
			}
		}
	}
	p.cleared[target.Table] = struct{}{}
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

		switch {
		case referenceViolation(err):
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
		case keel.IsNotNull(err):
			return false, nil
		case protectedByRule(err):
			p.repo.l.Info(
				"a table's own rules refuse a direct delete; its rows go with the organization's cascade",
				zap.String("table", target.Table),
			)
			p.protected[target.Table] = struct{}{}
			return false, nil
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
			if referenceViolation(err) || protectedByRule(err) {
				return false, nil
			}
			return false, fmt.Errorf("purge %s: %w", target.Table, err)
		}
		p.record(target.Table, affected)
		return true, nil
	}

	if Kept(fk.ChildTable) {
		return false, nil
	}
	if _, childIsTarget := p.plan.byTable[fk.ChildTable]; childIsTarget {
		return p.detach(ctx, fk, target)
	}

	affected, err := p.repo.deleteReferencing(ctx, p.reference(fk, target), p.batchSize)
	if err != nil {
		if referenceViolation(err) || protectedByRule(err) {
			return false, nil
		}
		return false, fmt.Errorf(
			"purge %s rows referencing %s: %w",
			fk.ChildTable,
			target.Table,
			err,
		)
	}
	p.record(fk.ChildTable, affected)
	return affected > 0, nil
}

func (p *purgeRun) detach(
	ctx context.Context,
	fk *foreignKey,
	target purgeTarget,
) (bool, error) {
	detached := false
	for p.batches < p.maxBatches {
		p.batches++
		affected, err := p.repo.nullReferencing(ctx, p.reference(fk, target), p.batchSize)
		if err != nil {
			if keel.IsNotNull(err) || protectedByRule(err) ||
				referenceViolation(err) {
				return detached, nil
			}
			return false, fmt.Errorf("detach %s from %s: %w", fk.ChildTable, target.Table, err)
		}
		if p.beat != nil {
			p.beat()
		}
		detached = detached || affected > 0
		if affected < int64(p.batchSize) {
			return detached, nil
		}
	}
	return detached, nil
}

func (p *purgeRun) record(table string, affected int64) {
	if p.beat != nil {
		p.beat()
	}
	if affected <= 0 {
		return
	}
	p.deleted[table] += affected
	p.total += affected
}

func (p *purgeRun) result(
	left map[string]struct{},
	complete bool,
) *cloudtenantports.PurgeRowsResult {
	result := &cloudtenantports.PurgeRowsResult{
		Deleted:  p.total,
		Tables:   make([]cloudtenantports.PurgeTableOutcome, 0, len(p.deleted)),
		Left:     sortedKeys(left),
		Complete: complete,
	}
	for table, deleted := range p.deleted {
		result.Tables = append(result.Tables, cloudtenantports.PurgeTableOutcome{
			Table:   table,
			Deleted: deleted,
		})
	}
	slices.SortFunc(result.Tables, func(a, b cloudtenantports.PurgeTableOutcome) int {
		return strings.Compare(a.Table, b.Table)
	})
	return result
}

func (r *repository) deleteBatch(
	ctx context.Context,
	target purgeTarget,
	limit int,
) (int64, error) {
	table := quoteIdent(target.Table)
	sql := "DELETE FROM " + table + " WHERE (tableoid, ctid) IN (SELECT tableoid, ctid FROM " +
		table + " WHERE " + quoteIdent(target.Column) + " = $1 LIMIT $2)"
	return r.exec(ctx, sql, target.Value.String(), limit)
}

func (r *repository) deleteAll(ctx context.Context, target purgeTarget) (int64, error) {
	sql := "DELETE FROM " + quoteIdent(
		target.Table,
	) + " WHERE " + quoteIdent(
		target.Column,
	) + " = $1"
	return r.exec(ctx, sql, target.Value.String())
}

type referencing struct {
	fk     *foreignKey
	target purgeTarget
	child  tenantTable
	tenant dbscope.Tenant
}

func (p *purgeRun) reference(fk *foreignKey, target purgeTarget) *referencing {
	return &referencing{
		fk:     fk,
		target: target,
		child:  p.plan.tables[fk.ChildTable],
		tenant: p.tenant,
	}
}

func (ref *referencing) childRows(limit int) (query string, args []any) {
	fk := ref.fk
	args = []any{ref.target.Value.String(), limit}
	where := "(" + joinedColumns(fk.ChildColumns, "c") + ") IN (SELECT " +
		joinedColumns(fk.ParentColumns, "p") + " FROM " + quoteIdent(fk.ParentTable) +
		" p WHERE p." + quoteIdent(ref.target.Column) + " = $1)"
	if own := ref.child.belongs("c"); own != "" {
		where += " AND " + strings.NewReplacer("$1", "$3", "$2", "$4").Replace(own)
		args = append(args, ref.tenant.OrganizationID.String(), ref.tenant.BusinessUnitID.String())
		if !strings.Contains(own, "$1") {
			where += " AND $3::text IS NOT NULL"
		}
	}
	return "SELECT c.tableoid, c.ctid FROM " + quoteIdent(fk.ChildTable) + " c WHERE " + where +
		" LIMIT $2", args
}

func (r *repository) deleteReferencing(
	ctx context.Context,
	ref *referencing,
	limit int,
) (int64, error) {
	fk := ref.fk
	if len(fk.ChildColumns) == 0 || len(fk.ChildColumns) != len(fk.ParentColumns) {
		return 0, nil
	}
	rows, args := ref.childRows(limit)
	sql := "DELETE FROM " + quoteIdent(fk.ChildTable) + " WHERE (tableoid, ctid) IN (" + rows + ")"
	return r.exec(ctx, sql, args...)
}

func (r *repository) nullReferencing(
	ctx context.Context,
	ref *referencing,
	limit int,
) (int64, error) {
	fk := ref.fk
	own := make([]string, 0, len(fk.ChildColumns))
	for _, c := range fk.ChildColumns {
		if c != organizationColumn && c != businessUnitColumn {
			own = append(own, quoteIdent(c)+" = NULL")
		}
	}
	if len(own) == 0 || len(fk.ChildColumns) != len(fk.ParentColumns) {
		return 0, nil
	}
	rows, args := ref.childRows(limit)
	sql := "UPDATE " + quoteIdent(fk.ChildTable) + " SET " + strings.Join(own, ", ") +
		" WHERE (tableoid, ctid) IN (" + rows + ")"
	return r.exec(ctx, sql, args...)
}

func (r *repository) exec(ctx context.Context, sql string, args ...any) (int64, error) {
	ctx = dbscope.WithSystem(ctx, purgeReason)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		db := r.db.Keel(ctx)
		if err := bound(ctx, db); err != nil {
			return 0, err
		}
		return keel.Exec(ctx, db, sql, args...)
	})
}

func (r *repository) DeleteOrganization(
	ctx context.Context,
	tenant dbscope.Tenant,
) (*cloudtenantports.DeleteResult, error) {
	if !tenant.Valid() {
		return nil, ErrTenantRequired
	}
	if err := r.refuseInbound(ctx, tenant); err != nil {
		return nil, err
	}
	ctx = dbscope.WithSystem(ctx, organizationReason)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*cloudtenantports.DeleteResult, error) {
		db := r.db.Keel(ctx)
		if err := bound(ctx, db); err != nil {
			return nil, err
		}
		scope := tenant.OrganizationID.String()
		fs, fc, us := managedfeaturesdb.States, managedfeaturesdb.Changes, managedusagedb.States
		if _, err := fc.Where(fc.ScopeKey.Eq(scope)).Delete(ctx, db); err != nil {
			return nil, fmt.Errorf("delete the tenant's plan history: %w", err)
		}
		if _, err := fs.Where(fs.ScopeKey.Eq(scope)).Delete(ctx, db); err != nil {
			return nil, fmt.Errorf("delete the tenant's plan: %w", err)
		}
		if _, err := us.Where(us.ScopeKey.Eq(scope)).Delete(ctx, db); err != nil {
			return nil, fmt.Errorf("delete the tenant's usage state: %w", err)
		}
		result := new(cloudtenantports.DeleteResult)
		err := dbtx.Savepoint(ctx, r.db, func(ctx context.Context) error {
			o := tenantdb.Organizations
			_, derr := o.Where(
				o.ID.Eq(tenant.OrganizationID),
				o.BusinessUnitID.Eq(tenant.BusinessUnitID),
			).AllTenants().Delete(ctx, r.db.Keel(ctx))
			return derr
		})
		switch {
		case err == nil:
			result.Deleted = true
			return result, nil
		case referenceViolation(err) || protectedByRule(err):
			result.BlockedBy = r.blocked("organization", tenant, err)
			return result, nil
		default:
			return nil, fmt.Errorf("delete the tenant's organization: %w", err)
		}
	})
}

func (r *repository) DeleteBusinessUnit(
	ctx context.Context,
	tenant dbscope.Tenant,
) (*cloudtenantports.DeleteResult, error) {
	if !tenant.Valid() {
		return nil, ErrTenantRequired
	}
	if err := r.refuseInbound(ctx, tenant); err != nil {
		return nil, err
	}
	ctx = dbscope.WithSystem(ctx, unitReason)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*cloudtenantports.DeleteResult, error) {
		db := r.db.Keel(ctx)
		if err := bound(ctx, db); err != nil {
			return nil, err
		}
		result := new(cloudtenantports.DeleteResult)
		o := tenantdb.Organizations
		others, err := o.Where(o.BusinessUnitID.Eq(tenant.BusinessUnitID)).
			AllTenants().
			Count(ctx, db)
		if err != nil {
			return nil, fmt.Errorf("count the business unit's organizations: %w", err)
		}
		if others > 0 {
			result.BlockedBy = "organizations"
			return result, nil
		}
		err = dbtx.Savepoint(ctx, r.db, func(ctx context.Context) error {
			b := tenantdb.BusinessUnits
			_, derr := b.Where(b.ID.Eq(tenant.BusinessUnitID)).Delete(ctx, r.db.Keel(ctx))
			return derr
		})
		switch {
		case err == nil:
			result.Deleted = true
			return result, nil
		case referenceViolation(err) || protectedByRule(err):
			result.BlockedBy = r.blocked("business unit", tenant, err)
			return result, nil
		default:
			return nil, fmt.Errorf("delete the tenant's business unit: %w", err)
		}
	})
}

func (r *repository) blocked(what string, tenant dbscope.Tenant, err error) string {
	by := dberror.ExtractConstraintName(err)
	if by == "" {
		var refusal *pgconn.PgError
		if errors.As(err, &refusal) {
			by = refusal.Message
		}
	}
	r.l.Warn(
		"the database refuses to delete a removed tenant's "+what,
		zap.String("organizationId", tenant.OrganizationID.String()),
		zap.String("businessUnitId", tenant.BusinessUnitID.String()),
		zap.String("blockedBy", by),
		zap.Error(err),
	)
	return by
}

func (r *repository) Remaining(
	ctx context.Context,
	tenant dbscope.Tenant,
) ([]cloudtenantports.TableCount, error) {
	if !tenant.Valid() {
		return nil, ErrTenantRequired
	}
	ctx = dbscope.WithSystem(ctx, countReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]cloudtenantports.TableCount, error) {
		db := r.db.Keel(ctx)
		tables, err := keel.QueryAll(
			ctx,
			db,
			pgx.RowToStructByName[tenantTable],
			catalogTablesQuery,
		)
		if err != nil {
			return nil, fmt.Errorf("read the tenant tables: %w", err)
		}
		counts, err := keel.QueryAll(
			ctx,
			db,
			pgx.RowToStructByName[cloudtenantports.TableCount],
			RemainingQuery(tables),
			tenant.OrganizationID.String(),
			tenant.BusinessUnitID.String(),
		)
		if err != nil {
			return nil, fmt.Errorf("count what is left of the tenant: %w", err)
		}
		return counts, nil
	})
}

func RemainingQuery(tables []tenantTable) string {
	parts := []string{
		"SELECT 'organizations'::text AS table_name, count(*)::bigint AS row_count FROM \"organizations\" WHERE \"id\" = $1",
		"SELECT 'business_units'::text, count(*)::bigint FROM \"business_units\" WHERE \"id\" = $2",
	}
	for _, t := range tables {
		if Kept(t.Name) {
			continue
		}
		if where := tenantFilter(t, true); where != "" {
			parts = append(parts, "SELECT "+literal(t.Name)+"::text, count(*)::bigint FROM "+
				quoteIdent(t.Name)+" WHERE "+where)
		}
	}
	return strings.Join(parts, "\nUNION ALL\n")
}

func HoldingQuery(tables []tenantTable, exclusiveBU bool) string {
	parts := make([]string, 0, len(tables))
	for _, t := range tables {
		if Kept(t.Name) {
			continue
		}
		if where := tenantFilter(t, exclusiveBU); where != "" {
			parts = append(parts, "SELECT "+literal(t.Name)+"::text WHERE EXISTS (SELECT 1 FROM "+
				quoteIdent(t.Name)+" WHERE "+where+")")
		}
	}
	parts = append(parts, "SELECT NULL::text WHERE $1::text IS NULL AND $2::text IS NULL")
	return strings.Join(parts, "\nUNION ALL\n")
}

func tenantFilter(t tenantTable, exclusiveBU bool) string {
	switch {
	case t.HasOrg:
		return "\"organization_id\" = $1"
	case t.HasBU && exclusiveBU:
		return "\"business_unit_id\" = $2"
	case t.PartyBU != "" && exclusiveBU:
		return quoteIdent(t.PartyBU) + " = $2"
	default:
		return ""
	}
}

func literal(name string) string {
	return "'" + strings.ReplaceAll(name, "'", "''") + "'"
}

func holdingTables(tables []tenantTable, holding []string) []tenantTable {
	held := make(map[string]struct{}, len(holding))
	for _, name := range holding {
		held[name] = struct{}{}
	}
	out := make([]tenantTable, 0, len(holding))
	for _, t := range tables {
		if _, ok := held[t.Name]; ok {
			out = append(out, t)
		}
	}
	return out
}

func referenceViolation(err error) bool {
	code := dberror.ExtractCode(err)
	return code == pgerrcode.ForeignKeyViolation || code == pgerrcode.RestrictViolation
}

func protectedByRule(err error) bool {
	switch dberror.ExtractCode(err) {
	case pgerrcode.InsufficientPrivilege,
		pgerrcode.RaiseException,
		pgerrcode.IntegrityConstraintViolation:
		return true
	default:
		return false
	}
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
