package insightquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
)

var (
	insightCategories = agenttoolschema.Source("insight.category", insightCategoryNames())
	insightSeverities = agenttoolschema.Source("insight.severity", insightSeverityNames())
	insightStatuses   = agenttoolschema.Source("insight.status", insightStatusNames())
)
