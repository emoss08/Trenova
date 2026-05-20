package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AllocateEDIControlNumbersRequest struct {
	TenantInfo     pagination.TenantInfo   `json:"tenantInfo"`
	PartnerID      pulid.ID                `json:"partnerId"`
	DocumentTypeID pulid.ID                `json:"documentTypeId"`
	Kinds          []edi.ControlNumberKind `json:"kinds"`
}

type EDIControlNumberRepository interface {
	AllocateControlNumbers(
		ctx context.Context,
		req AllocateEDIControlNumbersRequest,
	) (map[edi.ControlNumberKind]int64, error)
}
