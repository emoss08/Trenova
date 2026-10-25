package aianalyticsrepository

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentsuggestion/agentsuggestiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics/aianalyticsdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage/aiusagedb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const toolExampleCandidatesPerExample = 25

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) aianalyticsports.AnalyticsRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.ai-analytics-repository")}
}

func floorHour(at int64) int64 {
	return at - at%3600
}

func ceilHour(at int64) int64 {
	if at%3600 == 0 {
		return at
	}

	return floorHour(at) + 3600
}

func (r *repository) RefreshState(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*aianalytics.RefreshState, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*aianalytics.RefreshState, error) {
		state, err := aianalyticsdb.RefreshStates.Query().One(ctx, r.db.Keel(ctx))
		if errors.Is(err, keel.ErrNotFound) {
			return nil, nil
		}
		if err != nil {
			r.l.Error("failed to read the ai analytics refresh state", zap.Error(err))
			return nil, fmt.Errorf("read ai analytics refresh state: %w", err)
		}

		return state, nil
	})
}

func (r *repository) SaveRefreshState(ctx context.Context, state *aianalytics.RefreshState) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		rs := aianalyticsdb.RefreshStates
		_, err := rs.Upsert(
			ctx,
			r.db.Keel(ctx),
			[]*aianalytics.RefreshState{state},
			keel.OnConflict(rs.OrganizationID, rs.BusinessUnitID).
				Update(rs.RefreshedThrough, rs.RefreshedAt),
		)
		if err != nil {
			r.l.Error("failed to save the ai analytics refresh state", zap.Error(err))
			return fmt.Errorf("save ai analytics refresh state: %w", err)
		}

		return nil
	})
}

func (r *repository) markChanged(ctx context.Context, tables ...*schema.Table) {
	names := make([]string, 0, len(tables))
	for _, table := range tables {
		names = append(names, table.Name)
	}
	keel.MarkChanged(ctx, r.db.Keel(ctx), names...)
}

func (r *repository) lockRefresh(ctx context.Context, tenant pagination.TenantInfo) error {
	key := "ai-analytics-refresh:" + tenant.OrgID.String() + ":" + tenant.BuID.String()
	if err := keel.LockKey(ctx, r.db.Keel(ctx), key); err != nil {
		return fmt.Errorf("lock ai analytics refresh: %w", err)
	}

	return nil
}

func (r *repository) HasActivity(
	ctx context.Context,
	req aianalyticsports.RefreshRequest,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		runs := agentdb.AgentRuns
		turns := conversationdb.AssistantTurns
		usage := aiusagedb.AIUsageRecords
		var ran, talked, used bool
		err := keel.Batch(ctx, r.db.Keel(ctx),
			keel.ExistsInto(runs.Where(
				runs.UpdatedAt.Gte(req.From),
				runs.UpdatedAt.Lt(req.To),
			), &ran),
			keel.ExistsInto(turns.Where(
				turns.UpdatedAt.Gte(req.From),
				turns.UpdatedAt.Lt(req.To),
			), &talked),
			keel.ExistsInto(usage.Where(
				usage.CreatedAt.Gte(req.From),
				usage.CreatedAt.Lt(req.To),
			), &used),
		)
		if err != nil {
			r.l.Error("failed to check ai analytics activity", zap.Error(err))
			return false, fmt.Errorf("read ai analytics activity: %w", err)
		}

		return ran || talked || used, nil
	})
}

