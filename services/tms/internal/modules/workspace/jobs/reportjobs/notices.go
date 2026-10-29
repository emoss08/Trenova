package reportjobs

import (
	"github.com/emoss08/trenova/internal/modules/workspace/jobs/reportnotify"
)

func (a *Activities) notices() *reportnotify.Renderer {
	return reportnotify.NewRenderer(a.templates, a.l)
}
