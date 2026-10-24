package tender

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/shopspring/decimal"
)

type TenderOffer struct {
	keel.Table `keel:"tender_offers"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:tof_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	TenderID       pulid.ID `json:"tenderId" keel:"tender_id"`
	CarrierID      pulid.ID `json:"carrierId" keel:"carrier_id"`

	Rank                 int16                      `json:"rank" keel:"rank"`
	RateMethod           shipment.CarrierRateMethod `json:"rateMethod" keel:"rate_method"`
	Rate                 decimal.Decimal            `json:"rate" keel:"rate"`
	OfferTTLSeconds      int32                      `json:"offerTtlSeconds" keel:"offer_ttl_seconds"`
	Channel              Channel                    `json:"channel" keel:"channel"`
	Status               OfferStatus                `json:"status" keel:"status"`
	RecipientEmail       string                     `json:"recipientEmail" keel:"recipient_email,nullzero"`
	SentAt               *int64                     `json:"sentAt" keel:"sent_at"`
	ExpiresAt            *int64                     `json:"expiresAt" keel:"expires_at"`
	RespondedAt          *int64                     `json:"respondedAt" keel:"responded_at"`
	ResponseSource       ResponseSource             `json:"responseSource" keel:"response_source,nullzero"`
	DeclineReason        string                     `json:"declineReason" keel:"decline_reason,nullzero"`
	DeliveryError        string                     `json:"deliveryError" keel:"delivery_error,nullzero"`
	EDIPartnerID         *pulid.ID                  `json:"ediPartnerId" keel:"edi_partner_id"`
	EDIMessageID         *pulid.ID                  `json:"ediMessageId" keel:"edi_message_id"`
	LateResponseAction   string                     `json:"lateResponseAction" keel:"late_response_action,nullzero"`
	LateResponseAt       *int64                     `json:"lateResponseAt" keel:"late_response_at"`
	CounterRate          decimal.NullDecimal        `json:"counterRate" keel:"counter_rate"`
	CounterNote          string                     `json:"counterNote" keel:"counter_note,nullzero"`
	CounteredAt          *int64                     `json:"counteredAt" keel:"countered_at"`
	CounterMargin        decimal.NullDecimal        `json:"counterMargin" keel:"counter_margin_percent"`
	CounterNeedsApproval bool                       `json:"counterNeedsApproval" keel:"counter_needs_approval"`
	CounterDecidedAt     *int64                     `json:"counterDecidedAt" keel:"counter_decided_at"`
	CounterDecidedByID   *pulid.ID                  `json:"counterDecidedById" keel:"counter_decided_by_id"`
	RevokedAt            *int64                     `json:"revokedAt" keel:"revoked_at"`
	RevokeReason         string                     `json:"revokeReason" keel:"revoke_reason,nullzero"`
	RevokedByID          *pulid.ID                  `json:"revokedById" keel:"revoked_by_id"`
	ReofferedFromID      *pulid.ID                  `json:"reofferedFromId" keel:"reoffered_from_id"`
	Version              int64                      `json:"version" keel:"version,version"`
	CreatedAt            int64                      `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt            int64                      `json:"updatedAt" keel:"updated_at,autoupdate"`

	Tender  *Tender          `json:"tender,omitempty" keel:"belongs:tender_id"`
	Carrier *carrier.Carrier `json:"carrier,omitempty" keel:"belongs:carrier_id"`
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

func (to *TenderOffer) BeforeCreate(_ context.Context) error {
	if to.RateMethod == "" {
		to.RateMethod = shipment.CarrierRateMethodFlat
	}
	if to.Status == "" {
		to.Status = OfferStatusPending
	}

	return nil
}
