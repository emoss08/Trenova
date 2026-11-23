package aianalyticsrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/rollupread"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage/aiusagedb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"go.uber.org/zap"
)

const (
	dailyUsageIDPrefix = "aiud_"
	nextLocalDaySpan   = 26 * secondsPerHour
)

var dailyUsageKeyColumns = []string{
	"usage_date", "user_id", "feature", "surface", "model", "provider_kind",
}

var dailyUsageRowColumns = append(append([]string{}, dailyUsageKeyColumns...),
	"failed", "unpriced", "input_tokens", "cache_read_tokens", "cache_write_tokens",
	"output_tokens", "reasoning_tokens", "advisor_tokens", "cost_usd",
)

type dailyUsageColumn interface {
	keel.Column
	keel.Expr
	Excluded() keel.Expr
}

func localDay(epoch keel.NumberValue, zone keel.StringValue) keel.Int64Expr {
	return keel.Epoch(keel.DateTrunc("day", keel.ToTimestamp(epoch).AtTimeZone(zone)).
		AtTimeZone(zone))
}

func dailyUsageWindow(req aianalyticsports.RefreshRequest) (lo, hi keel.Int64Expr) {
	zone := rollupread.OrganizationZone(keel.Text(req.TenantInfo.OrgID.String()))
	lo = localDay(keel.Int64(req.From), zone)
	lastDay := localDay(keel.Int64(max(req.To-1, req.From)), zone)
	hi = localDay(lastDay.Add(keel.ConstInt64(nextLocalDaySpan)), zone)
	return lo, hi
}

func uncachedInput(u *aiusagedb.AIUsageRecordsTable) keel.Int64Expr {
	return keel.CaseInt64().
		When(u.ProviderKind.Eq(aiprovider.KindAnthropicMessages), u.InputTokens).
		Else(u.InputTokens.Sub(u.CacheReadTokens).
			Sub(u.CacheWriteTokens).
			Greatest(keel.ConstInt64(0)))
}

func dailyUsageSource(req aianalyticsports.RefreshRequest) keel.Query[*keel.Row] {
	u := aiusagedb.AIUsageRecords
	lo, hi := dailyUsageWindow(req)
	zone := rollupread.OrganizationZone(keel.Text(req.TenantInfo.OrgID.String()))
	createdAt := keel.Int64Of(u.CreatedAt)

	calls := keel.CTE("usage_calls", dailyUsageRowColumns, u.
		Pick(
			localDay(u.CreatedAt, zone),
			u.UserID,
			u.Feature,
			u.Surface,
			u.Model,
			u.ProviderKind,
			keel.CaseInt64().When(u.Succeeded.IsFalse(), keel.ConstInt64(1)).
				Else(keel.ConstInt64(0)),
			keel.CaseInt64().When(u.CostUSD.IsNull(), keel.ConstInt64(1)).
				Else(keel.ConstInt64(0)),
			uncachedInput(u),
			u.CacheReadTokens,
			u.CacheWriteTokens,
			u.OutputTokens,
			u.ReasoningTokens,
			u.AdvisorInputTokens.
				Add(u.AdvisorCacheReadTokens).
				Add(u.AdvisorCacheWriteTokens).
				Add(u.AdvisorOutputTokens),
			u.CostUSD,
		).
		Where(createdAt.GteExpr(lo), createdAt.LtExpr(hi)))
	col := calls.Col

	day := keel.Int64Of(col("usage_date"))
	key := keel.CastText(day).Cat(
		keel.Text("|"), keel.StringOf(col("user_id")).Coalesce(""),
		keel.Text("|"), keel.StringOf(col("feature")).Coalesce(""),
		keel.Text("|"), keel.StringOf(col("surface")),
		keel.Text("|"), keel.StringOf(col("model")),
		keel.Text("|"), keel.StringOf(col("provider_kind")),
	)
	total := keel.Int64Of(col("input_tokens")).
		Add(keel.Int64Of(col("cache_read_tokens"))).
		Add(keel.Int64Of(col("cache_write_tokens"))).
		Add(keel.Int64Of(col("output_tokens"))).
		Add(keel.Int64Of(col("advisor_tokens")))

	group := make([]keel.Expr, 0, len(dailyUsageKeyColumns))
	for _, name := range dailyUsageKeyColumns {
		group = append(group, col(name))
	}

	picks := []keel.Expr{keel.Text(dailyUsageIDPrefix).Cat(key.MD5())}
	picks = append(picks, group...)
	picks = append(picks,
		keel.Count(),
		keel.CastInt64(summed(col("failed"))),
		keel.CastInt64(summed(col("unpriced"))),
		keel.CastInt64(summed(col("input_tokens"))),
		keel.CastInt64(summed(col("cache_read_tokens"))),
		keel.CastInt64(summed(col("cache_write_tokens"))),
		keel.CastInt64(summed(col("output_tokens"))),
		keel.CastInt64(summed(col("reasoning_tokens"))),
		keel.CastInt64(summed(col("advisor_tokens"))),
		keel.CastInt64(summed(total)),
		summed(col("cost_usd")),
		keel.NowEpoch(),
	)

	return keel.RowsOf(calls.TableRef()).
		With(calls).
		Pick(picks...).
		Group(group...)
}

func dailyUsageConflict() keel.Conflict {
	d := aiusagedb.DailyUsages
	tracked := []dailyUsageColumn{
		d.Calls, d.FailedCalls, d.UnpricedCalls, d.InputTokens, d.CacheReadTokens,
		d.CacheWriteTokens, d.OutputTokens, d.ReasoningTokens, d.AdvisorTokens, d.TotalTokens,
		d.CostUSD,
	}
	updated := make([]keel.Column, 0, len(tracked)+1)
	stored := make([]any, 0, len(tracked))
	proposed := make([]any, 0, len(tracked))
	for _, col := range tracked {
		updated = append(updated, col)
		stored = append(stored, col)
		proposed = append(proposed, col.Excluded())
	}
	updated = append(updated, d.UpdatedAt)

	return keel.OnConflict(d.ID, d.BusinessUnitID, d.OrganizationID).
		Update(updated...).
		UpdateWhere(keel.Tuple(stored...).IsDistinctFrom(keel.Tuple(proposed...)))
}

func (r *repository) RefreshDailyUsage(
	ctx context.Context,
	req aianalyticsports.RefreshRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		if err := r.lockRefresh(ctx, req.TenantInfo); err != nil {
			return 0, err
		}
		d := aiusagedb.DailyUsages
		written, err := d.UpsertFrom(
			ctx,
			r.db.Keel(ctx),
			dailyUsageSource(req),
			dailyUsageConflict(),
			d.ID, d.UsageDate, d.UserID, d.Feature, d.Surface, d.Model, d.ProviderKind,
			d.Calls, d.FailedCalls, d.UnpricedCalls, d.InputTokens, d.CacheReadTokens,
			d.CacheWriteTokens, d.OutputTokens, d.ReasoningTokens, d.AdvisorTokens,
			d.TotalTokens, d.CostUSD, d.UpdatedAt,
		)
		if err != nil {
			r.l.Error("failed to refresh daily ai usage",
				zap.String("organization", req.TenantInfo.OrgID.String()),
				zap.Error(err),
			)
			return 0, fmt.Errorf("refresh daily ai usage: %w", err)
		}
		r.markChanged(ctx, aiusagedb.DailyUsagesSchema())

		return int(written), nil
	})
}
