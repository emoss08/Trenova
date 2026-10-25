package rateconfirmationrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/stretchr/testify/assert"
)

func TestDeadTokensQuery_PurgesAcrossTenantsInBoundedBatches(t *testing.T) {
	t.Parallel()

	ctx := dbscope.WithSystem(t.Context(), "test")
	sql, args := keeltest.RenderIn(
		t,
		ctx,
		deadTokensQuery(tenderports.PurgeDeadTokensRequest{DeadBefore: 1_000}, 25).Limit(1),
	)

	assert.Contains(t, sql, `coalesce(dead."used_at", coalesce(dead."revoked_at", dead."expires_at")) < $1`)
	assert.Contains(t, sql, "LIMIT 25")
	assert.NotContains(t, sql, `"organization_id" = $`)
	assert.Contains(t, args, int64(1_000))
}
