package aicontrolfactsrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentroster"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicontrolsummary"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage/aiusagedb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentrosterports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) agentrosterports.AIControlFactsRepository {
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
	req *agentrosterports.AgentCountsRequest,
) (aicontrolsummary.AgentCounts, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (aicontrolsummary.AgentCounts, error) {
		d := agentdefinitiondb.Definitions
		run := agentdb.AgentRuns
		p := agentdb.AgentProposals
		shadowRun := agentdb.AgentRuns.As("shadow_run")

		var (
			agents                           definitionCounts
			working, waiting, shadowRecorded int64
		)
		err := keel.Batch(ctx, r.db.Keel(ctx),
			keel.IntoValue(d.Pick(
				keel.Count(),
				keel.Count().Filter(d.Enabled.IsTrue()),
				keel.Count().Filter(keel.And(d.Enabled.IsTrue(), d.ShadowMode.IsTrue())),
			), &agents),
			keel.CountInto(run.Where(run.Status.In(workingStatuses...)), &working),
			keel.CountInto(p.Where(p.Status.Eq(agent.ProposalStatusPending)), &waiting),
			keel.CountInto(p.
				Join(shadowRun, shadowRun.ID.EqCol(p.RunID)).
				Where(
					shadowRun.Status.Eq(agent.RunStatusShadowCompleted),
					shadowRun.CreatedAt.Gte(req.ShadowSince),
				), &shadowRecorded),
		)
		if err != nil {
			return aicontrolsummary.AgentCounts{}, fmt.Errorf("count agents: %w", err)
		}

		return aicontrolsummary.AgentCounts{
			Total:          int(agents.Total),
			On:             int(agents.On),
			Shadow:         int(agents.Shadow),
			Working:        int(working),
			Waiting:        int(waiting),
			ShadowRecorded: int(shadowRecorded),
		}, nil
	})
}

type definitionCounts struct {
	Total  int64
	On     int64
	Shadow int64
}

type failureRow struct {
	ProviderID    string
	FailedCalls   int
	LastFailureAt int64
}

func (r *repository) ProviderFailures(
	ctx context.Context,
	req *agentrosterports.ProviderFailuresRequest,
) ([]agentrosterports.ProviderFailureCount, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]agentrosterports.ProviderFailureCount, error) {
			u := aiusagedb.AIUsageRecords
			lastFailure := keel.DynOf(u.CreatedAt).
				Max().
				Filter(keel.DynOf(u.Succeeded).Not()).
				Coalesce(0)
			lastSuccess := keel.DynOf(u.CreatedAt).
				Max().
				Filter(keel.DynOf(u.Succeeded)).
				Coalesce(0)
			rows, err := keel.Into[failureRow](ctx, r.db.Keel(ctx), u.
				Where(u.CreatedAt.Gte(req.Since), u.ProviderID.IsNotNull()).
				Group(u.ProviderID).
				Having(lastFailure.Gt(lastSuccess)).
				Pick(u.ProviderID, keel.Count().Filter(u.Succeeded.IsFalse()), lastFailure))
			if err != nil {
				return nil, fmt.Errorf("list failing providers: %w", err)
			}

			out := make([]agentrosterports.ProviderFailureCount, 0, len(rows))
			for idx := range rows {
				id, parseErr := pulid.Parse(rows[idx].ProviderID)
				if parseErr != nil {
					continue
				}
				out = append(out, agentrosterports.ProviderFailureCount{
					ProviderID:    id,
					FailedCalls:   rows[idx].FailedCalls,
					LastFailureAt: rows[idx].LastFailureAt,
				})
			}
			return out, nil
		},
	)
}

type rosterRunRow struct {
	AgentID string
	Day     int
	Runs    int
}

type rosterDecisionRow struct {
	AgentID  string
	Approved int
	Modified int
	Rejected int
	Failed   int
	Shadow   int
}

func (r *repository) AgentRoster(
	ctx context.Context,
	req *agentrosterports.AgentRosterRequest,
) ([]*agentroster.Stat, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentroster.Stat, error) {
		run := agentdb.AgentRuns
		p := agentdb.AgentProposals
		decidedRun := agentdb.AgentRuns.As("decided_run")
		day := keel.DynFunc("floor", keel.DynOf(run.CreatedAt).Sub(req.RunsSince).Div(86400)).
			Cast(keel.TypeInt64)

		var (
			runs      []rosterRunRow
			decisions []rosterDecisionRow
		)
		err := keel.Batch(ctx, r.db.Keel(ctx),
			keel.IntoSlice(run.
				Where(run.AgentDefinitionID.IsNotNull(), run.CreatedAt.Gte(req.RunsSince)).
				Group(run.AgentDefinitionID, day).
				Pick(
					run.AgentDefinitionID,
					day,
					keel.Count(),
				), &runs),
			keel.IntoSlice(p.
				Join(decidedRun, decidedRun.ID.EqCol(p.RunID)).
				Where(decidedRun.AgentDefinitionID.IsNotNull(), p.CreatedAt.Gte(req.DecisionsSince)).
				Group(decidedRun.AgentDefinitionID).
				Pick(
					decidedRun.AgentDefinitionID,
					keel.Count().Filter(p.Status.Eq(agent.ProposalStatusAccepted)),
					keel.Count().Filter(p.Status.Eq(agent.ProposalStatusModified)),
					keel.Count().Filter(p.Status.Eq(agent.ProposalStatusRejected)),
					keel.Count().Filter(p.Status.Eq(agent.ProposalStatusExecutionFailed)),
					keel.Count().Filter(decidedRun.Status.Eq(agent.RunStatusShadowCompleted)),
				), &decisions),
		)
		if err != nil {
			return nil, fmt.Errorf("count agent runs and decisions: %w", err)
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
