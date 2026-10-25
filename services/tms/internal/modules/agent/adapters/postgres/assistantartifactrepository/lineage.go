package assistantartifactrepository

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact/assistantartifactdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

var lineageColumns = []string{"root", "pinned", "last", "family"}

// rootExpr is the lineage an artifact belongs to: the first version's id,
// which is its own id for the first version.
func rootExpr() keel.StringExpr {
	a := assistantartifactdb.Artifacts

	return a.LineageID.Coalesce(a.ID)
}

func rootIs(root string) keel.Pred {
	a := assistantartifactdb.Artifacts

	return rootExpr().Eq(root)
}

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
func familyExpr() keel.Expr {
	a := assistantartifactdb.Artifacts
	family := keel.DynCase()
	for _, f := range assistantartifact.AllFamilies() {
		kinds, viewPath := assistantartifact.FamilyKinds(f)
		if len(kinds) == 0 {
			continue
		}
		family = family.When(
			familyCondition(a, kinds, viewPath),
			keel.Text(string(f)),
		)
	}

	return family.Else(keel.Text(string(assistantartifact.FamilyDoc)))
}

func familyCondition(
	a *assistantartifactdb.ArtifactsTable,
	kinds []assistantartifact.Kind,
	viewPath *bool,
) keel.Pred {
	listed := make([]assistantartifact.Kind, 0, len(kinds))
	for _, kind := range kinds {
		if kind == assistantartifact.KindTableView && viewPath != nil {
			continue
		}
		listed = append(listed, kind)
	}
	var clauses []keel.Pred
	if len(listed) > 0 {
		clauses = append(clauses, a.Kind.In(listed...))
	}
	if viewPath != nil {
		opens := a.Payload.HasKey("path")
		if !*viewPath {
			opens = keel.Not(opens)
		}
		clauses = append(clauses, keel.And(a.Kind.Eq(assistantartifact.KindTableView), opens))
	}

	return keel.Or(clauses...)
}

func escapeLike(text string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(text)
}

// matching restricts a read to the thread's artifacts that match the search.
// A lineage matches when any of its versions does.
func matching(req *assistantartifactports.ListArtifactsRequest) []keel.Pred {
	a := assistantartifactdb.Artifacts
	preds := []keel.Pred{a.ThreadID.Eq(req.ThreadID)}
	if needle := strings.TrimSpace(req.Query); needle != "" {
		pattern := "%" + escapeLike(needle) + "%"
		preds = append(preds, keel.Or(
			a.Title.ILike(pattern),
			a.Payload.Text("tool").ILike(pattern),
		))
	}

	return preds
}

type familyCount struct {
	Family string
	Pinned bool
	Count  int
}

type lineageRow struct {
	Root   string
	Pinned bool
	Last   int64
}

func (r *repository) ListPage(
	ctx context.Context,
	req assistantartifactports.ListArtifactsRequest,
) (*assistantartifactports.ArtifactPage, error) {
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

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*assistantartifactports.ArtifactPage, error) {
			page := &assistantartifactports.ArtifactPage{
				Counts: assistantartifactports.ArtifactCounts{
					Families: map[assistantartifact.Family]int{},
				},
			}

			lineages := lineageQuery(&req)
			var (
				counted []familyCount
				rows    []lineageRow
			)
			err := keel.Batch(ctx, r.db.Keel(ctx),
				keel.IntoSlice(lineages.Query().
					With(lineages).
					Group(lineages.Col("family"), lineages.Col("pinned")).
					Pick(lineages.Col("family"), lineages.Col("pinned"), keel.Count()), &counted),
				keel.IntoSlice(pageQuery(lineages, &req, cursor, limit), &rows),
			)
			if err != nil {
				r.l.Error("failed to page assistant artifacts",
					zap.String("threadId", req.ThreadID.String()),
					zap.Error(err))

				return nil, fmt.Errorf("page assistant artifacts: %w", err)
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
			artifacts, err := r.versionsOf(ctx, versionsRequest{
				threadID: req.ThreadID,
				roots:    roots,
				summary:  req.Summary,
			})
			if err != nil {
				return nil, err
			}
			page.Artifacts = artifacts

			return page, nil
		},
	)
}

