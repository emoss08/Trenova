package assistantartifactrepository

import (
	"strings"
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

	for _, key := range []string{"tool", "entity", "path", "documentId"} {
		assert.Contains(t, args, key)
		assert.Contains(t, args, []string{key})
	}
	assert.Equal(t, 4, strings.Count(sql, `CASE WHEN aa."payload" ? $`))
	assert.Contains(t, sql, `jsonb_build_object()`)
	assert.Contains(t, sql, `aa."payload" #> $`)
	assert.Contains(t, args, []string{"aart_a", "aart_b"})
}

func TestVersionsQueryReadsTheWholePayloadInLineageOrder(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, versionsQuery(versionsTestRequest()))

	assert.NotContains(t, sql, "jsonb_build_object")
	assert.Contains(t, sql, `aa."payload"`)
	assert.Contains(t, sql, `coalesce(aa."lineage_id", aa."id") = ANY(`)
	assert.Contains(
		t,
		sql,
		`ORDER BY aa."lineage_seq" ASC, aa."created_at" ASC, aa."id" ASC`,
	)
}
