package homelayoutservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/homelayout"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/reporting/canned"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestBuiltInLayoutsReferenceRealCannedReports is the cross-package compile
// gate the domain package cannot host: a canned report retired from the library
// must break the build, not a customer's shipped home screen. It lives here
// because the service layer is the first place allowed to see both packages.
func TestBuiltInLayoutsReferenceRealCannedReports(t *testing.T) {
	t.Parallel()

	registry := canned.Default()

	responsibilities := []permission.CoreResponsibility{
		permission.CoreResponsibilityOperations,
		permission.CoreResponsibilityBilling,
		permission.CoreResponsibilityFinance,
		permission.CoreResponsibilityLeadership,
		"",
	}

	for _, responsibility := range responsibilities {
		for _, widget := range homelayout.BuiltInLayout(responsibility).Widgets {
			if widget.Config.CannedKey == "" {
				continue
			}
			_, ok := registry.Get(widget.Config.CannedKey)
			assert.True(t, ok,
				"shipped layout %q references unknown canned report %q",
				responsibility, widget.Config.CannedKey)
		}
	}
}

func savePresetRequest(req *Request, mutate ...func(*SavePresetRequest)) *SavePresetRequest {
	save := &SavePresetRequest{
		Request: *req,
		Name:    "Dispatch Home",
		Layout: &homelayout.Layout{Widgets: []homelayout.Widget{{
			ID: "widget_1", Key: homelayout.WidgetAttention, W: 4, H: 4,
		}}},
	}
	for _, apply := range mutate {
		apply(save)
	}
	return save
}

// TestCreatePreset_ClaimsOrgDefault: the demote-then-insert handover happens
// inside the repository transaction, so the service's job is to pass the flag
// through untouched.
func TestCreatePreset_ClaimsOrgDefault(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()
	roleID := pulid.MustNew("rol_")
	userA := pulid.MustNew("usr_")
	userB := pulid.MustNew("usr_")

	h.roles.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&permission.Role{ID: roleID}, nil)
	h.repo.EXPECT().
		CreatePreset(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *homelayout.Preset) (*homelayout.Preset, error) {
			entity.ID = pulid.MustNew("hlp_")
			return entity, nil
		})
	h.repo.EXPECT().
		ListRoleUserAssignments(mock.Anything, mock.Anything).
		Return([]repositories.HomeRoleUserAssignment{
			{RoleID: roleID, UserID: userA},
			{RoleID: roleID, UserID: userB},
			{RoleID: pulid.MustNew("rol_"), UserID: pulid.MustNew("usr_")},
		}, nil)

	view, err := h.svc.CreatePreset(t.Context(), savePresetRequest(req,
		func(s *SavePresetRequest) {
			s.IsOrgDefault = true
			s.RoleIDs = []pulid.ID{roleID}
		}))
	require.NoError(t, err)

	assert.True(t, view.Preset.IsOrgDefault)
	assert.Equal(t, 2, view.AssignedUserCount)
}

func TestCreatePreset_RejectsUnknownRoleID(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()

	h.roles.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("Role not found"))

	_, err := h.svc.CreatePreset(t.Context(), savePresetRequest(req,
		func(s *SavePresetRequest) { s.RoleIDs = []pulid.ID{pulid.MustNew("rol_")} }))

	require.Error(t, err)
	h.repo.AssertNotCalled(t, "CreatePreset", mock.Anything, mock.Anything)
}

func TestCreatePreset_TrimsName(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()

	h.repo.EXPECT().
		CreatePreset(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *homelayout.Preset) (*homelayout.Preset, error) {
			entity.ID = pulid.MustNew("hlp_")
			return entity, nil
		})
	h.repo.EXPECT().
		ListRoleUserAssignments(mock.Anything, mock.Anything).
		Return(nil, nil)

	view, err := h.svc.CreatePreset(t.Context(), savePresetRequest(req,
		func(s *SavePresetRequest) { s.Name = "  Dispatch Home  " }))
	require.NoError(t, err)

	assert.Equal(t, "Dispatch Home", view.Preset.Name)
}

func TestCreatePreset_RejectsWhitespaceOnlyName(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()

	_, err := h.svc.CreatePreset(t.Context(), savePresetRequest(req,
		func(s *SavePresetRequest) { s.Name = "   " }))

	require.Error(t, err)
}

func TestCreatePreset_RejectsInvalidLayout(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()

	_, err := h.svc.CreatePreset(t.Context(), savePresetRequest(req, func(s *SavePresetRequest) {
		s.Layout = &homelayout.Layout{Widgets: []homelayout.Widget{{
			ID: "widget_1", Key: "not-a-widget", W: 4, H: 4,
		}}}
	}))

	require.Error(t, err)
}

func TestCreatePreset_RejectsMissingName(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()

	_, err := h.svc.CreatePreset(t.Context(), savePresetRequest(req,
		func(s *SavePresetRequest) { s.Name = "" }))

	require.Error(t, err)
}

