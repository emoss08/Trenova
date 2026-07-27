package homelayoutservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/homelayout"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var _ repositories.HomeLayoutRepository = (*mocks.MockHomeLayoutRepository)(nil)

type harness struct {
	svc    *Service
	repo   *mocks.MockHomeLayoutRepository
	rbac   *mocks.MockRBACRepository
	roles  *mocks.MockRoleRepository
	engine *mocks.MockPermissionEngine
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	repo := mocks.NewMockHomeLayoutRepository(t)
	rbac := mocks.NewMockRBACRepository(t)
	roles := mocks.NewMockRoleRepository(t)
	engine := mocks.NewMockPermissionEngine(t)

	return &harness{
		svc: New(Params{
			Logger:           zap.NewNop(),
			Repo:             repo,
			RBACRepo:         rbac,
			RoleRepo:         roles,
			PermissionEngine: engine,
		}),
		repo:   repo,
		rbac:   rbac,
		roles:  roles,
		engine: engine,
	}
}

func testRequest() *Request {
	userID := pulid.MustNew("usr_")
	return &Request{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: userID,
		},
		Principal: services.PrincipalInfo{UserID: userID},
	}
}

// allowBatch answers every permission check, denying only the named resources.
func allowBatch(engine *mocks.MockPermissionEngine, denied ...permission.Resource) {
	deniedSet := make(map[string]struct{}, len(denied))
	for _, resource := range denied {
		deniedSet[resource.String()] = struct{}{}
	}

	engine.EXPECT().
		CheckBatch(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *services.BatchPermissionCheckRequest,
		) (*services.BatchPermissionCheckResult, error) {
			results := make([]services.PermissionCheckResult, 0, len(req.Checks))
			for _, check := range req.Checks {
				_, deny := deniedSet[check.Resource]
				results = append(results, services.PermissionCheckResult{Allowed: !deny})
			}
			return &services.BatchPermissionCheckResult{Results: results}, nil
		})
}

func role(responsibility permission.CoreResponsibility) *permission.Role {
	return &permission.Role{ID: pulid.MustNew("rol_"), CoreResponsibility: responsibility}
}

func preset(name string, opts ...func(*homelayout.Preset)) *homelayout.Preset {
	entity := &homelayout.Preset{
		ID:   pulid.MustNew("hlp_"),
		Name: name,
		Layout: &homelayout.Layout{Widgets: []homelayout.Widget{{
			ID: "widget_1", Key: homelayout.WidgetAttention, W: 4, H: 4,
		}}},
	}
	for _, opt := range opts {
		opt(entity)
	}
	return entity
}

func widgetKeys(layout *homelayout.Layout) []string {
	keys := make([]string, 0, len(layout.Widgets))
	for _, widget := range layout.Widgets {
		keys = append(keys, widget.Key)
	}
	return keys
}

func TestResolve_FallsBackToBuiltInForCoreResponsibility(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()

	h.repo.EXPECT().GetPreference(mock.Anything, mock.Anything).Return(nil, false, nil)
	h.rbac.EXPECT().
		GetAuthorizedRoles(mock.Anything, mock.Anything, mock.Anything).
		Return([]*permission.Role{role(permission.CoreResponsibilityBilling)}, nil)
	h.repo.EXPECT().ListPresets(mock.Anything, mock.Anything).Return(nil, nil)
	allowBatch(h.engine)

	effective, err := h.svc.GetEffective(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, SourceBuiltIn, effective.Source)
	assert.Equal(t, "Billing Home", effective.PresetName)
	assert.True(t, effective.CanCustomize)
	assert.False(t, effective.Locked)
	assert.NotEmpty(t, effective.Layout.Widgets)
}

func TestResolve_PrefersRolePresetOverOrgDefault(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()
	dispatcher := role(permission.CoreResponsibilityOperations)

	assigned := preset("Dispatch Home", func(p *homelayout.Preset) {
		p.RoleIDs = []pulid.ID{dispatcher.ID}
	})
	fallback := preset("Everyone", func(p *homelayout.Preset) { p.IsOrgDefault = true })

	h.repo.EXPECT().GetPreference(mock.Anything, mock.Anything).Return(nil, false, nil)
	h.rbac.EXPECT().
		GetAuthorizedRoles(mock.Anything, mock.Anything, mock.Anything).
		Return([]*permission.Role{dispatcher}, nil)
	h.repo.EXPECT().
		ListPresets(mock.Anything, mock.Anything).
		Return([]*homelayout.Preset{fallback, assigned}, nil)
	allowBatch(h.engine)

	effective, err := h.svc.GetEffective(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, SourceRolePreset, effective.Source)
	assert.Equal(t, "Dispatch Home", effective.PresetName)
	assert.Equal(t, assigned.ID, effective.PresetID)
}

