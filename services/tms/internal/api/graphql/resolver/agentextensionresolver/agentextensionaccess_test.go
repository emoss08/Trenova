package agentextensionresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionwebwatch"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func grantKey(resource permission.Resource, operation permission.Operation) string {
	return resource.String() + "|" + string(operation)
}

func TestAgentExtensionResolvers_CheckTheAgentExtensionPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("apst_").String()
	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"agentExtensionCatalog", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).AgentExtensionCatalog(ctx)
			return err
		}},
		{"agentExtensionConfig", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).AgentExtensionConfig(ctx, "Exa")
			return err
		}},
		{"agentPortalSites", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).AgentPortalSites(ctx)
			return err
		}},
		{"agentPortalRuns", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).AgentPortalRuns(ctx, nil, nil, nil)
			return err
		}},
		{"agentPortalRunRecording", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).AgentPortalRunRecording(ctx, id)
			return err
		}},
		{"agentWebWatches", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).AgentWebWatches(ctx, nil, nil, nil)
			return err
		}},
		{"agentWebWatchFindings", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).AgentWebWatchFindings(ctx, id)
			return err
		}},
		{
			"updateAgentExtensionConfig",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateAgentExtensionConfig(
					ctx,
					"Exa",
					gqlmodel.UpdateAgentExtensionConfigInput{},
				)
				return err
			},
		},
		{"testAgentExtension", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).TestAgentExtension(ctx, "Exa")
			return err
		}},
		{"createAgentPortalSite", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateAgentPortalSite(
				ctx,
				gqlmodel.SaveAgentPortalSiteInput{},
			)
			return err
		}},
		{"updateAgentPortalSite", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateAgentPortalSite(
				ctx,
				id,
				gqlmodel.SaveAgentPortalSiteInput{},
			)
			return err
		}},
		{"deleteAgentPortalSite", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).DeleteAgentPortalSite(ctx, id)
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(
				t,
				err,
				engine,
				permission.ResourceAgentExtension,
				call.operation,
			)
		})
	}
}

func TestPortalRunAccess_AdministersOnlyWithTheExtensionPermission(t *testing.T) {
	t.Parallel()

	runID := pulid.MustNew("aprn_")
	cases := []struct {
		name        string
		granted     []string
		administer  permission.Operation
		administers bool
		refused     bool
	}{
		{
			name:        "someone who administers agents reaches any run",
			granted:     []string{grantKey(permission.ResourceAgentExtension, permission.OpRead)},
			administer:  permission.OpRead,
			administers: true,
		},
		{
			name: "the person a run was asked for reaches it through portal automation",
			granted: []string{
				grantKey(permission.ResourcePortalAutomation, permission.OpRead),
			},
			administer: permission.OpUpdate,
		},
		{
			name: "reading agent extensions does not take over another's live browser",
			granted: []string{
				grantKey(permission.ResourceAgentExtension, permission.OpRead),
				grantKey(permission.ResourcePortalAutomation, permission.OpRead),
			},
			administer: permission.OpUpdate,
		},
		{
			name:       "neither is refused",
			administer: permission.OpRead,
			refused:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			for _, key := range tc.granted {
				engine.Granted[key] = true
			}
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			authCtx := resolvertest.UserAuthContext()
			ctx := resolvertest.AuthedContext(t, authCtx)

			req, err := deps.portalRunAccess(ctx, runID.String(), tc.administer)

			if tc.refused {
				var authzErr *errortypes.AuthorizationError
				require.ErrorAs(t, err, &authzErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, runID, req.RunID)
			assert.Equal(t, tc.administers, req.Administers)
			assert.Equal(t, authCtx.UserID, req.TenantInfo.UserID)
			assert.Equal(t, authCtx.OrganizationID, req.TenantInfo.OrgID)
		})
	}
}

func TestStopAgentWebWatch_RefusesWithoutEitherPermission(t *testing.T) {
	t.Parallel()

	engine := resolvertest.RefusingEngine()
	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

	_, err := (&MutationResolver{deps}).StopAgentWebWatch(ctx, pulid.MustNew("awwt_").String())

	var authzErr *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authzErr)
	asked := make([]string, 0, len(engine.Requests))
	for _, req := range engine.Requests {
		asked = append(asked, req.Resource+"|"+string(req.Operation))
	}
	assert.ElementsMatch(t, []string{
		grantKey(permission.ResourceAgentExtension, permission.OpUpdate),
		grantKey(permission.ResourceWebResearch, permission.OpRead),
	}, asked)
}

