package carrierintelrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	defaultUsageDeleteBatchSize = 5000
	maxUsageDeleteBatchSize     = 50000
	usageSummaryCapacity        = 16
	usageDailyCapacity          = 64
	usageDailyConflictClause    = "ON CONFLICT (organization_id, business_unit_id, provider, endpoint, day) DO UPDATE SET"
)

type usageRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewUsageRepository(p Params) repositories.CarrierIntelUsageRepository {
	return &usageRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-intel-usage-repository"),
	}
}

func (r *usageRepository) Insert(
	ctx context.Context,
	entity *carrierintel.CarrierIntelUsageRecord,
) (bool, error) {
	result, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		On("CONFLICT DO NOTHING").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to insert carrier intel usage record", zap.Error(err))
		return false, fmt.Errorf("insert carrier intel usage record: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("insert carrier intel usage record rows affected: %w", err)
	}

	return affected == 1, nil
}

func (r *usageRepository) ExistsDedupeKey(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider integration.Type,
	dedupeKey string,
) (bool, error) {
	cols := buncolgen.CarrierIntelUsageRecordColumns
	exists, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*carrierintel.CarrierIntelUsageRecord)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelUsageRecordScopeTenant(sq, tenantInfo).
				Where(cols.Provider.Eq(), provider).
				Where(cols.Billable.IsTrue()).
				Where(cols.DedupeKey.Eq(), dedupeKey)
		}).
		Exists(ctx)
	if err != nil {
		r.l.Error("failed to check carrier intel usage dedupe key", zap.Error(err))
		return false, fmt.Errorf("check carrier intel usage dedupe key: %w", err)
	}

	return exists, nil
}

func (r *usageRepository) CostSince(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	since int64,
) (decimal.Decimal, error) {
	cols := buncolgen.CarrierIntelUsageRecordColumns
	var total decimal.Decimal
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*carrierintel.CarrierIntelUsageRecord)(nil)).
		ColumnExpr(cols.EstimatedCost.Expr("COALESCE(SUM({}), 0) AS ?"), bun.Ident(cols.EstimatedCost.Bare())).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelUsageRecordScopeTenant(sq, tenantInfo).
				Where(cols.CreatedAt.Gte(), since)
		}).
		Scan(ctx, &total)
	if err != nil {
		r.l.Error("failed to sum carrier intel usage cost", zap.Error(err))
		return decimal.Zero, fmt.Errorf("sum carrier intel usage cost: %w", err)
	}

	return total, nil
}

func (r *usageRepository) CountBillableSince(
	ctx context.Context,
	req *repositories.CountBillableUsageRequest,
) (int, error) {
	cols := buncolgen.CarrierIntelUsageRecordColumns
	count, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*carrierintel.CarrierIntelUsageRecord)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelUsageRecordScopeTenant(sq, req.TenantInfo).
				Where(cols.Provider.Eq(), req.Provider).
				Where(cols.Endpoint.Eq(), req.Endpoint).
				Where(cols.Billable.IsTrue()).
				Where(cols.CreatedAt.Gte(), req.Since)
		}).
		Count(ctx)
	if err != nil {
		r.l.Error("failed to count billable carrier intel usage", zap.Error(err))
		return 0, fmt.Errorf("count billable carrier intel usage: %w", err)
	}

	return count, nil
}

func (r *usageRepository) Summary(
	ctx context.Context,
	req *repositories.ListCarrierIntelUsageRequest,
) ([]repositories.CarrierIntelUsageSummaryRow, error) {
	cols := buncolgen.CarrierIntelUsageRecordColumns
	daily := buncolgen.CarrierIntelUsageDailyColumns
	rows := make([]repositories.CarrierIntelUsageSummaryRow, 0, usageSummaryCapacity)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*carrierintel.CarrierIntelUsageRecord)(nil)).
		Column(cols.Provider.Bare(), cols.Endpoint.Bare()).
		ColumnExpr(buncolgen.Count(daily.Calls.Bare())).
		ColumnExpr(buncolgen.Sum(cols.BillableUnits, daily.BillableUnits.Bare())).
		ColumnExpr(buncolgen.Sum(cols.EstimatedCost, daily.EstimatedCost.Bare())).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelUsageRecordScopeTenant(sq, req.TenantInfo).
				Where(cols.CreatedAt.Gte(), req.Since)
		}).
		Group(cols.Provider.Qualified(), cols.Endpoint.Qualified()).
		Order(cols.Provider.OrderAsc(), cols.Endpoint.OrderAsc())
	if req.Until > 0 {
		q = q.Where(cols.CreatedAt.Lt(), req.Until)
	}

	if err := q.Scan(ctx, &rows); err != nil {
		r.l.Error("failed to summarize carrier intel usage", zap.Error(err))
		return nil, fmt.Errorf("summarize carrier intel usage: %w", err)
	}

	return rows, nil
}

