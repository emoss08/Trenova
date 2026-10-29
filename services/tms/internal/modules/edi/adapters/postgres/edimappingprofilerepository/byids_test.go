package edimappingprofilerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMappingProfilesByIDsIsOneBoundedStatementInsideTheTenant(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, mappingProfilesByIDs([]pulid.ID{"emp_1", "emp_2"}))

	assert.Contains(t, sql, "= ANY(")
	assert.Contains(t, sql, "LIMIT 2")
	assert.Contains(t, sql, `"organization_id" = `)
	assert.Contains(t, args, string(keeltest.TestOrganizationID))
}

func TestGetMappingProfilesByIDsSkipsTheQueryForAnEmptyIDSet(t *testing.T) {
	t.Parallel()

	repo := &repository{db: postgres.NewTestConnection(keeltest.RenderDB(t)), l: zap.NewNop()}

	entities, err := repo.GetMappingProfilesByIDs(
		keeltest.TenantContext(t),
		edipartnerports.GetMappingProfilesByIDsRequest{},
	)
	require.NoError(t, err)
	assert.Empty(t, entities)
}
