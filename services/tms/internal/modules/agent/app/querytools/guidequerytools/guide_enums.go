package guidequerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
)

const (
	paramEntity = "entity"
)

var (
	guidePageActions = agenttoolschema.Source("guide.pageAction", []string{createAction})
)
