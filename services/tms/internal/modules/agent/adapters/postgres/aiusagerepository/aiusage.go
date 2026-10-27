package aiusagerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider/aiproviderdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage/aiusagedb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
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

func New(p Params) aiusageports.AIUsageRepository {
	return &repository{db: p.DB, l: p.Logger.Named("repository.aiusage")}
}

func (r *repository) Create(ctx context.Context, record *aiusage.AIUsageRecord) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := aiusagedb.AIUsageRecords.Create(ctx, r.db.Keel(ctx), record); err != nil {
			r.l.Error("failed to record ai usage", zap.Error(err))
			return fmt.Errorf("record ai usage: %w", err)
		}

		return nil
	})
}

// totalsRow is the shape every aggregate query decodes into: up to three
// grouping keys, then the totals. Percentiles are taken over successful calls
// only, in SQL, because a failure refused in a millisecond is not a fast
// answer and would drag the median down.
type totalsRow struct {
	First           string
	Second          string
	Third           string
	Calls           int
	Failed          int
	InputTokens     int64
	OutputTokens    int64
	ReasoningTokens int64
	CostUSD         string
	PricedCalls     int
	LatencyP50      float64
	LatencyP95      float64
}

func (row *totalsRow) totals() aiusageports.AIUsageTotals {
	cost := row.CostUSD
	if cost == "" {
		cost = "0"
	}

	return aiusageports.AIUsageTotals{
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

func latencyPercentile(fraction float64) keel.Expr {
	u := aiusagedb.AIUsageRecords

	return keel.CastFloat64(keel.Float64Of(
		keel.PercentileCont(fraction, u.LatencyMs.Asc()).Filter(keel.And(
			keel.BoolOf(u.Succeeded),
			u.Surface.Ne(aiusage.SurfaceBackground),
		)),
	).Coalesce(0))
}

func totalsPicks(keys ...keel.Expr) []keel.Expr {
	u := aiusagedb.AIUsageRecords
	picks := make([]keel.Expr, 0, 12)
	picks = append(picks, keys...)
	for len(picks) < 3 {
		picks = append(picks, keel.Text(""))
	}

	return append(picks,
		keel.Count(),
		keel.Count().Filter(u.Succeeded.IsFalse()),
		keel.NumericOf(u.InputTokens.Sum()).Coalesce(0),
		keel.NumericOf(u.OutputTokens.Sum()).Coalesce(0),
		keel.NumericOf(u.ReasoningTokens.Sum()).Coalesce(0),
		keel.CastText(keel.NumericOf(u.CostUSD.Sum()).Coalesce(0)),
		u.CostUSD.Count(),
		latencyPercentile(0.5),
		latencyPercentile(0.95),
	)
}

func (r *repository) Summary(
	ctx context.Context,
	req aiusageports.AIUsageSummaryRequest,
) (*aiusageports.AIUsageSummaryRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*aiusageports.AIUsageSummaryRow, error) {
		u := aiusagedb.AIUsageRecords
		provider := aiproviderdb.Providers.As("usage_provider")
		since := u.CreatedAt.Gte(req.Since)

		var (
			total      totalsRow
			byProvider []totalsRow
			byFeature  []totalsRow
		)
		err := keel.Batch(ctx, r.db.Keel(ctx),
			keel.IntoValue(u.Where(since).Pick(totalsPicks()...), &total),
			keel.IntoSlice(u.
				LeftJoin(provider, provider.ID.EqCol(u.ProviderID)).
				Where(since).
				Group(u.ProviderID, provider.Name, u.Model).
				Pick(totalsPicks(
					u.ProviderID.Coalesce(""),
					provider.Name.Coalesce(""),
					u.Model,
				)...).
				Order(keel.Count().Desc()), &byProvider),
			keel.IntoSlice(u.
				Where(since).
				Group(u.Feature).
				Pick(totalsPicks(u.Feature.Coalesce(""))...).
				Order(keel.Count().Desc()), &byFeature),
		)
		if err != nil {
			return nil, fmt.Errorf("summarise ai usage: %w", err)
		}

		summary := &aiusageports.AIUsageSummaryRow{
			Totals:     total.totals(),
			ByProvider: make([]aiusageports.AIUsageProviderTotals, 0, len(byProvider)),
			ByFeature:  make([]aiusageports.AIUsageFeatureTotals, 0, len(byFeature)),
		}
		for i := range byFeature {
			row := &byFeature[i]
			summary.ByFeature = append(summary.ByFeature, aiusageports.AIUsageFeatureTotals{
				Feature:       aiusage.Feature(row.First),
				AIUsageTotals: row.totals(),
			})
		}
		for i := range byProvider {
			row := &byProvider[i]
			slice := aiusageports.AIUsageProviderTotals{
				ProviderName:  row.Second,
				Model:         row.Third,
				AIUsageTotals: row.totals(),
			}
			if row.First != "" {
				slice.ProviderID = pulidFrom(row.First)
			}
			summary.ByProvider = append(summary.ByProvider, slice)
		}

		return summary, nil
	})
}