func (r *repository) RefreshUnits(
	ctx context.Context,
	req aianalyticsports.RefreshRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		if err := r.lockRefresh(ctx, req.TenantInfo); err != nil {
			return 0, err
		}
		db := r.db.Keel(ctx)
		refresh := &unitRefresh{from: req.From, to: req.To, stamp: req.Stamp}
		w := aianalyticsdb.WorkUnits

		total := 0
		for _, step := range []struct {
			name            string
			source          *keel.Query[*keel.Row]
			withDecisionRun bool
		}{
			{"runs", ownedUnitsSource(&runSource, refresh), true},
			{"turns", ownedUnitsSource(&turnSource, refresh), true},
			{"calls", callUnitsSource(refresh), false},
		} {
			written, err := w.UpsertFrom(
				ctx,
				db,
				step.source,
				unitConflict(),
				unitInsertColumns(w, step.withDecisionRun)...,
			)
			if err != nil {
				r.l.Error("failed to refresh ai work units",
					zap.String("source", step.name),
					zap.String("organization", req.TenantInfo.OrgID.String()),
					zap.Error(err),
				)
				return 0, fmt.Errorf("refresh ai work units from %s: %w", step.name, err)
			}
			total += int(written)
		}

		if _, err := clearUnitToolsQuery(req.Stamp).Delete(ctx, db); err != nil {
			r.l.Error("failed to clear ai work unit tools", zap.Error(err))
			return 0, fmt.Errorf("clear ai work unit tools: %w", err)
		}
		t := aianalyticsdb.UnitTools
		if _, err := t.InsertFrom(
			ctx,
			db,
			unitToolsSource(req.Stamp),
			unitToolColumns(t)...); err != nil {
			r.l.Error("failed to write ai work unit tools", zap.Error(err))
			return 0, fmt.Errorf("write ai work unit tools: %w", err)
		}
		r.markChanged(ctx, aianalyticsdb.WorkUnitsSchema(), aianalyticsdb.UnitToolsSchema())

		return total, nil
	})
}

func (r *repository) RefreshProviderHours(
	ctx context.Context,
	req aianalyticsports.RefreshRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		if err := r.lockRefresh(ctx, req.TenantInfo); err != nil {
			return 0, err
		}
		db := r.db.Keel(ctx)
		fromHour := floorHour(req.From)
		toHour := ceilHour(req.To)

		if _, err := clearProviderHoursQuery(fromHour, toHour).Delete(ctx, db); err != nil {
			r.l.Error("failed to clear ai provider hours", zap.Error(err))
			return 0, fmt.Errorf("clear ai provider hours: %w", err)
		}
		h := aianalyticsdb.ProviderHours
		written, err := h.InsertFrom(
			ctx,
			db,
			providerHoursSource(fromHour, toHour),
			providerHourColumns(h)...,
		)
		if err != nil {
			r.l.Error("failed to write ai provider hours", zap.Error(err))
			return 0, fmt.Errorf("write ai provider hours: %w", err)
		}
		r.markChanged(ctx, aianalyticsdb.ProviderHoursSchema())

		return int(written), nil
	})
}

func (r *repository) UnkeyedModels(
	ctx context.Context,
	req aianalyticsports.ModelKeyRequest,
) ([]string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]string, error) {
		w := aianalyticsdb.WorkUnits
		h := aianalyticsdb.ProviderHours
		var units, hours []string
		err := keel.Batch(ctx, r.db.Keel(ctx),
			keel.IntoSlice(w.
				Where(w.RefreshStamp.Eq(req.Stamp), w.ModelKey.Eq("")).
				Group(w.Model).
				Pick(w.Model), &units),
			keel.IntoSlice(h.
				Where(h.Hour.Gte(req.FromHour), h.ModelKey.Eq("")).
				Group(h.Model).
				Pick(h.Model), &hours),
		)
		if err != nil {
			r.l.Error("failed to list unkeyed ai models", zap.Error(err))
			return nil, fmt.Errorf("list unkeyed ai models: %w", err)
		}

		models := make([]string, 0, len(units)+len(hours))
		seen := make(map[string]struct{}, len(units)+len(hours))
		for _, model := range slices.Concat(units, hours) {
			if _, ok := seen[model]; ok {
				continue
			}
			seen[model] = struct{}{}
			models = append(models, model)
		}

		return models, nil
	})
}

