package iamservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	metadataMappingID   = "mappingId"
	metadataTokenID     = "tokenId"
	metadataTokenPrefix = "tokenPrefix"
)

type iamChange struct {
	tenantInfo pagination.TenantInfo
	resource   permission.Resource
	resourceID pulid.ID
	operation  permission.Operation
	before     any
	after      any
	comment    string
	metadata   map[string]any
}

func (s *service) recordChange(ctx context.Context, change *iamChange) {
	if s.auditor == nil {
		return
	}

	var actor services.AuditActor
	if change.tenantInfo.UserID.IsNotNil() {
		actor = services.UserActor(change.tenantInfo).AuditActor()
	}

	s.auditor.RecordChange(ctx, &services.SecurityChange{
		Resource:       change.resource,
		ResourceID:     change.resourceID.String(),
		Operation:      change.operation,
		Actor:          actor,
		OrganizationID: change.tenantInfo.OrgID,
		BusinessUnitID: change.tenantInfo.BuID,
		Before:         change.before,
		After:          change.after,
		Comment:        change.comment,
		Metadata:       change.metadata,
	})
}
