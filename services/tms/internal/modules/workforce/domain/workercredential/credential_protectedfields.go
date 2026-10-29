package workercredential

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
)

func (c *WorkerCredential) WithholdFields(visible protectedfield.GroupSet) {
	if c == nil || visible.Has(protectedfield.Personal) {
		return
	}
	c.Number = ""
}

func (c *WorkerCredential) KeepFields(stored *WorkerCredential, visible protectedfield.GroupSet) {
	if c == nil || stored == nil || visible.Has(protectedfield.Personal) {
		return
	}
	c.Number = stored.Number
}
