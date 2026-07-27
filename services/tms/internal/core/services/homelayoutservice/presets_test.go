package homelayoutservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/homelayout"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/reporting/canned"
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

// TestCreatePreset_DemotesIncumbentOrgDefault: the partial unique index would
// reject a second default outright, so promotion has to be a handover.
func TestCreatePreset_DemotesIncumbentOrgDefault(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()

	h.repo.EXPECT().
		ClearOrgDefault(mock.Anything, mock.Anything).
		Return(nil).
		Once()
	h.repo.EXPECT().
		CreatePreset(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *homelayout.Preset) (*homelayout.Preset, error) {
			entity.ID = pulid.MustNew("hlp_")
			return entity, nil
		})
	h.repo.EXPECT().
		CountUsersForRoles(mock.Anything, mock.Anything, mock.Anything).
		Return(4, nil)

	view, err := h.svc.CreatePreset(t.Context(), savePresetRequest(req,
		func(s *SavePresetRequest) { s.IsOrgDefault = true }))
	require.NoError(t, err)

	assert.True(t, view.Preset.IsOrgDefault)
	assert.Equal(t, 4, view.AssignedUserCount)
}

// TestCreatePreset_LeavesDefaultAloneWhenNotClaimingIt guards against every
// save quietly demoting whichever preset was the fallback.
func TestCreatePreset_LeavesDefaultAloneWhenNotClaimingIt(t *testing.T) {
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
		CountUsersForRoles(mock.Anything, mock.Anything, mock.Anything).
		Return(0, nil)

	_, err := h.svc.CreatePreset(t.Context(), savePresetRequest(req))
	require.NoError(t, err)

	h.repo.AssertNotCalled(t, "ClearOrgDefault", mock.Anything, mock.Anything)
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

func TestUpdatePreset_DemotesEveryOtherDefault(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()
	presetID := pulid.MustNew("hlp_")

	h.repo.EXPECT().
		GetPreset(mock.Anything, mock.Anything).
		Return(&homelayout.Preset{ID: presetID, Name: "Dispatch Home", Version: 2}, nil)
	h.repo.EXPECT().
		ClearOrgDefault(mock.Anything, mock.MatchedBy(
			func(r *repositories.ClearHomeLayoutOrgDefaultRequest) bool {
				return r.ExceptID == presetID
			})).
		Return(nil).
		Once()
	h.repo.EXPECT().
		UpdatePreset(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *homelayout.Preset) (*homelayout.Preset, error) {
			return entity, nil
		})
	h.repo.EXPECT().
		CountUsersForRoles(mock.Anything, mock.Anything, mock.Anything).
		Return(2, nil)

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

func TestListPresets_ReportsReachPerPreset(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := testRequest()
	roleID := pulid.MustNew("rol_")

	h.repo.EXPECT().
		ListPresets(mock.Anything, mock.Anything).
		Return([]*homelayout.Preset{
			{ID: pulid.MustNew("hlp_"), Name: "Dispatch Home", RoleIDs: []pulid.ID{roleID}},
			{ID: pulid.MustNew("hlp_"), Name: "Everyone", IsOrgDefault: true},
		}, nil)
	h.repo.EXPECT().
		CountUsersForRoles(mock.Anything, mock.Anything, []pulid.ID{roleID}).
		Return(7, nil)
	h.repo.EXPECT().
		CountUsersForRoles(mock.Anything, mock.Anything, []pulid.ID(nil)).
		Return(0, nil)

	views, err := h.svc.ListPresets(t.Context(), req)
	require.NoError(t, err)

	require.Len(t, views, 2)
	assert.Equal(t, 7, views[0].AssignedUserCount)
	assert.Equal(t, 0, views[1].AssignedUserCount)
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