func TestCreatePreset_RejectsUnknownCoreResponsibility(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()

	_, err := h.svc.CreatePreset(t.Context(), savePresetRequest(req,
		func(s *SavePresetRequest) { s.CoreResponsibility = "Warehousing" }))

	require.Error(t, err)
}

func TestUpdatePreset_KeepsOrgDefaultFlag(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()
	presetID := pulid.MustNew("hlp_")

	h.repo.EXPECT().
		GetPreset(mock.Anything, mock.Anything).
		Return(&homelayout.Preset{ID: presetID, Name: "Dispatch Home", Version: 2}, nil)
	h.repo.EXPECT().
		UpdatePreset(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *homelayout.Preset) (*homelayout.Preset, error) {
			return entity, nil
		})
	h.repo.EXPECT().
		ListRoleUserAssignments(mock.Anything, mock.Anything).
		Return(nil, nil)

	view, err := h.svc.UpdatePreset(t.Context(), savePresetRequest(req,
		func(s *SavePresetRequest) {
			s.PresetID = presetID
			s.IsOrgDefault = true
			s.Version = 2
		}))
	require.NoError(t, err)

	assert.Equal(t, presetID, view.Preset.ID)
	assert.True(t, view.Preset.IsOrgDefault)
}

// TestListPresets_ReportsReachPerPreset covers all three audience paths in one
// tenant snapshot: an explicit role, a core responsibility (which an explicit
// role list would miss), and an unassigned org default.
func TestListPresets_ReportsReachPerPreset(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()
	roleID := pulid.MustNew("rol_")
	opsRole := pulid.MustNew("rol_")
	sharedUser := pulid.MustNew("usr_")

	h.repo.EXPECT().
		ListPresets(mock.Anything, mock.Anything).
		Return([]*homelayout.Preset{
			{ID: pulid.MustNew("hlp_"), Name: "Dispatch Home", RoleIDs: []pulid.ID{roleID}},
			{
				ID:                 pulid.MustNew("hlp_"),
				Name:               "Ops",
				CoreResponsibility: permission.CoreResponsibilityOperations,
			},
			{ID: pulid.MustNew("hlp_"), Name: "Everyone", IsOrgDefault: true},
		}, nil)
	h.repo.EXPECT().
		ListRoleUserAssignments(mock.Anything, mock.Anything).
		Return([]repositories.HomeRoleUserAssignment{
			{RoleID: roleID, UserID: sharedUser},
			{RoleID: roleID, UserID: pulid.MustNew("usr_")},
			// The same user through two matching roles must count once.
			{
				RoleID:             opsRole,
				UserID:             sharedUser,
				CoreResponsibility: permission.CoreResponsibilityOperations,
			},
		}, nil).
		Once()

	views, err := h.svc.ListPresets(t.Context(), req)
	require.NoError(t, err)

	require.Len(t, views, 3)
	assert.Equal(t, 2, views[0].AssignedUserCount)
	assert.Equal(t, 1, views[1].AssignedUserCount)
	assert.Equal(t, 0, views[2].AssignedUserCount)
}

func TestPreviewPreset_RequiresAPresetOrRole(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()

	_, err := h.svc.PreviewPreset(t.Context(), &PreviewRequest{Request: *req})

	require.Error(t, err)
}

func TestPreviewPreset_RendersNamedPreset(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()
	presetID := pulid.MustNew("hlp_")

	h.repo.EXPECT().
		GetPreset(mock.Anything, mock.Anything).
		Return(&homelayout.Preset{
			ID:   presetID,
			Name: "Dispatch Home",
			Layout: &homelayout.Layout{Widgets: []homelayout.Widget{{
				ID: "widget_1", Key: homelayout.WidgetAttention, W: 4, H: 4,
			}}},
		}, nil)
	allowBatch(h.engine)

	effective, err := h.svc.PreviewPreset(t.Context(), &PreviewRequest{
		Request:  *req,
		PresetID: presetID,
	})
	require.NoError(t, err)

	assert.Equal(t, "Dispatch Home", effective.PresetName)
	assert.False(t, effective.CanCustomize, "a preview is never editable in place")
	assert.Equal(t, []string{homelayout.WidgetAttention}, widgetKeys(effective.Layout))
}

// TestPreviewPreset_FallsBackToShippedLayoutForRole shows an administrator what
// a role sees before any preset has been assigned to it.
func TestPreviewPreset_FallsBackToShippedLayoutForRole(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()
	roleID := pulid.MustNew("rol_")

	h.repo.EXPECT().ListPresets(mock.Anything, mock.Anything).Return(nil, nil)
	h.roles.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&permission.Role{
			ID:                 roleID,
			CoreResponsibility: permission.CoreResponsibilityLeadership,
		}, nil)
	allowBatch(h.engine)

	effective, err := h.svc.PreviewPreset(t.Context(), &PreviewRequest{
		Request: *req,
		RoleID:  roleID,
	})
	require.NoError(t, err)

	assert.Equal(t, SourceBuiltIn, effective.Source)
	assert.Equal(t, "Leadership Home", effective.PresetName)
	assert.NotEmpty(t, effective.Layout.Widgets)
}
