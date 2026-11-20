package glaccount

import "github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"

func (g *GLAccount) ApplyTextFormat(format dataentrycontrol.TextFormat) {
	g.AccountCode = format.Code(g.AccountCode)
	g.Name = format.Name(g.Name)
}
