package aianalyticsrepository

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/pkg/buncolgen/aianalyticscol"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun/dialect/pgdialect"
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

type sqlArgs struct {
	Org        string                `bun:"org"`
	Bu         string                `bun:"bu"`
	From       int64                 `bun:"from"`
	To         int64                 `bun:"to"`
	FromHour   int64                 `bun:"from_hour"`
	ToHour     int64                 `bun:"to_hour"`
	Stamp      int64                 `bun:"stamp"`
	Since      int64                 `bun:"since"`
	SinceHour  int64                 `bun:"since_hour"`
	Agent      string                `bun:"agent"`
	Task       string                `bun:"task"`
	Starts     *pgdialect.ArrayValue `bun:"starts"`
	SeriesKey  string                `bun:"series_key"`
	ToolName   string                `bun:"tool_name"`
	Limit      int                   `bun:"limit"`
	Candidates int                   `bun:"candidates"`
	Pending    int64                 `bun:"pending_since"`
	Model      string                `bun:"model"`
	ModelKey   string                `bun:"model_key"`
	Before     int64                 `bun:"before"`
	BeforeHour int64                 `bun:"before_hour"`
}

func tenantArgs(tenant pagination.TenantInfo) sqlArgs {
	return sqlArgs{Org: tenant.OrgID.String(), Bu: tenant.BuID.String()}
}

func filterArgs(filter aianalyticsports.Filter) sqlArgs {
	args := tenantArgs(filter.TenantInfo)
	args.Since = filter.Since
	args.SinceHour = ceilHour(filter.Since)
	args.Agent = filter.AgentDefinitionID.String()
	args.Task = string(filter.Task)

	return args
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
	tenant pagination.TenantInfo,
) (*aianalytics.RefreshState, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*aianalytics.RefreshState, error) {
		state := new(aianalytics.RefreshState)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(state).
			Apply(aianalyticscol.RefreshStateApplyTenant(tenant)).
			Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
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
		cols := aianalyticscol.RefreshStateColumns
		_, err := r.db.DBForContext(ctx).NewInsert().
			Model(state).
			On("CONFLICT (organization_id, business_unit_id) DO UPDATE").
			Set(cols.RefreshedThrough.SetExcluded()).
			Set(cols.RefreshedAt.SetExcluded()).
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to save the ai analytics refresh state", zap.Error(err))
			return fmt.Errorf("save ai analytics refresh state: %w", err)
		}

		return nil
	})
}

func (r *repository) exec(ctx context.Context, query string, args sqlArgs) (int, error) {
	result, err := r.db.DBForContext(ctx).NewRaw(query, &args).Exec(ctx)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count affected rows: %w", err)
	}

	return int(affected), nil
}

func (r *repository) lockRefresh(ctx context.Context, tenant pagination.TenantInfo) error {
	_, err := r.db.DBForContext(ctx).
		NewRaw(
			"SELECT pg_advisory_xact_lock(hashtextextended(?, 0))",
			"ai-analytics-refresh:"+tenant.OrgID.String()+":"+tenant.BuID.String(),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("lock ai analytics refresh: %w", err)
	}

	return nil
}

const activitySQL = `SELECT EXISTS (
		SELECT 1 FROM agent_runs AS o
		WHERE o.organization_id = ?org AND o.business_unit_id = ?bu
			AND o.updated_at >= ?from AND o.updated_at < ?to
	) OR EXISTS (
		SELECT 1 FROM assistant_turns AS o
		WHERE o.organization_id = ?org AND o.business_unit_id = ?bu
			AND o.updated_at >= ?from AND o.updated_at < ?to
	) OR EXISTS (
		SELECT 1 FROM ai_usage_records AS u
		WHERE u.organization_id = ?org AND u.business_unit_id = ?bu
			AND u.created_at >= ?from AND u.created_at < ?to
	) AS active`

func (r *repository) HasActivity(
	ctx context.Context,
	req aianalyticsports.RefreshRequest,
) (bool, error) {
	args := tenantArgs(req.TenantInfo)
	args.From = req.From
	args.To = req.To
	rows, err := scanRaw[activityScan](ctx, r, "activity", activitySQL, args)
	if err != nil {
		return false, err
	}

	return len(rows) > 0 && rows[0].Active, nil
}

