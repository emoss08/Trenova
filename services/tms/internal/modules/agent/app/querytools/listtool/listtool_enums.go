package listtool

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
)

var (
	SortDirections = agenttoolschema.Source("list.sortDirection", []string{"asc", "desc"})
)
