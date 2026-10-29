package worker

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
)

func (w *Worker) WithholdFields(visible protectedfield.GroupSet) {
	if w == nil {
		return
	}
	w.Profile.WithholdFields(visible)
}

func (w *Worker) KeepFields(stored *Worker, visible protectedfield.GroupSet) {
	if w == nil || stored == nil {
		return
	}
	w.Profile.KeepFields(stored.Profile, visible)
}

func (wp *WorkerProfile) WithholdFields(visible protectedfield.GroupSet) {
	if wp == nil || visible.Has(protectedfield.Personal) {
		return
	}
	wp.DOB = 0
	wp.LicenseNumber = ""
	wp.TWICCardNumber = ""
	wp.LastDrugTest = 0
}

func (wp *WorkerProfile) KeepFields(stored *WorkerProfile, visible protectedfield.GroupSet) {
	if wp == nil || stored == nil || visible.Has(protectedfield.Personal) {
		return
	}
	wp.DOB = stored.DOB
	wp.LicenseNumber = stored.LicenseNumber
	wp.TWICCardNumber = stored.TWICCardNumber
	wp.LastDrugTest = stored.LastDrugTest
}
