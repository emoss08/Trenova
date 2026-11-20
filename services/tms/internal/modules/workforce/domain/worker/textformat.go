package worker

import "github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"

func (w *Worker) ApplyTextFormat(format dataentrycontrol.TextFormat) {
	w.FirstName = format.Name(w.FirstName)
	w.LastName = format.Name(w.LastName)
	w.EmergencyContactName = format.Name(w.EmergencyContactName)
	w.City = format.City(w.City)
	w.Email = format.Email(w.Email)
}
