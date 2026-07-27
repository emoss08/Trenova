package homelayout_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/homelayout"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func customDocument(widgets ...homelayout.Widget) *homelayout.Document {
	return &homelayout.Document{
		SchemaVersion: homelayout.DocumentSchemaVersion,
		Mode:          homelayout.ModeCustom,
		Density:       homelayout.DensityComfortable,
		Layout:        &homelayout.Layout{Widgets: widgets},
	}
}

func validateDocument(t *testing.T, doc *homelayout.Document) *errortypes.MultiError {
	t.Helper()
	multiErr := errortypes.NewMultiError()
	doc.Validate(multiErr)
	return multiErr
}

func TestDefaultDocumentIsValid(t *testing.T) {
	require.False(t, validateDocument(t, homelayout.DefaultDocument()).HasErrors())
}

func TestValidateRejectsWrongSchemaVersion(t *testing.T) {
	doc := homelayout.DefaultDocument()
	doc.SchemaVersion = homelayout.DocumentSchemaVersion + 1

	require.True(t, validateDocument(t, doc).HasErrors())
}

func TestValidateRejectsUnknownModeAndDensity(t *testing.T) {
	doc := homelayout.DefaultDocument()
	doc.Mode = "sideways"
	doc.Density = "roomy"

	require.True(t, validateDocument(t, doc).HasErrors())
}

func TestValidateRequiresLayoutWhenCustomized(t *testing.T) {
	doc := homelayout.DefaultDocument()
	doc.Mode = homelayout.ModeCustom

	require.True(t, validateDocument(t, doc).HasErrors())
}

func TestValidateRejectsUnknownWidgetKey(t *testing.T) {
	doc := customDocument(homelayout.Widget{
		ID: "widget_1", Key: "not-a-widget", W: 4, H: 4,
	})

	require.True(t, validateDocument(t, doc).HasErrors())
}

func TestValidateRejectsDuplicateWidgetID(t *testing.T) {
	doc := customDocument(
		homelayout.Widget{ID: "widget_1", Key: homelayout.WidgetAttention, W: 4, H: 4},
		homelayout.Widget{ID: "widget_1", Key: homelayout.WidgetFavorites, W: 4, H: 4},
	)

	require.True(t, validateDocument(t, doc).HasErrors())
}

func TestValidateRejectsGeometryOutsideWidgetBounds(t *testing.T) {
	doc := customDocument(homelayout.Widget{
		ID: "widget_1", Key: homelayout.WidgetAttention, W: 1, H: 4,
	})

	require.True(t, validateDocument(t, doc).HasErrors())
}

func TestValidateRejectsTooManyWidgets(t *testing.T) {
	widgets := make([]homelayout.Widget, 0, homelayout.MaxWidgets+1)
	for i := 0; i <= homelayout.MaxWidgets; i++ {
		widgets = append(widgets, homelayout.Widget{
			ID:  homelayout.NextWidgetID(widgets),
			Key: homelayout.WidgetAttention,
			W:   4, H: 4,
		})
	}

	require.True(t, validateDocument(t, customDocument(widgets...)).HasErrors())
}

func TestValidateReportWidgetRequiresExactlyOneSource(t *testing.T) {
	definitionID := pulid.MustNew("rdf_")

	t.Run("neither", func(t *testing.T) {
		doc := customDocument(homelayout.Widget{
			ID: "widget_1", Key: homelayout.WidgetReport, W: 6, H: 5,
		})
		require.True(t, validateDocument(t, doc).HasErrors())
	})

	t.Run("both", func(t *testing.T) {
		doc := customDocument(homelayout.Widget{
			ID: "widget_1", Key: homelayout.WidgetReport, W: 6, H: 5,
			Config: homelayout.WidgetConfig{
				DefinitionID: definitionID,
				CannedKey:    "revenue-by-customer",
			},
		})
		require.True(t, validateDocument(t, doc).HasErrors())
	})

	t.Run("canned only", func(t *testing.T) {
		doc := customDocument(homelayout.Widget{
			ID: "widget_1", Key: homelayout.WidgetReport, W: 6, H: 5,
			Config: homelayout.WidgetConfig{CannedKey: "revenue-by-customer"},
		})
		require.False(t, validateDocument(t, doc).HasErrors())
	})
}