type activityScan struct {
	Active bool `bun:"active"`
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

		return written, nil
	})
}

const unkeyedModelsSQL = `SELECT DISTINCT model FROM (
		SELECT w.model FROM ai_work_units AS w
		WHERE w.organization_id = ?org AND w.business_unit_id = ?bu
			AND w.refresh_stamp = ?stamp AND w.model_key = ''
		UNION ALL
		SELECT h.model FROM ai_provider_hourly_stats AS h
		WHERE h.organization_id = ?org AND h.business_unit_id = ?bu
			AND h.hour >= ?from_hour AND h.model_key = ''
	) AS models`

func (r *repository) UnkeyedModels(
	ctx context.Context,
	req aianalyticsports.ModelKeyRequest,
) ([]string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]string, error) {
		args := tenantArgs(req.TenantInfo)
		args.Stamp = req.Stamp
		args.FromHour = req.FromHour

		models := make([]string, 0, 8)
		if err := r.db.DBForContext(ctx).NewRaw(unkeyedModelsSQL, &args).Scan(ctx, &models); err != nil {
			r.l.Error("failed to list unkeyed ai models", zap.Error(err))
			return nil, fmt.Errorf("list unkeyed ai models: %w", err)
		}

		return models, nil
	})
}

const (
	keyUnitsSQL = `UPDATE ai_work_units
	SET model_key = ?model_key
	WHERE organization_id = ?org AND business_unit_id = ?bu
		AND refresh_stamp = ?stamp AND model = ?model AND model_key <> ?model_key`
	keyHoursSQL = `UPDATE ai_provider_hourly_stats
	SET model_key = ?model_key
	WHERE organization_id = ?org AND business_unit_id = ?bu
		AND hour >= ?from_hour AND model = ?model AND model_key <> ?model_key`
	keyToolsSQL = `UPDATE ai_work_unit_tools AS t
	SET model_key = w.model_key
	FROM ai_work_units AS w
	WHERE t.organization_id = ?org AND t.business_unit_id = ?bu
		AND w.organization_id = t.organization_id AND w.business_unit_id = t.business_unit_id
		AND w.id = t.unit_id AND w.refresh_stamp = ?stamp AND t.model_key <> w.model_key`
)

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
		args := tenantArgs(req.TenantInfo)
		args.Stamp = req.Stamp
		args.FromHour = req.FromHour
		for _, key := range keys {
			args.Model = key.Model
			args.ModelKey = key.Key
			if _, err := r.exec(ctx, keyUnitsSQL, args); err != nil {
				r.l.Error("failed to key ai work units", zap.Error(err))
				return fmt.Errorf("key ai work units: %w", err)
			}
			if _, err := r.exec(ctx, keyHoursSQL, args); err != nil {
				r.l.Error("failed to key ai provider hours", zap.Error(err))
				return fmt.Errorf("key ai provider hours: %w", err)
			}
		}
		if _, err := r.exec(ctx, keyToolsSQL, args); err != nil {
			r.l.Error("failed to key ai work unit tools", zap.Error(err))
			return fmt.Errorf("key ai work unit tools: %w", err)
		}

		return nil
	})
}

const (
	pruneUnitsSQL = `DELETE FROM ai_work_units
	WHERE organization_id = ?org AND business_unit_id = ?bu AND finished_at < ?before`
	pruneUnitToolsSQL = `DELETE FROM ai_work_unit_tools
	WHERE organization_id = ?org AND business_unit_id = ?bu AND finished_at < ?before`
	pruneHoursSQL = `DELETE FROM ai_provider_hourly_stats
	WHERE organization_id = ?org AND business_unit_id = ?bu AND hour < ?before_hour`
	pruneSuggestionsSQL = `DELETE FROM agent_suggestions
	WHERE organization_id = ?org AND business_unit_id = ?bu AND created_at < ?before`
)

func (r *repository) Prune(ctx context.Context, req aianalyticsports.PruneRequest) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := r.lockRefresh(ctx, req.TenantInfo); err != nil {
			return err
		}
		args := tenantArgs(req.TenantInfo)
		args.Before = req.Before
		args.BeforeHour = floorHour(req.Before)
		for _, query := range []string{
			pruneUnitToolsSQL,
			pruneUnitsSQL,
			pruneHoursSQL,
			pruneSuggestionsSQL,
		} {
			if _, err := r.exec(ctx, query, args); err != nil {
				r.l.Error("failed to prune ai analytics", zap.Error(err))
				return fmt.Errorf("prune ai analytics: %w", err)
			}
		}

		return nil
	})
}

