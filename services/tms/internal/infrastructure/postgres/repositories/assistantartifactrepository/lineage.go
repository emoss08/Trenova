package assistantartifactrepository

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// rootExpr is the lineage an artifact belongs to: the first version's id,
// which is its own id for the first version.
const rootExpr = "COALESCE(aart.lineage_id, aart.id)"

// ErrBadCursor is a cursor this repository did not hand out.
var ErrBadCursor = errors.New("artifact cursor is not one a previous page returned")

type lineageCursor struct {
	pinned bool
	last   int64
	root   string
}

func (c lineageCursor) encode() string {
	pinned := "0"
	if c.pinned {
		pinned = "1"
	}

	return base64.RawURLEncoding.EncodeToString(
		[]byte(pinned + "." + strconv.FormatInt(c.last, 10) + "." + c.root),
	)
}

func decodeCursor(raw string) (lineageCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return lineageCursor{}, ErrBadCursor
	}
	parts := strings.SplitN(string(decoded), ".", 3)
	if len(parts) != 3 || (parts[0] != "0" && parts[0] != "1") || parts[2] == "" {
		return lineageCursor{}, ErrBadCursor
	}
	last, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return lineageCursor{}, ErrBadCursor
	}

	return lineageCursor{pinned: parts[0] == "1", last: last, root: parts[2]}, nil
}

// familyExpr is the family of the row's kind, written out from the domain's
// own table so the server filters on the families the Desk draws.
func familyExpr() string {
	var b strings.Builder
	b.WriteString("CASE")
	for _, family := range assistantartifact.AllFamilies() {
		kinds, viewPath := assistantartifact.FamilyKinds(family)
		if len(kinds) == 0 {
			continue
		}
		b.WriteString(" WHEN ")
		b.WriteString(familyCondition(kinds, viewPath))
		b.WriteString(" THEN '")
		b.WriteString(string(family))
		b.WriteString("'")
	}
	b.WriteString(" ELSE '")
	b.WriteString(string(assistantartifact.FamilyDoc))
	b.WriteString("' END")

	return b.String()
}

// familyCondition is the SQL that holds for a row in a family. The kinds are
// the domain's own constants, never input, so they are written in place.
func familyCondition(kinds []assistantartifact.Kind, viewPath *bool) string {
	quoted := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		if kind == assistantartifact.KindTableView && viewPath != nil {
			continue
		}
		quoted = append(quoted, "'"+string(kind)+"'")
	}
	var clauses []string
	if len(quoted) > 0 {
		clauses = append(clauses, "aart.kind IN ("+strings.Join(quoted, ", ")+")")
	}
	if viewPath != nil {
		opens := "IS NULL"
		if *viewPath {
			opens = "IS NOT NULL"
		}
		clauses = append(clauses,
			"(aart.kind = 'table_view' AND aart.payload->'path' "+opens+")")
	}

	return "(" + strings.Join(clauses, " OR ") + ")"
}

func escapeLike(text string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(text)
}

// matching restricts a read to the thread's artifacts that match the search.
// A lineage matches when any of its versions does.
func matching(q *bun.SelectQuery, req *repositories.ListArtifactsRequest) *bun.SelectQuery {
	cols := buncolgen.ArtifactColumns

	q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return buncolgen.ArtifactScopeTenant(sq, req.TenantInfo).
			Where(cols.ThreadID.Eq(), req.ThreadID)
	})
	if needle := strings.TrimSpace(req.Query); needle != "" {
		pattern := "%" + escapeLike(needle) + "%"
		q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("aart.title ILIKE ?", pattern).
				WhereOr("aart.payload->>'tool' ILIKE ?", pattern)
		})
	}

	return q
}

type lineageRow struct {
	Root   string `bun:"root"`
	Pinned bool   `bun:"pinned"`
	Last   int64  `bun:"last"`
	Family string `bun:"family"`
	Count  int    `bun:"count"`
}

