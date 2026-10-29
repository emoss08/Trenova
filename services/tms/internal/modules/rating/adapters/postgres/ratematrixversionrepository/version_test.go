package ratematrixversionrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrixversion"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
)

func TestVersionsByKeysQuery_MatchesExactKeyPairs(t *testing.T) {
	t.Parallel()

	keys := uniqueKeys([]ratematrixversion.VersionKey{
		{RateMatrixID: "rmx_1", VersionNumber: 1},
		{RateMatrixID: "rmx_2", VersionNumber: 3},
		{RateMatrixID: "rmx_1", VersionNumber: 1},
	})
	sql, args := keeltest.Render(t, versionsByKeysQuery(keys))

	assert.Len(t, keys, 2)
	assert.Contains(t, sql, "unnest(")
	assert.Contains(t, sql, "LIMIT 2")
	assert.Contains(t, args, []string{"rmx_1", "rmx_2"})
	assert.Contains(t, args, []int64{1, 3})
}

func TestServedStampQuery_HashesEveryServedMatrix(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, servedStampQuery())

	assert.Contains(t, sql, "string_agg(")
	assert.Contains(t, sql, `"published_version_number" > $`)
}
