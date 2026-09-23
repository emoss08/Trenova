package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/homelayout"
	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/homelayoutservice"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeHomeLayouts struct {
	effective *homelayoutservice.EffectiveLayout
	catalog   *homelayoutservice.WidgetCatalog
	updated   *homelayoutservice.UpdateRequest
}

func (f *fakeHomeLayouts) GetEffective(
	context.Context,
	*homelayoutservice.Request,
) (*homelayoutservice.EffectiveLayout, error) {
	return f.effective, nil
}

func (f *fakeHomeLayouts) GetCatalog(
	context.Context,
	*homelayoutservice.Request,
) (*homelayoutservice.WidgetCatalog, error) {
	return f.catalog, nil
}

func (f *fakeHomeLayouts) Update(
	_ context.Context,
	req *homelayoutservice.UpdateRequest,
) (*homelayoutservice.EffectiveLayout, error) {
	f.updated = req
	return f.effective, nil
}

type fakeHomeTargets struct {
	reports map[pulid.ID]bool
}

func (f *fakeHomeTargets) UnavailableTileTargets(
	_ context.Context,
	_ pagination.TenantInfo,
	tiles []report.DashboardTile,
) []reporting.UnavailableTile {
	var missing []reporting.UnavailableTile
	for idx := range tiles {
		if !f.reports[tiles[idx].DefinitionID] {
			missing = append(missing, reporting.UnavailableTile{Index: idx})
		}
	}
	return missing
}

func (f *fakeHomeTargets) GetDashboard(
	context.Context,
	*reporting.GetDashboardRequest,
) (*report.Dashboard, error) {
	return nil, errortypes.NewNotFoundError("Dashboard not found")
}

func widgetDefinition(t *testing.T, key string) homelayout.WidgetDefinition {
	t.Helper()
	definition, ok := homelayout.WidgetDefinitionFor(key)
	require.True(t, ok, key)
	return definition
}

func homeFixture(t *testing.T) (*fakeHomeLayouts, homeEditor) {
	t.Helper()

	layouts := &fakeHomeLayouts{
		effective: &homelayoutservice.EffectiveLayout{
			Layout: &homelayout.Layout{Widgets: []homelayout.Widget{
				{ID: "widget_1", Key: homelayout.WidgetAttention, W: 12, H: 3},
				{ID: "widget_2", Key: homelayout.WidgetAnnouncement, W: 6, H: 2,
					Config: homelayout.WidgetConfig{Text: "Safety week"}},
			}},
			Density:      homelayout.DensityComfortable,
			Source:       homelayoutservice.SourceRolePreset,
			CanCustomize: true,
			Version:      3,
		},
		catalog: &homelayoutservice.WidgetCatalog{
			Widgets: []homelayout.WidgetDefinition{
				widgetDefinition(t, homelayout.WidgetAttention),
				widgetDefinition(t, homelayout.WidgetAnnouncement),
				widgetDefinition(t, homelayout.WidgetKPI),
				widgetDefinition(t, homelayout.WidgetReport),
			},
			Metrics: []homelayout.MetricDefinition{{Key: "onTimePercent", Label: "On-Time %"}},
		},
	}

	return layouts, homeEditor{layouts: layouts, targets: &fakeHomeTargets{}}
}

// homeParams is a call as the runtime hands it over: the person's own, with the
// owner the runtime recorded.
func homeParams(args map[string]any) serviceports.ToolExecuteParams {
	params := executeParams(args)
	params.Actor.PrincipalType = serviceports.PrincipalTypeUser
	params.Params[serviceports.SelfScopeOwnerParam] = params.Actor.UserID.String()
	return params
}

/*
A widget goes onto the person's own home page, in the place they asked for,
at the widget's own size, and the page is saved as theirs at the version it
was read at.
*/
func TestAddHomeWidget_PutsItWhereAsked(t *testing.T) {
	t.Parallel()

	layouts, editor := homeFixture(t)
	tool := &addHomeWidgetTool{homeToolBase{editor: editor}}

	params := homeParams(map[string]any{
		"version":  float64(3),
		"key":      homelayout.WidgetKPI,
		"position": float64(1),
		"config":   map[string]any{"metric": "onTimePercent"},
	})
	require.NoError(t, tool.Validate(t.Context(), params))
	require.NoError(t, tool.Execute(t.Context(), params))

	require.NotNil(t, layouts.updated)
	assert.Equal(t, int64(3), layouts.updated.Version)
	assert.Equal(t, homelayout.ModeCustom, layouts.updated.Document.Mode)
	widgets := layouts.updated.Document.Layout.Widgets
	require.Len(t, widgets, 3)
	assert.Equal(t, homelayout.WidgetKPI, widgets[1].Key)
	assert.Equal(t, "onTimePercent", widgets[1].Config.Metric)
	assert.Equal(t, widgetDefinition(t, homelayout.WidgetKPI).DefaultW, widgets[1].W)
	assert.NotEqual(t, "widget_1", widgets[1].ID)
}

