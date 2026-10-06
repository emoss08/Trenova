package assistantartifactrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func pageRequest() *repositories.ListArtifactsRequest {
	return &repositories.ListArtifactsRequest{
		ThreadID:   pulid.MustNew("athr_"),
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}
}

/*
A page is cut where the last one ended, in the order the pane shows: a cursor
handed back reads the next lineages after it, never the same ones again and
never skipping one that shares a timestamp with the boundary.
*/
func TestCursorRoundTripsThroughItsEncoding(t *testing.T) {
	t.Parallel()

	for _, cursor := range []lineageCursor{
		{pinned: true, last: 1_790_000_000, root: "art_01J0000000000000000000000"},
		{pinned: false, last: 0, root: "art_x.y"},
	} {
		decoded, err := decodeCursor(cursor.encode())
		require.NoError(t, err)
		assert.Equal(t, cursor, decoded)
	}

	for _, bad := range []string{"", "not base64!", "MQ", "Mi4xLmFydA"} {
		_, err := decodeCursor(bad)
		assert.ErrorIs(t, err, ErrBadCursor, bad)
	}
}

func TestPageQueryStartsAfterTheCursorInPaneOrder(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(nil, pgdialect.New())
	req := pageRequest()
	cursor := &lineageCursor{pinned: false, last: 42, root: "art_b"}

	sql := pageQuery(db, lineageQuery(db, req), req, cursor, 60).String()

	assert.Contains(t, sql, "(lineages.pinned::int, lineages.last, lineages.root) < (0, 42, 'art_b')")
	assert.Contains(t, sql, "ORDER BY lineages.pinned DESC, lineages.last DESC, lineages.root DESC")
	assert.Contains(t, sql, "LIMIT 61", "one more than the page says whether another follows")
	assert.Contains(t, sql, "GROUP BY COALESCE(aart.lineage_id, aart.id)")
}

// The search, the family and the pinned filter are the server's, so the
// browser's "All N" and its pages agree with each other at any size.
func TestPageQueryFiltersOnTheServer(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(nil, pgdialect.New())
	req := pageRequest()
	req.Query = "50%_off"
	req.Family = assistantartifact.FamilyView
	req.PinnedOnly = true

	sql := pageQuery(db, lineageQuery(db, req), req, nil, 10).String()

	assert.Contains(t, sql, `aart.title ILIKE '%50\%\_off%'`)
	assert.Contains(t, sql, `aart.payload->>'tool' ILIKE`)
	assert.Contains(t, sql, "lineages.family = 'view'")
	assert.Contains(t, sql, "AND (lineages.pinned)")
	assert.NotContains(t, sql, "lineages.root) <")
}

// A table_view that names a page is a view and one that does not is a table,
// as the Desk draws them; every other kind belongs to one family.
func TestFamilyExprMatchesTheDomainsFamilies(t *testing.T) {
	t.Parallel()

	expr := familyExpr()

	assert.Contains(t, expr, "WHEN ((aart.kind = 'table_view' AND aart.payload->'path' IS NULL)) THEN 'table'")
	assert.Contains(t, expr, "(aart.kind = 'table_view' AND aart.payload->'path' IS NOT NULL)) THEN 'view'")
	assert.Contains(t, expr, "WHEN (aart.kind IN ('extraction')) THEN 'extract'")
	assert.Contains(t, expr, "WHEN (aart.kind IN ('report_preview', 'report_run')) THEN 'report'")
}
