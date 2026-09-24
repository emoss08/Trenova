package aiusagerepository

import (
	"context"
	"fmt"
	"github.com/shopspring/decimal"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
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

func New(p Params) repositories.AIUsageRepository {
	return &repository{db: p.DB, l: p.Logger.Named("repository.aiusage")}
}

func (r *repository) Create(ctx context.Context, record *aiusage.AIUsageRecord) error {
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
		Model((*aiusage.AIUsageRecord)(nil)).
		ColumnExpr(aggregateColumns).
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(cols.CreatedAt.Gte(), req.Since).
		Scan(ctx, &total); err != nil {
		return nil, fmt.Errorf("summarise ai usage: %w", err)
	}

	var byProvider []totalsRow
	if err := db.NewSelect().
		Model((*aiusage.AIUsageRecord)(nil)).
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

// RecentFailures lists the newest failed attempts in a window, with the
// provider's own message, so the reason behind a failure count is readable
// where the count is shown.
func (r *repository) RecentFailures(
	ctx context.Context,
	req repositories.AIUsageFailuresRequest,
) ([]repositories.AIUsageFailure, error) {
	cols := buncolgen.AIUsageRecordColumns
	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}

	var rows []failureRow
	if err := r.db.DBForContext(ctx).NewSelect().
		Model((*aiusage.AIUsageRecord)(nil)).
		ColumnExpr("COALESCE(aiu.provider_id, '') AS provider_id").
		ColumnExpr("COALESCE(aiprv.name, '') AS provider_name").
		ColumnExpr("aiu.model AS model").
		ColumnExpr("aiu.task AS task").
		ColumnExpr("COALESCE(aiu.error_class, '') AS error_class").
		ColumnExpr("COALESCE(aiu.error_message, '') AS error_message").
		ColumnExpr("aiu.created_at AS created_at").
		Join("LEFT JOIN ai_providers AS aiprv ON aiprv.id = aiu.provider_id "+
			"AND aiprv.organization_id = aiu.organization_id "+
			"AND aiprv.business_unit_id = aiu.business_unit_id").
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(cols.CreatedAt.Gte(), req.Since).
		Where(cols.Succeeded.Eq(), false).
		OrderExpr("aiu.created_at DESC").
		Limit(limit).
		Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("list ai usage failures: %w", err)
	}

	failures := make([]repositories.AIUsageFailure, 0, len(rows))
	for i := range rows {
		row := &rows[i]
		failure := repositories.AIUsageFailure{
			ProviderName: row.ProviderName,
			Model:        row.Model,
			Task:         row.Task,
			ErrorClass:   row.ErrorClass,
			Message:      row.ErrorMessage,
			At:           row.CreatedAt,
		}
		if row.ProviderID != "" {
			failure.ProviderID = pulidFrom(row.ProviderID)
		}
		failures = append(failures, failure)
	}

	return failures, nil
}

type failureRow struct {
	ProviderID   string `bun:"provider_id"`
	ProviderName string `bun:"provider_name"`
	Model        string `bun:"model"`
	Task         string `bun:"task"`
	ErrorClass   string `bun:"error_class"`
	ErrorMessage string `bun:"error_message"`
	CreatedAt    int64  `bun:"created_at"`
}

// CostByDefinition sums what one agent's calls cost since an instant. Calls
// without a price are counted, not summed: a budget check that read them as
// free would let an agent on an unpriced provider run without end.
func (r *repository) CostByDefinition(
	ctx context.Context,
	req repositories.AIUsageCostRequest,
) (*repositories.AIUsageCost, error) {
	cols := buncolgen.AIUsageRecordColumns

	q := costSums(r.db.DBForContext(ctx).NewSelect()).
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(cols.AgentDefinitionID.Eq(), req.DefinitionID).
		Where(cols.CreatedAt.Gte(), req.Since).
		Where(cols.Surface.NotEq(), aiusage.SurfaceEvaluation)

	return scanCost(ctx, q, "ai usage cost")
}

type costRow struct {
	CostUSD       string `bun:"cost_usd"`
	Calls         int    `bun:"calls"`
	UnpricedCalls int    `bun:"unpriced_calls"`
}

func costSums(q *bun.SelectQuery) *bun.SelectQuery {
	cost := buncolgen.AIUsageRecordColumns.CostUSD.Qualified()

	return q.Model((*aiusage.AIUsageRecord)(nil)).
		ColumnExpr("COALESCE(SUM(" + cost + "), 0)::text AS cost_usd").
		ColumnExpr("COUNT(*) AS calls").
		ColumnExpr("COUNT(*) FILTER (WHERE " + cost + " IS NULL) AS unpriced_calls")
}

func scanCost(
	ctx context.Context,
	q *bun.SelectQuery,
	what string,
) (*repositories.AIUsageCost, error) {
	var row costRow
	if err := q.Scan(ctx, &row); err != nil {
		return nil, fmt.Errorf("sum %s: %w", what, err)
	}

	cost, err := decimal.NewFromString(row.CostUSD)
	if err != nil {
		return nil, fmt.Errorf("read %s %q: %w", what, row.CostUSD, err)
	}

	return &repositories.AIUsageCost{
		CostUSD:       cost,
		Calls:         row.Calls,
		UnpricedCalls: row.UnpricedCalls,
	}, nil
}

func (r *repository) SurfaceCost(
	ctx context.Context,
	req repositories.AIUsageSurfaceCostRequest,
) (*repositories.AIUsageCost, error) {
	cols := buncolgen.AIUsageRecordColumns

	q := costSums(r.db.DBForContext(ctx).NewSelect()).
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(cols.Surface.Eq(), req.Surface)
	if req.Since > 0 {
		q = q.Where(cols.CreatedAt.Gte(), req.Since)
	}

	return scanCost(ctx, q, string(req.Surface)+" cost")
}

func (r *repository) EvaluationCost(
	ctx context.Context,
	req repositories.AIUsageEvaluationCostRequest,
) (*repositories.AIUsageCost, error) {
	cols := buncolgen.AIUsageRecordColumns
	evals := buncolgen.EvaluationColumns
	dba := r.db.DBForContext(ctx)

	q := costSums(dba.NewSelect()).
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(cols.Surface.Eq(), aiusage.SurfaceEvaluation)
	if req.Since > 0 {
		q = q.Where(cols.CreatedAt.Gte(), req.Since)
	}
	if req.SuiteRunID.IsNotNil() {
		replays := dba.NewSelect().
			Model((*agent.Evaluation)(nil)).
			ColumnExpr(evals.ID.Qualified()).
			Where(evals.OrganizationID.Eq(), req.TenantInfo.OrgID).
			Where(evals.BusinessUnitID.Eq(), req.TenantInfo.BuID).
			Where(evals.SuiteRunID.Eq(), req.SuiteRunID)
		q = q.Where(cols.RunID.In(), replays)
	}

	return scanCost(ctx, q, "evaluation cost")
}
