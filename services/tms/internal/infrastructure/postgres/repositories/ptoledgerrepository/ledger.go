package ptoledgerrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.PTOLedgerRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.pto-ledger-repository"),
	}
}

func balanceScope(
	sq *bun.SelectQuery,
	key *repositories.PTOBalanceKey,
) *bun.SelectQuery {
	cols := buncolgen.WorkerPTOBalanceColumns
	return buncolgen.WorkerPTOBalanceScopeTenant(sq, key.TenantInfo).
		Where(cols.WorkerID.Eq(), key.WorkerID).
		Where(cols.PTOType.Eq(), key.PTOType)
}

func (r *repository) EnsureBalance(ctx context.Context, key *repositories.PTOBalanceKey) error {
	entity := &worker.WorkerPTOBalance{
		OrganizationID: key.TenantInfo.OrgID,
		BusinessUnitID: key.TenantInfo.BuID,
		WorkerID:       key.WorkerID,
		PTOType:        key.PTOType,
	}

	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		On("CONFLICT (organization_id, business_unit_id, worker_id, pto_type) DO NOTHING").
		Exec(ctx); err != nil {
		r.l.Error("failed to ensure PTO balance row", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) LockBalance(
	ctx context.Context,
	key *repositories.PTOBalanceKey,
) (*worker.WorkerPTOBalance, error) {
	entity := new(worker.WorkerPTOBalance)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return balanceScope(sq, key)
		}).
		For("UPDATE").
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerPTOBalance")
	}

	return entity, nil
}

func (r *repository) GetBalance(
	ctx context.Context,
	key *repositories.PTOBalanceKey,
) (*worker.WorkerPTOBalance, error) {
	entity := new(worker.WorkerPTOBalance)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return balanceScope(sq, key)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerPTOBalance")
	}

	return entity, nil
}

func (r *repository) ListBalances(
	ctx context.Context,
	req *repositories.ListPTOBalancesRequest,
) ([]*worker.WorkerPTOBalance, error) {
	cols := buncolgen.WorkerPTOBalanceColumns
	entities := make([]*worker.WorkerPTOBalance, 0, 4)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerPTOBalanceScopeTenant(sq, req.TenantInfo).
				Where(cols.WorkerID.Eq(), req.WorkerID)
		}).
		Order(cols.PTOType.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list PTO balances", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) ListOrgBalances(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*worker.WorkerPTOBalance, error) {
	cols := buncolgen.WorkerPTOBalanceColumns
	entities := make([]*worker.WorkerPTOBalance, 0, 64)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.WorkerPTOBalanceRelations.Worker).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerPTOBalanceScopeTenant(sq, tenantInfo)
		}).
		Order(cols.WorkerID.OrderAsc()).
		Order(cols.PTOType.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list organisation PTO balances", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) UpdateBalance(
	ctx context.Context,
	entity *worker.WorkerPTOBalance,
) (*worker.WorkerPTOBalance, error) {
	ov := entity.Version
	entity.Version++

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerPTOBalanceColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update PTO balance", zap.Error(err))
		return nil, err
	}
	if err = dberror.CheckRowsAffected(res, "WorkerPTOBalance", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) InsertEntry(
	ctx context.Context,
	entry *worker.WorkerPTOLedgerEntry,
) (*worker.WorkerPTOLedgerEntry, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entry).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, repositories.ErrDuplicatePTOLedgerEntry
		}
		r.l.Error("failed to insert PTO ledger entry", zap.Error(err))
		return nil, err
	}

	return entry, nil
}

func (r *repository) applyEntryFilters(
	q *bun.SelectQuery,
	req *repositories.ListPTOLedgerRequest,
) *bun.SelectQuery {
	cols := buncolgen.WorkerPTOLedgerEntryColumns
	if !req.WorkerID.IsNil() {
		q = q.Where(cols.WorkerID.Eq(), req.WorkerID)
	}
	if req.PTOType != "" {
		q = q.Where(cols.PTOType.Eq(), req.PTOType)
	}
	if req.EntryType != "" {
		q = q.Where(cols.EntryType.Eq(), req.EntryType)
	}
	if req.EffectiveFrom > 0 {
		q = q.Where(cols.EffectiveAt.Gte(), req.EffectiveFrom)
	}
	if req.EffectiveTo > 0 {
		q = q.Where(cols.EffectiveAt.Lte(), req.EffectiveTo)
	}
	return q
}