func (r *repository) SetModelKeys(
	ctx context.Context,
	req aianalyticsports.ModelKeyRequest,
	keys []aianalyticsports.ModelKey,
) error {
	if len(keys) == 0 {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := r.lockRefresh(ctx, req.TenantInfo); err != nil {
			return err
		}
		db := r.db.Keel(ctx)
		w := aianalyticsdb.WorkUnits
		h := aianalyticsdb.ProviderHours
		for _, key := range keys {
			if _, err := w.Where(
				w.RefreshStamp.Eq(req.Stamp),
				w.Model.Eq(key.Model),
				w.ModelKey.Ne(key.Key),
			).Update(ctx, db, w.ModelKey.To(key.Key)); err != nil {
				r.l.Error("failed to key ai work units", zap.Error(err))
				return fmt.Errorf("key ai work units: %w", err)
			}
			if _, err := h.Where(
				h.Hour.Gte(req.FromHour),
				h.Model.Eq(key.Model),
				h.ModelKey.Ne(key.Key),
			).Update(ctx, db, h.ModelKey.To(key.Key)); err != nil {
				r.l.Error("failed to key ai provider hours", zap.Error(err))
				return fmt.Errorf("key ai provider hours: %w", err)
			}
		}

		t := aianalyticsdb.UnitTools
		unit := aianalyticsdb.WorkUnits.As("keyed_unit")
		if _, err := t.Join(unit,
			unit.ID.EqCol(t.UnitID),
			unit.RefreshStamp.Eq(req.Stamp),
		).
			Where(t.ModelKey.NeCol(unit.ModelKey)).
			Update(ctx, db, t.ModelKey.ToExpr(unit.ModelKey)); err != nil {
			r.l.Error("failed to key ai work unit tools", zap.Error(err))
			return fmt.Errorf("key ai work unit tools: %w", err)
		}

		return nil
	})
}

func (r *repository) Prune(ctx context.Context, req aianalyticsports.PruneRequest) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := r.lockRefresh(ctx, req.TenantInfo); err != nil {
			return err
		}
		db := r.db.Keel(ctx)
		t := aianalyticsdb.UnitTools
		w := aianalyticsdb.WorkUnits
		h := aianalyticsdb.ProviderHours
		s := agentsuggestiondb.Suggestions
		for _, query := range []interface {
			Delete(context.Context, *keel.DB) (int64, error)
		}{
			t.Where(t.FinishedAt.Lt(req.Before)),
			w.Where(w.FinishedAt.Lt(req.Before)),
			h.Where(h.Hour.Lt(floorHour(req.Before))),
			s.Where(s.CreatedAt.Lt(req.Before)),
		} {
			if _, err := query.Delete(ctx, db); err != nil {
				r.l.Error("failed to prune ai analytics", zap.Error(err))
				return fmt.Errorf("prune ai analytics: %w", err)
			}
		}

		return nil
	})
}

type deploymentScan struct {
	ModelKey   string
	ProviderID string
}

func (s deploymentScan) key() aianalyticsports.DeploymentKey {
	return aianalyticsports.DeploymentKey{ModelKey: s.ModelKey, ProviderID: s.ProviderID}
}

func deploymentKey(modelKey, providerID string) aianalyticsports.DeploymentKey {
	return aianalyticsports.DeploymentKey{ModelKey: modelKey, ProviderID: providerID}
}

type countsScan struct {
	Runs              int
	Completed         int
	HandedOff         int
	Stopped           int
	Failed            int
	ModelCalls        int
	FailedCalls       int
	ToolCalls         int
	ToolFailuresModel int
	ToolFailuresTool  int
	CostUSD           string
	PricedRuns        int
	InputTokens       int64
	OutputTokens      int64
	CacheReadTokens   int64
	CacheWriteTokens  int64
	AdvisorCalls      int64
	AdvisorPrompt     int64
	AdvisorOutput     int64
	AdvisorCostUSD    string
	LastRanAt         int64
}

func (s *countsScan) counts() aianalyticsports.Counts {
	cost, err := decimal.NewFromString(s.CostUSD)
	if err != nil {
		cost = decimal.Zero
	}
	advice, err := decimal.NewFromString(s.AdvisorCostUSD)
	if err != nil {
		advice = decimal.Zero
	}

	return aianalyticsports.Counts{
		Runs:              s.Runs,
		Completed:         s.Completed,
		HandedOff:         s.HandedOff,
		Stopped:           s.Stopped,
		Failed:            s.Failed,
		ModelCalls:        s.ModelCalls,
		FailedCalls:       s.FailedCalls,
		ToolCalls:         s.ToolCalls,
		ToolFailuresModel: s.ToolFailuresModel,
		ToolFailuresTool:  s.ToolFailuresTool,
		CostUSD:           cost,
		PricedRuns:        s.PricedRuns,
		LastRanAt:         s.LastRanAt,
		Tokens: aianalyticsports.TokenTotals{
			Input:      s.InputTokens,
			Output:     s.OutputTokens,
			CacheRead:  s.CacheReadTokens,
			CacheWrite: s.CacheWriteTokens,
		},
		Advisor: aianalyticsports.AdvisorTotals{
			Calls:        s.AdvisorCalls,
			PromptTokens: s.AdvisorPrompt,
			OutputTokens: s.AdvisorOutput,
			CostUSD:      advice,
		},
	}
}

