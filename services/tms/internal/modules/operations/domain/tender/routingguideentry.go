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
	"github.com/shopspring/decimal"
)

const (
	MinOfferTTLSeconds     = int32(300)
	MaxOfferTTLSeconds     = int32(604800)
	DefaultOfferTTLSeconds = int32(7200)
)

type RoutingGuideEntry struct {
	keel.Table `keel:"routing_guide_entries"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:rge_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	RoutingGuideID pulid.ID `json:"routingGuideId" keel:"routing_guide_id"`
	CarrierID      pulid.ID `json:"carrierId"      keel:"carrier_id"`

	Rank       int16                      `json:"rank"       keel:"rank"`
	RateMethod shipment.CarrierRateMethod `json:"rateMethod" keel:"rate_method"`
	Rate       decimal.Decimal            `json:"rate"       keel:"rate"`

	// UseContractRate offers what the carrier's contract says today rather than
	// the rate frozen on this entry. A guide written a year ago carries the
	// number somebody typed a year ago; a contract carries the one that was
	// negotiated. The rate above stays as the fallback for a lane no contract
	// covers.
	UseContractRate bool    `json:"useContractRate" keel:"use_contract_rate"`
	OfferTTLSeconds int32   `json:"offerTtlSeconds" keel:"offer_ttl_seconds"`
	Channel         Channel `json:"channel"         keel:"channel"`
	Version         int64   `json:"version"         keel:"version,version"`
	CreatedAt       int64   `json:"createdAt"       keel:"created_at,autocreate"`
	UpdatedAt       int64   `json:"updatedAt"       keel:"updated_at,autoupdate"`

	RoutingGuide *RoutingGuide    `json:"routingGuide,omitempty" keel:"belongs:routing_guide_id"`
	Carrier      *carrier.Carrier `json:"carrier,omitempty"      keel:"belongs:carrier_id"`
}

func (rge *RoutingGuideEntry) applyDefaults() {
	if rge.RateMethod == "" {
		rge.RateMethod = shipment.CarrierRateMethodFlat
	}
	if rge.Channel == "" {
		rge.Channel = ChannelEmail
	}
	if rge.OfferTTLSeconds == 0 {
		rge.OfferTTLSeconds = DefaultOfferTTLSeconds
	}
}

func (rge *RoutingGuideEntry) Validate(multiErr *errortypes.MultiError) {
	rge.applyDefaults()

	multiErr.AddOzzoError(validation.ValidateStruct(rge,
		validation.Field(&rge.CarrierID,
			validation.Required.Error("Carrier is required"),
		),
		validation.Field(&rge.Rank,
			validation.Min(1).Error("Rank must be at least 1"),
		),
		validation.Field(&rge.RateMethod,
			validation.Required.Error("Rate method is required"),
			domainvalidation.ValidEnum[shipment.CarrierRateMethod]("Rate method is invalid"),
		),
		validation.Field(&rge.Channel,
			validation.Required.Error("Channel is required"),
			domainvalidation.ValidEnum[Channel]("Channel is invalid"),
		),
		validation.Field(&rge.OfferTTLSeconds,
			validation.Min(MinOfferTTLSeconds).
				Error("Offer expiry must be at least 5 minutes"),
			validation.Max(MaxOfferTTLSeconds).
				Error("Offer expiry cannot exceed 7 days"),
		),
	))

	if rge.Rate.IsNegative() {
		multiErr.Add("rate", errortypes.ErrInvalid, "Rate cannot be negative")
	}
}

func (rge *RoutingGuideEntry) GetID() pulid.ID {
	return rge.ID
}

func (rge *RoutingGuideEntry) GetCreatedAt() int64 {
	return rge.CreatedAt
}

func (rge *RoutingGuideEntry) GetTableName() string {
	return "routing_guide_entries"
}

func (rge *RoutingGuideEntry) GetOrganizationID() pulid.ID {
	return rge.OrganizationID
}

func (rge *RoutingGuideEntry) GetBusinessUnitID() pulid.ID {
	return rge.BusinessUnitID
}

func (rge *RoutingGuideEntry) BeforeCreate(_ context.Context) error {
	if rge.RateMethod == "" {
		rge.RateMethod = shipment.CarrierRateMethodFlat
	}
	if rge.Channel == "" {
		rge.Channel = ChannelEmail
	}

	return nil
}
