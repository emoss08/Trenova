package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// PortalAccessRevoker shuts off a worker's driver-portal sign-in. Employment
// termination calls it so somebody who has left the company cannot keep using
// Dash, and it reports whether there was any access to shut off so the office
// is never told access was revoked for a worker who never had any.
type PortalAccessRevoker interface {
	RevokeOnTermination(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		actor *RequestActor,
	) (bool, error)
}
