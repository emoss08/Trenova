package assistantartifactrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact/assistantartifactdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pageRequest() *assistantartifactports.ListArtifactsRequest {
	return &assistantartifactports.ListArtifactsRequest{
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

	req := pageRequest()
	cursor := &lineageCursor{pinned: false, last: 42, root: "art_b"}

	sql, args := keeltest.Render(t, pageQuery(lineageQuery(req), req, cursor, 60))

	assert.Contains(t, sql, `WITH "lineages"`)
	assert.Contains(t, sql, `ROW(l."pinned", l."last", l."root") < ROW(`)
	assert.Contains(t, sql, "DESC")
	assert.Contains(t, sql, "LIMIT 61", "one more than the page says whether another follows")
	assert.Contains(t, sql, `GROUP BY 1`)
	assert.Contains(t, args, false)
	assert.Contains(t, args, int64(42))
	assert.Contains(t, args, "art_b")
}

func TestPageQueryFiltersOnTheServer(t *testing.T) {
	t.Parallel()

	req := pageRequest()
	req.Query = "50%_off"
	req.Family = assistantartifact.FamilyView
	req.PinnedOnly = true

	sql, args := keeltest.Render(t, pageQuery(lineageQuery(req), req, nil, 10))

	assert.Contains(t, sql, `aa."title" ILIKE `)
	assert.Regexp(t, `\(aa\."payload" #>> \$\d+\) ILIKE `, sql)
	assert.Contains(t, args, []string{"tool"})
	assert.Contains(t, args, `%50\%\_off%`)
	assert.Contains(t, args, string(assistantartifact.FamilyView))
	assert.NotContains(t, sql, ") < ROW(")
}

func TestFamilyExprMatchesTheDomainsFamilies(t *testing.T) {
	t.Parallel()

	a := assistantartifactdb.Artifacts
	sql, args := keeltest.Render(t, a.Pick(familyExpr()))

	assert.Regexp(t,
		`WHEN \(\(aa\."kind" = 'table_view' AND NOT \(aa\."payload" \? \$3\)\)\) THEN \$4::text `, sql)
	assert.Regexp(t,
		`\(aa\."kind" = 'table_view' AND aa\."payload" \? \$3\)\) THEN \$12::text `, sql)
	assert.Contains(t, sql, `WHEN (aa."kind" IN ('extraction', 'edi_connection_draft')) THEN $14::text `)
	assert.Contains(t, sql,
		`WHEN (aa."kind" IN ('report_preview', 'report_run', 'analysis', 'portal_task', 'board')) `+
			`THEN $9::text `)
	assert.Equal(t, "path", args[2])
	assert.Equal(t, "table", args[3])
	assert.Equal(t, "view", args[11])
	assert.Equal(t, "extract", args[13])
	assert.Equal(t, "report", args[8])
}
