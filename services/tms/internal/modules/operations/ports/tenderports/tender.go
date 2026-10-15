package tenderports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type GetTenderByIDRequest struct {
	TenantInfo    pagination.TenantInfo `json:"-"`
	TenderID      pulid.ID              `json:"tenderId"`
	IncludeOffers bool                  `json:"includeOffers"`
}

type GetTenderOfferByIDRequest struct {
	TenantInfo    pagination.TenantInfo `json:"-"`
	OfferID       pulid.ID              `json:"offerId"`
	IncludeTender bool                  `json:"includeTender"`
}

type ListTendersByShipmentRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ShipmentID pulid.ID              `json:"shipmentId"`
}

type GetLiveTenderByMoveRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	MoveID     pulid.ID              `json:"moveId"`
}

type ListLiveTendersByMovesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	MoveIDs    []pulid.ID            `json:"moveIds"`
}

type UpdateTenderStatusRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	TenderID   pulid.ID              `json:"tenderId"`
	FromStatus []tender.Status       `json:"fromStatus"`
	ToStatus   tender.Status         `json:"toStatus"`

	CurrentRank        *int16    `json:"currentRank"`
	CanceledByID       *pulid.ID `json:"canceledById"`
	CancellationReason string    `json:"cancellationReason"`
	AcceptedOfferID    *pulid.ID `json:"acceptedOfferId"`
	AcceptedAt         *int64    `json:"acceptedAt"`
	ExhaustedAt        *int64    `json:"exhaustedAt"`
	CanceledAt         *int64    `json:"canceledAt"`
}

type UpdateOfferStatusRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	OfferID    pulid.ID              `json:"offerId"`
	FromStatus []tender.OfferStatus  `json:"fromStatus"`
	ToStatus   tender.OfferStatus    `json:"toStatus"`

	SentAt         *int64                `json:"sentAt"`
	ExpiresAt      *int64                `json:"expiresAt"`
	RespondedAt    *int64                `json:"respondedAt"`
	ResponseSource tender.ResponseSource `json:"responseSource"`
	DeclineReason  string                `json:"declineReason"`
	DeliveryError  string                `json:"deliveryError"`
	EDIMessageID   *pulid.ID             `json:"ediMessageId"`

	Counter        *OfferCounter  `json:"counter"`
	CounterDecided *DecisionStamp `json:"counterDecided"`
}

// OfferCounter is a carrier's counter-offer as it is recorded on the offer.
type OfferCounter struct {
	Rate          decimal.Decimal     `json:"rate"`
	Note          string              `json:"note"`
	At            int64               `json:"at"`
	MarginPercent decimal.NullDecimal `json:"marginPercent"`
	NeedsApproval bool                `json:"needsApproval"`
}

// DecisionStamp is who decided something on an offer, and when. A nil user
// is a decision the policy made on its own.
type DecisionStamp struct {
	At     int64     `json:"at"`
	UserID *pulid.ID `json:"userId"`
}

// ReopenTenderRequest walks an accepted tender back to live after its carrier
// backs out: the acceptance is revoked and the carriers it cut short are
// offered the move again.
type ReopenTenderRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	TenderID       pulid.ID              `json:"tenderId"`
	RevokedOfferID pulid.ID              `json:"revokedOfferId"`
	Reason         string                `json:"reason"`
	RevokedByID    *pulid.ID             `json:"revokedById"`
	At             int64                 `json:"at"`
	Reoffers       []*tender.TenderOffer `json:"reoffers"`
}

type BulkOfferStatusRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	TenderID   pulid.ID              `json:"tenderId"`
	FromStatus []tender.OfferStatus  `json:"fromStatus"`
	ToStatus   tender.OfferStatus    `json:"toStatus"`
	ExceptID   pulid.ID              `json:"exceptId"`
}

type RecordLateOfferResponseRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	OfferID    pulid.ID              `json:"offerId"`
	Action     tender.ResponseAction `json:"action"`
	Source     tender.ResponseSource `json:"source"`
	OccurredAt int64                 `json:"occurredAt"`
}

type GetAcceptedOfferForPartnerReferenceRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	PartnerID  pulid.ID              `json:"partnerId"`
	OfferID    pulid.ID              `json:"offerId"`
	References []string              `json:"references"`
}

type ListTendersForSweepRequest struct {
	TenantInfo *pagination.TenantInfo `json:"-"`
	ExpiredBy  int64                  `json:"expiredBy"`
	Limit      int                    `json:"limit"`
}

type ListAcceptedMissingRateConfirmationRequest struct {
	AcceptedAfter  int64 `json:"acceptedAfter"`
	AcceptedBefore int64 `json:"acceptedBefore"`
	Limit          int   `json:"limit"`
}

type PurgeDeadTokensRequest struct {
	DeadBefore int64 `json:"deadBefore"`
	Limit      int   `json:"limit"`
}

type MarkTokenUsedRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	TokenID    pulid.ID              `json:"tokenId"`
	UsedAt     int64                 `json:"usedAt"`
}

type TenderRepository interface {
	Create(ctx context.Context, entity *tender.Tender) (*tender.Tender, error)
	GetByID(ctx context.Context, req GetTenderByIDRequest) (*tender.Tender, error)
	GetLiveByMoveID(
		ctx context.Context,
		req GetLiveTenderByMoveRequest,
	) (*tender.Tender, error)
	ListLiveByMoveIDs(
		ctx context.Context,
		req *ListLiveTendersByMovesRequest,
	) ([]*tender.Tender, error)
	GetOfferByID(
		ctx context.Context,
		req GetTenderOfferByIDRequest,
	) (*tender.TenderOffer, error)
	GetAcceptedOfferForPartnerReference(
		ctx context.Context,
		req GetAcceptedOfferForPartnerReferenceRequest,
	) (*tender.TenderOffer, error)
	ListByShipment(
		ctx context.Context,
		req ListTendersByShipmentRequest,
	) ([]*tender.Tender, error)
	ListForSweep(
		ctx context.Context,
		req ListTendersForSweepRequest,
	) ([]*tender.Tender, error)
	ListAcceptedMissingRateConfirmation(
		ctx context.Context,
		req ListAcceptedMissingRateConfirmationRequest,
	) ([]*tender.Tender, error)
	SetWorkflowID(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		tenderID pulid.ID,
		workflowID string,
	) error
	UpdateTenderStatus(
		ctx context.Context,
		req *UpdateTenderStatusRequest,
	) (bool, error)
	UpdateOfferStatus(
		ctx context.Context,
		req *UpdateOfferStatusRequest,
	) (bool, error)
	Reopen(
		ctx context.Context,
		req *ReopenTenderRequest,
	) (bool, error)
	BulkUpdateOfferStatus(
		ctx context.Context,
		req *BulkOfferStatusRequest,
	) (int64, error)
	RecordLateOfferResponse(
		ctx context.Context,
		req *RecordLateOfferResponseRequest,
	) error
	CreateToken(
		ctx context.Context,
		entity *tender.TenderOfferToken,
	) (*tender.TenderOfferToken, error)
	GetTokenByHash(
		ctx context.Context,
		tokenHash string,
	) (*tender.TenderOfferToken, error)
	MarkTokenUsed(
		ctx context.Context,
		req *MarkTokenUsedRequest,
	) (bool, error)
	RevokeTokensForOffer(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		offerID pulid.ID,
		revokedAt int64,
	) error
	PurgeDeadOfferTokens(
		ctx context.Context,
		req PurgeDeadTokensRequest,
	) (int64, error)
}