func (r *repository) ListPage(
	ctx context.Context,
	req repositories.ListArtifactsRequest,
) (*repositories.ArtifactPage, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	limit = min(limit, maxListLimit)

	var cursor *lineageCursor
	if req.Cursor != "" {
		decoded, err := decodeCursor(req.Cursor)
		if err != nil {
			return nil, err
		}
		cursor = &decoded
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*repositories.ArtifactPage, error) {
		db := r.db.DBForContext(ctx)
		page := &repositories.ArtifactPage{
			Counts: repositories.ArtifactCounts{Families: map[assistantartifact.Family]int{}},
		}

		lineages := lineageQuery(db, &req)

		var counted []lineageRow
		if err := db.NewSelect().
			TableExpr("(?) AS lineages", lineages).
			ColumnExpr("lineages.family, lineages.pinned").
			ColumnExpr("count(*) AS count").
			GroupExpr("lineages.family, lineages.pinned").
			Scan(ctx, &counted); err != nil {
			return nil, fmt.Errorf("count assistant artifacts: %w", err)
		}
		for _, row := range counted {
			family := assistantartifact.Family(row.Family)
			page.Counts.All += row.Count
			page.Counts.Families[family] += row.Count
			if row.Pinned {
				page.Counts.Pinned += row.Count
			}
			if (req.Family == "" || req.Family == family) && (!req.PinnedOnly || row.Pinned) {
				page.Total += row.Count
			}
		}

		var rows []lineageRow
		if err := pageQuery(db, lineages, &req, cursor, limit).Scan(ctx, &rows); err != nil {
			r.l.Error("failed to page assistant artifacts",
				zap.String("threadId", req.ThreadID.String()),
				zap.Error(err))

			return nil, fmt.Errorf("page assistant artifacts: %w", err)
		}
		if len(rows) > limit {
			last := rows[limit-1]
			page.NextCursor = lineageCursor{
				pinned: last.Pinned, last: last.Last, root: last.Root,
			}.encode()
			rows = rows[:limit]
		}
		if len(rows) == 0 {
			page.Artifacts = []*assistantartifact.Artifact{}
			return page, nil
		}

		roots := make([]string, 0, len(rows))
		for _, row := range rows {
			roots = append(roots, row.Root)
		}
		artifacts, err := r.versionsOf(ctx, req.TenantInfo, req.ThreadID, roots)
		if err != nil {
			return nil, err
		}
		page.Artifacts = artifacts

		return page, nil
	})
}

// lineageQuery is one row per lineage that matches the search: its root,
// whether any version is pinned, when it was last added to, and its family.
func lineageQuery(db bun.IDB, req *repositories.ListArtifactsRequest) *bun.SelectQuery {
	lineages := db.NewSelect().
		Model((*assistantartifact.Artifact)(nil)).
		ColumnExpr(rootExpr + " AS root").
		ColumnExpr("bool_or(aart.pinned) AS pinned").
		ColumnExpr("max(aart.created_at) AS last").
		ColumnExpr("max(" + familyExpr() + ") AS family")

	return matching(lineages, req).GroupExpr(rootExpr)
}

// pageQuery is one page of lineages after the cursor, in the pane's order:
// pinned first, then most recently added to. It reads one more than the page
// to know whether another follows.
func pageQuery(
	db bun.IDB,
	lineages *bun.SelectQuery,
	req *repositories.ListArtifactsRequest,
	cursor *lineageCursor,
	limit int,
) *bun.SelectQuery {
	window := db.NewSelect().
		TableExpr("(?) AS lineages", lineages).
		ColumnExpr("lineages.root, lineages.pinned, lineages.last")
	if req.Family != "" {
		window = window.Where("lineages.family = ?", string(req.Family))
	}
	if req.PinnedOnly {
		window = window.Where("lineages.pinned")
	}
	if cursor != nil {
		window = window.Where(
			"(lineages.pinned::int, lineages.last, lineages.root) < (?, ?, ?)",
			boolInt(cursor.pinned), cursor.last, cursor.root,
		)
	}

	return window.
		OrderExpr("lineages.pinned DESC, lineages.last DESC, lineages.root DESC").
		Limit(limit + 1)
}

func boolInt(value bool) int {
	if value {
		return 1
	}

	return 0
}

// versionsOf reads every version of the given lineages, oldest first.
func (r *repository) versionsOf(
	ctx context.Context,
	tenant pagination.TenantInfo,
	threadID pulid.ID,
	roots []string,
) ([]*assistantartifact.Artifact, error) {
	cols := buncolgen.ArtifactColumns
	artifacts := make([]*assistantartifact.Artifact, 0, len(roots))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&artifacts).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ArtifactScopeTenant(sq, tenant).
				Where(cols.ThreadID.Eq(), threadID).
				Where(rootExpr+" IN (?)", bun.List(roots))
		}).
		Order(cols.LineageSeq.OrderAsc(), cols.CreatedAt.OrderAsc(), cols.ID.OrderAsc()).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("read artifact versions: %w", err)
	}

	return artifacts, nil
}

// rootOf is the lineage an artifact belongs to, read from the artifact.
func (r *repository) rootOf(
	ctx context.Context,
	tenant pagination.TenantInfo,
	threadID, id pulid.ID,
) (string, error) {
	cols := buncolgen.ArtifactColumns
	var root string
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*assistantartifact.Artifact)(nil)).
		ColumnExpr(rootExpr).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ArtifactScopeTenant(sq, tenant).
				Where(cols.ThreadID.Eq(), threadID).
				Where(cols.ID.Eq(), id)
		}).
		Scan(ctx, &root)
	if err != nil {
		return "", dberror.HandleNotFoundError(err, "AssistantArtifact")
	}

	return root, nil
}

