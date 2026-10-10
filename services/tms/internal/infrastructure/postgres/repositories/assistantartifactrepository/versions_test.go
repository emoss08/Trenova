package assistantartifactrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func versionsTestRequest(summary bool) *versionsRequest {
	return &versionsRequest{
		tenant:   pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		threadID: pulid.MustNew("athr_"),
		roots:    []string{"aart_a", "aart_b"},
		summary:  summary,
	}
}

func TestVersionsQueryKeepsOnlyTheSummaryKeysOfAPayload(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(nil, pgdialect.New())
	var dest []*assistantartifact.Artifact

	sql := versionsQuery(db, &dest, versionsTestRequest(true)).String()

	assert.Contains(
		t,
		sql,
		"COALESCE((SELECT jsonb_object_agg(e.key, e.value) FROM jsonb_each(aart.payload) AS e WHERE e.key IN ('tool', 'entity', 'path')), jsonb_build_object()) AS payload",
	)
	assert.NotContains(t, sql, `"aart"."payload"`, "the full payload is not read alongside it")
	assert.NotContains(t, sql, "'aart.payload'",
		"Column.Expr replaces every {}, so an empty-object literal written as '{}' became the column's name and Postgres refused it as JSON")
	assert.Contains(t, sql, `"aart"."title"`, "every other column is still read")
	assert.Contains(t, sql, "COALESCE(aart.lineage_id, aart.id) IN ('aart_a', 'aart_b')")
}

func TestVersionsQueryReadsTheWholePayloadOtherwise(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(nil, pgdialect.New())
	var dest []*assistantartifact.Artifact

	sql := versionsQuery(db, &dest, versionsTestRequest(false)).String()

	assert.NotContains(t, sql, "jsonb_object_agg")
	assert.Contains(t, sql, `"aart"."payload"`)
	assert.Contains(
		t,
		sql,
		`ORDER BY "aart"."lineage_seq" ASC, "aart"."created_at" ASC, "aart"."id" ASC`,
	)
}
