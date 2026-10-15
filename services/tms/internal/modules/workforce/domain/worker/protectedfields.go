package worker

import "github.com/emoss08/trenova/internal/modules/identity/domain/permission"

func (w *Worker) WithholdFields(visible permission.FieldGroupSet) {
	if w == nil {
		return
	}
	w.Profile.WithholdFields(visible)
}

func (w *Worker) KeepFields(stored *Worker, visible permission.FieldGroupSet) {
	if w == nil || stored == nil {
		return
	}
	w.Profile.KeepFields(stored.Profile, visible)
}

func (wp *WorkerProfile) WithholdFields(visible permission.FieldGroupSet) {
	if wp == nil || visible.Has(permission.FieldGroupPersonal) {
		return
	}
	wp.DOB = 0
	wp.LicenseNumber = ""
	wp.TWICCardNumber = ""
	wp.LastDrugTest = 0
}

func (wp *WorkerProfile) KeepFields(stored *WorkerProfile, visible permission.FieldGroupSet) {
	if wp == nil || stored == nil || visible.Has(permission.FieldGroupPersonal) {
		return
	}
	wp.DOB = stored.DOB
	wp.LicenseNumber = stored.LicenseNumber
	wp.TWICCardNumber = stored.TWICCardNumber
	wp.LastDrugTest = stored.LastDrugTest
}

func (c *WorkerCredential) WithholdFields(visible permission.FieldGroupSet) {
	if c == nil || visible.Has(permission.FieldGroupPersonal) {
		return
	}
	c.Number = ""
}

func (c *WorkerCredential) KeepFields(stored *WorkerCredential, visible permission.FieldGroupSet) {
	if c == nil || stored == nil || visible.Has(permission.FieldGroupPersonal) {
		return
	}
	c.Number = stored.Number
}
