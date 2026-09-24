package agentextensionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agentextension"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const usageConflict = "CONFLICT (organization_id, business_unit_id, extension_type, day) DO UPDATE"

type usageRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewUsageRepository(p Params) repositories.AgentExtensionUsageRepository {
	return &usageRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-extension-usage-repository"),
	}
}

func buildReserve(
	db bun.IDB,
	params repositories.ReserveExtensionRequestParams,
) *bun.InsertQuery {
	cols := buncolgen.ExtensionUsageDailyColumns

	return db.NewInsert().
		Model(&agentextension.ExtensionUsageDaily{
			OrganizationID: params.TenantInfo.OrgID,
			BusinessUnitID: params.TenantInfo.BuID,
			ExtensionType:  params.Type,
			Day:            params.Day,
			Requests:       1,
			CostUSD:        decimal.Zero,
			UpdatedAt:      params.Now,
		}).
		On(usageConflict).
		Set(cols.Requests.IncConflict(1)).
		Set(cols.UpdatedAt.SetExcluded()).
		Where(cols.Requests.Lt(), params.Limit)
}

func (r *usageRepository) Reserve(
	ctx context.Context,
	params repositories.ReserveExtensionRequestParams,
) (bool, error) {
	result, err := buildReserve(r.db.DBForContext(ctx), params).Exec(ctx)
	if err != nil {
		r.l.Error("failed to reserve agent extension request", zap.Error(err))
		return false, fmt.Errorf("reserve agent extension request: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("reserve agent extension request rows affected: %w", err)
	}

	return affected == 1, nil
}

func buildRecordOutcome(
	db bun.IDB,
	params repositories.RecordExtensionOutcomeParams,
) *bun.UpdateQuery {
	cols := buncolgen.ExtensionUsageDailyColumns
	failures := 0
	if params.Failed {
		failures = 1
	}
	cost := params.CostUSD
	if cost.IsNegative() {
		cost = decimal.Zero
	}

	return db.NewUpdate().
		Model((*agentextension.ExtensionUsageDaily)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ExtensionUsageDailyScopeTenantUpdate(uq, params.TenantInfo).
				Where(cols.ExtensionType.Eq(), params.Type).
				Where(cols.Day.Eq(), params.Day)
		}).
		Set(cols.Failures.SetExpr("{} + ?"), failures).
		Set(cols.CostUSD.SetExpr("{} + ?"), cost).
		Set(cols.UpdatedAt.Set(), params.Now)
}

func (r *usageRepository) RecordOutcome(
	ctx context.Context,
	params repositories.RecordExtensionOutcomeParams,
) error {
	if _, err := buildRecordOutcome(r.db.DBForContext(ctx), params).Exec(ctx); err != nil {
		r.l.Error("failed to record agent extension outcome", zap.Error(err))
		return fmt.Errorf("record agent extension outcome: %w", err)
	}

	return nil
}

type usageTotals struct {
	ExtensionType     agentextension.Type `bun:"extension_type"`
	RequestsToday     int                 `bun:"requests_today"`
	RequestsThisMonth int                 `bun:"requests_month"`
	FailuresThisMonth int                 `bun:"failures_month"`
	CostThisMonth     decimal.Decimal     `bun:"cost_month"`
}

func buildSummarize(
	db bun.IDB,
	params repositories.SummarizeExtensionUsageParams,
) *bun.SelectQuery {
	cols := buncolgen.ExtensionUsageDailyColumns

	return db.NewSelect().
		Model((*agentextension.ExtensionUsageDaily)(nil)).
		ColumnExpr(cols.ExtensionType.Qualified()+" AS extension_type").
		ColumnExpr(
			cols.Requests.Expr("COALESCE(SUM({}) FILTER (WHERE "+cols.Day.Qualified()+" = ?), 0)")+
				" AS requests_today",
			params.Today,
		).
		ColumnExpr(cols.Requests.Expr("COALESCE(SUM({}), 0)")+" AS requests_month").
		ColumnExpr(cols.Failures.Expr("COALESCE(SUM({}), 0)")+" AS failures_month").
		ColumnExpr(cols.CostUSD.Expr("COALESCE(SUM({}), 0)")+" AS cost_month").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ExtensionUsageDailyScopeTenant(sq, params.TenantInfo).
				Where(cols.Day.Gte(), params.FromDay).
				Where(cols.Day.Lte(), params.Today)
		}).
		Group(cols.ExtensionType.Qualified())
}

func (r *usageRepository) Summarize(
	ctx context.Context,
	params repositories.SummarizeExtensionUsageParams,
) (map[agentextension.Type]agentextension.UsageSummary, error) {
	rows := make([]usageTotals, 0, len(agentextension.AllTypes()))
	if err := buildSummarize(r.db.DBForContext(ctx), params).Scan(ctx, &rows); err != nil {
		r.l.Error("failed to summarize agent extension usage", zap.Error(err))
		return nil, fmt.Errorf("summarize agent extension usage: %w", err)
	}

	summaries := make(map[agentextension.Type]agentextension.UsageSummary, len(rows))
	for idx := range rows {
		summaries[rows[idx].ExtensionType] = agentextension.UsageSummary{
			RequestsToday:     rows[idx].RequestsToday,
			RequestsThisMonth: rows[idx].RequestsThisMonth,
			FailuresThisMonth: rows[idx].FailuresThisMonth,
			CostThisMonthUSD:  rows[idx].CostThisMonth,
		}
	}

	return summaries, nil
}
