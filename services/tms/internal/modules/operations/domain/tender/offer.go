package tender

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*TenderOffer)(nil)

type TenderOffer struct {
	bun.BaseModel `bun:"table:tender_offers,alias:tof" json:"-"`
	keel.Table    `keel:"tender_offers"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull" keel:"id,pk,autoid:tof_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	TenderID       pulid.ID `json:"tenderId"       bun:"tender_id,type:VARCHAR(100),notnull" keel:"tender_id"`
	CarrierID      pulid.ID `json:"carrierId"      bun:"carrier_id,type:VARCHAR(100),notnull" keel:"carrier_id"`

	Rank                 int16                      `json:"rank"                 bun:"rank,type:SMALLINT,notnull" keel:"rank"`
	RateMethod           shipment.CarrierRateMethod `json:"rateMethod"           bun:"rate_method,type:VARCHAR(50),notnull,default:'Flat'" keel:"rate_method"`
	Rate                 decimal.Decimal            `json:"rate"                 bun:"rate,type:NUMERIC(19,4),notnull" keel:"rate"`
	OfferTTLSeconds      int32                      `json:"offerTtlSeconds"      bun:"offer_ttl_seconds,type:INTEGER,notnull" keel:"offer_ttl_seconds"`
	Channel              Channel                    `json:"channel"              bun:"channel,type:VARCHAR(50),notnull" keel:"channel"`
	Status               OfferStatus                `json:"status"               bun:"status,type:VARCHAR(50),notnull,default:'Pending'" keel:"status"`
	RecipientEmail       string                     `json:"recipientEmail"       bun:"recipient_email,type:VARCHAR(255),nullzero" keel:"recipient_email,nullzero"`
	SentAt               *int64                     `json:"sentAt"               bun:"sent_at,type:BIGINT,nullzero" keel:"sent_at"`
	ExpiresAt            *int64                     `json:"expiresAt"            bun:"expires_at,type:BIGINT,nullzero" keel:"expires_at"`
	RespondedAt          *int64                     `json:"respondedAt"          bun:"responded_at,type:BIGINT,nullzero" keel:"responded_at"`
	ResponseSource       ResponseSource             `json:"responseSource"       bun:"response_source,type:VARCHAR(50),nullzero" keel:"response_source,nullzero"`
	DeclineReason        string                     `json:"declineReason"        bun:"decline_reason,type:TEXT,nullzero" keel:"decline_reason,nullzero"`
	DeliveryError        string                     `json:"deliveryError"        bun:"delivery_error,type:TEXT,nullzero" keel:"delivery_error,nullzero"`
	EDIPartnerID         *pulid.ID                  `json:"ediPartnerId"         bun:"edi_partner_id,type:VARCHAR(100),nullzero" keel:"edi_partner_id"`
	EDIMessageID         *pulid.ID                  `json:"ediMessageId"         bun:"edi_message_id,type:VARCHAR(100),nullzero" keel:"edi_message_id"`
	LateResponseAction   string                     `json:"lateResponseAction"   bun:"late_response_action,type:VARCHAR(50),nullzero" keel:"late_response_action,nullzero"`
	LateResponseAt       *int64                     `json:"lateResponseAt"       bun:"late_response_at,type:BIGINT,nullzero" keel:"late_response_at"`
	CounterRate          decimal.NullDecimal        `json:"counterRate"          bun:"counter_rate,type:NUMERIC(19,4),nullzero" keel:"counter_rate"`
	CounterNote          string                     `json:"counterNote"          bun:"counter_note,type:TEXT,nullzero" keel:"counter_note,nullzero"`
	CounteredAt          *int64                     `json:"counteredAt"          bun:"countered_at,type:BIGINT,nullzero" keel:"countered_at"`
	CounterMargin        decimal.NullDecimal        `json:"counterMargin"        bun:"counter_margin_percent,type:NUMERIC(9,4),nullzero" keel:"counter_margin_percent"`
	CounterNeedsApproval bool                       `json:"counterNeedsApproval" bun:"counter_needs_approval,type:BOOLEAN,notnull" keel:"counter_needs_approval"`
	CounterDecidedAt     *int64                     `json:"counterDecidedAt"     bun:"counter_decided_at,type:BIGINT,nullzero" keel:"counter_decided_at"`
	CounterDecidedByID   *pulid.ID                  `json:"counterDecidedById"   bun:"counter_decided_by_id,type:VARCHAR(100),nullzero" keel:"counter_decided_by_id"`
	RevokedAt            *int64                     `json:"revokedAt"            bun:"revoked_at,type:BIGINT,nullzero" keel:"revoked_at"`
	RevokeReason         string                     `json:"revokeReason"         bun:"revoke_reason,type:TEXT,nullzero" keel:"revoke_reason,nullzero"`
	RevokedByID          *pulid.ID                  `json:"revokedById"          bun:"revoked_by_id,type:VARCHAR(100),nullzero" keel:"revoked_by_id"`
	ReofferedFromID      *pulid.ID                  `json:"reofferedFromId"      bun:"reoffered_from_id,type:VARCHAR(100),nullzero" keel:"reoffered_from_id"`
	Version              int64                      `json:"version"              bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt            int64                      `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt            int64                      `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Tender  *Tender          `json:"tender,omitempty"  bun:"rel:belongs-to,join:tender_id=id" keel:"belongs:tender_id"`
	Carrier *carrier.Carrier `json:"carrier,omitempty" bun:"rel:belongs-to,join:carrier_id=id" keel:"belongs:carrier_id"`
}

