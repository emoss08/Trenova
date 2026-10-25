package ratematrixrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
)

func TestMisfitQuery_TestsEveryAxisAgainstItsMatchMode(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, misfitQuery(&ratematrixports.CountMisfitCellsRequest{
		RateMatrixID:  "rmx_1",
		VersionNumber: 2,
		Dimensions: []*ratematrix.RateMatrixDimension{
			{Position: 0, MatchMode: ratematrix.MatchModeExact},
		},
	}).Limit(1))

	assert.Contains(t, sql, `"d0_key" = $`)
	assert.Contains(t, sql, `"d0_min" IS NOT NULL`)
	assert.Contains(t, sql, `"d3_min" IS NOT NULL`)
	assert.Contains(t, args, "rmx_1")
	assert.Contains(t, args, int64(2))
}

func TestSelectOptionsQuery_OffersOnlyServedMatrices(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, selectOptionsQuery().Limit(10))

	assert.Contains(t, sql, `"published_version_number" > $`)
	assert.Contains(t, sql, `"status" <> $`)
}
