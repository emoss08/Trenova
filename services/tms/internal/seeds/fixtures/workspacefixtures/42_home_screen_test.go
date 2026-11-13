package workspacefixtures

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHomeScreenDocument_IsAValidCustomLayout(t *testing.T) {
	t.Parallel()

	document := HomeScreenDocument()

	multiErr := errortypes.NewMultiError()
	document.Validate(multiErr)
	require.False(t, multiErr.HasErrors(), "validation errors: %v", multiErr)
	assert.Equal(t, homelayout.ModeCustom, document.Mode)
}

func TestHomeScreenDocument_SurvivesNormalize(t *testing.T) {
	t.Parallel()

	document := HomeScreenDocument()
	normalized := document.Normalize()

	require.Len(t, normalized.Layout.Widgets, len(document.Layout.Widgets))
	for i, widget := range document.Layout.Widgets {
		assert.Equal(t, widget, normalized.Layout.Widgets[i])
	}
}

func TestHomeScreenDocument_FillsTheCanvasWithDistinctWidgets(t *testing.T) {
	t.Parallel()

	widgets := HomeScreenDocument().Layout.Widgets
	require.Len(t, widgets, homelayout.MaxWidgets)

	keys := make(map[string]struct{}, len(widgets))
	ids := make(map[string]struct{}, len(widgets))
	for _, widget := range widgets {
		_, known := homelayout.WidgetDefinitionFor(widget.Key)
		assert.True(t, known, "%s is not in the catalog", widget.Key)

		_, repeated := keys[widget.Key]
		assert.False(t, repeated, "%s is on the canvas twice", widget.Key)
		keys[widget.Key] = struct{}{}

		_, duplicate := ids[widget.ID]
		assert.False(t, duplicate, "widget id %s is used twice", widget.ID)
		ids[widget.ID] = struct{}{}
	}
}

func TestHomeScreenDocument_ReportsOnACannedReportThatExists(t *testing.T) {
	t.Parallel()

	registry, err := canned.Default()
	require.NoError(t, err)

	for _, widget := range HomeScreenDocument().Layout.Widgets {
		if widget.Config.CannedKey == "" {
			continue
		}
		_, ok := registry.Get(widget.Config.CannedKey)
		assert.True(t, ok, "unknown canned report %q", widget.Config.CannedKey)
	}
}

func TestHomeFavorites_PointInsideTheApplicationOnce(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{})
	for _, favorite := range homeFavorites() {
		assert.Regexp(t, `^/\S+$`, favorite.url)
		assert.NotEmpty(t, favorite.title)

		_, repeated := seen[favorite.url]
		assert.False(t, repeated, "%s is pinned twice", favorite.url)
		seen[favorite.url] = struct{}{}
	}
}
