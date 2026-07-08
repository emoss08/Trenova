package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
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

type EDIPartnerReadinessRow struct {
	PartnerID             pulid.ID `bun:"partner_id"`
	ContactEmail          string   `bun:"contact_email"`
	Timezone              string   `bun:"timezone"`
	HasActiveProfile      bool     `bun:"has_active_profile"`
	HasMappingProfile     bool     `bun:"has_mapping_profile"`
	HasInboundDocProfile  bool     `bun:"has_inbound_doc_profile"`
	HasOutboundDocProfile bool     `bun:"has_outbound_doc_profile"`
	HasPassingTestCase    bool     `bun:"has_passing_test_case"`
	EnabledForInbound     bool     `bun:"enabled_for_inbound"`
	EnabledForOutbound    bool     `bun:"enabled_for_outbound"`
	Kind                  string   `bun:"kind"`
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
}