func (r *usageRepository) ListDaily(
	ctx context.Context,
	req *repositories.ListCarrierIntelUsageDailyRequest,
) ([]*carrierintel.CarrierIntelUsageDaily, error) {
	cols := buncolgen.CarrierIntelUsageDailyColumns
	entities := make([]*carrierintel.CarrierIntelUsageDaily, 0, usageDailyCapacity)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelUsageDailyScopeTenant(sq, req.TenantInfo).
				Where(cols.Day.Between(), req.FromDay, req.ToDay)
		}).
		Order(cols.Day.OrderAsc(), cols.Provider.OrderAsc(), cols.Endpoint.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list carrier intel daily usage", zap.Error(err))
		return nil, fmt.Errorf("list carrier intel daily usage: %w", err)
	}

	return entities, nil
}

func buildUsageRollup(db bun.IDB, dayStart int64) *bun.RawQuery {
	cols := buncolgen.CarrierIntelUsageRecordColumns
	daily := buncolgen.CarrierIntelUsageDailyColumns

	aggregate := db.NewSelect().
		Model((*carrierintel.CarrierIntelUsageRecord)(nil)).
		Column(
			cols.OrganizationID.Bare(),
			cols.BusinessUnitID.Bare(),
			cols.Provider.Bare(),
			cols.Endpoint.Bare(),
		).
		ColumnExpr("?", timeutils.DayKeyUTC(dayStart)).
		ColumnExpr(buncolgen.Count(daily.Calls.Bare())).
		ColumnExpr(cols.BillableUnits.Expr("COALESCE(SUM({}), 0)")).
		ColumnExpr(cols.EstimatedCost.Expr("COALESCE(SUM({}), 0)")).
		Where(cols.CreatedAt.Gte(), dayStart).
		Where(cols.CreatedAt.Lt(), dayStart+timeutils.SecondsPerDay).
		Group(
			cols.OrganizationID.Qualified(),
			cols.BusinessUnitID.Qualified(),
			cols.Provider.Qualified(),
			cols.Endpoint.Qualified(),
		)

	return db.NewRaw(
		"INSERT INTO ? (?, ?, ?, ?, ?, ?, ?, ?) ? "+usageDailyConflictClause+" ?, ?, ?",
		bun.Ident(buncolgen.CarrierIntelUsageDailyTable.Name),
		bun.Ident(daily.OrganizationID.Bare()),
		bun.Ident(daily.BusinessUnitID.Bare()),
		bun.Ident(daily.Provider.Bare()),
		bun.Ident(daily.Endpoint.Bare()),
		bun.Ident(daily.Day.Bare()),
		bun.Ident(daily.Calls.Bare()),
		bun.Ident(daily.BillableUnits.Bare()),
		bun.Ident(daily.EstimatedCost.Bare()),
		aggregate,
		bun.Safe(daily.Calls.SetExcluded()),
		bun.Safe(daily.BillableUnits.SetExcluded()),
		bun.Safe(daily.EstimatedCost.SetExcluded()),
	)
}

func (r *usageRepository) RollupDay(ctx context.Context, dayStart int64) (int, error) {
	result, err := buildUsageRollup(r.db.DBForContext(ctx), timeutils.DayStartUTC(dayStart)).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to roll up carrier intel usage", zap.Error(err))
		return 0, fmt.Errorf("roll up carrier intel usage: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("roll up carrier intel usage rows affected: %w", err)
	}

	return int(affected), nil
}

func (r *usageRepository) DeleteOlderThan(
	ctx context.Context,
	before int64,
	limit int,
) (int, error) {
	cols := buncolgen.CarrierIntelUsageRecordColumns
	batch := intutils.Clamp(
		intutils.WithDefault(max(limit, 0), defaultUsageDeleteBatchSize),
		1,
		maxUsageDeleteBatchSize,
	)

	dba := r.db.DBForContext(ctx)
	stale := dba.NewSelect().
		Model((*carrierintel.CarrierIntelUsageRecord)(nil)).
		Column(cols.ID.Bare()).
		Where(cols.CreatedAt.Lt(), before).
		Order(cols.CreatedAt.OrderAsc()).
		Limit(batch)

	result, err := dba.NewDelete().
		Model((*carrierintel.CarrierIntelUsageRecord)(nil)).
		Where(cols.ID.In(), stale).
		Where(cols.CreatedAt.Lt(), before).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to delete old carrier intel usage records", zap.Error(err))
		return 0, fmt.Errorf("delete old carrier intel usage records: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("delete old carrier intel usage records rows affected: %w", err)
	}

	return int(affected), nil
}
