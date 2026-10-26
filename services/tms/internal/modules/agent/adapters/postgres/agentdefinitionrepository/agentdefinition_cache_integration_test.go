//go:build integration

package agentdefinitionrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/testutil/cachetest"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetByID_IsServedFromTheResultCacheUntilTheDefinitionChanges(t *testing.T) {
	ctx, db, dsn, cleanup := seededtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	row := findDefinitions(t, ctx, db, "TRUE", 1)[0]
	tctx, tenant := row.tenant(ctx)
	cached := cachetest.Open(t, ctx, dsn)
	repo := New(Params{DB: postgres.NewTestConnection(cached), Logger: zap.NewNop()})
	request := agentdefinitionports.GetAgentDefinitionByIDRequest{ID: row.ID, TenantInfo: tenant}

	var definition *agentdefinition.Definition
	read := func() error {
		var err error
		definition, err = repo.GetByID(tctx, request)
		return err
	}

	cachetest.RequireMiss(t, cached, read)
	cachetest.RequireHit(t, cached, read)
	require.NotNil(t, definition)
	assert.Equal(t, row.ID, definition.ID)

	definition.Name = "Cached Agent Renamed"
	_, err := repo.Update(tctx, definition)
	require.NoError(t, err)

	cachetest.RequireMiss(t, cached, read)
	assert.Equal(t, "Cached Agent Renamed", definition.Name)
	cachetest.RequireHit(t, cached, read)
	assert.Equal(t, "Cached Agent Renamed", definition.Name)
}
