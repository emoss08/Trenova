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
	"github.com/jackc/pgx/v5"
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

func (r *repository) exec(ctx context.Context, query string, args sqlArgs) (int, error) {
	text, values, err := bind(query, &args)
	if err != nil {
		return 0, err
	}
	affected, err := keel.Exec(ctx, r.db.Keel(ctx), text, values...)
	if err != nil {
		return 0, err
	}

	return int(affected), nil
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
		args := tenantArgs(req.TenantInfo)
		args.From = req.From
		args.To = req.To
		args.Stamp = req.Stamp

		total := 0
		for _, step := range []struct {
			name  string
			query string
		}{
			{"runs", runUnitsSQL},
			{"turns", turnUnitsSQL},
			{"calls", callUnitsSQL},
		} {
			written, err := r.exec(ctx, step.query, args)
			if err != nil {
				r.l.Error("failed to refresh ai work units",
					zap.String("source", step.name),
					zap.String("organization", args.Org),
					zap.Error(err),
				)
				return 0, fmt.Errorf("refresh ai work units from %s: %w", step.name, err)
			}
			total += written
		}

		if _, err := r.exec(ctx, clearUnitToolsSQL, args); err != nil {
			r.l.Error("failed to clear ai work unit tools", zap.Error(err))
			return 0, fmt.Errorf("clear ai work unit tools: %w", err)
		}
		if _, err := r.exec(ctx, insertUnitToolsSQL, args); err != nil {
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
		args := tenantArgs(req.TenantInfo)
		args.FromHour = floorHour(req.From)
		args.ToHour = ceilHour(req.To)

		if _, err := r.exec(ctx, clearProviderHoursSQL, args); err != nil {
			r.l.Error("failed to clear ai provider hours", zap.Error(err))
			return 0, fmt.Errorf("clear ai provider hours: %w", err)
		}
		written, err := r.exec(ctx, insertProviderHoursSQL, args)
		if err != nil {
			r.l.Error("failed to write ai provider hours", zap.Error(err))
			return 0, fmt.Errorf("write ai provider hours: %w", err)
		}
		r.markChanged(ctx, aianalyticsdb.ProviderHoursSchema())

		return written, nil
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
	ModelKey   string `db:"model_key"`
	ProviderID string `db:"provider_id"`
}

func (s deploymentScan) key() aianalyticsports.DeploymentKey {
	return aianalyticsports.DeploymentKey{ModelKey: s.ModelKey, ProviderID: s.ProviderID}
}

type countsScan struct {
	Runs              int    `db:"runs"`
	Completed         int    `db:"completed"`
	HandedOff         int    `db:"handed_off"`
	Stopped           int    `db:"stopped"`
	Failed            int    `db:"failed"`
	ModelCalls        int    `db:"model_calls"`
	FailedCalls       int    `db:"failed_calls"`
	ToolCalls         int    `db:"tool_calls"`
	ToolFailuresModel int    `db:"tool_failures_model"`
	ToolFailuresTool  int    `db:"tool_failures_tool"`
	CostUSD           string `db:"cost_usd"`
	PricedRuns        int    `db:"priced_runs"`
	LastRanAt         int64  `db:"last_ran_at"`
	InputTokens       int64  `db:"input_tokens"`
	OutputTokens      int64  `db:"output_tokens"`
	CacheReadTokens   int64  `db:"cache_read_tokens"`
	CacheWriteTokens  int64  `db:"cache_write_tokens"`
	AdvisorCalls      int64  `db:"advisor_calls"`
	AdvisorPrompt     int64  `db:"advisor_prompt_tokens"`
	AdvisorOutput     int64  `db:"advisor_output_tokens"`
	AdvisorCostUSD    string `db:"advisor_cost_usd"`
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

func scanRaw[T any](
	ctx context.Context,
	r *repository,
	what string,
	query string,
	args sqlArgs,
) ([]T, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]T, error) {
		text, values, err := bind(query, &args)
		if err != nil {
			return nil, err
		}
		rows, err := keel.QueryAll(ctx, r.db.Keel(ctx), pgx.RowToStructByName[T], text, values...)
		if err != nil {
			r.l.Error("failed to read ai analytics", zap.String("what", what), zap.Error(err))
			return nil, fmt.Errorf("read ai analytics %s: %w", what, err)
		}

		return rows, nil
	})
}

func scanMapped[T, R any](
	ctx context.Context,
	r *repository,
	what string,
	query string,
	args sqlArgs,
	mapRow func(*T) R,
) ([]R, error) {
	rows, err := scanRaw[T](ctx, r, what, query, args)
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
	countsScan
	CompletionCount int        `db:"completion_count"`
	Completion      []*float64 `db:"completion"`
}

