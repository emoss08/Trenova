package repositories

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ThreadOwnersRequest struct {
	TenantInfo pagination.TenantInfo
	ThreadIDs  []pulid.ID
}

type ThreadOwnerRepository interface {
	ThreadOwners(ctx context.Context, req ThreadOwnersRequest) (map[pulid.ID]pulid.ID, error)
}