func durationStats(count int, percentiles []*float64) aianalyticsports.DurationStats {
	stats := aianalyticsports.DurationStats{Count: count}
	if count == 0 || len(percentiles) < 4 {
		return stats
	}
	values := make([]int64, 4)
	for index := range values {
		if percentiles[index] == nil {
			return stats
		}
		values[index] = int64(*percentiles[index] + 0.5)
	}
	stats.P25Ms, stats.P50Ms, stats.P75Ms, stats.P95Ms = values[0], values[1], values[2], values[3]
	stats.HasAny = true

	return stats
}

func readRows[T any](
	ctx context.Context,
	r *repository,
	what string,
	query keel.Projection,
) ([]T, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]T, error) {
		rows, err := keel.Into[T](ctx, r.db.Keel(ctx), query)
		if err != nil {
			r.l.Error("failed to read ai analytics", zap.String("what", what), zap.Error(err))
			return nil, fmt.Errorf("read ai analytics %s: %w", what, err)
		}

		return rows, nil
	})
}

func readMapped[T, R any](
	ctx context.Context,
	r *repository,
	what string,
	query keel.Projection,
	mapRow func(*T) R,
) ([]R, error) {
	rows, err := readRows[T](ctx, r, what, query)
	if err != nil {
		return nil, err
	}
	out := make([]R, 0, len(rows))
	for index := range rows {
		out = append(out, mapRow(&rows[index]))
	}

	return out, nil
}

type totalsScan struct {
	Runs              int
	Completed         int
	HandedOff         int
	Stopped           int
	Failed            int
	ModelCalls        int
	FailedCalls       int
	ToolCalls         int
	ToolFailuresModel int
	ToolFailuresTool  int
	CostUSD           string
	PricedRuns        int
	InputTokens       int64
	OutputTokens      int64
	CacheReadTokens   int64
	CacheWriteTokens  int64
	AdvisorCalls      int64
	AdvisorPrompt     int64
	AdvisorOutput     int64
	AdvisorCostUSD    string
	LastRanAt         int64
	CompletionCount   int
	CompletionP25     *float64
	CompletionP50     *float64
	CompletionP75     *float64
	CompletionP95     *float64
}

func (r *repository) Totals(
	ctx context.Context,
	filter aianalyticsports.Filter,
) (*aianalyticsports.TotalsRow, error) {
	rows, err := readRows[totalsScan](ctx, r, "totals", totalsQuery(&filter))
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return &aianalyticsports.TotalsRow{
			Counts: aianalyticsports.Counts{CostUSD: decimal.Zero},
		}, nil
	}
	row := &rows[0]
	counts := countsScan{
		Runs:              row.Runs,
		Completed:         row.Completed,
		HandedOff:         row.HandedOff,
		Stopped:           row.Stopped,
		Failed:            row.Failed,
		ModelCalls:        row.ModelCalls,
		FailedCalls:       row.FailedCalls,
		ToolCalls:         row.ToolCalls,
		ToolFailuresModel: row.ToolFailuresModel,
		ToolFailuresTool:  row.ToolFailuresTool,
		CostUSD:           row.CostUSD,
		PricedRuns:        row.PricedRuns,
		InputTokens:       row.InputTokens,
		OutputTokens:      row.OutputTokens,
		CacheReadTokens:   row.CacheReadTokens,
		CacheWriteTokens:  row.CacheWriteTokens,
		AdvisorCalls:      row.AdvisorCalls,
		AdvisorPrompt:     row.AdvisorPrompt,
		AdvisorOutput:     row.AdvisorOutput,
		AdvisorCostUSD:    row.AdvisorCostUSD,
		LastRanAt:         row.LastRanAt,
	}

	return &aianalyticsports.TotalsRow{
		Counts: counts.counts(),
		Completion: durationStats(row.CompletionCount, []*float64{
			row.CompletionP25, row.CompletionP50, row.CompletionP75, row.CompletionP95,
		}),
	}, nil
}

