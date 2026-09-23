package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/homelayout"
	"github.com/emoss08/trenova/internal/core/services/homelayoutservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeHomeLayoutReader struct {
	effective *homelayoutservice.EffectiveLayout
	catalog   *homelayoutservice.WidgetCatalog
	asked     *homelayoutservice.Request
}

func (f *fakeHomeLayoutReader) GetEffective(
	_ context.Context,
	req *homelayoutservice.Request,
) (*homelayoutservice.EffectiveLayout, error) {
	f.asked = req
	return f.effective, nil
}

func (f *fakeHomeLayoutReader) GetCatalog(
	context.Context,
	*homelayoutservice.Request,
) (*homelayoutservice.WidgetCatalog, error) {
	return f.catalog, nil
}

/*
"What is on my dashboard" is answered from the person's home page.

The Homepage Widget Builder could only read the report dashboards under
Reports, which were not what the person meant, and had nothing to say about
the page they were looking at.
*/
func TestGetMyHomeLayout_ReadsThePersonsOwnHomePage(t *testing.T) {
	t.Parallel()

	kpi, ok := homelayout.WidgetDefinitionFor(homelayout.WidgetKPI)
	require.True(t, ok)
	reader := &fakeHomeLayoutReader{effective: &homelayoutservice.EffectiveLayout{
		Layout: &homelayout.Layout{Widgets: []homelayout.Widget{
			{ID: "widget_1", Key: homelayout.WidgetKPI, W: 3, H: 2,
				Config: homelayout.WidgetConfig{Metric: "onTimePercent"}},
		}},
		Source:       homelayoutservice.SourceRolePreset,
		PresetName:   "Dispatch",
		CanCustomize: true,
		Version:      4,
	}}
	tool := &getMyHomeLayoutTool{layouts: reader}
	params := testParams(map[string]any{})

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	view := result.(homeLayoutView)
	assert.Equal(
		t,
		params.Actor.UserID,
		reader.asked.Principal.UserID,
		"the page resolves for the reader",
	)
	assert.Equal(t, "rolePreset", view.Source)
	assert.Equal(t, int64(4), view.Version)
	require.Len(t, view.Widgets, 1)
	assert.Equal(t, kpi.Label, view.Widgets[0].Label)
	require.NotNil(t, view.Widgets[0].Config)
	assert.Equal(t, "onTimePercent", view.Widgets[0].Config.Metric)
	assert.Contains(t, view.Note, "add_home_widget")
}

func TestGetMyHomeLayout_SaysWhenThePageCannotBeChanged(t *testing.T) {
	t.Parallel()

	tool := &getMyHomeLayoutTool{layouts: &fakeHomeLayoutReader{
		effective: &homelayoutservice.EffectiveLayout{Locked: true},
	}}

	result, err := tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	view := result.(homeLayoutView)
	assert.Empty(t, view.Widgets)
	assert.Contains(t, view.Note, "administrator")
}

func TestListHomeWidgets_NamesWhatEachNeeds(t *testing.T) {
	t.Parallel()

	kpi, _ := homelayout.WidgetDefinitionFor(homelayout.WidgetKPI)
	attention, _ := homelayout.WidgetDefinitionFor(homelayout.WidgetAttention)
	tool := &listHomeWidgetsTool{
		layouts: &fakeHomeLayoutReader{catalog: &homelayoutservice.WidgetCatalog{
			Widgets:    []homelayout.WidgetDefinition{kpi, attention},
			Metrics:    []homelayout.MetricDefinition{{Key: "onTimePercent", Label: "On-Time %"}},
			MaxWidgets: homelayout.MaxWidgets,
		}},
	}

	result, err := tool.Query(t.Context(), testParams(map[string]any{"category": kpi.Category}))
	require.NoError(t, err)

	view := result.(homeWidgetCatalogView)
	require.Len(t, view.Widgets, 1)
	assert.Equal(t, homelayout.WidgetKPI, view.Widgets[0].Key)
	assert.Equal(t, string(homelayout.ConfigKindMetric), view.Widgets[0].Needs)
	assert.Equal(t, []homeMetricOption{{Key: "onTimePercent", Label: "On-Time %"}}, view.Metrics)
}
