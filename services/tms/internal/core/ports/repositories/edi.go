package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDIPartnersRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetEDIPartnerByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type EDIPartnerSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
	Kind               edi.PartnerKind                `json:"kind"`
	EnabledForOutbound bool                           `json:"enabledForOutbound"`
}

type GetReciprocalInternalPartnerRequest struct {
	SourceOrganizationID pulid.ID `json:"sourceOrganizationId"`
	TargetOrganizationID pulid.ID `json:"targetOrganizationId"`
	BusinessUnitID       pulid.ID `json:"businessUnitId"`
}

type CreateInternalPartnerPairRequest struct {
	SourcePartner        *edi.EDIPartner       `json:"sourcePartner"`
	TargetPartner        *edi.EDIPartner       `json:"targetPartner"`
	SourceOrganizationID pulid.ID              `json:"sourceOrganizationId"`
	TargetOrganizationID pulid.ID              `json:"targetOrganizationId"`
	BusinessUnitID       pulid.ID              `json:"businessUnitId"`
	TenantInfo           pagination.TenantInfo `json:"tenantInfo"`
}

type GetMappingProfileRequest struct {
	PartnerID  pulid.ID              `json:"partnerId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type SaveMappingItemsRequest struct {
	PartnerID  pulid.ID                     `json:"partnerId"`
	TenantInfo pagination.TenantInfo        `json:"tenantInfo"`
	ActorID    pulid.ID                     `json:"actorId"`
	Items      []*edi.EDIMappingProfileItem `json:"items"`
}

type GetMappingItemsRequest struct {
	PartnerID   pulid.ID                `json:"partnerId"`
	TenantInfo  pagination.TenantInfo   `json:"tenantInfo"`
	EntityTypes []edi.MappingEntityType `json:"entityTypes"`
	SourceIDs   []pulid.ID              `json:"sourceIds"`
}

type DeleteMappingItemRequest struct {
	PartnerID     pulid.ID              `json:"partnerId"`
	MappingItemID pulid.ID              `json:"mappingItemId"`
	TenantInfo    pagination.TenantInfo `json:"tenantInfo"`
}

type ListEDITransfersRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetEDITransferByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Direction  string                `json:"direction"`
}

type GetEDITransferForUpdateRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Direction  string                `json:"direction"`
}

type SetEDITransferApprovalWorkflowRunIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	RunID      string                `json:"runId"`
}

type EDILoadTenderTransferRepository interface {
	ListInbound(
		ctx context.Context,
		req *ListEDITransfersRequest,
	) (*pagination.ListResult[*edi.EDITransfer], error)
	ListOutbound(
		ctx context.Context,
		req *ListEDITransfersRequest,
	) (*pagination.ListResult[*edi.EDITransfer], error)
	GetTransferByID(
		ctx context.Context,
		req GetEDITransferByIDRequest,
	) (*edi.EDITransfer, error)
	GetTransferForUpdate(
		ctx context.Context,
		req GetEDITransferForUpdateRequest,
	) (*edi.EDITransfer, error)
	CreateTransfer(
		ctx context.Context,
		entity *edi.EDITransfer,
	) (*edi.EDITransfer, error)
	UpdateTransfer(
		ctx context.Context,
		entity *edi.EDITransfer,
	) (*edi.EDITransfer, error)
	SetApprovalWorkflowRunID(
		ctx context.Context,
		req SetEDITransferApprovalWorkflowRunIDRequest,
	) (*edi.EDITransfer, error)
}

type EDIPartnerRepository interface {
	List(
		ctx context.Context,
		req *ListEDIPartnersRequest,
	) (*pagination.ListResult[*edi.EDIPartner], error)
	SelectOptions(
		ctx context.Context,
		req *EDIPartnerSelectOptionsRequest,
	) (*pagination.ListResult[*edi.EDIPartner], error)
	GetByID(
		ctx context.Context,
		req GetEDIPartnerByIDRequest,
	) (*edi.EDIPartner, error)
	Create(
		ctx context.Context,
		entity *edi.EDIPartner,
	) (*edi.EDIPartner, error)
	Update(
		ctx context.Context,
		entity *edi.EDIPartner,
	) (*edi.EDIPartner, error)
	CreateInternalPair(
		ctx context.Context,
		req *CreateInternalPartnerPairRequest,
	) (*edi.InternalPartnerPair, error)
	GetReciprocalInternalPartner(
		ctx context.Context,
		req GetReciprocalInternalPartnerRequest,
	) (*edi.EDIPartner, error)
	GetMappingProfile(
		ctx context.Context,
		req GetMappingProfileRequest,
	) (*edi.EDIMappingProfile, error)
	SaveMappingItems(
		ctx context.Context,
		req *SaveMappingItemsRequest,
	) ([]*edi.EDIMappingProfileItem, error)
	GetMappingItems(
		ctx context.Context,
		req GetMappingItemsRequest,
	) ([]*edi.EDIMappingProfileItem, error)
	DeleteMappingItem(ctx context.Context, req DeleteMappingItemRequest) error
}