func TestPortalRunsRequest_BoundsThePage(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_")}
	site := pulid.MustNew("apst_")
	siteRaw := site.String()
	before, limit := 1_790_000_000, 25

	req, err := portalRunsRequest(tenant, &siteRaw, &before, &limit)
	require.NoError(t, err)
	assert.Equal(t, site, req.SiteID)
	assert.Equal(t, int64(1_790_000_000), req.Before)
	assert.Equal(t, 25, req.Limit)
	assert.Equal(t, tenant, req.TenantInfo)

	tooMany := maxPortalRunsPerPage + 1
	_, err = portalRunsRequest(tenant, nil, nil, &tooMany)
	var invalid *errortypes.Error
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "limit", invalid.Field)

	negative := -1
	_, err = portalRunsRequest(tenant, nil, &negative, nil)
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "before", invalid.Field)
}

func TestWebWatchesRequest_TakesOnlyAKnownStatus(t *testing.T) {
	t.Parallel()

	active := string(extensionwebwatch.WebWatchStatusActive)
	req, err := webWatchesRequest(pagination.TenantInfo{}, &active, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, extensionwebwatch.WebWatchStatusActive, req.Status)

	unknown := "Snoozed"
	_, err = webWatchesRequest(pagination.TenantInfo{}, &unknown, nil, nil)
	var invalid *errortypes.Error
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "status", invalid.Field)

	tooMany := maxWebWatchesPerPage + 1
	_, err = webWatchesRequest(pagination.TenantInfo{}, nil, nil, &tooMany)
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "limit", invalid.Field)
}

func TestUpdateConfigRequest_MapsTheSettingsByKey(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_")}
	req, err := updateConfigRequest(tenant, &gqlmodel.UpdateAgentExtensionConfigInput{
		Enabled:      true,
		Availability: string(agentextension.AvailabilityAllAgents),
		Configuration: []*gqlmodel.AgentExtensionSettingInput{
			{Key: "apiKey", Value: "exa-secret"},
		},
		Version: 4,
	})
	require.NoError(t, err)

	assert.Equal(t, tenant, req.TenantInfo)
	assert.True(t, req.Enabled)
	assert.Equal(t, agentextension.AvailabilityAllAgents, req.Availability)
	assert.Equal(t, map[string]string{"apiKey": "exa-secret"}, req.Configuration)
	assert.Equal(t, int64(4), req.Version)

	_, err = updateConfigRequest(tenant, &gqlmodel.UpdateAgentExtensionConfigInput{
		Configuration: []*gqlmodel.AgentExtensionSettingInput{
			{Key: "apiKey", Value: "a"},
			{Key: "apiKey", Value: "b"},
		},
	})
	var invalid *errortypes.Error
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "configuration", invalid.Field)
}

func TestPortalSiteRequest_CarriesTheSignInWithoutKeepingIt(t *testing.T) {
	t.Parallel()

	req := portalSiteRequest(&gqlmodel.SaveAgentPortalSiteInput{
		Name:        "Acme DC",
		Domains:     []string{"portal.acme.com"},
		StartURL:    "https://portal.acme.com/login",
		Username:    "dispatch@trenova.example",
		Password:    "hunter2",
		ClearSignIn: false,
		Enabled:     true,
		Version:     2,
	})

	assert.Equal(t, "Acme DC", req.Name)
	assert.Equal(t, []string{"portal.acme.com"}, req.Domains)
	assert.Equal(t, "https://portal.acme.com/login", req.StartURL)
	assert.Equal(t, "dispatch@trenova.example", req.Username)
	assert.Equal(t, "hunter2", req.Password)
	assert.True(t, req.Enabled)
	assert.Equal(t, int64(2), req.Version)
	assert.True(t, req.ID.IsNil(), "the site id comes from the mutation, not the input")
}

func TestAgentPortalRunFields_CarryTheDetailsAndCostForTheWire(t *testing.T) {
	t.Parallel()

	run := &agentextensionports.PortalRunView{
		Details: map[string]string{"po": "PO-118"},
		CostUSD: decimal.RequireFromString("0.0300"),
	}
	resolver := &AgentPortalRunResolver{}

	details, err := resolver.Details(t.Context(), run)
	require.NoError(t, err)
	cost, err := resolver.CostUsd(t.Context(), run)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{"po": "PO-118"}, details)
	assert.Equal(t, "0.03", cost)
}