func (r *repository) Totals(
	ctx context.Context,
	filter aianalyticsports.Filter,
) (*aianalyticsports.TotalsRow, error) {
	rows, err := scanRaw[totalsScan](ctx, r, "totals", totalsSQL, filterArgs(&filter))
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return &aianalyticsports.TotalsRow{
			Counts: aianalyticsports.Counts{CostUSD: decimal.Zero},
		}, nil
	}

	return &aianalyticsports.TotalsRow{
		Counts:     rows[0].counts(),
		Completion: durationStats(rows[0].CompletionCount, rows[0].Completion),
	}, nil
}

type deploymentCountsScan struct {
	deploymentScan
	countsScan
}

func (r *repository) Deployments(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.DeploymentRow, error) {
	return scanMapped(ctx, r, "deployments", deploymentsSQL, filterArgs(&filter),
		func(row *deploymentCountsScan) aianalyticsports.DeploymentRow {
			return aianalyticsports.DeploymentRow{DeploymentKey: row.key(), Counts: row.counts()}
		})
}

type outcomeDurationScan struct {
	deploymentScan
	Outcome     string     `db:"outcome"`
	Count       int        `db:"count"`
	Percentiles []*float64 `db:"percentiles"`
}

func (r *repository) OutcomeDurations(
	ctx context.Context,
	filter aianalyticsports.Filter,
	grouping aianalytics.Grouping,
) ([]aianalyticsports.OutcomeDurationRow, error) {
	return scanMapped(
		ctx,
		r,
		"outcome durations",
		outcomeDurationsSQL(grouping),
		filterArgs(&filter),
		func(row *outcomeDurationScan) aianalyticsports.OutcomeDurationRow {
			return aianalyticsports.OutcomeDurationRow{
				DeploymentKey: row.key(),
				Outcome:       aianalytics.Outcome(row.Outcome),
				DurationStats: durationStats(row.Count, row.Percentiles),
			}
		},
	)
}

type trendScan struct {
	deploymentScan
	BucketStart  int64           `db:"bucket_start"`
	Runs         int             `db:"runs"`
	Completed    int             `db:"completed"`
	ToolCalls    int             `db:"tool_calls"`
	ToolFailures int             `db:"tool_failures"`
	MedianRunMs  sql.NullFloat64 `db:"median_run_ms"`
}