type deploymentCountsScan struct {
	ModelKey          string
	ProviderID        string
	Runs              int
	Completed         int
	HandedOff         int
	Stopped           int
	Failed            int
	ModelCalls        int
	FailedCalls       int
	ToolCalls         int
	ToolFailuresModel int
	ToolFailuresTool  int
	CostUSD           string
	PricedRuns        int
	InputTokens       int64
	OutputTokens      int64
	CacheReadTokens   int64
	CacheWriteTokens  int64
	AdvisorCalls      int64
	AdvisorPrompt     int64
	AdvisorOutput     int64
	AdvisorCostUSD    string
	LastRanAt         int64
}

func (s *deploymentCountsScan) counts() aianalyticsports.Counts {
	counts := countsScan{
		Runs:              s.Runs,
		Completed:         s.Completed,
		HandedOff:         s.HandedOff,
		Stopped:           s.Stopped,
		Failed:            s.Failed,
		ModelCalls:        s.ModelCalls,
		FailedCalls:       s.FailedCalls,
		ToolCalls:         s.ToolCalls,
		ToolFailuresModel: s.ToolFailuresModel,
		ToolFailuresTool:  s.ToolFailuresTool,
		CostUSD:           s.CostUSD,
		PricedRuns:        s.PricedRuns,
		InputTokens:       s.InputTokens,
		OutputTokens:      s.OutputTokens,
		CacheReadTokens:   s.CacheReadTokens,
		CacheWriteTokens:  s.CacheWriteTokens,
		AdvisorCalls:      s.AdvisorCalls,
		AdvisorPrompt:     s.AdvisorPrompt,
		AdvisorOutput:     s.AdvisorOutput,
		AdvisorCostUSD:    s.AdvisorCostUSD,
		LastRanAt:         s.LastRanAt,
	}

	return counts.counts()
}

func (r *repository) Deployments(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.DeploymentRow, error) {
	return readMapped(ctx, r, "deployments", deploymentsQuery(&filter),
		func(row *deploymentCountsScan) aianalyticsports.DeploymentRow {
			return aianalyticsports.DeploymentRow{
				DeploymentKey: deploymentKey(row.ModelKey, row.ProviderID),
				Counts:        row.counts(),
			}
		})
}

type outcomeDurationScan struct {
	ModelKey   string
	ProviderID string
	Outcome    string
	Count      int
	P25        *float64
	P50        *float64
	P75        *float64
	P95        *float64
}

func (r *repository) OutcomeDurations(
	ctx context.Context,
	filter aianalyticsports.Filter,
	grouping aianalytics.Grouping,
) ([]aianalyticsports.OutcomeDurationRow, error) {
	return readMapped(
		ctx,
		r,
		"outcome durations",
		outcomeDurationsQuery(&filter, grouping),
		func(row *outcomeDurationScan) aianalyticsports.OutcomeDurationRow {
			return aianalyticsports.OutcomeDurationRow{
				DeploymentKey: deploymentKey(row.ModelKey, row.ProviderID),
				Outcome:       aianalytics.Outcome(row.Outcome),
				DurationStats: durationStats(
					row.Count,
					[]*float64{row.P25, row.P50, row.P75, row.P95},
				),
			}
		},
	)
}

type trendScan struct {
	ModelKey     string
	ProviderID   string
	BucketStart  int64
	Runs         int
	Completed    int
	ToolCalls    int
	ToolFailures int
	MedianRunMs  sql.NullFloat64
}

func (r *repository) Trend(
	ctx context.Context,
	req aianalyticsports.TrendRequest,
) ([]aianalyticsports.TrendRow, error) {
	if len(req.BucketStarts) == 0 {
		return []aianalyticsports.TrendRow{}, nil
	}

	return readMapped(ctx, r, "trend", trendQuery(&req),
		func(row *trendScan) aianalyticsports.TrendRow {
			return aianalyticsports.TrendRow{
				DeploymentKey: deploymentKey(row.ModelKey, row.ProviderID),
				BucketStart:   row.BucketStart,
				Runs:          row.Runs,
				Completed:     row.Completed,
				ToolCalls:     row.ToolCalls,
				ToolFailures:  row.ToolFailures,
				MedianRunMs:   int64(row.MedianRunMs.Float64 + 0.5),
				HasMedian:     row.MedianRunMs.Valid,
			}
		})
}

