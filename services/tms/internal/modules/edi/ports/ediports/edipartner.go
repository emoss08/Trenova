package ediports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDIPartnersRequest struct {
	Filter             *pagination.QueryOptions `json:"filter"`
	Cursor             pagination.CursorInfo    `json:"-"`
	CustomerID         pulid.ID                 `json:"customerId"`
	EnabledForOutbound bool                     `json:"enabledForOutbound"`
	Status             domaintypes.Status       `json:"status"`
}

type GetEDIPartnerByIDRequest struct {
	ID           pulid.ID              `json:"id"`
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	IncludeLinks bool                  `json:"-"`
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

type ListEDIPartnersByCustomerIDsRequest struct {
	CustomerIDs []pulid.ID            `json:"customerIds"`
	TenantInfo  pagination.TenantInfo `json:"tenantInfo"`
}

type EDIPartnerReadinessRow struct {
	PartnerID          pulid.ID
	ContactEmail       string
	Timezone           string
	HasActiveProfile   bool
	HasMappingProfile  bool
	HasLadingProject   bool
	HasActiveVersion   bool
	EnabledForInbound  bool
	EnabledForOutbound bool
	Kind               string
}

type GetEDIPartnerReadinessRequest struct {
	TenantInfo pagination.TenantInfo
	PartnerIDs []pulid.ID
}

type EDIPartnerRepository interface {
	GetReadiness(
		ctx context.Context,
		req *GetEDIPartnerReadinessRequest,
	) ([]*EDIPartnerReadinessRow, error)
	List(
		ctx context.Context,
		req *ListEDIPartnersRequest,
	) (*pagination.ListResult[*edi.EDIPartner], error)
	ListCursor(
		ctx context.Context,
		req *ListEDIPartnersRequest,
	) (*pagination.CursorListResult[*edi.EDIPartner], error)
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
	GetReciprocalInternalPartner(
		ctx context.Context,
		req GetReciprocalInternalPartnerRequest,
	) (*edi.EDIPartner, error)
	ListInternalOutboundPartnersByCustomerIDs(
		ctx context.Context,
		req ListEDIPartnersByCustomerIDsRequest,
	) ([]*edi.EDIPartner, error)
	// ListOutboundPartnersByCustomerIDs returns every active partner enabled
	// for outbound documents, of any kind, oldest first per customer.
	ListOutboundPartnersByCustomerIDs(
		ctx context.Context,
		req ListEDIPartnersByCustomerIDsRequest,
	) ([]*edi.EDIPartner, error)
}
