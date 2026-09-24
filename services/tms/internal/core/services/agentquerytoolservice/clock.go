package agentquerytoolservice

import (
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/filtercatalog"
)

// clock is the frame a query reads dates in. It lives in filtercatalog, which
// owns the filter vocabulary the tools and the table composer share; this name
// stays so a tool reads the way it always did.
type clock = filtercatalog.Clock

const secondsPerDay = filtercatalog.SecondsPerDay

func clockFor(params *serviceports.QueryToolParams) clock {
	return filtercatalog.NewClock(params.Timezone)
}