type toolCellScan struct {
	ModelKey   string
	ProviderID string
	ToolName   string
	Cause      string
	Calls      int
}

func (r *repository) ToolCells(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.ToolCellRow, error) {
	return readMapped(ctx, r, "tool cells", toolCellsQuery(&filter),
		func(row *toolCellScan) aianalyticsports.ToolCellRow {
			return aianalyticsports.ToolCellRow{
				DeploymentKey: deploymentKey(row.ModelKey, row.ProviderID),
				ToolName:      row.ToolName,
				Cause:         aianalytics.Cause(row.Cause),
				Calls:         row.Calls,
			}
		})
}

type taskCellScan struct {
	ModelKey   string
	ProviderID string
	Task       string
	Runs       int
	Completed  int
}

func (r *repository) TaskCells(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.TaskCellRow, error) {
	return readMapped(ctx, r, "task cells", taskCellsQuery(&filter),
		func(row *taskCellScan) aianalyticsports.TaskCellRow {
			return aianalyticsports.TaskCellRow{
				DeploymentKey: deploymentKey(row.ModelKey, row.ProviderID),
				Task:          aiprovider.Task(row.Task),
				Runs:          row.Runs,
				Completed:     row.Completed,
			}
		})
}

type decisionScan struct {
	ModelKey   string
	ProviderID string
	Accepted   int
	Modified   int
	Rejected   int
}

func (r *repository) Decisions(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.DecisionRow, error) {
	return readMapped(ctx, r, "decisions", decisionsQuery(&filter),
		func(row *decisionScan) aianalyticsports.DecisionRow {
			return aianalyticsports.DecisionRow{
				DeploymentKey: deploymentKey(row.ModelKey, row.ProviderID),
				Accepted:      row.Accepted,
				Modified:      row.Modified,
				Rejected:      row.Rejected,
			}
		})
}

type ratingScan struct {
	ModelKey   string
	ProviderID string
	Rated      int
	Helpful    int
}

func (r *repository) Ratings(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.RatingRow, error) {
	return readMapped(ctx, r, "ratings", ratingsQuery(&filter),
		func(row *ratingScan) aianalyticsports.RatingRow {
			return aianalyticsports.RatingRow{
				DeploymentKey: deploymentKey(row.ModelKey, row.ProviderID),
				Rated:         row.Rated,
				Helpful:       row.Helpful,
			}
		})
}

type suggestionScan struct {
	AgentDefinitionID pulid.ID
	AgentName         string
	Switched          int
	SentThrough       int
	AutoSwitched      int
	Kept              int
	PassedOver        int
}

func (r *repository) Suggestions(
	ctx context.Context,
	req *aianalyticsports.SuggestionRequest,
) ([]aianalyticsports.SuggestionRow, error) {
	return readMapped(ctx, r, "suggestions", suggestionsQuery(req),
		func(row *suggestionScan) aianalyticsports.SuggestionRow {
			return aianalyticsports.SuggestionRow{
				AgentDefinitionID: row.AgentDefinitionID,
				AgentName:         row.AgentName,
				SuggestionCounts: aianalyticsports.SuggestionCounts{
					Switched:     row.Switched,
					SentThrough:  row.SentThrough,
					AutoSwitched: row.AutoSwitched,
					Kept:         row.Kept,
					PassedOver:   row.PassedOver,
				},
			}
		})
}

type providerCallScan struct {
	ModelKey   string
	ProviderID string
	ErrorClass string
	Calls      int
	FellToNext int
	LastHour   int64
}

func (r *repository) ProviderCalls(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.ProviderCallRow, error) {
	return readMapped(ctx, r, "provider calls", providerCallsQuery(&filter),
		func(row *providerCallScan) aianalyticsports.ProviderCallRow {
			return aianalyticsports.ProviderCallRow{
				DeploymentKey: deploymentKey(row.ModelKey, row.ProviderID),
				Error:         aianalytics.ProviderError(row.ErrorClass),
				Calls:         row.Calls,
				FellToNext:    row.FellToNext,
				LastHour:      row.LastHour,
			}
		})
}

type histogramScan struct {
	ModelKey   string
	ProviderID string
	Bucket     int
	Count      int64
}