func (r *repository) ListLineage(
	ctx context.Context,
	req repositories.LineageRequest,
) ([]*assistantartifact.Artifact, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*assistantartifact.Artifact, error) {
		root, err := r.rootOf(ctx, req.TenantInfo, req.ThreadID, req.ID)
		if err != nil {
			return nil, err
		}

		return r.versionsOf(ctx, req.TenantInfo, req.ThreadID, []string{root})
	})
}

func (r *repository) FindBySlug(
	ctx context.Context,
	req repositories.SlugRequest,
) (*assistantartifact.Artifact, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*assistantartifact.Artifact, error) {
		cols := buncolgen.ArtifactColumns
		entity := new(assistantartifact.Artifact)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.ArtifactScopeTenant(sq, req.TenantInfo).
					Where(cols.ThreadID.Eq(), req.ThreadID).
					Where(cols.Slug.Eq(), req.Slug)
			}).
			Order(cols.LineageSeq.OrderAsc()).
			Limit(1).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "AssistantArtifact")
		}

		return entity, nil
	})
}

func (r *repository) TakenSlugs(
	ctx context.Context,
	threadID pulid.ID,
	tenant pagination.TenantInfo,
	base string,
) (map[string]bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[string]bool, error) {
		cols := buncolgen.ArtifactColumns
		var slugs []string
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*assistantartifact.Artifact)(nil)).
			ColumnExpr(cols.Slug.Qualified()).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.ArtifactScopeTenant(sq, tenant).
					Where(cols.ThreadID.Eq(), threadID).
					Where(cols.LineageID.IsNull()).
					Where(cols.Slug.Qualified()+" LIKE ?", escapeLike(base)+"%")
			}).
			Scan(ctx, &slugs)
		if err != nil {
			return nil, fmt.Errorf("read taken artifact slugs: %w", err)
		}

		taken := make(map[string]bool, len(slugs))
		for _, slug := range slugs {
			taken[slug] = true
		}

		return taken, nil
	})
}

func (r *repository) InsertVersion(
	ctx context.Context,
	artifact *assistantartifact.Artifact,
) (*assistantartifact.Artifact, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*assistantartifact.Artifact, error) {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(artifact).
			Returning("*").
			Exec(ctx); err != nil {
			r.l.Error("failed to insert assistant artifact version",
				zap.String("threadId", artifact.ThreadID.String()),
				zap.Error(err))

			return nil, fmt.Errorf("insert assistant artifact version: %w", err)
		}

		return artifact, nil
	})
}

type turnQuestion struct {
	ID       pulid.ID `bun:"id"`
	Question string   `bun:"question"`
}

func (r *repository) TurnQuestions(
	ctx context.Context,
	threadID pulid.ID,
	tenant pagination.TenantInfo,
	messageIDs []pulid.ID,
) (map[pulid.ID]string, error) {
	if len(messageIDs) == 0 {
		return map[pulid.ID]string{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]string, error) {
		var rows []turnQuestion
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*conversation.Message)(nil)).
			ColumnExpr("amsg.id").
			ColumnExpr(`(
				SELECT asked.content FROM assistant_messages AS asked
				WHERE asked.organization_id = amsg.organization_id
					AND asked.business_unit_id = amsg.business_unit_id
					AND asked.thread_id = amsg.thread_id
					AND asked.role = ?
					AND asked.sequence < amsg.sequence
				ORDER BY asked.sequence DESC
				LIMIT 1
			) AS question`, conversation.RoleUser).
			Where("amsg.organization_id = ?", tenant.OrgID).
			Where("amsg.business_unit_id = ?", tenant.BuID).
			Where("amsg.thread_id = ?", threadID).
			Where("amsg.id IN (?)", bun.List(messageIDs)).
			Scan(ctx, &rows)
		if err != nil {
			return nil, fmt.Errorf("read turn questions: %w", err)
		}

		out := make(map[pulid.ID]string, len(rows))
		for _, row := range rows {
			out[row.ID] = row.Question
		}

		return out, nil
	})
}

func (r *repository) ToolCallArguments(
	ctx context.Context,
	threadID pulid.ID,
	tenant pagination.TenantInfo,
	callID string,
) (map[string]any, error) {
	if callID == "" {
		return nil, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[string]any, error) {
		var arguments []map[string]any
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*conversation.Message)(nil)).
			ColumnExpr("call->'arguments'").
			Join("CROSS JOIN LATERAL jsonb_array_elements(amsg.tool_calls) AS call").
			Where("amsg.organization_id = ?", tenant.OrgID).
			Where("amsg.business_unit_id = ?", tenant.BuID).
			Where("amsg.thread_id = ?", threadID).
			Where("jsonb_typeof(amsg.tool_calls) = 'array'").
			Where("call->>'id' = ?", callID).
			Limit(1).
			Scan(ctx, &arguments)
		if err != nil {
			return nil, fmt.Errorf("read tool call arguments: %w", err)
		}
		if len(arguments) == 0 {
			return nil, nil
		}

		return arguments[0], nil
	})
}
