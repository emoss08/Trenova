package agentextensionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension/agentextensiondb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type usageRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewUsageRepository(p Params) agentextensionports.AgentExtensionUsageRepository {
	return &usageRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-extension-usage-repository"),
	}
}

func reserve(
	params *agentextensionports.ReserveExtensionRequestParams,
) (*keel.RowsTable, keel.Conflict, []keel.Column) {
	u := agentextensiondb.ExtensionUsageDailies
	src := keel.Rows(
		"req",
		[]string{"extension_type", "day", "requests", "failures", "cost_usd", "updated_at"},
		keel.Text(string(params.Type)),
		keel.Int64(int64(params.Day)),
		keel.Int64(1),
		keel.Int64(0),
		keel.Int64(0),
		keel.Int64(params.Now),
	)
	conflict := keel.OnConflict(u.OrganizationID, u.BusinessUnitID, u.ExtensionType, u.Day).
		Set(u.Requests.Add(1), u.UpdatedAt.ToExpr(u.UpdatedAt.Excluded())).
		UpdateWhere(u.Requests.Lt(int64(params.Limit)))

	return src, conflict, []keel.Column{
		u.ExtensionType,
		u.Day,
		u.Requests,
		u.Failures,
		u.CostUSD,
		u.UpdatedAt,
	}
}

func (r *usageRepository) Reserve(
	ctx context.Context,
	params agentextensionports.ReserveExtensionRequestParams,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		u := agentextensiondb.ExtensionUsageDailies
		src, conflict, cols := reserve(&params)
		affected, err := u.UpsertFrom(ctx, r.db.Keel(ctx), src, conflict, cols...)
		if err != nil {
			r.l.Error("failed to reserve agent extension request", zap.Error(err))
			return false, fmt.Errorf("reserve agent extension request: %w", err)
		}

		return affected == 1, nil
	})
}

func recordOutcome(
	params agentextensionports.RecordExtensionOutcomeParams,
) (*keel.Query[*agentextension.ExtensionUsageDaily], []keel.Assignment) {
	u := agentextensiondb.ExtensionUsageDailies
	var failures int64
	if params.Failed {
		failures = 1
	}
	cost := params.CostUSD
	if cost.IsNegative() {
		cost = decimal.Zero
	}

	return u.Where(u.ExtensionType.Eq(params.Type), u.Day.Eq(int64(params.Day))),
		[]keel.Assignment{
			u.Failures.Add(failures),
			u.CostUSD.Add(cost),
			u.UpdatedAt.To(params.Now),
		}
}

func (r *usageRepository) RecordOutcome(
	ctx context.Context,
	params agentextensionports.RecordExtensionOutcomeParams,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		q, sets := recordOutcome(params)
		if _, err := q.Update(ctx, r.db.Keel(ctx), sets...); err != nil {
			r.l.Error("failed to record agent extension outcome", zap.Error(err))
			return fmt.Errorf("record agent extension outcome: %w", err)
		}

		return nil
	})
}

type usageTotals struct {
	ExtensionType     agentextension.Type
	RequestsToday     int
	RequestsThisMonth int
	FailuresThisMonth int
	CostThisMonth     decimal.Decimal
}

func summarize(
	params agentextensionports.SummarizeExtensionUsageParams,
) *keel.Query[*agentextension.ExtensionUsageDaily] {
	u := agentextensiondb.ExtensionUsageDailies

	return u.Group(u.ExtensionType).
		Pick(
			u.ExtensionType,
			keel.NumericOf(u.Requests.Sum().Filter(u.Day.Eq(int64(params.Today)))).Coalesce(0),
			keel.NumericOf(u.Requests.Sum()).Coalesce(0),
			keel.NumericOf(u.Failures.Sum()).Coalesce(0),
			keel.NumericOf(u.CostUSD.Sum()).Coalesce(0),
		).
		Where(u.Day.Gte(int64(params.FromDay)), u.Day.Lte(int64(params.Today)))
}

func (r *usageRepository) Summarize(
	ctx context.Context,
	params agentextensionports.SummarizeExtensionUsageParams,
) (map[agentextension.Type]agentextension.UsageSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[agentextension.Type]agentextension.UsageSummary, error) {
			rows, err := keel.Into[usageTotals](ctx, r.db.Keel(ctx), summarize(params))
			if err != nil {
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
		},
	)
}