// lineageQuery is one row per lineage that matches the search: its root,
// whether any version is pinned, when it was last added to, and its family.
func lineageQuery(req *assistantartifactports.ListArtifactsRequest) *keel.CTETable {
	a := assistantartifactdb.Artifacts

	return keel.CTE("lineages", lineageColumns, a.
		Where(matching(req)...).
		Group(rootExpr()).
		Pick(
			rootExpr(),
			keel.BoolOr(keel.BoolOf(a.Pinned)),
			a.CreatedAt.Max(),
			keel.Max(familyExpr()),
		))
}

// pageQuery is one page of lineages after the cursor, in the pane's order:
// pinned first, then most recently added to. It reads one more than the page
// to know whether another follows.
func pageQuery(
	lineages *keel.CTETable,
	req *assistantartifactports.ListArtifactsRequest,
	cursor *lineageCursor,
	limit int,
) *keel.Query[*keel.Row] {
	root, pinned, last := lineages.Col("root"), lineages.Col("pinned"), lineages.Col("last")
	window := lineages.Query().With(lineages)
	if req.Family != "" {
		window.Where(lineages.Col("family").Eq(string(req.Family)))
	}
	if req.PinnedOnly {
		window.Where(keel.BoolOf(pinned))
	}
	if cursor != nil {
		window.Where(keel.DynRow(pinned, last, root).Lt(
			keel.DynRow(cursor.pinned, cursor.last, cursor.root),
		))
	}

	return window.
		Pick(root, pinned, last).
		Order(pinned.Desc(), last.Desc(), root.Desc()).
		Limit(limit + 1)
}

type versionsRequest struct {
	threadID pulid.ID
	roots    []string
	summary  bool
}

type summaryPayload struct {
	ID      pulid.ID
	Payload map[string]any
}

func (r *repository) versionsOf(
	ctx context.Context,
	req versionsRequest,
) ([]*assistantartifact.Artifact, error) {
	a := assistantartifactdb.Artifacts
	db := r.db.Keel(ctx)
	if !req.summary {
		artifacts, err := versionsQuery(&req).All(ctx, db)
		if err != nil {
			return nil, fmt.Errorf("read artifact versions: %w", err)
		}

		return artifacts, nil
	}

	var (
		artifacts []*assistantartifact.Artifact
		payloads  []summaryPayload
	)
	err := keel.Batch(ctx, db,
		keel.ListInto(versionsQuery(&req).Omit(a.Payload), &artifacts),
		keel.IntoSlice(summaryQuery(&req), &payloads),
	)
	if err != nil {
		return nil, fmt.Errorf("read artifact versions: %w", err)
	}

	byID := make(map[pulid.ID]map[string]any, len(payloads))
	for _, row := range payloads {
		byID[row.ID] = row.Payload
	}
	for _, artifact := range artifacts {
		artifact.Payload = byID[artifact.ID]
	}

	return artifacts, nil
}

func summaryQuery(req *versionsRequest) *keel.Query[*assistantartifact.Artifact] {
	a := assistantartifactdb.Artifacts

	return versionsQuery(req).Pick(a.ID, summaryPayloadOf(a.Payload))
}

func summaryPayloadOf(payload keel.JSONBCol) keel.JSONBExpr {
	summary := keel.JSONBuildObject()
	for _, key := range assistantartifact.SummaryPayloadKeys() {
		summary = summary.Concat(keel.JSONBOf(keel.DynCase().
			When(payload.HasKey(key), keel.JSONBuildObject(keel.Pair(key, payload.Path(key)))).
			Else(keel.JSONBuildObject())))
	}

	return summary
}

