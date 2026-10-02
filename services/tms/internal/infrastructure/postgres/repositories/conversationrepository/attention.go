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
	ThreadID         pulid.ID                         `bun:"thread_id"`
	PendingDecisions int                              `bun:"pending_decisions"`
	LastTurnStatus   conversation.AssistantTurnStatus `bun:"last_turn_status"`
}

const threadAttentionQuery = `
SELECT
	t.id AS thread_id,
	(
		SELECT count(*)
		FROM agent_proposals AS p
		JOIN agent_runs AS r
			ON r.id = p.run_id
			AND r.organization_id = p.organization_id
			AND r.business_unit_id = p.business_unit_id
		WHERE p.organization_id = t.organization_id
			AND p.business_unit_id = t.business_unit_id
			AND p.status = ?
			AND p.plan_id IS NULL
			AND r.subject_type = ?
			AND r.subject_id = t.id
	) + (
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
	) AS pending_decisions,
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

func (r *repository) ListThreadAttention(
	ctx context.Context,
	req repositories.ListThreadAttentionRequest,
) (map[pulid.ID]conversation.ThreadAttentionSignals, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]conversation.ThreadAttentionSignals, error) {
			out := make(map[pulid.ID]conversation.ThreadAttentionSignals, len(req.ThreadIDs))
			if len(req.ThreadIDs) == 0 {
				return out, nil
			}

			rows := make([]threadAttentionRow, 0, len(req.ThreadIDs))
			err := r.db.DBForContext(ctx).
				NewRaw(
					threadAttentionQuery,
					agent.ProposalStatusPending,
					agent.SubjectAssistantThread,
					agent.PlanStatusPending,
					agent.SubjectAssistantThread,
					req.TenantInfo.OrgID,
					req.TenantInfo.BuID,
					req.UserID,
					bun.In(req.ThreadIDs),
				).
				Scan(ctx, &rows)
			if err != nil {
				return nil, fmt.Errorf("list thread attention: %w", err)
			}

			for _, row := range rows {
				out[row.ThreadID] = conversation.ThreadAttentionSignals{
					PendingDecisions: row.PendingDecisions,
					LastTurnStatus:   row.LastTurnStatus,
				}
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