func (r *repository) Trend(
	ctx context.Context,
	req aianalyticsports.TrendRequest,
) ([]aianalyticsports.TrendRow, error) {
	if len(req.BucketStarts) == 0 {
		return []aianalyticsports.TrendRow{}, nil
	}
	args := filterArgs(&req.Filter)
	args.Starts = req.BucketStarts

	return scanMapped(ctx, r, "trend", trendSQL(req.Grouping), args,
		func(row *trendScan) aianalyticsports.TrendRow {
			return aianalyticsports.TrendRow{
				DeploymentKey: row.key(),
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
	deploymentScan
	ToolName string `db:"tool_name"`
	Cause    string `db:"cause"`
	Calls    int    `db:"calls"`
}

func (r *repository) ToolCells(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.ToolCellRow, error) {
	return scanMapped(ctx, r, "tool cells", toolCellsSQL, filterArgs(&filter),
		func(row *toolCellScan) aianalyticsports.ToolCellRow {
			return aianalyticsports.ToolCellRow{
				DeploymentKey: row.key(),
				ToolName:      row.ToolName,
				Cause:         aianalytics.Cause(row.Cause),
				Calls:         row.Calls,
			}
		})
}

type taskCellScan struct {
	deploymentScan
	Task      string `db:"task"`
	Runs      int    `db:"runs"`
	Completed int    `db:"completed"`
}

func (r *repository) TaskCells(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.TaskCellRow, error) {
	return scanMapped(ctx, r, "task cells", taskCellsSQL, filterArgs(&filter),
		func(row *taskCellScan) aianalyticsports.TaskCellRow {
			return aianalyticsports.TaskCellRow{
				DeploymentKey: row.key(),
				Task:          aiprovider.Task(row.Task),
				Runs:          row.Runs,
				Completed:     row.Completed,
			}
		})
}

type decisionScan struct {
	deploymentScan
	Accepted int `db:"accepted"`
	Modified int `db:"modified"`
	Rejected int `db:"rejected"`
}

func (r *repository) Decisions(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.DecisionRow, error) {
	return scanMapped(ctx, r, "decisions", decisionsSQL, filterArgs(&filter),
		func(row *decisionScan) aianalyticsports.DecisionRow {
			return aianalyticsports.DecisionRow{
				DeploymentKey: row.key(),
				Accepted:      row.Accepted,
				Modified:      row.Modified,
				Rejected:      row.Rejected,
			}
		})
}

type ratingScan struct {
	deploymentScan
	Rated   int `db:"rated"`
	Helpful int `db:"helpful"`
}

func (r *repository) Ratings(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.RatingRow, error) {
	return scanMapped(ctx, r, "ratings", ratingsSQL, filterArgs(&filter),
		func(row *ratingScan) aianalyticsports.RatingRow {
			return aianalyticsports.RatingRow{
				DeploymentKey: row.key(),
				Rated:         row.Rated,
				Helpful:       row.Helpful,
			}
		})
}

type suggestionScan struct {
	AgentDefinitionID pulid.ID `db:"agent_definition_id"`
	AgentName         string   `db:"agent_name"`
	Switched          int      `db:"switched"`
	SentThrough       int      `db:"sent_through"`
	AutoSwitched      int      `db:"auto_switched"`
	Kept              int      `db:"kept"`
	PassedOver        int      `db:"passed_over"`
}

func (r *repository) Suggestions(
	ctx context.Context,
	req *aianalyticsports.SuggestionRequest,
) ([]aianalyticsports.SuggestionRow, error) {
	args := filterArgs(&req.Filter)
	args.Pending = req.PendingSince

	return scanMapped(ctx, r, "suggestions", suggestionsSQL, args,
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
	deploymentScan
	ErrorClass string `db:"error_class"`
	Calls      int    `db:"calls"`
	FellToNext int    `db:"fell_to_next"`
	LastHour   int64  `db:"last_hour"`
}

func (r *repository) ProviderCalls(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.ProviderCallRow, error) {
	return scanMapped(ctx, r, "provider calls", providerCallsSQL, filterArgs(&filter),
		func(row *providerCallScan) aianalyticsports.ProviderCallRow {
			return aianalyticsports.ProviderCallRow{
				DeploymentKey: row.key(),
				Error:         aianalytics.ProviderError(row.ErrorClass),
				Calls:         row.Calls,
				FellToNext:    row.FellToNext,
				LastHour:      row.LastHour,
			}
		})
}

type histogramScan struct {
	deploymentScan
	Bucket int   `db:"bucket"`
	Count  int64 `db:"count"`
}

func (r *repository) FirstTokenHistograms(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.HistogramRow, error) {
	return scanMapped(ctx, r, "first token histograms", firstTokenSQL, filterArgs(&filter),
		func(row *histogramScan) aianalyticsports.HistogramRow {
			return aianalyticsports.HistogramRow{
				DeploymentKey: row.key(),
				Bucket:        row.Bucket,
				Count:         row.Count,
			}
		})
}

type providerHourScan struct {
	ProviderID string `db:"provider_id"`
	Hour       int64  `db:"hour"`
	Calls      int    `db:"calls"`
	Failed     int    `db:"failed"`
	AuthFailed int    `db:"auth_failed"`
}

func (r *repository) ProviderHours(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.ProviderHourRow, error) {
	return scanMapped(ctx, r, "provider hours", providerHoursSQL, filterArgs(&filter),
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
	ProviderID string `db:"provider_id"`
	Version    int64  `db:"version"`
	Name       string `db:"name"`
	Model      string `db:"model"`
	Enabled    bool   `db:"enabled"`
	CreatedAt  int64  `db:"created_at"`
}

func (r *repository) ProviderVersions(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.ProviderVersionRow, error) {
	return scanMapped(ctx, r, "provider versions", providerVersionsSQL, filterArgs(&filter),
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
	Cause      string `db:"cause"`
	ProviderID string `db:"provider_id"`
	Calls      int    `db:"calls"`
}

type exampleScan struct {
	UnitID            string `db:"unit_id"`
	Source            string `db:"source"`
	ToolName          string `db:"tool_name"`
	Message           string `db:"message"`
	AgentDefinitionID string `db:"agent_definition_id"`
	At                int64  `db:"at"`
}

func (r *repository) ToolFailures(
	ctx context.Context,
	req aianalyticsports.ToolFailureRequest,
) (*aianalyticsports.ToolFailureRows, error) {
	args := filterArgs(&req.Filter)
	args.SeriesKey = req.SeriesKey
	args.ToolName = req.ToolName
	args.Limit = req.Limit
	args.Candidates = req.Limit * toolExampleCandidatesPerExample

	causes, err := scanRaw[causeScan](ctx, r, "tool causes", toolCausesSQL(req.CompareBy), args)
	if err != nil {
		return nil, err
	}
	examples, err := scanRaw[exampleScan](
		ctx,
		r,
		"tool examples",
		toolExamplesSQL(req.CompareBy),
		args,
	)
	if err != nil {
		return nil, err
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
	tenant pagination.TenantInfo,
	since int64,
) ([]aianalyticsports.DeploymentKey, error) {
	args := tenantArgs(tenant)
	args.SinceHour = floorHour(since)

	return scanMapped(ctx, r, "palette", paletteSQL, args,
		func(row *deploymentScan) aianalyticsports.DeploymentKey { return row.key() })
}
