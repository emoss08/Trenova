package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDIPartnerDocumentProfilesRequest struct {
	Filter         *pagination.QueryOptions `json:"filter"`
	TransactionSet edi.TransactionSet       `json:"transactionSet"`
	Direction      edi.DocumentDirection    `json:"direction"`
	Status         edi.DocumentStatus       `json:"status"`
	PartnerID      pulid.ID                 `json:"partnerId"`
}

type EDIPartnerDocumentProfileSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
	TransactionSet     edi.TransactionSet             `json:"transactionSet"`
	Direction          edi.DocumentDirection          `json:"direction"`
	Status             edi.DocumentStatus             `json:"status"`
	PartnerID          pulid.ID                       `json:"partnerId"`
}

type GetEDIPartnerDocumentProfileByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetActiveEDIPartnerDocumentProfileRequest struct {
	PartnerID      pulid.ID              `json:"partnerId"`
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	TransactionSet edi.TransactionSet    `json:"transactionSet"`
	Direction      edi.DocumentDirection `json:"direction"`
}

type EDIPartnerDocumentProfileRepository interface {
	ListPartnerDocumentProfiles(
		ctx context.Context,
		req *ListEDIPartnerDocumentProfilesRequest,
	) (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error)
	SelectPartnerDocumentProfileOptions(
		ctx context.Context,
		req *EDIPartnerDocumentProfileSelectOptionsRequest,
	) (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error)
	GetPartnerDocumentProfileByID(
		ctx context.Context,
		req GetEDIPartnerDocumentProfileByIDRequest,
	) (*edi.EDIPartnerDocumentProfile, error)
	GetActivePartnerDocumentProfile(
		ctx context.Context,
		req GetActiveEDIPartnerDocumentProfileRequest,
	) (*edi.EDIPartnerDocumentProfile, error)
	CreatePartnerDocumentProfile(
		ctx context.Context,
		entity *edi.EDIPartnerDocumentProfile,
	) (*edi.EDIPartnerDocumentProfile, error)
	UpdatePartnerDocumentProfile(
		ctx context.Context,
		entity *edi.EDIPartnerDocumentProfile,
	) (*edi.EDIPartnerDocumentProfile, error)
}