func (r *repository) FirstTokenHistograms(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.HistogramRow, error) {
	return readMapped(ctx, r, "first token histograms", firstTokenQuery(&filter),
		func(row *histogramScan) aianalyticsports.HistogramRow {
			return aianalyticsports.HistogramRow{
				DeploymentKey: deploymentKey(row.ModelKey, row.ProviderID),
				Bucket:        row.Bucket,
				Count:         row.Count,
			}
		})
}

type providerHourScan struct {
	ProviderID string
	Hour       int64
	Calls      int
	Failed     int
	AuthFailed int
}

func (r *repository) ProviderHours(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.ProviderHourRow, error) {
	return readMapped(ctx, r, "provider hours", providerHoursQuery(&filter),
		func(row *providerHourScan) aianalyticsports.ProviderHourRow {
			return aianalyticsports.ProviderHourRow{
				ProviderID: row.ProviderID,
				Hour:       row.Hour,
				Calls:      row.Calls,
				Failed:     row.Failed,
				AuthFailed: row.AuthFailed,
			}
		})
}

type providerVersionScan struct {
	ProviderID string
	Version    int64
	Name       string
	Model      string
	Enabled    bool
	CreatedAt  int64
}

func (r *repository) ProviderVersions(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.ProviderVersionRow, error) {
	return readMapped(ctx, r, "provider versions", providerVersionsQuery(&filter),
		func(row *providerVersionScan) aianalyticsports.ProviderVersionRow {
			return aianalyticsports.ProviderVersionRow{
				ProviderID: row.ProviderID,
				Version:    row.Version,
				Name:       row.Name,
				Model:      row.Model,
				Enabled:    row.Enabled,
				CreatedAt:  row.CreatedAt,
			}
		})
}

type causeScan struct {
	Cause      string
	ProviderID string
	Calls      int
}

type exampleScan struct {
	UnitID            string
	Source            string
	ToolName          string
	Message           string
	AgentDefinitionID string
	At                int64
}

func (r *repository) ToolFailures(
	ctx context.Context,
	req aianalyticsports.ToolFailureRequest,
) (*aianalyticsports.ToolFailureRows, error) {
	causes, err := readRows[causeScan](ctx, r, "tool causes", toolCausesQuery(&req))
	if err != nil {
		return nil, err
	}
	var examples []exampleScan
	if req.Limit > 0 {
		examples, err = readRows[exampleScan](ctx, r, "tool examples", toolExamplesQuery(&req))
		if err != nil {
			return nil, err
		}
	}

	out := &aianalyticsports.ToolFailureRows{
		Causes:   make([]aianalyticsports.CauseRow, 0, len(causes)),
		Examples: make([]aianalyticsports.ToolFailureExample, 0, len(examples)),
	}
	byCause := make(map[string]int, len(causes))
	byProvider := make(map[string]int, len(causes))
	for index := range causes {
		row := &causes[index]
		out.Calls += row.Calls
		if row.ProviderID != "" {
			byProvider[row.ProviderID] += row.Calls
		}
		if row.Cause == "" {
			continue
		}
		out.Failed += row.Calls
		byCause[row.Cause] += row.Calls
	}
	for cause, calls := range byCause {
		out.Causes = append(out.Causes, aianalyticsports.CauseRow{
			Cause: aianalytics.Cause(cause),
			Calls: calls,
		})
	}
	out.Providers = slices.Collect(maps.Keys(byProvider))
	slices.SortFunc(out.Providers, func(left, right string) int {
		return cmp.Or(cmp.Compare(byProvider[right], byProvider[left]), cmp.Compare(left, right))
	})
	for index := range examples {
		row := &examples[index]
		example := aianalyticsports.ToolFailureExample{
			UnitID:   row.UnitID,
			Source:   aianalytics.Source(row.Source),
			ToolName: row.ToolName,
			Message:  row.Message,
			At:       row.At,
		}
		if row.AgentDefinitionID != "" {
			example.AgentDefinitionID = pulid.ID(row.AgentDefinitionID)
		}
		out.Examples = append(out.Examples, example)
	}

	return out, nil
}

func (r *repository) Palette(
	ctx context.Context,
	_ pagination.TenantInfo,
	since int64,
) ([]aianalyticsports.DeploymentKey, error) {
	return readMapped(ctx, r, "palette", paletteQuery(since),
		func(row *deploymentScan) aianalyticsports.DeploymentKey { return row.key() })
}
