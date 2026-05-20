package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDIConnectionsRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetEDIConnectionByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetEDIConnectionForUpdateRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetActiveEDIConnectionForPartnerRequest struct {
	PartnerID  pulid.ID              `json:"partnerId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Method     edi.ConnectionMethod  `json:"method"`
}

type CreateInternalEDIConnectionAcceptanceRequest struct {
	Connection    *edi.EDIConnection           `json:"connection"`
	SourcePartner *edi.EDIPartner              `json:"sourcePartner"`
	TargetPartner *edi.EDIPartner              `json:"targetPartner"`
	SourceProfile *edi.EDICommunicationProfile `json:"sourceProfile"`
	TargetProfile *edi.EDICommunicationProfile `json:"targetProfile"`
	TenantInfo    pagination.TenantInfo        `json:"tenantInfo"`
}

type EDIConnectionRepository interface {
	ListConnections(
		ctx context.Context,
		req *ListEDIConnectionsRequest,
	) (*pagination.ListResult[*edi.EDIConnection], error)
	GetConnectionByID(
		ctx context.Context,
		req GetEDIConnectionByIDRequest,
	) (*edi.EDIConnection, error)
	GetConnectionForUpdate(
		ctx context.Context,
		req GetEDIConnectionForUpdateRequest,
	) (*edi.EDIConnection, error)
	GetActiveConnectionForPartner(
		ctx context.Context,
		req GetActiveEDIConnectionForPartnerRequest,
	) (*edi.EDIConnection, error)
	CreateConnection(
		ctx context.Context,
		entity *edi.EDIConnection,
	) (*edi.EDIConnection, error)
	UpdateConnection(
		ctx context.Context,
		entity *edi.EDIConnection,
	) (*edi.EDIConnection, error)
	AcceptInternalConnection(
		ctx context.Context,
		req *CreateInternalEDIConnectionAcceptanceRequest,
	) (*edi.EDIConnection, error)
}
