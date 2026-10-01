package apikeyservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/apikey"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type keyChange struct {
	tenantInfo pagination.TenantInfo
	actorID    pulid.ID
	key        *apikey.Key
	operation  permission.Operation
	before     any
	after      any
	comment    string
}

func (s *Service) recordChange(ctx context.Context, change *keyChange) {
	if s.auditor == nil {
		return
	}

	actor := change.tenantInfo
	if change.actorID.IsNotNil() {
		actor.UserID = change.actorID
	}

	s.auditor.RecordChange(ctx, &services.SecurityChange{
		Resource:       permission.ResourceAPIKey,
		ResourceID:     change.key.ID.String(),
		Operation:      change.operation,
		Actor:          services.UserActor(actor).AuditActor(),
		OrganizationID: change.key.OrganizationID,
		BusinessUnitID: change.key.BusinessUnitID,
		Before:         change.before,
		After:          change.after,
		Comment:        change.comment,
		Metadata:       map[string]any{"keyPrefix": change.key.KeyPrefix},
	})
}
