package tender

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Tender struct {
	keel.Table `keel:"tenders"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:tnd_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	ShipmentID     pulid.ID `json:"shipmentId" keel:"shipment_id"`
	ShipmentMoveID pulid.ID `json:"shipmentMoveId" keel:"shipment_move_id"`

	RoutingGuideID     *pulid.ID `json:"routingGuideId" keel:"routing_guide_id"`
	Mode               Mode      `json:"mode" keel:"mode"`
	Status             Status    `json:"status" keel:"status"`
	CurrentRank        int16     `json:"currentRank" keel:"current_rank"`
	WorkflowID         string    `json:"workflowId" keel:"workflow_id,nullzero"`
	CreatedByID        *pulid.ID `json:"createdById" keel:"created_by_id"`
	CanceledByID       *pulid.ID `json:"canceledById" keel:"canceled_by_id"`
	CancellationReason string    `json:"cancellationReason" keel:"cancellation_reason,nullzero"`
	AcceptedOfferID    *pulid.ID `json:"acceptedOfferId" keel:"accepted_offer_id"`
	AcceptedAt         *int64    `json:"acceptedAt" keel:"accepted_at"`
	ExhaustedAt        *int64    `json:"exhaustedAt" keel:"exhausted_at"`
	CanceledAt         *int64    `json:"canceledAt" keel:"canceled_at"`
	BounceCount        int16     `json:"bounceCount" keel:"bounce_count"`
	LastBouncedAt      *int64    `json:"lastBouncedAt" keel:"last_bounced_at"`
	EscalatedFromID    *pulid.ID `json:"escalatedFromId" keel:"escalated_from_tender_id"`
	EscalationRound    int16     `json:"escalationRound" keel:"escalation_round"`
	Version            int64     `json:"version" keel:"version,version"`
	CreatedAt          int64     `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt          int64     `json:"updatedAt" keel:"updated_at,autoupdate"`

	Shipment     *shipment.Shipment     `json:"shipment,omitempty" keel:"belongs:shipment_id"`
	ShipmentMove *shipment.ShipmentMove `json:"shipmentMove,omitempty" keel:"belongs:shipment_move_id"`
	RoutingGuide *RoutingGuide          `json:"routingGuide,omitempty" keel:"belongs:routing_guide_id"`
	Offers       []*TenderOffer         `json:"offers,omitempty" keel:"many:tender_id"`
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

func (t *Tender) BeforeCreate(_ context.Context) error {
	if t.Status == "" {
		t.Status = StatusActive
	}

	return nil
}
