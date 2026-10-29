package querytool

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
)

func ClockFor(params *agenttoolports.QueryToolParams) filtercatalog.Clock {
	return filtercatalog.NewClock(params.Timezone)
}
