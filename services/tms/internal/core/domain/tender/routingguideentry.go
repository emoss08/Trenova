package tender

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const (
	MinOfferTTLSeconds     = int32(300)
	MaxOfferTTLSeconds     = int32(604800)
	DefaultOfferTTLSeconds = int32(7200)
)

var _ bun.BeforeAppendModelHook = (*RoutingGuideEntry)(nil)

type RoutingGuideEntry struct {
	bun.BaseModel `bun:"table:routing_guide_entries,alias:rge" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	RoutingGuideID pulid.ID `json:"routingGuideId" bun:"routing_guide_id,type:VARCHAR(100),notnull"`
	CarrierID      pulid.ID `json:"carrierId"      bun:"carrier_id,type:VARCHAR(100),notnull"`

	Rank            int16                      `json:"rank"            bun:"rank,type:SMALLINT,notnull"`
	RateMethod      shipment.CarrierRateMethod `json:"rateMethod"      bun:"rate_method,type:VARCHAR(50),notnull,default:'Flat'"`
	Rate            decimal.Decimal            `json:"rate"            bun:"rate,type:NUMERIC(19,4),notnull"`
	OfferTTLSeconds int32                      `json:"offerTtlSeconds" bun:"offer_ttl_seconds,type:INTEGER,notnull"`
	Channel         Channel                    `json:"channel"         bun:"channel,type:VARCHAR(50),notnull,default:'Email'"`
	Version         int64                      `json:"version"         bun:"version,type:BIGINT"`
	CreatedAt       int64                      `json:"createdAt"       bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt       int64                      `json:"updatedAt"       bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	RoutingGuide *RoutingGuide    `json:"routingGuide,omitempty" bun:"rel:belongs-to,join:routing_guide_id=id"`
	Carrier      *carrier.Carrier `json:"carrier,omitempty"      bun:"rel:belongs-to,join:carrier_id=id"`
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

func (rge *RoutingGuideEntry) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if rge.ID.IsNil() {
			rge.ID = pulid.MustNew("rge_")
		}
		rge.CreatedAt = now
	case *bun.UpdateQuery:
		rge.UpdatedAt = now
	}

	return nil
}
