package location

import "github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"

func (l *Location) ApplyTextFormat(format dataentrycontrol.TextFormat) {
	l.Code = format.Code(l.Code)
	l.Name = format.Name(l.Name)
	l.City = format.City(l.City)
}
