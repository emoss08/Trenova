package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDICommunicationProfilesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type EDICommunicationProfileSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
	Status             domaintypes.Status             `json:"status"`
	Method             edi.ConnectionMethod           `json:"method"`
	PartnerID          pulid.ID                       `json:"partnerId"`
}

type GetEDICommunicationProfileByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetActiveEDICommunicationProfileByPartnerRequest struct {
	PartnerID  pulid.ID              `json:"partnerId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Method     edi.ConnectionMethod  `json:"method"`
}

type EDICommunicationProfileRepository interface {
	ListProfiles(
		ctx context.Context,
		req *ListEDICommunicationProfilesRequest,
	) (*pagination.ListResult[*edi.EDICommunicationProfile], error)
	SelectProfileOptions(
		ctx context.Context,
		req *EDICommunicationProfileSelectOptionsRequest,
	) (*pagination.ListResult[*edi.EDICommunicationProfile], error)
	GetProfileByID(
		ctx context.Context,
		req GetEDICommunicationProfileByIDRequest,
	) (*edi.EDICommunicationProfile, error)
	GetActiveProfileByPartner(
		ctx context.Context,
		req GetActiveEDICommunicationProfileByPartnerRequest,
	) (*edi.EDICommunicationProfile, error)
	CreateProfile(
		ctx context.Context,
		entity *edi.EDICommunicationProfile,
	) (*edi.EDICommunicationProfile, error)
	UpdateProfile(
		ctx context.Context,
		entity *edi.EDICommunicationProfile,
	) (*edi.EDICommunicationProfile, error)
}