func versionsQuery(req *versionsRequest) *keel.Query[*assistantartifact.Artifact] {
	a := assistantartifactdb.Artifacts

	return a.Where(
		a.ThreadID.Eq(req.threadID),
		rootExpr().In(req.roots...),
	).
		Order(a.LineageSeq.Asc(), a.CreatedAt.Asc(), a.ID.Asc())
}

// rootOf is the lineage an artifact belongs to, read from the artifact.
func rootOf(ctx context.Context, db *keel.DB, threadID, id pulid.ID) (string, error) {
	a := assistantartifactdb.Artifacts
	root, err := keel.IntoOne[string](ctx, db, a.
		Where(a.ThreadID.Eq(threadID), a.ID.Eq(id)).
		Pick(rootExpr()))
	if err != nil {
		return "", artifactErrors.Map(err, id.String())
	}

	return root, nil
}

func (r *repository) ListLineage(
	ctx context.Context,
	req assistantartifactports.LineageRequest,
) ([]*assistantartifact.Artifact, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*assistantartifact.Artifact, error) {
		root, err := rootOf(ctx, r.db.Keel(ctx), req.ThreadID, req.ID)
		if err != nil {
			return nil, err
		}

		return r.versionsOf(ctx, versionsRequest{
			threadID: req.ThreadID,
			roots:    []string{root},
		})
	})
}

func (r *repository) FindBySlug(
	ctx context.Context,
	req assistantartifactports.SlugRequest,
) (*assistantartifact.Artifact, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*assistantartifact.Artifact, error) {
		a := assistantartifactdb.Artifacts
		entity, err := a.Where(a.ThreadID.Eq(req.ThreadID), a.Slug.Eq(req.Slug)).
			Order(a.LineageSeq.Asc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, artifactErrors.Map(err, req.Slug)
		}

		return entity, nil
	})
}

func (r *repository) TakenSlugs(
	ctx context.Context,
	threadID pulid.ID,
	_ pagination.TenantInfo,
	base string,
) (map[string]bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[string]bool, error) {
		a := assistantartifactdb.Artifacts
		slugs, err := keel.Into[string](ctx, r.db.Keel(ctx), a.
			Where(
				a.ThreadID.Eq(threadID),
				a.LineageID.IsNull(),
				a.Slug.Like(escapeLike(base)+"%"),
			).
			Pick(a.Slug))
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
		if err := assistantartifactdb.Artifacts.Create(ctx, r.db.Keel(ctx), artifact); err != nil {
			r.l.Error("failed to insert assistant artifact version",
				zap.String("threadId", artifact.ThreadID.String()),
				zap.Error(err))

			return nil, fmt.Errorf("insert assistant artifact version: %w", err)
		}

		return artifact, nil
	})
}

type turnQuestion struct {
	ID       pulid.ID
	Question string
}

func (r *repository) TurnQuestions(
	ctx context.Context,
	threadID pulid.ID,
	_ pagination.TenantInfo,
	messageIDs []pulid.ID,
) (map[pulid.ID]string, error) {
	if len(messageIDs) == 0 {
		return map[pulid.ID]string{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]string, error) {
		m := conversationdb.Messages
		asked := conversationdb.Messages.As("asked")
		rows, err := keel.Into[turnQuestion](ctx, r.db.Keel(ctx), m.
			Where(m.ThreadID.Eq(threadID), m.ID.In(messageIDs...)).
			Pick(m.ID, keel.StringOf(keel.Scalar(asked.
				Where(
					asked.ThreadID.EqCol(m.ThreadID),
					asked.Role.Eq(conversation.RoleUser),
					keel.Int64Of(asked.Sequence).LtExpr(m.Sequence),
				).
				Pick(asked.Content).
				Order(asked.Sequence.Desc()).
				Limit(1))).Coalesce("")))
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
