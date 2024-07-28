package audit

import (
	"github.com/google/uuid"
)

type AuditUser struct {
	ID       uuid.UUID
	Username string
}
