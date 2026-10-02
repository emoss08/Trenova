package conversationrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type threadAttentionRow struct {
	ThreadID       pulid.ID                         `bun:"thread_id"`
	PendingPlans   int                              `bun:"pending_plans"`
	LastTurnStatus conversation.AssistantTurnStatus `bun:"last_turn_status"`
}

type pendingProposalRunRow struct {
	ThreadID pulid.ID `bun:"thread_id"`
	RunID    pulid.ID `bun:"run_id"`
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
			AND pl.status = ?
			AND r.subject_type = ?
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
WHERE t.organization_id = ?
	AND t.business_unit_id = ?
	AND t.user_id = ?
	AND t.id IN (?)
`

const pendingProposalRunsQuery = `
SELECT r.subject_id AS thread_id, p.run_id AS run_id
FROM agent_proposals AS p
JOIN agent_runs AS r
	ON r.id = p.run_id
	AND r.organization_id = p.organization_id
	AND r.business_unit_id = p.business_unit_id
WHERE p.organization_id = ?
	AND p.business_unit_id = ?
	AND p.status = ?
	AND p.plan_id IS NULL
	AND (p.expires_at IS NULL OR p.expires_at = 0 OR p.expires_at > extract(epoch from now())::bigint)
	AND r.subject_type = ?
	AND r.subject_id IN (?)
`

func (r *repository) ListThreadAttention(
	ctx context.Context,
	req repositories.ListThreadAttentionRequest,
) (map[pulid.ID]repositories.ThreadAttentionRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]repositories.ThreadAttentionRow, error) {
			out := make(map[pulid.ID]repositories.ThreadAttentionRow, len(req.ThreadIDs))
			if len(req.ThreadIDs) == 0 {
				return out, nil
			}

			db := r.db.DBForContext(ctx)
			rows := make([]threadAttentionRow, 0, len(req.ThreadIDs))
			if err := db.NewRaw(
				threadAttentionQuery,
				agent.PlanStatusPending,
				agent.SubjectAssistantThread,
				req.TenantInfo.OrgID,
				req.TenantInfo.BuID,
				req.UserID,
				bun.In(req.ThreadIDs),
			).Scan(ctx, &rows); err != nil {
				return nil, fmt.Errorf("list thread attention: %w", err)
			}

			owned := make([]pulid.ID, 0, len(rows))
			for _, row := range rows {
				owned = append(owned, row.ThreadID)
				out[row.ThreadID] = repositories.ThreadAttentionRow{
					PendingPlans:   row.PendingPlans,
					LastTurnStatus: row.LastTurnStatus,
				}
			}
			if len(owned) == 0 {
				return out, nil
			}

			runs := make([]pendingProposalRunRow, 0)
			if err := db.NewRaw(
				pendingProposalRunsQuery,
				req.TenantInfo.OrgID,
				req.TenantInfo.BuID,
				agent.ProposalStatusPending,
				agent.SubjectAssistantThread,
				bun.In(owned),
			).Scan(ctx, &runs); err != nil {
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
	req repositories.MarkThreadReadRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.ThreadColumns
		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*conversation.Thread)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.ThreadScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ThreadID).
					Where(cols.UserID.Eq(), req.UserID)
			}).
			Set(cols.LastReadAt.SetExpr("GREATEST({}, ?)"), req.ReadAt).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("mark thread read: %w", err)
		}

		return dberror.CheckRowsAffected(res, "Thread", req.ThreadID.String())
	})
}
