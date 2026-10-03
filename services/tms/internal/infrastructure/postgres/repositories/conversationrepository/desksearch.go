package conversationrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/uptrace/bun"
)

// The person's own listed conversations, which every kind is read through.
const deskThreads = `
	t.organization_id = ? AND t.business_unit_id = ? AND t.user_id = ?
	AND t.origin NOT IN (?)`

// Each query takes the tenant, the person, the unlisted origins, the term
// twice (empty matches everything) and a limit.
var deskSearchQueries = map[string]string{
	"chat": `
SELECT 'chat' AS kind, t.id AS id, t.id AS thread_id, t.agent_definition_id AS agent_id,
	coalesce(t.title, '') AS title, '' AS thread_title, '' AS artifact_kind, t.status AS status,
	greatest(t.last_message_at, t.created_at) AS at
FROM assistant_threads AS t
WHERE ` + deskThreads + `
	AND (? = '' OR t.title ILIKE ?)
ORDER BY t.pinned DESC, greatest(t.last_message_at, t.created_at) DESC
LIMIT ?`,
	// The same words said twice in one conversation are one result: the
	// latest time they were said.
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
		AND (? = '' OR m.content ILIKE ?)
	ORDER BY m.thread_id, md5(m.content), m.created_at DESC
) AS distinct_messages
ORDER BY at DESC
LIMIT ?`,
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
		AND (? = '' OR a.title ILIKE ?)
	ORDER BY coalesce(a.lineage_id, a.id), a.lineage_seq DESC, a.created_at DESC
) AS latest
ORDER BY at DESC
LIMIT ?`,
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
	AND (? = '' OR a.title ILIKE ?)
ORDER BY a.created_at DESC
LIMIT ?`,
}

func (r *repository) SearchDesk(
	ctx context.Context,
	req repositories.SearchDeskRequest,
) ([]repositories.DeskSearchRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]repositories.DeskSearchRow, error) {
		db := r.db.DBForContext(ctx)
		term := strings.TrimSpace(req.Query)
		contains := ""
		if term != "" {
			contains = "%" + likeEscape(term) + "%"
		}
		out := make([]repositories.DeskSearchRow, 0, len(req.Kinds)*req.LimitPerKind)
		for _, kind := range req.Kinds {
			query, ok := deskSearchQueries[kind]
			if !ok {
				continue
			}
			rows := make([]repositories.DeskSearchRow, 0, req.LimitPerKind)
			if err := db.NewRaw(
				query,
				req.TenantInfo.OrgID,
				req.TenantInfo.BuID,
				req.UserID,
				bun.In(conversation.UnlistedOrigins()),
				term,
				contains,
				req.LimitPerKind,
			).Scan(ctx, &rows); err != nil {
				return nil, fmt.Errorf("search desk %s: %w", kind, err)
			}
			out = append(out, rows...)
		}
		return out, nil
	})
}

func (r *repository) CountQuestionsSince(
	ctx context.Context,
	req repositories.CountQuestionsSinceRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		var count int
		err := r.db.DBForContext(ctx).NewRaw(`
SELECT count(*)
FROM assistant_messages AS m
JOIN assistant_threads AS t ON t.id = m.thread_id
	AND t.organization_id = m.organization_id AND t.business_unit_id = m.business_unit_id
WHERE m.organization_id = ? AND m.business_unit_id = ? AND t.user_id = ?
	AND m.role = 'User' AND m.kind = 'Message' AND m.created_at >= ?`,
			req.TenantInfo.OrgID, req.TenantInfo.BuID, req.UserID, req.Since,
		).Scan(ctx, &count)
		if err != nil {
			return 0, fmt.Errorf("count questions: %w", err)
		}
		return count, nil
	})
}