// TestResolve_MatchesPresetByCoreResponsibility covers the assignment an
// administrator makes without naming individual roles.
func TestResolve_MatchesPresetByCoreResponsibility(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()

	assigned := preset("Finance Home", func(p *homelayout.Preset) {
		p.CoreResponsibility = permission.CoreResponsibilityFinance
	})

	h.repo.EXPECT().GetPreference(mock.Anything, mock.Anything).Return(nil, false, nil)
	h.rbac.EXPECT().
		GetAuthorizedRoles(mock.Anything, mock.Anything, mock.Anything).
		Return([]*permission.Role{role(permission.CoreResponsibilityFinance)}, nil)
	h.repo.EXPECT().
		ListPresets(mock.Anything, mock.Anything).
		Return([]*homelayout.Preset{assigned}, nil)
	allowBatch(h.engine)

	effective, err := h.svc.GetEffective(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, SourceRolePreset, effective.Source)
	assert.Equal(t, "Finance Home", effective.PresetName)
}

func TestResolve_UsesOrgDefaultWhenNoRoleMatches(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()
	fallback := preset("Everyone", func(p *homelayout.Preset) { p.IsOrgDefault = true })

	h.repo.EXPECT().GetPreference(mock.Anything, mock.Anything).Return(nil, false, nil)
	h.rbac.EXPECT().
		GetAuthorizedRoles(mock.Anything, mock.Anything, mock.Anything).
		Return([]*permission.Role{role("")}, nil)
	h.repo.EXPECT().
		ListPresets(mock.Anything, mock.Anything).
		Return([]*homelayout.Preset{fallback}, nil)
	allowBatch(h.engine)

	effective, err := h.svc.GetEffective(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, SourceOrgDefault, effective.Source)
	assert.Equal(t, fallback.ID, effective.PresetID)
}

func TestResolve_UserLayoutWinsOverAssignedPreset(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()
	dispatcher := role(permission.CoreResponsibilityOperations)
	assigned := preset("Dispatch Home", func(p *homelayout.Preset) {
		p.RoleIDs = []pulid.ID{dispatcher.ID}
	})

	h.repo.EXPECT().GetPreference(mock.Anything, mock.Anything).Return(&homelayout.Preference{
		Version: 3,
		Preferences: &homelayout.Document{
			SchemaVersion: homelayout.DocumentSchemaVersion,
			Mode:          homelayout.ModeCustom,
			Density:       homelayout.DensityCompact,
			Layout: &homelayout.Layout{Widgets: []homelayout.Widget{{
				ID: "widget_1", Key: homelayout.WidgetFavorites, W: 4, H: 4,
			}}},
		},
	}, true, nil)
	h.rbac.EXPECT().
		GetAuthorizedRoles(mock.Anything, mock.Anything, mock.Anything).
		Return([]*permission.Role{dispatcher}, nil)
	h.repo.EXPECT().
		ListPresets(mock.Anything, mock.Anything).
		Return([]*homelayout.Preset{assigned}, nil)
	allowBatch(h.engine)

	effective, err := h.svc.GetEffective(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, SourceUser, effective.Source)
	assert.Equal(t, []string{homelayout.WidgetFavorites}, widgetKeys(effective.Layout))
	assert.Equal(t, homelayout.DensityCompact, effective.Density)
	assert.Equal(t, int64(3), effective.Version)
	// The user is told what they diverged from, so "reset" can be labelled.
	assert.Equal(t, "Dispatch Home", effective.PresetName)
}

// TestResolve_LockedPresetOverridesUserLayout is the guarantee an administrator
// buys when they lock a preset — and it must not destroy what the user saved.
func TestResolve_LockedPresetOverridesUserLayout(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()
	dispatcher := role(permission.CoreResponsibilityOperations)
	assigned := preset("Dispatch Home", func(p *homelayout.Preset) {
		p.RoleIDs = []pulid.ID{dispatcher.ID}
		p.Locked = true
	})

	h.repo.EXPECT().GetPreference(mock.Anything, mock.Anything).Return(&homelayout.Preference{
		Version: 2,
		Preferences: &homelayout.Document{
			SchemaVersion: homelayout.DocumentSchemaVersion,
			Mode:          homelayout.ModeCustom,
			Density:       homelayout.DensityComfortable,
			Layout: &homelayout.Layout{Widgets: []homelayout.Widget{{
				ID: "widget_1", Key: homelayout.WidgetFavorites, W: 4, H: 4,
			}}},
		},
	}, true, nil)
	h.rbac.EXPECT().
		GetAuthorizedRoles(mock.Anything, mock.Anything, mock.Anything).
		Return([]*permission.Role{dispatcher}, nil)
	h.repo.EXPECT().
		ListPresets(mock.Anything, mock.Anything).
		Return([]*homelayout.Preset{assigned}, nil)
	allowBatch(h.engine)

	effective, err := h.svc.GetEffective(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, SourceRolePreset, effective.Source)
	assert.Equal(t, []string{homelayout.WidgetAttention}, widgetKeys(effective.Layout))
	assert.True(t, effective.Locked)
	assert.False(t, effective.CanCustomize)
}