type deploymentScan struct {
	ModelKey   string `bun:"model_key"`
	ProviderID string `bun:"provider_id"`
}

func (s deploymentScan) key() aianalyticsports.DeploymentKey {
	return aianalyticsports.DeploymentKey{ModelKey: s.ModelKey, ProviderID: s.ProviderID}
}

type countsScan struct {
	Runs              int    `bun:"runs"`
	Completed         int    `bun:"completed"`
	HandedOff         int    `bun:"handed_off"`
	Stopped           int    `bun:"stopped"`
	Failed            int    `bun:"failed"`
	ModelCalls        int    `bun:"model_calls"`
	FailedCalls       int    `bun:"failed_calls"`
	ToolCalls         int    `bun:"tool_calls"`
	ToolFailuresModel int    `bun:"tool_failures_model"`
	ToolFailuresTool  int    `bun:"tool_failures_tool"`
	CostUSD           string `bun:"cost_usd"`
	PricedRuns        int    `bun:"priced_runs"`
	LastRanAt         int64  `bun:"last_ran_at"`
}

func (s *countsScan) counts() aianalyticsports.Counts {
	cost, err := decimal.NewFromString(s.CostUSD)
	if err != nil {
		cost = decimal.Zero
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
	}
}

func durationStats(count int, percentiles []sql.NullFloat64) aianalyticsports.DurationStats {
	stats := aianalyticsports.DurationStats{Count: count}
	if count == 0 || len(percentiles) < 4 {
		return stats
	}
	values := make([]int64, 4)
	for index := range values {
		if !percentiles[index].Valid {
			return stats
		}
		values[index] = int64(percentiles[index].Float64 + 0.5)
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
		rows := make([]T, 0, 16)
		if err := r.db.DBForContext(ctx).NewRaw(query, &args).Scan(ctx, &rows); err != nil {
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
	CompletionCount int               `bun:"completion_count"`
	Completion      []sql.NullFloat64 `bun:"completion,array"`
}

func (r *repository) Totals(
	ctx context.Context,
	filter aianalyticsports.Filter,
) (*aianalyticsports.TotalsRow, error) {
	rows, err := scanRaw[totalsScan](ctx, r, "totals", totalsSQL, filterArgs(filter))
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
	return scanMapped(ctx, r, "deployments", deploymentsSQL, filterArgs(filter),
		func(row *deploymentCountsScan) aianalyticsports.DeploymentRow {
			return aianalyticsports.DeploymentRow{DeploymentKey: row.key(), Counts: row.counts()}
		})
}

type outcomeDurationScan struct {
	deploymentScan
	Outcome     string            `bun:"outcome"`
	Count       int               `bun:"count"`
	Percentiles []sql.NullFloat64 `bun:"percentiles,array"`
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
		filterArgs(filter),
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
	BucketStart  int64           `bun:"bucket_start"`
	Runs         int             `bun:"runs"`
	Completed    int             `bun:"completed"`
	ToolCalls    int             `bun:"tool_calls"`
	ToolFailures int             `bun:"tool_failures"`
	MedianRunMs  sql.NullFloat64 `bun:"median_run_ms"`
}

func (r *repository) Trend(
	ctx context.Context,
	req aianalyticsports.TrendRequest,
) ([]aianalyticsports.TrendRow, error) {
	if len(req.BucketStarts) == 0 {
		return []aianalyticsports.TrendRow{}, nil
	}
	args := filterArgs(req.Filter)
	args.Starts = pgdialect.Array(req.BucketStarts)

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
	ToolName string `bun:"tool_name"`
	Cause    string `bun:"cause"`
	Calls    int    `bun:"calls"`
}

func (r *repository) ToolCells(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.ToolCellRow, error) {
	return scanMapped(ctx, r, "tool cells", toolCellsSQL, filterArgs(filter),
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
	Task      string `bun:"task"`
	Runs      int    `bun:"runs"`
	Completed int    `bun:"completed"`
}

func (r *repository) TaskCells(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.TaskCellRow, error) {
	return scanMapped(ctx, r, "task cells", taskCellsSQL, filterArgs(filter),
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
	Accepted int `bun:"accepted"`
	Modified int `bun:"modified"`
	Rejected int `bun:"rejected"`
}

func (r *repository) Decisions(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.DecisionRow, error) {
	return scanMapped(ctx, r, "decisions", decisionsSQL, filterArgs(filter),
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
	Rated   int `bun:"rated"`
	Helpful int `bun:"helpful"`
}

func (r *repository) Ratings(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.RatingRow, error) {
	return scanMapped(ctx, r, "ratings", ratingsSQL, filterArgs(filter),
		func(row *ratingScan) aianalyticsports.RatingRow {
			return aianalyticsports.RatingRow{
				DeploymentKey: row.key(),
				Rated:         row.Rated,
				Helpful:       row.Helpful,
			}
		})
}

type suggestionScan struct {
	AgentDefinitionID string `bun:"agent_definition_id"`
	AgentName         string `bun:"agent_name"`
	Switched          int    `bun:"switched"`
	SentThrough       int    `bun:"sent_through"`
	AutoSwitched      int    `bun:"auto_switched"`
	Kept              int    `bun:"kept"`
	PassedOver        int    `bun:"passed_over"`
}

func (r *repository) Suggestions(
	ctx context.Context,
	req aianalyticsports.SuggestionRequest,
) ([]aianalyticsports.SuggestionRow, error) {
	args := filterArgs(req.Filter)
	args.Pending = req.PendingSince

	return scanMapped(ctx, r, "suggestions", suggestionsSQL, args,
		func(row *suggestionScan) aianalyticsports.SuggestionRow {
			return aianalyticsports.SuggestionRow{
				AgentDefinitionID: pulid.ID(row.AgentDefinitionID),
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
	ErrorClass string `bun:"error_class"`
	Calls      int    `bun:"calls"`
	FellToNext int    `bun:"fell_to_next"`
	LastHour   int64  `bun:"last_hour"`
}

func (r *repository) ProviderCalls(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.ProviderCallRow, error) {
	return scanMapped(ctx, r, "provider calls", providerCallsSQL, filterArgs(filter),
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
	Bucket int   `bun:"bucket"`
	Count  int64 `bun:"count"`
}

func (r *repository) FirstTokenHistograms(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.HistogramRow, error) {
	return scanMapped(ctx, r, "first token histograms", firstTokenSQL, filterArgs(filter),
		func(row *histogramScan) aianalyticsports.HistogramRow {
			return aianalyticsports.HistogramRow{
				DeploymentKey: row.key(),
				Bucket:        row.Bucket,
				Count:         row.Count,
			}
		})
}

type providerHourScan struct {
	ProviderID string `bun:"provider_id"`
	Hour       int64  `bun:"hour"`
	Calls      int    `bun:"calls"`
	Failed     int    `bun:"failed"`
	AuthFailed int    `bun:"auth_failed"`
}

func (r *repository) ProviderHours(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.ProviderHourRow, error) {
	return scanMapped(ctx, r, "provider hours", providerHoursSQL, filterArgs(filter),
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
	ProviderID string `bun:"provider_id"`
	Version    int64  `bun:"version"`
	Name       string `bun:"name"`
	Model      string `bun:"model"`
	Enabled    bool   `bun:"enabled"`
	CreatedAt  int64  `bun:"created_at"`
}

func (r *repository) ProviderVersions(
	ctx context.Context,
	filter aianalyticsports.Filter,
) ([]aianalyticsports.ProviderVersionRow, error) {
	return scanMapped(ctx, r, "provider versions", providerVersionsSQL, filterArgs(filter),
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
	Cause      string `bun:"cause"`
	ProviderID string `bun:"provider_id"`
	Calls      int    `bun:"calls"`
}

type exampleScan struct {
	UnitID            string `bun:"unit_id"`
	Source            string `bun:"source"`
	ToolName          string `bun:"tool_name"`
	Message           string `bun:"message"`
	AgentDefinitionID string `bun:"agent_definition_id"`
	At                int64  `bun:"at"`
}

func (r *repository) ToolFailures(
	ctx context.Context,
	req aianalyticsports.ToolFailureRequest,
) (*aianalyticsports.ToolFailureRows, error) {
	args := filterArgs(req.Filter)
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
