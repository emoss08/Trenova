package homelayout_test

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/homelayout"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func responsibilities() []permission.CoreResponsibility {
	return []permission.CoreResponsibility{
		permission.CoreResponsibilityOperations,
		permission.CoreResponsibilityBilling,
		permission.CoreResponsibilityFinance,
		permission.CoreResponsibilityLeadership,
		"",
	}
}

// TestBuiltInLayoutsAreValid is the compile gate for the shipped home screens:
// a widget removed from the catalog, or a geometry outside its definition's
// bounds, breaks the build rather than a customer's first login.
func TestBuiltInLayoutsAreValid(t *testing.T) {
	for _, responsibility := range responsibilities() {
		t.Run(fmt.Sprintf("responsibility=%q", responsibility), func(t *testing.T) {
			layout := homelayout.BuiltInLayout(responsibility)
			require.NotNil(t, layout)
			require.NotEmpty(t, layout.Widgets, "a shipped layout must not be empty")

			multiErr := errortypes.NewMultiError()
			layout.Validate(multiErr, "layout")
			require.False(t, multiErr.HasErrors(), "validation errors: %v", multiErr)

			require.NotEmpty(t, homelayout.BuiltInLayoutName(responsibility))
		})
	}
}

// TestBuiltInLayoutsSurviveNormalize proves the shipped layouts are already in
// normal form. If Normalize changes one, the definition and the catalog have
// drifted apart.
func TestBuiltInLayoutsSurviveNormalize(t *testing.T) {
	for _, responsibility := range responsibilities() {
		t.Run(fmt.Sprintf("responsibility=%q", responsibility), func(t *testing.T) {
			layout := homelayout.BuiltInLayout(responsibility)
			normalized := layout.Normalize()

			require.Len(t, normalized.Widgets, len(layout.Widgets),
				"Normalize dropped a widget from a shipped layout")
			for i := range layout.Widgets {
				require.Equal(t, layout.Widgets[i].Key, normalized.Widgets[i].Key)
				require.Equal(t, layout.Widgets[i].W, normalized.Widgets[i].W)
				require.Equal(t, layout.Widgets[i].H, normalized.Widgets[i].H)
				require.Equal(t, layout.Widgets[i].Config, normalized.Widgets[i].Config)
			}
		})
	}
}

func TestBuiltInLayoutWidgetIDsAreUnique(t *testing.T) {
	for _, responsibility := range responsibilities() {
		t.Run(fmt.Sprintf("responsibility=%q", responsibility), func(t *testing.T) {
			seen := make(map[string]struct{})
			for _, widget := range homelayout.BuiltInLayout(responsibility).Widgets {
				require.NotEmpty(t, widget.ID)
				_, dup := seen[widget.ID]
				require.False(t, dup, "duplicate widget ID %q", widget.ID)
				seen[widget.ID] = struct{}{}
			}
		})
	}
}

func TestWidgetCatalogIsWellFormed(t *testing.T) {
	categories := make(map[string]struct{})
	for _, category := range homelayout.CategoryCatalog() {
		categories[category.Key] = struct{}{}
	}

	seen := make(map[string]struct{})
	for _, widget := range homelayout.WidgetCatalog() {
		t.Run(widget.Key, func(t *testing.T) {
			require.NotEmpty(t, widget.Label)
			require.NotEmpty(t, widget.Description)

			_, dup := seen[widget.Key]
			require.False(t, dup, "duplicate widget key")
			seen[widget.Key] = struct{}{}

			_, ok := categories[widget.Category]
			require.True(t, ok, "widget category %q is not in the catalog", widget.Category)

			require.GreaterOrEqual(t, widget.MinW, 1)
			require.LessOrEqual(t, widget.MaxW, homelayout.GridColumns)
			require.LessOrEqual(t, widget.MinW, widget.DefaultW)
			require.LessOrEqual(t, widget.DefaultW, widget.MaxW)

			require.GreaterOrEqual(t, widget.MinH, 1)
			require.LessOrEqual(t, widget.MaxH, homelayout.MaxWidgetHeight)
			require.LessOrEqual(t, widget.MinH, widget.DefaultH)
			require.LessOrEqual(t, widget.DefaultH, widget.MaxH)

			// A widget that carries an operation must carry the resource it
			// applies to, or the gate silently passes.
			if widget.Operation != "" {
				require.NotEmpty(t, string(widget.Resource),
					"widget declares an operation with no resource")
			}
		})
	}
}

func TestMetricCatalogIsWellFormed(t *testing.T) {
	seen := make(map[string]struct{})
	for _, metric := range homelayout.MetricCatalog() {
		t.Run(metric.Key, func(t *testing.T) {
			require.NotEmpty(t, metric.Label)
			require.NotEmpty(t, string(metric.Resource))
			require.NotEmpty(t, string(metric.Operation))

			_, dup := seen[metric.Key]
			require.False(t, dup, "duplicate metric key")
			seen[metric.Key] = struct{}{}
		})
	}
}
