package tenantports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetDocumentControlRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type DocumentControlRepository interface {
	Get(
		ctx context.Context,
		req GetDocumentControlRequest,
	) (*documentcontrol.DocumentControl, error)
	Create(
		ctx context.Context,
		entity *documentcontrol.DocumentControl,
	) (*documentcontrol.DocumentControl, error)
	Update(
		ctx context.Context,
		entity *documentcontrol.DocumentControl,
	) (*documentcontrol.DocumentControl, error)
	GetOrCreate(
		ctx context.Context,
		orgID, buID pulid.ID,
	) (*documentcontrol.DocumentControl, error)
}
