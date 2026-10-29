package edipartnerports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AllocateEDIControlNumbersRequest struct {
	TenantInfo     pagination.TenantInfo           `json:"tenantInfo"`
	PartnerID      pulid.ID                        `json:"partnerId"`
	DocumentTypeID pulid.ID                        `json:"documentTypeId"`
	Kinds          []edidocument.ControlNumberKind `json:"kinds"`
}

type ResetEDIControlNumberRequest struct {
	TenantInfo     pagination.TenantInfo         `json:"-"`
	PartnerID      pulid.ID                      `json:"partnerId"`
	DocumentTypeID pulid.ID                      `json:"documentTypeId"`
	Kind           edidocument.ControlNumberKind `json:"kind"`
	NextValue      int64                         `json:"nextValue"`
}

type EDIControlNumberRepository interface {
	AllocateControlNumbers(
		ctx context.Context,
		req AllocateEDIControlNumbersRequest,
	) (map[edidocument.ControlNumberKind]int64, error)
	ResetControlNumber(
		ctx context.Context,
		req *ResetEDIControlNumberRequest,
	) (*edidocument.EDIControlNumberSequence, error)
	StartControlNumber(
		ctx context.Context,
		req *ResetEDIControlNumberRequest,
	) (*edidocument.EDIControlNumberSequence, error)
}