func (r *repository) Daily(
	ctx context.Context,
	req aiusageports.AIUsageDailyQuery,
) ([]aiusageports.AIUsageDayTotals, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]aiusageports.AIUsageDayTotals, error) {
		u := aiusagedb.AIUsageRecords
		q := u.Where(u.CreatedAt.Gte(req.Since))
		if req.ProviderID.IsNotNil() {
			q.Where(u.ProviderID.Eq(req.ProviderID))
		}

		day := keel.ToTimestamp(u.CreatedAt).
			AtTimeZone(keel.Text(req.Timezone)).
			ToChar("YYYY-MM-DD")
		rows, err := keel.Into[totalsRow](ctx, r.db.Keel(ctx), q.
			Group(day).
			Pick(totalsPicks(day)...).
			Order(day.Asc()))
		if err != nil {
			return nil, fmt.Errorf("summarise ai usage by day: %w", err)
		}

		days := make([]aiusageports.AIUsageDayTotals, 0, len(rows))
		for idx := range rows {
			days = append(days, aiusageports.AIUsageDayTotals{
				Day:           rows[idx].First,
				AIUsageTotals: rows[idx].totals(),
			})
		}
		return days, nil
	})
}

type failureRow struct {
	ProviderID   string
	ProviderName string
	Model        string
	Task         string
	ErrorClass   string
	ErrorMessage string
	CreatedAt    int64
}

// RecentFailures lists the newest failed attempts in a window, with the
// provider's own message, so the reason behind a failure count is readable
// where the count is shown.
func (r *repository) RecentFailures(
	ctx context.Context,
	req aiusageports.AIUsageFailuresRequest,
) ([]aiusageports.AIUsageFailureRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]aiusageports.AIUsageFailureRow, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = 5
			}

			u := aiusagedb.AIUsageRecords
			provider := aiproviderdb.Providers.As("usage_provider")
			rows, err := keel.Into[failureRow](ctx, r.db.Keel(ctx), u.
				LeftJoin(provider, provider.ID.EqCol(u.ProviderID)).
				Where(u.CreatedAt.Gte(req.Since), u.Succeeded.IsFalse()).
				Pick(
					u.ProviderID.Coalesce(""),
					provider.Name.Coalesce(""),
					u.Model,
					u.Task,
					u.ErrorClass.Coalesce(""),
					u.ErrorMessage.Coalesce(""),
					u.CreatedAt,
				).
				Order(u.CreatedAt.Desc()).
				Limit(limit))
			if err != nil {
				return nil, fmt.Errorf("list ai usage failures: %w", err)
			}

			failures := make([]aiusageports.AIUsageFailureRow, 0, len(rows))
			for i := range rows {
				row := &rows[i]
				failure := aiusageports.AIUsageFailureRow{
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
		},
	)
}

type costRow struct {
	CostUSD       string
	Calls         int
	UnpricedCalls int
}

func costSums(q *keel.Query[*aiusage.AIUsageRecord]) *keel.Query[*aiusage.AIUsageRecord] {
	u := aiusagedb.AIUsageRecords

	return q.Pick(
		keel.CastText(keel.NumericOf(u.CostUSD.Sum()).Coalesce(0)),
		keel.Count(),
		keel.Count().Filter(u.CostUSD.IsNull()),
	)
}

func (r *repository) scanCost(
	ctx context.Context,
	q *keel.Query[*aiusage.AIUsageRecord],
	what string,
) (*aiusageports.AIUsageCost, error) {
	row, err := keel.IntoOne[costRow](ctx, r.db.Keel(ctx), costSums(q))
	if err != nil {
		return nil, fmt.Errorf("sum %s: %w", what, err)
	}

	cost, err := decimal.NewFromString(row.CostUSD)
	if err != nil {
		return nil, fmt.Errorf("read %s %q: %w", what, row.CostUSD, err)
	}

	return &aiusageports.AIUsageCost{
		CostUSD:       cost,
		Calls:         row.Calls,
		UnpricedCalls: row.UnpricedCalls,
	}, nil
}