// TestUpdate_RefusedWhileLocked keeps the lock honest against a direct API
// call rather than relying on the button being disabled.
func TestUpdate_RefusedWhileLocked(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()
	dispatcher := role(permission.CoreResponsibilityOperations)
	assigned := preset("Dispatch Home", func(p *homelayout.Preset) {
		p.RoleIDs = []pulid.ID{dispatcher.ID}
		p.Locked = true
	})

	h.rbac.EXPECT().
		GetAuthorizedRoles(mock.Anything, mock.Anything, mock.Anything).
		Return([]*permission.Role{dispatcher}, nil)
	h.repo.EXPECT().
		ListPresets(mock.Anything, mock.Anything).
		Return([]*homelayout.Preset{assigned}, nil)

	_, err := h.svc.Update(t.Context(), &UpdateRequest{
		Request: *req,
		Document: &homelayout.Document{
			SchemaVersion: homelayout.DocumentSchemaVersion,
			Mode:          homelayout.ModeCustom,
			Density:       homelayout.DensityComfortable,
			Layout: &homelayout.Layout{Widgets: []homelayout.Widget{{
				ID: "widget_1", Key: homelayout.WidgetFavorites, W: 4, H: 4,
			}}},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "administrator")
}

// TestResolve_DropsWidgetsTheViewerCannotRead is the fail-closed behaviour: a
// revoked resource costs a card, never an error.
func TestResolve_DropsWidgetsTheViewerCannotRead(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()

	assigned := preset("Mixed", func(p *homelayout.Preset) {
		p.IsOrgDefault = true
		p.Layout = &homelayout.Layout{Widgets: []homelayout.Widget{
			{ID: "widget_1", Key: homelayout.WidgetUnassigned, W: 4, H: 5},
			{ID: "widget_2", Key: homelayout.WidgetARSnapshot, W: 6, H: 3},
			{ID: "widget_3", Key: homelayout.WidgetFavorites, W: 4, H: 4},
		}}
	})

	h.repo.EXPECT().GetPreference(mock.Anything, mock.Anything).Return(nil, false, nil)
	h.rbac.EXPECT().
		GetAuthorizedRoles(mock.Anything, mock.Anything, mock.Anything).
		Return([]*permission.Role{role("")}, nil)
	h.repo.EXPECT().
		ListPresets(mock.Anything, mock.Anything).
		Return([]*homelayout.Preset{assigned}, nil)
	allowBatch(h.engine, permission.ResourceAccountsReceivable)

	effective, err := h.svc.GetEffective(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(
		t,
		[]string{homelayout.WidgetUnassigned, homelayout.WidgetFavorites},
		widgetKeys(effective.Layout),
	)
}

// TestResolve_DropsMetricsTheViewerCannotRead narrows inside a widget rather
// than dropping the whole strip, so a dispatcher keeps the shipment numbers.
func TestResolve_DropsMetricsTheViewerCannotRead(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()

	assigned := preset("Strip", func(p *homelayout.Preset) {
		p.IsOrgDefault = true
		p.Layout = &homelayout.Layout{Widgets: []homelayout.Widget{{
			ID: "widget_1", Key: homelayout.WidgetKPIRow, W: 12, H: 2,
			Config: homelayout.WidgetConfig{
				Metrics: []string{"activeShipments", "arTotalOpen", "onTimePercent"},
			},
		}}}
	})

	h.repo.EXPECT().GetPreference(mock.Anything, mock.Anything).Return(nil, false, nil)
	h.rbac.EXPECT().
		GetAuthorizedRoles(mock.Anything, mock.Anything, mock.Anything).
		Return([]*permission.Role{role("")}, nil)
	h.repo.EXPECT().
		ListPresets(mock.Anything, mock.Anything).
		Return([]*homelayout.Preset{assigned}, nil)
	allowBatch(h.engine, permission.ResourceAccountsReceivable)

	effective, err := h.svc.GetEffective(t.Context(), req)
	require.NoError(t, err)

	require.Len(t, effective.Layout.Widgets, 1)
	assert.Equal(
		t,
		[]string{"activeShipments", "onTimePercent"},
		effective.Layout.Widgets[0].Config.Metrics,
	)
}

// TestResolve_DropsMetricWidgetLeftWithNothing avoids rendering a card that
// resolved to no measure at all.
func TestResolve_DropsMetricWidgetLeftWithNothing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()

	assigned := preset("Single", func(p *homelayout.Preset) {
		p.IsOrgDefault = true
		p.Layout = &homelayout.Layout{Widgets: []homelayout.Widget{{
			ID: "widget_1", Key: homelayout.WidgetKPI, W: 3, H: 2,
			Config: homelayout.WidgetConfig{Metric: "arTotalOpen"},
		}}}
	})

	h.repo.EXPECT().GetPreference(mock.Anything, mock.Anything).Return(nil, false, nil)
	h.rbac.EXPECT().
		GetAuthorizedRoles(mock.Anything, mock.Anything, mock.Anything).
		Return([]*permission.Role{role("")}, nil)
	h.repo.EXPECT().
		ListPresets(mock.Anything, mock.Anything).
		Return([]*homelayout.Preset{assigned}, nil)
	allowBatch(h.engine, permission.ResourceAccountsReceivable)

	effective, err := h.svc.GetEffective(t.Context(), req)
	require.NoError(t, err)

	assert.Empty(t, effective.Layout.Widgets)
}

// TestResolve_DeletedPresetLeavesCustomLayoutIntact: a user's layout is
// self-contained, so removing the preset they forked from does not take their
// home screen with it.
func TestResolve_DeletedPresetLeavesCustomLayoutIntact(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()

	h.repo.EXPECT().GetPreference(mock.Anything, mock.Anything).Return(&homelayout.Preference{
		Version: 5,
		Preferences: &homelayout.Document{
			SchemaVersion:   homelayout.DocumentSchemaVersion,
			Mode:            homelayout.ModeCustom,
			Density:         homelayout.DensityComfortable,
			BasedOnPresetID: pulid.MustNew("hlp_"),
			Layout: &homelayout.Layout{Widgets: []homelayout.Widget{{
				ID: "widget_1", Key: homelayout.WidgetFavorites, W: 4, H: 4,
			}}},
		},
	}, true, nil)
	h.rbac.EXPECT().
		GetAuthorizedRoles(mock.Anything, mock.Anything, mock.Anything).
		Return([]*permission.Role{role(permission.CoreResponsibilityOperations)}, nil)
	h.repo.EXPECT().ListPresets(mock.Anything, mock.Anything).Return(nil, nil)
	allowBatch(h.engine)

	effective, err := h.svc.GetEffective(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, SourceUser, effective.Source)
	assert.Equal(t, []string{homelayout.WidgetFavorites}, widgetKeys(effective.Layout))
	assert.True(t, effective.PresetID.IsNil())
	assert.Equal(t, "Operations Home", effective.PresetName)
}

// TestResolve_DropsRetiredWidgetKeys proves a stored layout survives a widget
// being removed from the catalog.
func TestResolve_DropsRetiredWidgetKeys(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()

	h.repo.EXPECT().GetPreference(mock.Anything, mock.Anything).Return(&homelayout.Preference{
		Version: 1,
		Preferences: &homelayout.Document{
			SchemaVersion: homelayout.DocumentSchemaVersion,
			Mode:          homelayout.ModeCustom,
			Density:       homelayout.DensityComfortable,
			Layout: &homelayout.Layout{Widgets: []homelayout.Widget{
				{ID: "widget_1", Key: "retired-widget", W: 4, H: 4},
				{ID: "widget_2", Key: homelayout.WidgetFavorites, W: 4, H: 4},
			}},
		},
	}, true, nil)
	h.rbac.EXPECT().
		GetAuthorizedRoles(mock.Anything, mock.Anything, mock.Anything).
		Return([]*permission.Role{role("")}, nil)
	h.repo.EXPECT().ListPresets(mock.Anything, mock.Anything).Return(nil, nil)
	allowBatch(h.engine)

	effective, err := h.svc.GetEffective(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, []string{homelayout.WidgetFavorites}, widgetKeys(effective.Layout))
}

func TestGetCatalog_FiltersByPermission(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()

	allowBatch(h.engine, permission.ResourceAccountsReceivable)

	catalog, err := h.svc.GetCatalog(t.Context(), req)
	require.NoError(t, err)

	for _, widget := range catalog.Widgets {
		assert.NotEqual(t, homelayout.WidgetARSnapshot, widget.Key)
	}
	for _, metric := range catalog.Metrics {
		assert.NotEqual(t, "arTotalOpen", metric.Key)
	}

	// Ungated widgets stay: their contents are filtered as they render.
	keys := make(map[string]struct{}, len(catalog.Widgets))
	for _, widget := range catalog.Widgets {
		keys[widget.Key] = struct{}{}
	}
	assert.Contains(t, keys, homelayout.WidgetAttention)
	assert.Contains(t, keys, homelayout.WidgetQuickActions)
}
