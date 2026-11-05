package tenantports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type RevokeUserAccessRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	OrganizationID pulid.ID
}

type UserAccessRevoker interface {
	RevokeUserAccess(ctx context.Context, req RevokeUserAccessRequest) error
}
