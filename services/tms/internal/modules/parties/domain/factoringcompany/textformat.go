package factoringcompany

import "github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"

func (f *FactoringCompany) ApplyTextFormat(format dataentrycontrol.TextFormat) {
	f.Code = format.Code(f.Code)
	f.Name = format.Name(f.Name)
	f.City = format.City(f.City)
	f.Email = format.Email(f.Email)
}
