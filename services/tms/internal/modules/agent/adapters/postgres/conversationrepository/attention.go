package conversationrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/jackc/pgx/v5"
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

const threadAttentionQuery = `
SELECT
	t.id AS thread_id,
	(
		SELECT count(*)
		FROM agent_plans AS pl
		JOIN agent_runs AS r
			ON r.id = pl.run_id
			AND r.organization_id = pl.organization_id
			AND r.business_unit_id = pl.business_unit_id
		WHERE pl.organization_id = t.organization_id
			AND pl.business_unit_id = t.business_unit_id
			AND pl.status = $1
			AND r.subject_type = $2
			AND r.subject_id = t.id
	) AS pending_plans,
	COALESCE((
		SELECT tr.status
		FROM assistant_turns AS tr
		WHERE tr.organization_id = t.organization_id
			AND tr.business_unit_id = t.business_unit_id
			AND tr.thread_id = t.id
		ORDER BY tr.created_at DESC, tr.id DESC
		LIMIT 1
	), '') AS last_turn_status
FROM assistant_threads AS t
WHERE t.organization_id = $3
	AND t.business_unit_id = $4
	AND t.user_id = $5
	AND t.id = ANY($6::text[])
`

const pendingProposalRunsQuery = `
SELECT r.subject_id AS thread_id, p.run_id AS run_id
FROM agent_proposals AS p
JOIN agent_runs AS r
	ON r.id = p.run_id
	AND r.organization_id = p.organization_id
	AND r.business_unit_id = p.business_unit_id
WHERE p.organization_id = $1
	AND p.business_unit_id = $2
	AND p.status = $3
	AND p.plan_id IS NULL
	AND (p.expires_at IS NULL OR p.expires_at = 0
		OR p.expires_at > extract(epoch from current_timestamp)::bigint)
	AND r.subject_type = $4
	AND r.subject_id = ANY($5::text[])
`

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
			rows, err := keel.QueryAll(
				ctx,
				db,
				pgx.RowToStructByName[threadAttentionRow],
				threadAttentionQuery,
				string(agent.PlanStatusPending),
				string(agent.SubjectAssistantThread),
				req.TenantInfo.OrgID.String(),
				req.TenantInfo.BuID.String(),
				req.UserID.String(),
				sliceutils.Strings(req.ThreadIDs),
			)
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

			runs, err := keel.QueryAll(
				ctx,
				db,
				pgx.RowToStructByName[pendingProposalRunRow],
				pendingProposalRunsQuery,
				req.TenantInfo.OrgID.String(),
				req.TenantInfo.BuID.String(),
				string(agent.ProposalStatusPending),
				string(agent.SubjectAssistantThread),
				sliceutils.Strings(owned),
			)
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
