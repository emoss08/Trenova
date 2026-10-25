package assistantartifactrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func versionsTestRequest() *versionsRequest {
	return &versionsRequest{
		threadID: pulid.MustNew("athr_"),
		roots:    []string{"aart_a", "aart_b"},
	}
}

func TestSummaryQueryKeepsOnlyTheSummaryKeysOfAPayload(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, summaryQuery(versionsTestRequest()))

	assert.Contains(
		t,
		sql,
		`COALESCE((SELECT jsonb_object_agg(e.key, e.value) FROM jsonb_each(aa."payload") AS e WHERE e.key = ANY(`,
	)
	assert.Contains(t, sql, "jsonb_build_object())")
	assert.Contains(t, args, []string{"tool", "entity", "path", "documentId"})
	assert.Contains(t, args, []string{"aart_a", "aart_b"})
}

func TestVersionsQueryReadsTheWholePayloadInLineageOrder(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, versionsQuery(versionsTestRequest()))

	assert.NotContains(t, sql, "jsonb_object_agg")
	assert.Contains(t, sql, `aa."payload"`)
	assert.Contains(t, sql, `coalesce(aa."lineage_id", aa."id") = ANY(`)
	assert.Contains(
		t,
		sql,
		`ORDER BY aa."lineage_seq" ASC, aa."created_at" ASC, aa."id" ASC`,
	)
}
