package tender

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*Tender)(nil)

type Tender struct {
	bun.BaseModel `bun:"table:tenders,alias:tnd" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	ShipmentID     pulid.ID `json:"shipmentId"     bun:"shipment_id,type:VARCHAR(100),notnull"`
	ShipmentMoveID pulid.ID `json:"shipmentMoveId" bun:"shipment_move_id,type:VARCHAR(100),notnull"`

	RoutingGuideID     *pulid.ID `json:"routingGuideId"     bun:"routing_guide_id,type:VARCHAR(100),nullzero"`
	Mode               Mode      `json:"mode"               bun:"mode,type:VARCHAR(50),notnull"`
	Status             Status    `json:"status"             bun:"status,type:VARCHAR(50),notnull,default:'Active'"`
	CurrentRank        int16     `json:"currentRank"        bun:"current_rank,type:SMALLINT,notnull,default:0"`
	WorkflowID         string    `json:"workflowId"         bun:"workflow_id,type:VARCHAR(255),nullzero"`
	CreatedByID        *pulid.ID `json:"createdById"        bun:"created_by_id,type:VARCHAR(100),nullzero"`
	CanceledByID       *pulid.ID `json:"canceledById"       bun:"canceled_by_id,type:VARCHAR(100),nullzero"`
	CancellationReason string    `json:"cancellationReason" bun:"cancellation_reason,type:TEXT,nullzero"`
	AcceptedOfferID    *pulid.ID `json:"acceptedOfferId"    bun:"accepted_offer_id,type:VARCHAR(100),nullzero"`
	AcceptedAt         *int64    `json:"acceptedAt"         bun:"accepted_at,type:BIGINT,nullzero"`
	ExhaustedAt        *int64    `json:"exhaustedAt"        bun:"exhausted_at,type:BIGINT,nullzero"`
	CanceledAt         *int64    `json:"canceledAt"         bun:"canceled_at,type:BIGINT,nullzero"`
	Version            int64     `json:"version"            bun:"version,type:BIGINT"`
	CreatedAt          int64     `json:"createdAt"          bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64     `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Shipment     *shipment.Shipment     `json:"shipment,omitempty"     bun:"rel:belongs-to,join:shipment_id=id"`
	ShipmentMove *shipment.ShipmentMove `json:"shipmentMove,omitempty" bun:"rel:belongs-to,join:shipment_move_id=id"`
	RoutingGuide *RoutingGuide          `json:"routingGuide,omitempty" bun:"rel:belongs-to,join:routing_guide_id=id"`
	Offers       []*TenderOffer         `json:"offers,omitempty"       bun:"rel:has-many,join:id=tender_id"`
}

func (t *Tender) applyDefaults() {
	if t.Status == "" {
		t.Status = StatusActive
	}
}

func (t *Tender) IsLive() bool {
	return t != nil && (t.Status == StatusActive || t.Status == StatusNeedsReview)
}

func (t *Tender) FindOffer(offerID pulid.ID) *TenderOffer {
	if t == nil {
		return nil
	}
	for _, offer := range t.Offers {
		if offer != nil && offer.ID == offerID {
			return offer
		}
	}
	return nil
}

func (t *Tender) Validate(multiErr *errortypes.MultiError) {
	t.applyDefaults()

	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.ShipmentID,
			validation.Required.Error("Shipment is required"),
		),
		validation.Field(&t.ShipmentMoveID,
			validation.Required.Error("Shipment move is required"),
		),
		validation.Field(&t.Mode,
			validation.Required.Error("Mode is required"),
			domainvalidation.ValidEnum[Mode]("Mode is invalid"),
		),
		validation.Field(&t.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[Status]("Status is invalid"),
		),
		validation.Field(&t.CancellationReason,
			validation.When(
				t.Status == StatusCanceled,
				validation.Required.Error("A canceled tender must record its reason"),
			),
		),
	))

	if t.Mode == ModeWaterfall && (t.RoutingGuideID == nil || t.RoutingGuideID.IsNil()) {
		multiErr.Add(
			"routingGuideId",
			errortypes.ErrRequired,
			"A waterfall tender requires a routing guide",
		)
	}

	if len(t.Offers) == 0 {
		multiErr.Add("offers", errortypes.ErrRequired, "At least one offer is required")
	}

	for idx, offer := range t.Offers {
		if offer == nil {
			continue
		}
		offer.Validate(multiErr.WithIndex("offers", idx))
	}
}

func (t *Tender) GetID() pulid.ID {
	return t.ID
}

func (t *Tender) GetCreatedAt() int64 {
	return t.CreatedAt
}

func (t *Tender) GetTableName() string {
	return "tenders"
}

func (t *Tender) GetOrganizationID() pulid.ID {
	return t.OrganizationID
}

func (t *Tender) GetBusinessUnitID() pulid.ID {
	return t.BusinessUnitID
}

func (t *Tender) GetVersion() int64 {
	return t.Version
}

func (t *Tender) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("tnd_")
		}
		t.CreatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}

	return nil
}
