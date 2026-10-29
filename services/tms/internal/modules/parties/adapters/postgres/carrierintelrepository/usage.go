package carrierintelrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelusage"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelusage/carrierintelusagedb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	defaultUsageDeleteBatchSize = 5000
	maxUsageDeleteBatchSize     = 50000
)

type usageRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewUsageRepository(p Params) carrierintelports.CarrierIntelUsageRepository {
	return &usageRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-intel-usage-repository"),
	}
}

func (r *usageRepository) Insert(
	ctx context.Context,
	entity *carrierintelusage.CarrierIntelUsageRecord,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		inserted, err := carrierintelusagedb.CarrierIntelUsageRecords.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			[]*carrierintelusage.CarrierIntelUsageRecord{entity},
		)
		if err != nil {
			r.l.Error("failed to insert carrier intel usage record", zap.Error(err))
			return false, fmt.Errorf("insert carrier intel usage record: %w", err)
		}

		return inserted == 1, nil
	})
}

func (r *usageRepository) ExistsDedupeKey(
	ctx context.Context,
	_ pagination.TenantInfo,
	provider integration.Type,
	dedupeKey string,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		u := carrierintelusagedb.CarrierIntelUsageRecords
		exists, err := u.Where(
			u.Provider.Eq(provider),
			u.Billable.IsTrue(),
			u.DedupeKey.Eq(dedupeKey),
		).Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to check carrier intel usage dedupe key", zap.Error(err))
			return false, fmt.Errorf("check carrier intel usage dedupe key: %w", err)
		}

		return exists, nil
	})
}

func (r *usageRepository) CostSince(
	ctx context.Context,
	_ pagination.TenantInfo,
	since int64,
) (decimal.Decimal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (decimal.Decimal, error) {
		u := carrierintelusagedb.CarrierIntelUsageRecords
		total, err := keel.IntoOne[decimal.Decimal](
			ctx,
			r.db.Keel(ctx),
			u.Pick(keel.NumericOf(u.EstimatedCost.Sum()).Coalesce(0)).
				Where(u.CreatedAt.Gte(since)),
		)
		if err != nil {
			r.l.Error("failed to sum carrier intel usage cost", zap.Error(err))
			return decimal.Zero, fmt.Errorf("sum carrier intel usage cost: %w", err)
		}

		return total, nil
	})
}

func (r *usageRepository) CountBillableSince(
	ctx context.Context,
	req *carrierintelports.CountBillableUsageRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		u := carrierintelusagedb.CarrierIntelUsageRecords
		count, err := u.Where(
			u.Provider.Eq(req.Provider),
			u.Endpoint.Eq(req.Endpoint),
			u.Billable.IsTrue(),
			u.CreatedAt.Gte(req.Since),
		).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count billable carrier intel usage", zap.Error(err))
			return 0, fmt.Errorf("count billable carrier intel usage: %w", err)
		}

		return int(count), nil
	})
}

func (r *usageRepository) Summary(
	ctx context.Context,
	req *carrierintelports.ListCarrierIntelUsageRequest,
) ([]carrierintelports.CarrierIntelUsageSummaryRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]carrierintelports.CarrierIntelUsageSummaryRow, error) {
			u := carrierintelusagedb.CarrierIntelUsageRecords
			q := u.Pick(u.Provider, u.Endpoint, keel.Count(), u.BillableUnits.Sum(), u.EstimatedCost.Sum()).
				Where(
					u.CreatedAt.Gte(req.Since),
					keel.When(req.Until > 0, u.CreatedAt.Lt(req.Until)),
				).
				Group(u.Provider, u.Endpoint).
				Order(u.Provider.Asc(), u.Endpoint.Asc())

			rows, err := keel.Into[carrierintelports.CarrierIntelUsageSummaryRow](
				ctx,
				r.db.Keel(ctx),
				q,
			)
			if err != nil {
				r.l.Error("failed to summarize carrier intel usage", zap.Error(err))
				return nil, fmt.Errorf("summarize carrier intel usage: %w", err)
			}
			if rows == nil {
				rows = []carrierintelports.CarrierIntelUsageSummaryRow{}
			}

			return rows, nil
		},
	)
}

func (r *usageRepository) ListDaily(
	ctx context.Context,
	req *carrierintelports.ListCarrierIntelUsageDailyRequest,
) ([]*carrierintelusage.CarrierIntelUsageDaily, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carrierintelusage.CarrierIntelUsageDaily, error) {
			d := carrierintelusagedb.CarrierIntelUsageDailies
			entities, err := d.Where(d.Day.Between(int64(req.FromDay), int64(req.ToDay))).
				Order(d.Day.Asc(), d.Provider.Asc(), d.Endpoint.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list carrier intel daily usage", zap.Error(err))
				return nil, fmt.Errorf("list carrier intel daily usage: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *usageRepository) RollupDay(ctx context.Context, dayStart int64) (int, error) {
	ctx = dbscope.WithSystem(ctx, "roll up carrier intelligence usage across every organization")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		start := timeutils.DayStartUTC(dayStart)
		l := carrierintelusagedb.CarrierIntelUsageRecords
		d := carrierintelusagedb.CarrierIntelUsageDailies
		day := keel.Int64(int64(timeutils.DayKeyUTC(start)))
		affected, err := d.UpsertFrom(
			ctx,
			r.db.Keel(ctx),
			l.Pick(
				l.OrganizationID,
				l.BusinessUnitID,
				l.Provider,
				l.Endpoint,
				day,
				keel.Count(),
				keel.Int64Of(l.BillableUnits.Sum()).Coalesce(0),
				keel.NumericOf(l.EstimatedCost.Sum()).Coalesce(0),
			).
				Where(l.CreatedAt.Gte(start), l.CreatedAt.Lt(start+timeutils.SecondsPerDay)).
				Group(l.OrganizationID, l.BusinessUnitID, l.Provider, l.Endpoint),
			keel.OnConflict(d.OrganizationID, d.BusinessUnitID, d.Provider, d.Endpoint, d.Day).
				Update(d.Calls, d.BillableUnits, d.EstimatedCost),
			d.OrganizationID,
			d.BusinessUnitID,
			d.Provider,
			d.Endpoint,
			d.Day,
			d.Calls,
			d.BillableUnits,
			d.EstimatedCost,
		)
		if err != nil {
			r.l.Error("failed to roll up carrier intel usage", zap.Error(err))
			return 0, fmt.Errorf("roll up carrier intel usage: %w", err)
		}

		return int(affected), nil
	})
}

func (r *usageRepository) DeleteOlderThan(
	ctx context.Context,
	before int64,
	limit int,
) (int, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"delete carrier intelligence usage past retention across every organization",
	)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		u := carrierintelusagedb.CarrierIntelUsageRecords
		batch := intutils.Clamp(
			intutils.WithDefault(max(limit, 0), defaultUsageDeleteBatchSize),
			1,
			maxUsageDeleteBatchSize,
		)

		s := u.As("stale")
		stale := s.AllTenants().
			Pick(s.ID).
			Where(s.CreatedAt.Lt(before)).
			Order(s.CreatedAt.Asc()).
			Limit(batch)

		affected, err := u.AllTenants().
			Where(u.ID.InQuery(stale), u.CreatedAt.Lt(before)).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to delete old carrier intel usage records", zap.Error(err))
			return 0, fmt.Errorf("delete old carrier intel usage records: %w", err)
		}

		return int(affected), nil
	})
}
