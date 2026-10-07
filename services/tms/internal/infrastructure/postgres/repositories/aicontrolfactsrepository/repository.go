package aicontrolfactsrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/agentroster"
	"github.com/emoss08/trenova/internal/core/domain/aicontrolsummary"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) repositories.AIControlFactsRepository {
	return &repository{db: p.DB}
}

// workingStatuses are the run statuses of a run still going.
var workingStatuses = []agent.RunStatus{
	agent.RunStatusPending,
	agent.RunStatusGatheringContext,
	agent.RunStatusDiagnosing,
}

func (r *repository) AgentCounts(
	ctx context.Context,
	req *repositories.AgentCountsRequest,
) (aicontrolsummary.AgentCounts, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (aicontrolsummary.AgentCounts, error) {
		db := r.db.DBForContext(ctx)
		defCols := buncolgen.DefinitionColumns
		runCols := buncolgen.AgentRunColumns
		propCols := buncolgen.AgentProposalColumns
		var counts aicontrolsummary.AgentCounts

		definitions := func() *bun.SelectQuery {
			return db.NewSelect().
				Model((*agentdefinition.Definition)(nil)).
				Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
					return buncolgen.DefinitionScopeTenant(sq, req.TenantInfo)
				})
		}

		var err error
		if counts.Total, err = definitions().Count(ctx); err != nil {
			return counts, fmt.Errorf("count agents: %w", err)
		}
		if counts.On, err = definitions().Where(defCols.Enabled.IsTrue()).Count(ctx); err != nil {
			return counts, fmt.Errorf("count agents on: %w", err)
		}
		if counts.Shadow, err = definitions().
			Where(defCols.Enabled.IsTrue()).
			Where(defCols.ShadowMode.IsTrue()).
			Count(ctx); err != nil {
			return counts, fmt.Errorf("count agents in shadow: %w", err)
		}
		if counts.Working, err = db.NewSelect().
			Model((*agent.AgentRun)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.AgentRunScopeTenant(sq, req.TenantInfo)
			}).
			Where(runCols.Status.In(), bun.List(workingStatuses)).
			Count(ctx); err != nil {
			return counts, fmt.Errorf("count working runs: %w", err)
		}
		if counts.Waiting, err = db.NewSelect().
			Model((*agent.AgentProposal)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.AgentProposalScopeTenant(sq, req.TenantInfo)
			}).
			Where(propCols.Status.Eq(), agent.ProposalStatusPending).
			Count(ctx); err != nil {
			return counts, fmt.Errorf("count waiting proposals: %w", err)
		}
		if counts.ShadowRecorded, err = db.NewSelect().
			Model((*agent.AgentProposal)(nil)).
			Join("JOIN agent_runs AS ar ON "+runCols.ID.Qualified()+" = "+propCols.RunID.Qualified()+
				" AND "+runCols.OrganizationID.Qualified()+" = "+propCols.OrganizationID.Qualified()+
				" AND "+runCols.BusinessUnitID.Qualified()+" = "+propCols.BusinessUnitID.Qualified()).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.AgentProposalScopeTenant(sq, req.TenantInfo)
			}).
			Where(runCols.Status.Eq(), agent.RunStatusShadowCompleted).
			Where(runCols.CreatedAt.Gte(), req.ShadowSince).
			Count(ctx); err != nil {
			return counts, fmt.Errorf("count shadow recordings: %w", err)
		}

		return counts, nil
	})
}

type failureRow struct {
	ProviderID    string `bun:"provider_id"`
	FailedCalls   int    `bun:"failed_calls"`
	LastFailureAt int64  `bun:"last_failure_at"`
}

func (r *repository) ProviderFailures(
	ctx context.Context,
	req *repositories.ProviderFailuresRequest,
) ([]repositories.ProviderFailureCount, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]repositories.ProviderFailureCount, error) {
		cols := buncolgen.AIUsageRecordColumns
		var rows []failureRow
		err := r.db.DBForContext(ctx).NewSelect().
			Model((*aiusage.AIUsageRecord)(nil)).
			ColumnExpr(cols.ProviderID.Qualified()+" AS provider_id").
			ColumnExpr("COUNT(*) FILTER (WHERE NOT "+cols.Succeeded.Qualified()+") AS failed_calls").
			ColumnExpr("COALESCE(MAX("+cols.CreatedAt.Qualified()+") FILTER (WHERE NOT "+
				cols.Succeeded.Qualified()+"), 0) AS last_failure_at").
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.AIUsageRecordScopeTenant(sq, req.TenantInfo)
			}).
			Where(cols.CreatedAt.Gte(), req.Since).
			Where(cols.ProviderID.IsNotNull()).
			GroupExpr(cols.ProviderID.Qualified()).
			Having("COALESCE(MAX("+cols.CreatedAt.Qualified()+") FILTER (WHERE NOT "+cols.Succeeded.Qualified()+"), 0) > "+
				"COALESCE(MAX("+cols.CreatedAt.Qualified()+") FILTER (WHERE "+cols.Succeeded.Qualified()+"), 0)").
			Scan(ctx, &rows)
		if err != nil {
			return nil, fmt.Errorf("list failing providers: %w", err)
		}

		out := make([]repositories.ProviderFailureCount, 0, len(rows))
		for idx := range rows {
			id, parseErr := pulid.Parse(rows[idx].ProviderID)
			if parseErr != nil {
				continue
			}
			out = append(out, repositories.ProviderFailureCount{
				ProviderID:    id,
				FailedCalls:   rows[idx].FailedCalls,
				LastFailureAt: rows[idx].LastFailureAt,
			})
		}
		return out, nil
	})
}

