package conversationrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/jackc/pgx/v5"
)

const deskThreads = `
	t.organization_id = $1 AND t.business_unit_id = $2 AND t.user_id = $3
	AND t.origin <> ALL($4::text[])`

var deskSearchQueries = map[string]string{
	"chat": `
SELECT 'chat' AS kind, t.id AS id, t.id AS thread_id, t.agent_definition_id AS agent_id,
	coalesce(t.title, '') AS title, '' AS thread_title, '' AS artifact_kind, t.status AS status,
	greatest(t.last_message_at, t.created_at) AS at
FROM assistant_threads AS t
WHERE ` + deskThreads + `
	AND ($5 = '' OR t.title ILIKE $6)
ORDER BY t.pinned DESC, greatest(t.last_message_at, t.created_at) DESC
LIMIT $7`,
	"msg": `
SELECT * FROM (
	SELECT DISTINCT ON (m.thread_id, md5(m.content))
		'msg' AS kind, m.id AS id, t.id AS thread_id,
		coalesce(m.agent_definition_id, t.agent_definition_id) AS agent_id,
		left(m.content, 4000) AS title, coalesce(t.title, '') AS thread_title, '' AS artifact_kind,
		m.role AS status, m.created_at AS at
	FROM assistant_messages AS m
	JOIN assistant_threads AS t ON t.id = m.thread_id
		AND t.organization_id = m.organization_id AND t.business_unit_id = m.business_unit_id
	WHERE ` + deskThreads + `
		AND m.role IN ('User', 'Assistant') AND m.content <> ''
		AND ($5 = '' OR m.content ILIKE $6)
	ORDER BY m.thread_id, md5(m.content), m.created_at DESC
) AS distinct_messages
ORDER BY at DESC
LIMIT $7`,
	"art": `
SELECT * FROM (
	SELECT DISTINCT ON (coalesce(a.lineage_id, a.id))
		'art' AS kind, a.id AS id, t.id AS thread_id, t.agent_definition_id AS agent_id,
		a.title AS title, coalesce(t.title, '') AS thread_title, a.kind AS artifact_kind,
		a.status AS status, a.created_at AS at
	FROM assistant_artifacts AS a
	JOIN assistant_threads AS t ON t.id = a.thread_id
		AND t.organization_id = a.organization_id AND t.business_unit_id = a.business_unit_id
	WHERE ` + deskThreads + `
		AND a.kind <> 'decision_request'
		AND ($5 = '' OR a.title ILIKE $6)
	ORDER BY coalesce(a.lineage_id, a.id), a.lineage_seq DESC, a.created_at DESC
) AS latest
ORDER BY at DESC
LIMIT $7`,
	"dec": `
SELECT 'dec' AS kind, a.id AS id, t.id AS thread_id, t.agent_definition_id AS agent_id,
	a.title AS title, coalesce(t.title, '') AS thread_title, a.kind AS artifact_kind,
	coalesce(p.status::text, a.status) AS status, a.created_at AS at
FROM assistant_artifacts AS a
JOIN assistant_threads AS t ON t.id = a.thread_id
	AND t.organization_id = a.organization_id AND t.business_unit_id = a.business_unit_id
LEFT JOIN agent_proposals AS p ON p.id = a.proposal_id
	AND p.organization_id = a.organization_id AND p.business_unit_id = a.business_unit_id
WHERE ` + deskThreads + `
	AND a.kind = 'decision_request'
	AND ($5 = '' OR a.title ILIKE $6)
ORDER BY a.created_at DESC
LIMIT $7`,
}

func (r *repository) SearchDesk(
	ctx context.Context,
	req agentports.SearchDeskRequest,
) ([]agentports.DeskSearchRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]agentports.DeskSearchRow, error) {
		db := r.db.Keel(ctx)
		term := strings.TrimSpace(req.Query)
		contains := ""
		if term != "" {
			contains = "%" + likeEscape(term) + "%"
		}
		unlisted := sliceutils.Strings(conversation.UnlistedOrigins())
		out := make([]agentports.DeskSearchRow, 0, len(req.Kinds)*req.LimitPerKind)
		for _, kind := range req.Kinds {
			query, ok := deskSearchQueries[kind]
			if !ok {
				continue
			}
			rows, err := keel.QueryAll(
				ctx,
				db,
				pgx.RowToStructByName[agentports.DeskSearchRow],
				query,
				req.TenantInfo.OrgID.String(),
				req.TenantInfo.BuID.String(),
				req.UserID.String(),
				unlisted,
				term,
				contains,
				req.LimitPerKind,
			)
			if err != nil {
				return nil, fmt.Errorf("search desk %s: %w", kind, err)
			}
			out = append(out, rows...)
		}
		return out, nil
	})
}

const countQuestionsSinceQuery = `
SELECT count(*)
FROM assistant_messages AS m
JOIN assistant_threads AS t ON t.id = m.thread_id
	AND t.organization_id = m.organization_id AND t.business_unit_id = m.business_unit_id
WHERE m.organization_id = $1 AND m.business_unit_id = $2 AND t.user_id = $3
	AND m.role = 'User' AND m.kind = 'Message' AND m.created_at >= $4`

func (r *repository) CountQuestionsSince(
	ctx context.Context,
	req agentports.CountQuestionsSinceRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		count, err := keel.QueryValue[int64](
			ctx,
			r.db.Keel(ctx),
			countQuestionsSinceQuery,
			req.TenantInfo.OrgID.String(),
			req.TenantInfo.BuID.String(),
			req.UserID.String(),
			req.Since,
		)
		if err != nil {
			return 0, fmt.Errorf("count questions: %w", err)
		}
		return int(count), nil
	})
}
