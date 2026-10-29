package servicefailurereasoncoderepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestReasonCodesByIDsIsOneBoundedStatementInsideTheTenant(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, reasonCodesByIDs([]pulid.ID{"sfrc_1", "sfrc_2"}))

	assert.Contains(t, sql, "= ANY(")
	assert.Contains(t, sql, "LIMIT 2")
	assert.Contains(t, sql, `"organization_id" = `)
	assert.NotContains(t, sql, "users", "a select-option read loads no actors")
	assert.Contains(t, args, string(keeltest.TestOrganizationID))
}

func TestGetByIDsSkipsTheQueryForAnEmptyIDSet(t *testing.T) {
	t.Parallel()

	repo := &repository{db: postgres.NewTestConnection(keeltest.RenderDB(t)), l: zap.NewNop()}

	entities, err := repo.GetByIDs(
		keeltest.TenantContext(t),
		servicefailureports.GetServiceFailureReasonCodesByIDsRequest{},
	)
	require.NoError(t, err)
	assert.Empty(t, entities)
}
