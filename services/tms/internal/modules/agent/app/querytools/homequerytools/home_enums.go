package homequerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
)

var (
	homeWidgetCategories = agenttoolschema.Source("homeLayout.category", []string{
		homelayout.CategoryWork,
		homelayout.CategoryPulse,
		homelayout.CategoryInsight,
		homelayout.CategoryOrientation,
		homelayout.CategoryComms,
	})
)