func (to *TenderOffer) applyDefaults() {
	if to.Status == "" {
		to.Status = OfferStatusPending
	}
	if to.RateMethod == "" {
		to.RateMethod = shipment.CarrierRateMethodFlat
	}
	if to.OfferTTLSeconds == 0 {
		to.OfferTTLSeconds = DefaultOfferTTLSeconds
	}
}

func (to *TenderOffer) IsExpiredAt(now int64) bool {
	return to != nil &&
		to.Status == OfferStatusSent &&
		to.ExpiresAt != nil &&
		*to.ExpiresAt <= now
}

// AgreedRate is the rate the carrier is held to: its own counter once it has
// made one that still stands or was taken, the offered rate otherwise.
func (to *TenderOffer) AgreedRate() decimal.Decimal {
	if to.CounterRate.Valid && to.Status != OfferStatusCounterRejected {
		return to.CounterRate.Decimal
	}
	return to.Rate
}

func (to *TenderOffer) Validate(multiErr *errortypes.MultiError) {
	to.applyDefaults()

	multiErr.AddOzzoError(validation.ValidateStruct(to,
		validation.Field(&to.CarrierID,
			validation.Required.Error("Carrier is required"),
		),
		validation.Field(&to.Rank,
			validation.Min(1).Error("Rank must be at least 1"),
		),
		validation.Field(&to.RateMethod,
			validation.Required.Error("Rate method is required"),
			domainvalidation.ValidEnum[shipment.CarrierRateMethod]("Rate method is invalid"),
		),
		validation.Field(&to.Channel,
			validation.Required.Error("Channel is required"),
			domainvalidation.ValidEnum[Channel]("Channel is invalid"),
		),
		validation.Field(&to.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[OfferStatus]("Status is invalid"),
		),
		validation.Field(&to.OfferTTLSeconds,
			validation.Min(MinOfferTTLSeconds).
				Error("Offer expiry must be at least 5 minutes"),
			validation.Max(MaxOfferTTLSeconds).
				Error("Offer expiry cannot exceed 7 days"),
		),
		validation.Field(&to.RecipientEmail,
			validation.When(to.Channel == ChannelEmail,
				validation.Required.Error("An email offer requires a recipient email address"),
				is.EmailFormat.Error("Recipient email must be a valid email address"),
			),
		),
	))

	if to.Rate.IsNegative() {
		multiErr.Add("rate", errortypes.ErrInvalid, "Rate cannot be negative")
	}
	if to.Channel == ChannelEDI && (to.EDIPartnerID == nil || to.EDIPartnerID.IsNil()) {
		multiErr.Add(
			"ediPartnerId",
			errortypes.ErrRequired,
			"An EDI offer requires an EDI partner",
		)
	}
}

func (to *TenderOffer) GetID() pulid.ID {
	return to.ID
}

func (to *TenderOffer) GetCreatedAt() int64 {
	return to.CreatedAt
}

func (to *TenderOffer) GetTableName() string {
	return "tender_offers"
}

func (to *TenderOffer) GetOrganizationID() pulid.ID {
	return to.OrganizationID
}

func (to *TenderOffer) GetBusinessUnitID() pulid.ID {
	return to.BusinessUnitID
}

func (to *TenderOffer) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if to.ID.IsNil() {
			to.ID = pulid.MustNew("tof_")
		}
		to.CreatedAt = now
	case *bun.UpdateQuery:
		to.UpdatedAt = now
	}

	return nil
}

func (to *TenderOffer) BeforeCreate(_ context.Context) error {
	if to.RateMethod == "" {
		to.RateMethod = shipment.CarrierRateMethodFlat
	}
	if to.Status == "" {
		to.Status = OfferStatusPending
	}

	return nil
}