// CostByDefinition sums what one agent's calls cost since an instant. Calls
// without a price are counted, not summed: a budget check that read them as
// free would let an agent on an unpriced provider run without end.
func (r *repository) CostByDefinition(
	ctx context.Context,
	req aiusageports.AIUsageCostRequest,
) (*aiusageports.AIUsageCost, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*aiusageports.AIUsageCost, error) {
		u := aiusagedb.AIUsageRecords

		return r.scanCost(ctx, u.Where(
			u.AgentDefinitionID.Eq(req.DefinitionID),
			u.CreatedAt.Gte(req.Since),
			u.Surface.Ne(aiusage.SurfaceEvaluation),
		), "ai usage cost")
	})
}

func (r *repository) SurfaceCost(
	ctx context.Context,
	req aiusageports.AIUsageSurfaceCostRequest,
) (*aiusageports.AIUsageCost, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*aiusageports.AIUsageCost, error) {
		u := aiusagedb.AIUsageRecords
		q := u.Where(u.Surface.Eq(req.Surface))
		if req.Since > 0 {
			q.Where(u.CreatedAt.Gte(req.Since))
		}

		return r.scanCost(ctx, q, string(req.Surface)+" cost")
	})
}

func (r *repository) EvaluationCost(
	ctx context.Context,
	req aiusageports.AIUsageEvaluationCostRequest,
) (*aiusageports.AIUsageCost, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*aiusageports.AIUsageCost, error) {
		u := aiusagedb.AIUsageRecords
		q := u.Where(u.Surface.Eq(aiusage.SurfaceEvaluation))
		if req.Since > 0 {
			q.Where(u.CreatedAt.Gte(req.Since))
		}
		if req.SuiteRunID.IsNotNil() {
			e := agentdb.Evaluations
			q.Where(u.RunID.InQuery(e.Where(e.SuiteRunID.Eq(req.SuiteRunID)).Pick(e.ID)))
		}

		return r.scanCost(ctx, q, "evaluation cost")
	})
}

type providerTaskRow struct {
	ProviderID string
	Task       string
	Calls      int
	Failed     int
	Rescued    int
}

func (r *repository) ProviderTaskTotals(
	ctx context.Context,
	req aiusageports.AIUsageProviderTaskRequest,
) ([]aiusageports.AIUsageProviderTaskTotals, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]aiusageports.AIUsageProviderTaskTotals, error) {
			u := aiusagedb.AIUsageRecords
			rows, err := keel.Into[providerTaskRow](ctx, r.db.Keel(ctx), u.
				Where(u.CreatedAt.Gte(req.Since), u.ProviderID.IsNotNull()).
				Group(u.ProviderID, u.Task).
				Pick(
					u.ProviderID,
					u.Task,
					keel.Count(),
					keel.Count().Filter(u.Succeeded.IsFalse()),
					keel.Count().Filter(keel.And(u.Succeeded.IsTrue(), u.Failover.IsTrue())),
				))
			if err != nil {
				return nil, fmt.Errorf("total ai usage by provider and task: %w", err)
			}

			totals := make([]aiusageports.AIUsageProviderTaskTotals, 0, len(rows))
			for i := range rows {
				row := &rows[i]
				totals = append(totals, aiusageports.AIUsageProviderTaskTotals{
					ProviderID: pulidFrom(row.ProviderID),
					Task:       aiprovider.Task(row.Task),
					Calls:      row.Calls,
					Failed:     row.Failed,
					Rescued:    row.Rescued,
				})
			}
			return totals, nil
		},
	)
}

type providerSpendRow struct {
	ProviderID string
	CostUSD    string
}

func (r *repository) SpendByProvider(
	ctx context.Context,
	req aiusageports.AIUsageProviderSpendRequest,
) (map[pulid.ID]decimal.Decimal, error) {
	if len(req.ProviderIDs) == 0 {
		return map[pulid.ID]decimal.Decimal{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]decimal.Decimal, error) {
		u := aiusagedb.AIUsageRecords
		rows, err := keel.Into[providerSpendRow](ctx, r.db.Keel(ctx), u.
			Where(
				u.ProviderID.In(req.ProviderIDs...),
				u.CreatedAt.Gte(req.Since),
				u.CostUSD.IsNotNull(),
			).
			Group(u.ProviderID).
			Pick(u.ProviderID, keel.CastText(keel.NumericOf(u.CostUSD.Sum()).Coalesce(0))))
		if err != nil {
			return nil, fmt.Errorf("sum ai provider spend: %w", err)
		}

		spend := make(map[pulid.ID]decimal.Decimal, len(rows))
		for idx := range rows {
			cost, err := decimal.NewFromString(rows[idx].CostUSD)
			if err != nil {
				return nil, fmt.Errorf("read ai provider spend %q: %w", rows[idx].CostUSD, err)
			}
			spend[pulidFrom(rows[idx].ProviderID)] = cost
		}

		return spend, nil
	})
}
