package alerttools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHomeTools_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		build func(homeEditor) agentports.AgentTool
		args  map[string]any
	}{
		"add": {
			build: func(editor homeEditor) agentports.AgentTool {
				return &AddHomeWidgetTool{homeToolBase{editor: editor}}
			},
			args: map[string]any{
				"version":  float64(3),
				"key":      homelayout.WidgetKPI,
				"position": float64(0),
				"config":   map[string]any{"metric": "onTimePercent"},
			},
		},
		"remove": {
			build: func(editor homeEditor) agentports.AgentTool {
				return &RemoveHomeWidgetTool{homeToolBase{editor: editor}}
			},
			args: map[string]any{"version": float64(3), "widgetId": "widget_1"},
		},
		"arrange": {
			build: func(editor homeEditor) agentports.AgentTool {
				return &ArrangeHomeLayoutTool{homeToolBase{editor: editor}}
			},
			args: map[string]any{
				"version":   float64(3),
				"widgetIds": []any{"widget_2", "widget_1"},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			layouts, editor := homeFixture(t)
			tool := tc.build(editor)
			params := homeParams(tc.args)
			before := homePageOf(layouts.effective.Layout)

			preview := agenttoolstest.PreviewWithoutWrites(t, &layouts.guard, func() (*agent.ToolPreview, error) {
				return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
			})

			change := agenttoolstest.PreviewChange(t, preview, 0)
			assert.Equal(t, permission.ResourceHomeLayoutPreset, change.Resource)
			assert.Equal(t, homePageLabel, change.Label)
			require.NotNil(t, change.Version)
			assert.Equal(t, int64(3), *change.Version)
			assert.Contains(t, preview.Summary, "your own layout")

			require.NoError(t, tool.Execute(t.Context(), params))
			agenttoolstest.RequireUpdateParity(t, change, before, homePageOf(layouts.updated.Document.Layout),
				toolpreview.Labels(homePageLabels))
		})
	}
}

func TestRemoveHomeWidget_PreviewShowsWhatThePageKeeps(t *testing.T) {
	t.Parallel()

	layouts, editor := homeFixture(t)
	tool := &RemoveHomeWidgetTool{homeToolBase{editor: editor}}

	preview := agenttoolstest.PreviewWithoutWrites(t, &layouts.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), homeParams(map[string]any{
			"version": float64(3), "widgetId": "widget_1",
		}))
	})

	announcement := widgetDefinition(t, homelayout.WidgetAnnouncement).Label
	attention := widgetDefinition(t, homelayout.WidgetAttention).Label
	assert.Contains(t, preview.Summary, announcement)
	assert.NotContains(t, preview.Summary, attention)

	widgets := agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), "widgets")
	assert.Equal(t, "Widgets, in order", widgets.Label)
	assert.Contains(t, widgets.Before, attention)
	assert.NotContains(t, widgets.After, attention)
}

func TestHomeTools_PreviewWarnsOnAStaleVersion(t *testing.T) {
	t.Parallel()

	layouts, editor := homeFixture(t)
	tool := &RemoveHomeWidgetTool{homeToolBase{editor: editor}}

	preview := agenttoolstest.PreviewWithoutWrites(t, &layouts.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), homeParams(map[string]any{
			"version": float64(2), "widgetId": "widget_1",
		}))
	})

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
}
