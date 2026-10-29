package listtool

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

func SpecOf(tool agenttoolports.AgentQueryTool) Spec {
	return tool.(*Tool).Spec //nolint:errcheck,forcetypeassert // constructed above
}
