package conversationrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
)

type threadAttentionRow struct {
	ThreadID       pulid.ID
	PendingPlans   int
	LastTurnStatus conversation.AssistantTurnStatus
}

type pendingProposalRunRow struct {
	ThreadID pulid.ID
	RunID    pulid.ID
}

func threadAttentionQuery(
	req *agentports.ListThreadAttentionRequest,
) *keel.Query[*conversation.Thread] {
	th := conversationdb.Threads
	pl := agentdb.AgentPlans
	run := agentdb.AgentRuns
	tr := conversationdb.AssistantTurns

	pendingPlans := keel.Scalar(pl.Join(run, run.ID.EqCol(pl.RunID)).
		Pick(keel.Count()).
		Where(
			pl.Status.Eq(agent.PlanStatusPending),
			run.SubjectType.Eq(agent.SubjectAssistantThread),
			run.SubjectID.EqCol(th.ID),
		))
	lastTurnStatus := keel.DynOf(keel.Scalar(tr.Pick(tr.Status).
		Where(tr.ThreadID.EqCol(th.ID)).
		Order(tr.CreatedAt.Desc(), tr.ID.Desc()).
		Limit(1))).Coalesce("")

	return th.Pick(th.ID, pendingPlans, lastTurnStatus).
		Where(th.UserID.Eq(req.UserID), th.ID.In(req.ThreadIDs...))
}

func pendingProposalRunsQuery(threadIDs []pulid.ID) *keel.Query[*agent.AgentProposal] {
	p := agentdb.AgentProposals
	run := agentdb.AgentRuns
	now := keel.DynFunc("date_part", "epoch", keel.DynFunc("now")).Cast(keel.TypeInt64)

	return p.Join(run, run.ID.EqCol(p.RunID)).
		Pick(run.SubjectID, p.RunID).
		Where(
			p.Status.Eq(agent.ProposalStatusPending),
			p.PlanID.IsNull(),
			keel.Or(p.ExpiresAt.IsNull(), p.ExpiresAt.Eq(0), keel.DynOf(p.ExpiresAt).Gt(now)),
			run.SubjectType.Eq(agent.SubjectAssistantThread),
			run.SubjectID.In(threadIDs...),
		)
}

func (r *repository) ListThreadAttention(
	ctx context.Context,
	req agentports.ListThreadAttentionRequest,
) (map[pulid.ID]agentports.ThreadAttentionRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]agentports.ThreadAttentionRow, error) {
			out := make(map[pulid.ID]agentports.ThreadAttentionRow, len(req.ThreadIDs))
			if len(req.ThreadIDs) == 0 {
				return out, nil
			}

			db := r.db.Keel(ctx)
			rows, err := keel.Into[threadAttentionRow](ctx, db, threadAttentionQuery(&req))
			if err != nil {
				return nil, fmt.Errorf("list thread attention: %w", err)
			}

			owned := make([]pulid.ID, 0, len(rows))
			for _, row := range rows {
				owned = append(owned, row.ThreadID)
				out[row.ThreadID] = agentports.ThreadAttentionRow{
					PendingPlans:   row.PendingPlans,
					LastTurnStatus: row.LastTurnStatus,
				}
			}
			if len(owned) == 0 {
				return out, nil
			}

			runs, err := keel.Into[pendingProposalRunRow](ctx, db, pendingProposalRunsQuery(owned))
			if err != nil {
				return nil, fmt.Errorf("list pending proposal runs: %w", err)
			}

			for _, run := range runs {
				row := out[run.ThreadID]
				row.PendingProposalRuns = append(row.PendingProposalRuns, run.RunID)
				out[run.ThreadID] = row
			}

			return out, nil
		},
	)
}

func (r *repository) MarkThreadRead(
	ctx context.Context,
	req agentports.MarkThreadReadRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		th := conversationdb.Threads
		n, err := th.Where(th.ID.Eq(req.ThreadID), th.UserID.Eq(req.UserID)).Update(
			ctx,
			r.db.Keel(ctx),
			th.LastReadAt.ToExpr(keel.DynFunc("greatest", th.LastReadAt, req.ReadAt)),
			keepThreadVersion(),
			keepThreadUpdatedAt(),
		)
		if err != nil {
			return fmt.Errorf("mark thread read: %w", err)
		}

		return dberror.CheckAffected(n, "Thread", req.ThreadID.String())
	})
}

func (r *repository) WakeThread(
	ctx context.Context,
	req *agentports.WakeThreadRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		th := conversationdb.Threads
		n, err := th.Where(
			th.ID.Eq(req.ThreadID),
			th.UserID.Eq(req.UserID),
			th.SnoozedUntil.IsNotNull(),
		).Update(
			ctx,
			r.db.Keel(ctx),
			th.SnoozedUntil.ToNull(),
			th.SnoozeAnchor.ToNull(),
			th.SnoozeStopID.ToNull(),
		)
		if err != nil {
			return false, fmt.Errorf("wake thread: %w", err)
		}

		return n > 0, nil
	})
}
