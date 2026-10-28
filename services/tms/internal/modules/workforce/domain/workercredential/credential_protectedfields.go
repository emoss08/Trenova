package workercredential

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

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
