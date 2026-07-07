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
	Cursor pagination.CursorInfo    `json:"-"`
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
	PartnerID  pulid.ID               `json:"partnerId"`
	TenantInfo pagination.TenantInfo  `json:"tenantInfo"`
	Method     edi.ConnectionMethod   `json:"method"`
	Methods    []edi.ConnectionMethod `json:"methods"`
}

type GetActiveAS2ProfileByIdentifiersRequest struct {
	LocalAS2ID   string `json:"localAs2Id"`
	PartnerAS2ID string `json:"partnerAs2Id"`
}

type RecordEDIProfilePollOutcomeRequest struct {
	ProfileID  pulid.ID
	TenantInfo pagination.TenantInfo
	PolledAt   int64
	Success    bool
	Error      string
}

type EDICommunicationProfileRepository interface {
	GetActiveAS2ProfileByIdentifiers(
		ctx context.Context,
		req GetActiveAS2ProfileByIdentifiersRequest,
	) (*edi.EDICommunicationProfile, error)
	ListProfiles(
		ctx context.Context,
		req *ListEDICommunicationProfilesRequest,
	) (*pagination.ListResult[*edi.EDICommunicationProfile], error)
	ListProfilesCursor(
		ctx context.Context,
		req *ListEDICommunicationProfilesRequest,
	) (*pagination.CursorListResult[*edi.EDICommunicationProfile], error)
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
	ListInboundPollingProfiles(
		ctx context.Context,
	) ([]*edi.EDICommunicationProfile, error)
	RecordInboundPollOutcome(
		ctx context.Context,
		req RecordEDIProfilePollOutcomeRequest,
	) error
	CountStaleInboundPollingProfiles(ctx context.Context, staleBefore int64) (int64, error)
	CreateProfile(
		ctx context.Context,
		entity *edi.EDICommunicationProfile,
	) (*edi.EDICommunicationProfile, error)
	UpdateProfile(
		ctx context.Context,
		entity *edi.EDICommunicationProfile,
	) (*edi.EDICommunicationProfile, error)
}
