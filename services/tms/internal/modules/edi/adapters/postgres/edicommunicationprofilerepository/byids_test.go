package edicommunicationprofilerepository

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

func TestProfilesByIDsIsOneBoundedStatementInsideTheTenant(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, profilesByIDs([]pulid.ID{"ecp_1", "ecp_2"}))

	assert.Contains(t, sql, "= ANY(")
	assert.Contains(t, sql, "LIMIT 2")
	assert.Contains(t, sql, `"organization_id" = `)
	assert.Contains(t, args, string(keeltest.TestOrganizationID))
}

func TestGetProfilesByIDsSkipsTheQueryForAnEmptyIDSet(t *testing.T) {
	t.Parallel()

	repo := &repository{db: postgres.NewTestConnection(keeltest.RenderDB(t)), l: zap.NewNop()}

	entities, err := repo.GetProfilesByIDs(
		keeltest.TenantContext(t),
		edipartnerports.GetEDICommunicationProfilesByIDsRequest{},
	)
	require.NoError(t, err)
	assert.Empty(t, entities)
}