func (r *repository) ListEntries(
	ctx context.Context,
	req *repositories.ListPTOLedgerRequest,
) (*pagination.CursorListResult[*worker.WorkerPTOLedgerEntry], error) {
	log := r.l.With(zap.String("operation", "ListEntries"))

	dba := r.db.DBForContext(ctx)
	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*worker.WorkerPTOLedgerEntry)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.WorkerPTOLedgerEntryTable.Alias,
					req.Filter,
					(*worker.WorkerPTOLedgerEntry)(nil),
				)
				return r.applyEntryFilters(sq, req)
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count PTO ledger entries", zap.Error(err))
			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*worker.WorkerPTOLedgerEntry]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(items *[]*worker.WorkerPTOLedgerEntry) *bun.SelectQuery {
				q := dba.NewSelect().Model(items).
					ColumnExpr(buncolgen.WorkerPTOLedgerEntryTable.All())
				if req.IncludeWorker {
					q = q.Relation(buncolgen.WorkerPTOLedgerEntryRelations.Worker)
				}
				return q
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				sq, applyErr := querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.WorkerPTOLedgerEntryTable.Alias,
					req.Filter,
					req.Cursor,
					(*worker.WorkerPTOLedgerEntry)(nil),
				)
				if applyErr != nil {
					return sq, applyErr
				}
				return r.applyEntryFilters(sq, req), nil
			},
		},
	)
	if err != nil {
		log.Error("failed to list PTO ledger entries", zap.Error(err))
		return nil, err
	}

	return result, nil
}

type sumRow struct {
	Total decimal.Decimal `bun:"total"`
	Count int64           `bun:"entry_count"`
}

func (r *repository) SumEntries(
	ctx context.Context,
	key *repositories.PTOBalanceKey,
) (decimal.Decimal, int64, error) {
	cols := buncolgen.WorkerPTOLedgerEntryColumns
	row := new(sumRow)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerPTOLedgerEntry)(nil)).
		ColumnExpr("COALESCE(SUM("+cols.AmountDays.String()+"), 0) AS total").
		ColumnExpr("COUNT(*) AS entry_count").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerPTOLedgerEntryScopeTenant(sq, key.TenantInfo).
				Where(cols.WorkerID.Eq(), key.WorkerID).
				Where(cols.PTOType.Eq(), key.PTOType)
		}).
		Scan(ctx, row)
	if err != nil {
		r.l.Error("failed to sum PTO ledger entries", zap.Error(err))
		return decimal.Zero, 0, err
	}

	return row.Total, row.Count, nil
}

type pendingRow struct {
	Total decimal.Decimal `bun:"total"`
}

func (r *repository) PendingDays(
	ctx context.Context,
	req *repositories.PendingPTODaysRequest,
) (decimal.Decimal, error) {
	cols := buncolgen.WorkerPTOColumns
	row := new(pendingRow)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerPTO)(nil)).
		ColumnExpr("COALESCE(SUM("+cols.Days.String()+"), 0) AS total").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerPTOScopeTenant(sq, req.TenantInfo).
				Where(cols.WorkerID.Eq(), req.WorkerID).
				Where(cols.Type.Eq(), req.PTOType).
				Where(cols.Status.Eq(), worker.PTOStatusRequested)
		})
	if !req.ExcludeID.IsNil() {
		q = q.Where(cols.ID.Ne(), req.ExcludeID)
	}

	if err := q.Scan(ctx, row); err != nil {
		r.l.Error("failed to sum pending PTO days", zap.Error(err))
		return decimal.Zero, err
	}

	return row.Total, nil
}