func TestValidateMetricWidgets(t *testing.T) {
	t.Run("unknown metric", func(t *testing.T) {
		doc := customDocument(homelayout.Widget{
			ID: "widget_1", Key: homelayout.WidgetKPI, W: 3, H: 2,
			Config: homelayout.WidgetConfig{Metric: "notAMetric"},
		})
		require.True(t, validateDocument(t, doc).HasErrors())
	})

	t.Run("duplicate metric in strip", func(t *testing.T) {
		doc := customDocument(homelayout.Widget{
			ID: "widget_1", Key: homelayout.WidgetKPIRow, W: 12, H: 2,
			Config: homelayout.WidgetConfig{
				Metrics: []string{"activeShipments", "activeShipments"},
			},
		})
		require.True(t, validateDocument(t, doc).HasErrors())
	})

	t.Run("too many metrics in strip", func(t *testing.T) {
		metrics := make([]string, 0, homelayout.MaxKPIRowMetrics+1)
		for _, metric := range homelayout.MetricCatalog() {
			if len(metrics) > homelayout.MaxKPIRowMetrics {
				break
			}
			metrics = append(metrics, metric.Key)
		}

		doc := customDocument(homelayout.Widget{
			ID: "widget_1", Key: homelayout.WidgetKPIRow, W: 12, H: 2,
			Config: homelayout.WidgetConfig{Metrics: metrics},
		})
		require.True(t, validateDocument(t, doc).HasErrors())
	})
}

func TestValidateAnnouncementLength(t *testing.T) {
	t.Run("blank", func(t *testing.T) {
		doc := customDocument(homelayout.Widget{
			ID: "widget_1", Key: homelayout.WidgetAnnouncement, W: 12, H: 2,
			Config: homelayout.WidgetConfig{Text: "   "},
		})
		require.True(t, validateDocument(t, doc).HasErrors())
	})

	t.Run("too long", func(t *testing.T) {
		doc := customDocument(homelayout.Widget{
			ID: "widget_1", Key: homelayout.WidgetAnnouncement, W: 12, H: 2,
			Config: homelayout.WidgetConfig{
				Text: strings.Repeat("a", homelayout.MaxAnnouncementLength+1),
			},
		})
		require.True(t, validateDocument(t, doc).HasErrors())
	})
}

// TestNormalizeDropsRetiredWidgets is the behaviour that makes retiring a
// widget a one-line catalog change: stored layouts referencing it stop
// mentioning it rather than failing to load.
func TestNormalizeDropsRetiredWidgets(t *testing.T) {
	doc := customDocument(
		homelayout.Widget{ID: "widget_1", Key: "retired-widget", W: 4, H: 4},
		homelayout.Widget{ID: "widget_2", Key: homelayout.WidgetAttention, W: 4, H: 4},
	)

	normalized := doc.Normalize()

	require.Len(t, normalized.Layout.Widgets, 1)
	require.Equal(t, homelayout.WidgetAttention, normalized.Layout.Widgets[0].Key)
}

func TestNormalizeDropsDuplicateAndUnidentifiedWidgets(t *testing.T) {
	doc := customDocument(
		homelayout.Widget{ID: "", Key: homelayout.WidgetAttention, W: 4, H: 4},
		homelayout.Widget{ID: "widget_1", Key: homelayout.WidgetAttention, W: 4, H: 4},
		homelayout.Widget{ID: "widget_1", Key: homelayout.WidgetFavorites, W: 4, H: 4},
	)

	normalized := doc.Normalize()

	require.Len(t, normalized.Layout.Widgets, 1)
	require.Equal(t, "widget_1", normalized.Layout.Widgets[0].ID)
}