func TestAddHomeWidget_RefusesWhatThePersonCannotHave(t *testing.T) {
	t.Parallel()

	_, editor := homeFixture(t)
	tool := &addHomeWidgetTool{homeToolBase{editor: editor}}
	errorsFor := func(args map[string]any) map[string]string {
		args["version"] = float64(3)
		return fieldErrors(t, tool.Validate(t.Context(), homeParams(args)))
	}

	assert.Contains(t, errorsFor(map[string]any{"key": "made-up"})["key"], "list_home_widgets")
	assert.Contains(t,
		errorsFor(map[string]any{
			"key":    homelayout.WidgetKPI,
			"config": map[string]any{"metric": "revenueToday"},
		})["config.metric"],
		"list_home_widgets", "a metric outside what the person may see")
	assert.Contains(t,
		errorsFor(map[string]any{
			"key":    homelayout.WidgetReport,
			"config": map[string]any{"definitionId": "on_time_percentage"},
		})["config.definitionId"],
		"list_reports", "the id the Homepage Widget Builder invented")
	assert.Contains(t,
		errorsFor(map[string]any{
			"key":    homelayout.WidgetReport,
			"config": map[string]any{"definitionId": pulid.MustNew("rdef_").String()},
		})["config.definitionId"],
		"list_reports", "a report that does not exist")
}

// The page moved on since it was read: somebody changed it in the editor
// between the proposal and its approval. The edit is refused rather than
// overwriting theirs.
func TestHomeEdit_RefusesAStaleVersion(t *testing.T) {
	t.Parallel()

	_, editor := homeFixture(t)
	tool := &removeHomeWidgetTool{homeToolBase{editor: editor}}

	err := tool.Validate(t.Context(), homeParams(map[string]any{
		"version":  float64(2),
		"widgetId": "widget_1",
	}))

	assert.Contains(t, fieldErrors(t, err)["version"], "get_my_home_layout")
}

func TestHomeEdit_RefusesALockedHomePage(t *testing.T) {
	t.Parallel()

	layouts, editor := homeFixture(t)
	layouts.effective.Locked = true
	tool := &removeHomeWidgetTool{homeToolBase{editor: editor}}

	err := tool.Validate(t.Context(), homeParams(map[string]any{
		"version":  float64(3),
		"widgetId": "widget_1",
	}))

	assert.Contains(t, fieldErrors(t, err)["layout"], "administrator")
}

/*
The change runs only for the person it was proposed for.

Approved from somebody else's queue, it would otherwise run as the approver and
rearrange their home page instead.
*/
func TestHomeEdit_RunsOnlyForThePersonItWasProposedFor(t *testing.T) {
	t.Parallel()

	layouts, editor := homeFixture(t)
	tool := &removeHomeWidgetTool{homeToolBase{editor: editor}}

	params := homeParams(map[string]any{"version": float64(3), "widgetId": "widget_1"})
	params.Params[serviceports.SelfScopeOwnerParam] = pulid.MustNew("usr_").String()
	require.Error(t, tool.Execute(t.Context(), params))

	delete(params.Params, serviceports.SelfScopeOwnerParam)
	require.Error(t, tool.Execute(t.Context(), params))
	assert.Nil(t, layouts.updated)
}

func TestRemoveHomeWidget(t *testing.T) {
	t.Parallel()

	layouts, editor := homeFixture(t)
	tool := &removeHomeWidgetTool{homeToolBase{editor: editor}}

	assert.Contains(t,
		fieldErrors(t, tool.Validate(t.Context(), homeParams(map[string]any{
			"version": float64(3), "widgetId": "widget_9",
		})))["widgetId"],
		"get_my_home_layout")

	require.NoError(t, tool.Execute(t.Context(), homeParams(map[string]any{
		"version": float64(3), "widgetId": "widget_1",
	})))
	require.Len(t, layouts.updated.Document.Layout.Widgets, 1)
	assert.Equal(t, "widget_2", layouts.updated.Document.Layout.Widgets[0].ID)
}

func TestArrangeHomeLayout(t *testing.T) {
	t.Parallel()

	layouts, editor := homeFixture(t)
	tool := &arrangeHomeLayoutTool{homeToolBase{editor: editor}}

	errs := fieldErrors(t, tool.Validate(t.Context(), homeParams(map[string]any{
		"version": float64(3), "widgetIds": []any{"widget_2", "widget_2"},
	})))
	assert.Contains(t, errs, "widgetIds[1]", "a widget listed twice")
	assert.Contains(t, errs["widgetIds"], "widget_1", "a widget left out")

	require.NoError(t, tool.Execute(t.Context(), homeParams(map[string]any{
		"version": float64(3), "widgetIds": []any{"widget_2", "widget_1"},
	})))
	widgets := layouts.updated.Document.Layout.Widgets
	assert.Equal(t, []string{"widget_2", "widget_1"}, []string{widgets[0].ID, widgets[1].ID})
}

func TestHomeEdit_SaysWhatThePageWouldShow(t *testing.T) {
	t.Parallel()

	_, editor := homeFixture(t)
	tool := &removeHomeWidgetTool{homeToolBase{editor: editor}}

	simulation, err := tool.Simulate(t.Context(), homeParams(map[string]any{
		"version": float64(3), "widgetId": "widget_1",
	}))
	require.NoError(t, err)

	assert.True(t, simulation.Previewed)
	assert.Contains(t, simulation.Summary, widgetDefinition(t, homelayout.WidgetAnnouncement).Label)
	assert.NotContains(t, simulation.Summary, widgetDefinition(t, homelayout.WidgetAttention).Label)
}
