package aiusagerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
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

func New(p Params) repositories.AIUsageRepository {
	return &repository{db: p.DB, l: p.Logger.Named("repository.aiusage")}
}

func (r *repository) Create(ctx context.Context, record *aiusage.Record) error {
	if _, err := r.db.DB().NewInsert().Model(record).Exec(ctx); err != nil {
		r.l.Error("failed to record ai usage", zap.Error(err))
		return fmt.Errorf("record ai usage: %w", err)
	}

	return nil
}

// totalsRow is the shape both aggregate queries scan into. Percentiles are
// taken over successful calls only, in SQL, because a failure refused in a
// millisecond is not a fast answer and would drag the median down.
type totalsRow struct {
	ProviderID      string  `bun:"provider_id"`
	ProviderName    string  `bun:"provider_name"`
	Model           string  `bun:"model"`
	Calls           int     `bun:"calls"`
	Failed          int     `bun:"failed"`
	InputTokens     int64   `bun:"input_tokens"`
	OutputTokens    int64   `bun:"output_tokens"`
	ReasoningTokens int64   `bun:"reasoning_tokens"`
	CostUSD         string  `bun:"cost_usd"`
	PricedCalls     int     `bun:"priced_calls"`
	LatencyP50      float64 `bun:"latency_p50"`
	LatencyP95      float64 `bun:"latency_p95"`
}

func (row *totalsRow) totals() repositories.AIUsageTotals {
	cost := row.CostUSD
	if cost == "" {
		cost = "0"
	}

	return repositories.AIUsageTotals{
		Calls:           row.Calls,
		Failed:          row.Failed,
		InputTokens:     row.InputTokens,
		OutputTokens:    row.OutputTokens,
		ReasoningTokens: row.ReasoningTokens,
		CostUSD:         cost,
		PricedCalls:     row.PricedCalls,
		LatencyP50:      int64(row.LatencyP50),
		LatencyP95:      int64(row.LatencyP95),
	}
}

const aggregateColumns = `
	COUNT(*) AS calls,
	COUNT(*) FILTER (WHERE NOT aiu.succeeded) AS failed,
	COALESCE(SUM(aiu.input_tokens), 0) AS input_tokens,
	COALESCE(SUM(aiu.output_tokens), 0) AS output_tokens,
	COALESCE(SUM(aiu.reasoning_tokens), 0) AS reasoning_tokens,
	COALESCE(SUM(aiu.cost_usd), 0)::text AS cost_usd,
	COUNT(aiu.cost_usd) AS priced_calls,
	COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY aiu.latency_ms) FILTER (WHERE aiu.succeeded), 0) AS latency_p50,
	COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY aiu.latency_ms) FILTER (WHERE aiu.succeeded), 0) AS latency_p95`

func (r *repository) Summary(
	ctx context.Context,
	req repositories.AIUsageSummaryRequest,
) (*repositories.AIUsageSummary, error) {
	cols := buncolgen.AIUsageRecordColumns
	db := r.db.DBForContext(ctx)

	var total totalsRow
	if err := db.NewSelect().
		Model((*aiusage.Record)(nil)).
		ColumnExpr(aggregateColumns).
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(cols.CreatedAt.Gte(), req.Since).
		Scan(ctx, &total); err != nil {
		return nil, fmt.Errorf("summarise ai usage: %w", err)
	}

	var byProvider []totalsRow
	if err := db.NewSelect().
		Model((*aiusage.Record)(nil)).
		ColumnExpr("COALESCE(aiu.provider_id, '') AS provider_id").
		ColumnExpr("COALESCE(aiprv.name, '') AS provider_name").
		ColumnExpr("aiu.model AS model").
		ColumnExpr(aggregateColumns).
		Join("LEFT JOIN ai_providers AS aiprv ON aiprv.id = aiu.provider_id "+
			"AND aiprv.organization_id = aiu.organization_id "+
			"AND aiprv.business_unit_id = aiu.business_unit_id").
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(cols.CreatedAt.Gte(), req.Since).
		GroupExpr("aiu.provider_id, aiprv.name, aiu.model").
		OrderExpr("calls DESC").
		Scan(ctx, &byProvider); err != nil {
		return nil, fmt.Errorf("summarise ai usage by provider: %w", err)
	}

	summary := &repositories.AIUsageSummary{
		Totals:     total.totals(),
		ByProvider: make([]repositories.AIUsageProviderTotals, 0, len(byProvider)),
	}
	for i := range byProvider {
		row := &byProvider[i]
		slice := repositories.AIUsageProviderTotals{
			ProviderName:  row.ProviderName,
			Model:         row.Model,
			AIUsageTotals: row.totals(),
		}
		if row.ProviderID != "" {
			slice.ProviderID = pulidFrom(row.ProviderID)
		}
		summary.ByProvider = append(summary.ByProvider, slice)
	}

	return summary, nil
}