func TestNormalizeClampsGeometryToWidgetBounds(t *testing.T) {
	doc := customDocument(
		homelayout.Widget{ID: "widget_1", Key: homelayout.WidgetKPI, W: 99, H: 0},
	)

	widget := doc.Normalize().Layout.Widgets[0]
	definition, ok := homelayout.WidgetDefinitionFor(homelayout.WidgetKPI)

	require.True(t, ok)
	require.Equal(t, definition.MaxW, widget.W)
	require.Equal(t, definition.DefaultH, widget.H)
}

// TestNormalizeStripsConfigTheWidgetCannotUse keeps a stored layout from
// carrying settings that can never take effect — for example a metric left
// behind after a widget was switched to an announcement.
func TestNormalizeStripsConfigTheWidgetCannotUse(t *testing.T) {
	doc := customDocument(homelayout.Widget{
		ID: "widget_1", Key: homelayout.WidgetAnnouncement, W: 12, H: 2,
		Config: homelayout.WidgetConfig{
			Text:       "Safety push starts Monday",
			Metric:     "activeShipments",
			CannedKey:  "revenue-by-customer",
			WindowDays: 30,
		},
	})

	config := doc.Normalize().Layout.Widgets[0].Config

	require.Equal(t, "Safety push starts Monday", config.Text)
	require.Empty(t, config.Metric)
	require.Empty(t, config.CannedKey)
	require.Zero(t, config.WindowDays)
}

func TestNormalizeDropsUnknownMetricsFromStrip(t *testing.T) {
	doc := customDocument(homelayout.Widget{
		ID: "widget_1", Key: homelayout.WidgetKPIRow, W: 12, H: 2,
		Config: homelayout.WidgetConfig{
			Metrics: []string{"activeShipments", "retiredMetric", "activeShipments"},
		},
	})

	require.Equal(
		t,
		[]string{"activeShipments"},
		doc.Normalize().Layout.Widgets[0].Config.Metrics,
	)
}

// TestNormalizeReportConfigPrefersDefinition resolves a config carrying both
// sources the same way the validator describes, rather than falling open on the
// shared canned library.
func TestNormalizeReportConfigPrefersDefinition(t *testing.T) {
	definitionID := pulid.MustNew("rdf_")
	doc := customDocument(homelayout.Widget{
		ID: "widget_1", Key: homelayout.WidgetReport, W: 6, H: 5,
		Config: homelayout.WidgetConfig{
			DefinitionID: definitionID,
			CannedKey:    "revenue-by-customer",
		},
	})

	config := doc.Normalize().Layout.Widgets[0].Config

	require.Equal(t, definitionID, config.DefinitionID)
	require.Empty(t, config.CannedKey)
}

func TestNormalizeFallsBackToPresetModeAndComfortableDensity(t *testing.T) {
	doc := &homelayout.Document{
		SchemaVersion: 99,
		Mode:          "sideways",
		Density:       "roomy",
	}

	normalized := doc.Normalize()

	require.Equal(t, homelayout.DocumentSchemaVersion, normalized.SchemaVersion)
	require.Equal(t, homelayout.ModePreset, normalized.Mode)
	require.Equal(t, homelayout.DensityComfortable, normalized.Density)
	require.Nil(t, normalized.Layout, "a preset-mode document carries no layout of its own")
}

func TestNextWidgetIDSkipsTakenNames(t *testing.T) {
	widgets := []homelayout.Widget{
		{ID: "widget_1"},
		{ID: "widget_2"},
		{ID: "widget_3"},
	}

	require.Equal(t, "widget_4", homelayout.NextWidgetID(widgets))
	require.Equal(t, "widget_1", homelayout.NextWidgetID(nil))
}

func TestLayoutCloneDoesNotShareMetricSlices(t *testing.T) {
	layout := &homelayout.Layout{Widgets: []homelayout.Widget{{
		ID: "widget_1", Key: homelayout.WidgetKPIRow, W: 12, H: 2,
		Config: homelayout.WidgetConfig{Metrics: []string{"activeShipments"}},
	}}}

	clone := layout.Clone()
	clone.Widgets[0].Config.Metrics[0] = "revenueToday"

	require.Equal(t, "activeShipments", layout.Widgets[0].Config.Metrics[0])
}
