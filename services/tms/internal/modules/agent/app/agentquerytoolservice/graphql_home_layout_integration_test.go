//go:build integration

package agentquerytoolservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/rbacrepository"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/rolerepository"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/homelayoutrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/app/homelayoutservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// A person's home page is read as the page resolves it, under row-level
// security: the built-in one until they save their own, then theirs, with its
// version for the tools that change it and each widget named as the page
// names it.
func TestConvertedGets_HomeLayoutReadsThePersonsOwnPage(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	_, resolvers := convertedListCases(conn, admin)
	logger := zap.NewNop()
	engine := &mocks.DenyResourcesPermissionEngine{}
	layouts := homelayoutservice.New(homelayoutservice.Params{
		Logger:           logger,
		Repo:             homelayoutrepository.New(homelayoutrepository.Params{DB: conn, Logger: logger}),
		RBACRepo:         rbacrepository.New(rbacrepository.Params{DB: conn, Logger: logger}),
		RoleRepo:         rolerepository.New(rolerepository.Params{DB: conn, Logger: logger}),
		PermissionEngine: engine,
	})
	resolvers.HomeLayoutService = layouts
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: resolvers})
	get, err := agentquerytoolservice.ConvertedGetTool("home_get_layout", reader, engine)
	require.NoError(t, err)

	tenant := agentreadtest.SeedTenant(t, admin, "A")
	actor := tenant.Actor
	ctx := readscope.With(dbscope.WithTenant(t.Context(), tenant.Scope()), nil)

	page := callTool(t, ctx, get, actor, map[string]any{})
	assert.Equal(t, "BUILT_IN", page["source"])
	widgets, _ := page["widgets"].([]any)
	require.NotEmpty(t, widgets)
	for _, raw := range widgets {
		widget, _ := raw.(map[string]any)
		assert.NotEmpty(t, widget["label"], "each widget is named as the page names it")
	}
	assert.Contains(t, page["note"], "home_add_widget")

	request := homelayoutservice.Request{
		TenantInfo: pagination.TenantInfo{
			OrgID: actor.OrganizationID, BuID: actor.BusinessUnitID, UserID: actor.UserID,
		},
		Principal: kernelports.PrincipalInfo{
			Type: actor.PrincipalType, ID: actor.PrincipalID, UserID: actor.UserID,
		},
	}
	effective, err := layouts.GetEffective(ctx, &request)
	require.NoError(t, err)
	_, err = layouts.Update(ctx, &homelayoutservice.UpdateRequest{
		Request: request,
		Document: &homelayout.Document{
			SchemaVersion: homelayout.DocumentSchemaVersion,
			Mode:          homelayout.ModeCustom,
			Layout:        effective.Layout,
			Density:       homelayout.DensityComfortable,
		},
	})
	require.NoError(t, err)

	saved := callTool(t, ctx, get, actor, map[string]any{})
	assert.Equal(t, "USER", saved["source"])
	assert.EqualValues(t, 1, saved["version"])
}
