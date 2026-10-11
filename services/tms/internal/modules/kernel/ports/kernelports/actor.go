package kernelports

import (
	"github.com/emoss08/trenova/shared/pulid"
)

type AuditActor struct {
	PrincipalType PrincipalType
	PrincipalID   pulid.ID
	UserID        pulid.ID
	APIKeyID      pulid.ID
}
