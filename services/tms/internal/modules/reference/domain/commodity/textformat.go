package commodity

import "github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"

func (c *Commodity) ApplyTextFormat(format dataentrycontrol.TextFormat) {
	c.Name = format.Name(c.Name)
}
