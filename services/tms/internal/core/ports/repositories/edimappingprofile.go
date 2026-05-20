package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetMappingProfileRequest struct {
	PartnerID  pulid.ID              `json:"partnerId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListEDIMappingProfilesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type EDIMappingProfileSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
	PartnerID          pulid.ID                       `json:"partnerId"`
}

type GetMappingProfileByIDRequest struct {
	ProfileID  pulid.ID              `json:"profileId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type SaveMappingItemsRequest struct {
	PartnerID  pulid.ID                     `json:"partnerId"`
	TenantInfo pagination.TenantInfo        `json:"tenantInfo"`
	ActorID    pulid.ID                     `json:"actorId"`
	Items      []*edi.EDIMappingProfileItem `json:"items"`
}

type SaveMappingProfileItemsRequest struct {
	ProfileID  pulid.ID                     `json:"profileId"`
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

type DeleteMappingProfileItemRequest struct {
	ProfileID     pulid.ID              `json:"profileId"`
	MappingItemID pulid.ID              `json:"mappingItemId"`
	TenantInfo    pagination.TenantInfo `json:"tenantInfo"`
}

type EDIMappingProfileRepository interface {
	GetMappingProfile(
		ctx context.Context,
		req GetMappingProfileRequest,
	) (*edi.EDIMappingProfile, error)
	ListMappingProfiles(
		ctx context.Context,
		req *ListEDIMappingProfilesRequest,
	) (*pagination.ListResult[*edi.EDIMappingProfile], error)
	SelectMappingProfileOptions(
		ctx context.Context,
		req *EDIMappingProfileSelectOptionsRequest,
	) (*pagination.ListResult[*edi.EDIMappingProfile], error)
	GetMappingProfileByID(
		ctx context.Context,
		req GetMappingProfileByIDRequest,
	) (*edi.EDIMappingProfile, error)
	SaveMappingItems(
		ctx context.Context,
		req *SaveMappingItemsRequest,
	) ([]*edi.EDIMappingProfileItem, error)
	SaveMappingProfileItems(
		ctx context.Context,
		req *SaveMappingProfileItemsRequest,
	) ([]*edi.EDIMappingProfileItem, error)
	GetMappingItems(
		ctx context.Context,
		req GetMappingItemsRequest,
	) ([]*edi.EDIMappingProfileItem, error)
	DeleteMappingItem(ctx context.Context, req DeleteMappingItemRequest) error
	DeleteMappingProfileItem(ctx context.Context, req DeleteMappingProfileItemRequest) error
}