func (r *repository) HasEntry(
	ctx context.Context,
	req *repositories.HasPTOLedgerEntryRequest,
) (bool, error) {
	cols := buncolgen.WorkerPTOLedgerEntryColumns
	exists, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerPTOLedgerEntry)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerPTOLedgerEntryScopeTenant(sq, req.TenantInfo).
				Where(cols.SourcePTOID.Eq(), req.SourcePTOID).
				Where(cols.EntryType.Eq(), req.EntryType)
		}).
		Exists(ctx)
	if err != nil {
		r.l.Error("failed to check PTO ledger entry", zap.Error(err))
		return false, err
	}

	return exists, nil
}

type summaryRow struct {
	WorkersTracked int64           `bun:"workers_tracked"`
	TotalBalance   decimal.Decimal `bun:"total_balance"`
	AccruedYTD     decimal.Decimal `bun:"accrued_ytd"`
	UsedYTD        decimal.Decimal `bun:"used_ytd"`
}

type countRow struct {
	Count int64 `bun:"count"`
}

func (r *repository) Summary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*repositories.PTOBalanceSummary, error) {
	dba := r.db.DBForContext(ctx)
	bal := buncolgen.WorkerPTOBalanceColumns

	balances := new(summaryRow)
	if err := dba.NewSelect().
		Model((*worker.WorkerPTOBalance)(nil)).
		ColumnExpr("COUNT(DISTINCT "+bal.WorkerID.String()+") AS workers_tracked").
		ColumnExpr("COALESCE(SUM("+bal.BalanceDays.String()+"), 0) AS total_balance").
		ColumnExpr("COALESCE(SUM("+bal.AccruedYTDDays.String()+"), 0) AS accrued_ytd").
		ColumnExpr("COALESCE(SUM("+bal.UsedYTDDays.String()+"), 0) AS used_ytd").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerPTOBalanceScopeTenant(sq, tenantInfo)
		}).
		Scan(ctx, balances); err != nil {
		r.l.Error("failed to summarise PTO balances", zap.Error(err))
		return nil, err
	}

	pto := buncolgen.WorkerPTOColumns
	pending := new(pendingRow)
	if err := dba.NewSelect().
		Model((*worker.WorkerPTO)(nil)).
		ColumnExpr("COALESCE(SUM("+pto.Days.String()+"), 0) AS total").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerPTOScopeTenant(sq, tenantInfo).
				Where(pto.Status.Eq(), worker.PTOStatusRequested)
		}).
		Scan(ctx, pending); err != nil {
		r.l.Error("failed to summarise pending PTO", zap.Error(err))
		return nil, err
	}

	wrk := buncolgen.WorkerColumns
	asg := buncolgen.WorkerPTOPolicyAssignmentColumns
	unassigned := new(countRow)
	if err := dba.NewSelect().
		Model((*worker.Worker)(nil)).
		ColumnExpr("COUNT(*) AS count").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerScopeTenant(sq, tenantInfo).
				Where(wrk.Status.Eq(), "Active").
				Where("NOT EXISTS (SELECT 1 FROM " + buncolgen.WorkerPTOPolicyAssignmentTable.Name +
					" AS " + buncolgen.WorkerPTOPolicyAssignmentTable.Alias +
					" WHERE " + asg.WorkerID.String() + " = " + wrk.ID.String() +
					" AND " + asg.OrganizationID.String() + " = " + wrk.OrganizationID.String() +
					" AND " + asg.BusinessUnitID.String() + " = " + wrk.BusinessUnitID.String() +
					" AND " + asg.EffectiveTo.String() + " IS NULL)")
		}).
		Scan(ctx, unassigned); err != nil {
		r.l.Error("failed to count unassigned workers", zap.Error(err))
		return nil, err
	}

	return &repositories.PTOBalanceSummary{
		WorkersTracked:    int(balances.WorkersTracked),
		WorkersUnassigned: int(unassigned.Count),
		TotalBalanceDays:  balances.TotalBalance,
		TotalPendingDays:  pending.Total,
		AccruedYTDDays:    balances.AccruedYTD,
		UsedYTDDays:       balances.UsedYTD,
	}, nil
}