type rosterRunRow struct {
	AgentID string `bun:"agent_id"`
	Day     int    `bun:"day"`
	Runs    int    `bun:"runs"`
}

type rosterDecisionRow struct {
	AgentID  string `bun:"agent_id"`
	Approved int    `bun:"approved"`
	Modified int    `bun:"modified"`
	Rejected int    `bun:"rejected"`
	Failed   int    `bun:"failed"`
	Shadow   int    `bun:"shadow"`
}

func (r *repository) AgentRoster(
	ctx context.Context,
	req *repositories.AgentRosterRequest,
) ([]*agentroster.Stat, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentroster.Stat, error) {
		db := r.db.DBForContext(ctx)
		runCols := buncolgen.AgentRunColumns
		propCols := buncolgen.AgentProposalColumns

		var runs []rosterRunRow
		err := db.NewSelect().
			Model((*agent.AgentRun)(nil)).
			ColumnExpr(runCols.AgentDefinitionID.Qualified()+" AS agent_id").
			ColumnExpr("FLOOR(("+runCols.CreatedAt.Qualified()+" - ?) / 86400)::int AS day", req.RunsSince).
			ColumnExpr(buncolgen.Count("runs")).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.AgentRunScopeTenant(sq, req.TenantInfo)
			}).
			Where(runCols.AgentDefinitionID.IsNotNull()).
			Where(runCols.CreatedAt.Gte(), req.RunsSince).
			GroupExpr("1, 2").
			Scan(ctx, &runs)
		if err != nil {
			return nil, fmt.Errorf("count agent runs by day: %w", err)
		}

		var decisions []rosterDecisionRow
		err = db.NewSelect().
			Model((*agent.AgentProposal)(nil)).
			Join("JOIN agent_runs AS ar ON "+runCols.ID.Qualified()+" = "+propCols.RunID.Qualified()+
				" AND "+runCols.OrganizationID.Qualified()+" = "+propCols.OrganizationID.Qualified()+
				" AND "+runCols.BusinessUnitID.Qualified()+" = "+propCols.BusinessUnitID.Qualified()).
			ColumnExpr(runCols.AgentDefinitionID.Qualified()+" AS agent_id").
			ColumnExpr(buncolgen.CountFilter("approved", propCols.Status.Eq()), agent.ProposalStatusAccepted).
			ColumnExpr(buncolgen.CountFilter("modified", propCols.Status.Eq()), agent.ProposalStatusModified).
			ColumnExpr(buncolgen.CountFilter("rejected", propCols.Status.Eq()), agent.ProposalStatusRejected).
			ColumnExpr(
				buncolgen.CountFilter("failed", propCols.Status.Eq()),
				agent.ProposalStatusExecutionFailed,
			).
			ColumnExpr(buncolgen.CountFilter("shadow", runCols.Status.Eq()), agent.RunStatusShadowCompleted).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.AgentProposalScopeTenant(sq, req.TenantInfo)
			}).
			Where(runCols.AgentDefinitionID.IsNotNull()).
			Where(propCols.CreatedAt.Gte(), req.DecisionsSince).
			GroupExpr(runCols.AgentDefinitionID.Qualified()).
			Scan(ctx, &decisions)
		if err != nil {
			return nil, fmt.Errorf("count agent decisions: %w", err)
		}

		return assembleRoster(runs, decisions), nil
	})
}

func assembleRoster(runs []rosterRunRow, decisions []rosterDecisionRow) []*agentroster.Stat {
	byAgent := make(map[pulid.ID]*agentroster.Stat, len(decisions))
	order := make([]pulid.ID, 0, len(decisions))
	stat := func(raw string) *agentroster.Stat {
		id, err := pulid.Parse(raw)
		if err != nil {
			return nil
		}
		found, ok := byAgent[id]
		if !ok {
			found = agentroster.NewStat(id)
			byAgent[id] = found
			order = append(order, id)
		}
		return found
	}

	for idx := range runs {
		if found := stat(runs[idx].AgentID); found != nil {
			found.AddRuns(runs[idx].Day, runs[idx].Runs)
		}
	}
	for idx := range decisions {
		row := &decisions[idx]
		if found := stat(row.AgentID); found != nil {
			found.Approved = row.Approved
			found.Modified = row.Modified
			found.Rejected = row.Rejected
			found.Failed = row.Failed
			found.ShadowRecorded = row.Shadow
		}
	}

	out := make([]*agentroster.Stat, 0, len(order))
	for _, id := range order {
		out = append(out, byAgent[id])
	}
	return out
}
