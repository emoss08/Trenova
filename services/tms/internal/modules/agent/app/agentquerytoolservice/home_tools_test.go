package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/homelayoutservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/toolschema"
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
the page they were looking at. The page is read through homeLayout, as the
person's own, with each widget named as the page names it.
*/
func TestGetMyHomeLayout_ReadsThePersonsOwnHomePage(t *testing.T) {
	t.Parallel()

	kpi, ok := homelayout.WidgetDefinitionFor(homelayout.WidgetKPI)
	require.True(t, ok)
	reader := &keyedReader{answers: map[string]*graphqlreadports.GraphQLReadResult{
		"HomeLayout": {Data: map[string]any{"homeLayout": map[string]any{
			"source": "rolePreset", "presetName": "Dispatch", "canCustomize": true,
			"version": 4,
			"widgets": []any{map[string]any{
				"id": "widget_1", "key": homelayout.WidgetKPI, "w": 3, "h": 2,
				"config": map[string]any{"metric": "onTimePercent"},
			}},
		}}},
	}}
	spec := homeLayoutGetSpec()
	tool, err := newGraphQLGetTool(&spec, reader, &mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)
	assert.Equal(t, agent.ToolScopeSelf, tool.Policy().Scope, "it reads only the caller's page")
	assert.Empty(t, tool.ParamSchema()[toolschema.KeyRequired], "the page takes no id")

	result, err := tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	page := encodedDocument(t, result)
	widgets, _ := page["widgets"].([]any)
	require.Len(t, widgets, 1)
	widget, _ := widgets[0].(map[string]any)
	assert.Equal(t, kpi.Label, widget["label"])
	assert.Equal(t, map[string]any{"metric": "onTimePercent"}, widget["config"])
	assert.Contains(t, page["note"], "home_add_widget")
	assert.Empty(t, reader.sent("HomeLayout").Variables)
}

func TestGetMyHomeLayout_SaysWhenThePageCannotBeChanged(t *testing.T) {
	t.Parallel()

	record := map[string]any{"locked": true, "canCustomize": false}
	labelHomeLayout(record, nil, clock{})

	assert.Contains(t, record["note"], "administrator")
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
