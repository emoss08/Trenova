package tenantports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig"
	"github.com/emoss08/trenova/pkg/pagination"
)

type GetSequenceConfigRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type SequenceConfigRepository interface {
	GetByTenant(
		ctx context.Context,
		req GetSequenceConfigRequest,
	) (*sequenceconfig.SequenceConfigDocument, error)
	UpdateByTenant(
		ctx context.Context,
		doc *sequenceconfig.SequenceConfigDocument,
	) (*sequenceconfig.SequenceConfigDocument, error)
}
