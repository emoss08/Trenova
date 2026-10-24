package tender

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
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
	keel.Table    `keel:"tenders"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull" keel:"id,pk,autoid:tnd_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	ShipmentID     pulid.ID `json:"shipmentId"     bun:"shipment_id,type:VARCHAR(100),notnull" keel:"shipment_id"`
	ShipmentMoveID pulid.ID `json:"shipmentMoveId" bun:"shipment_move_id,type:VARCHAR(100),notnull" keel:"shipment_move_id"`

	RoutingGuideID     *pulid.ID `json:"routingGuideId"     bun:"routing_guide_id,type:VARCHAR(100),nullzero" keel:"routing_guide_id"`
	Mode               Mode      `json:"mode"               bun:"mode,type:VARCHAR(50),notnull" keel:"mode"`
	Status             Status    `json:"status"             bun:"status,type:VARCHAR(50),notnull,default:'Active'" keel:"status"`
	CurrentRank        int16     `json:"currentRank"        bun:"current_rank,type:SMALLINT,notnull,default:0" keel:"current_rank"`
	WorkflowID         string    `json:"workflowId"         bun:"workflow_id,type:VARCHAR(255),nullzero" keel:"workflow_id,nullzero"`
	CreatedByID        *pulid.ID `json:"createdById"        bun:"created_by_id,type:VARCHAR(100),nullzero" keel:"created_by_id"`
	CanceledByID       *pulid.ID `json:"canceledById"       bun:"canceled_by_id,type:VARCHAR(100),nullzero" keel:"canceled_by_id"`
	CancellationReason string    `json:"cancellationReason" bun:"cancellation_reason,type:TEXT,nullzero" keel:"cancellation_reason,nullzero"`
	AcceptedOfferID    *pulid.ID `json:"acceptedOfferId"    bun:"accepted_offer_id,type:VARCHAR(100),nullzero" keel:"accepted_offer_id"`
	AcceptedAt         *int64    `json:"acceptedAt"         bun:"accepted_at,type:BIGINT,nullzero" keel:"accepted_at"`
	ExhaustedAt        *int64    `json:"exhaustedAt"        bun:"exhausted_at,type:BIGINT,nullzero" keel:"exhausted_at"`
	CanceledAt         *int64    `json:"canceledAt"         bun:"canceled_at,type:BIGINT,nullzero" keel:"canceled_at"`
	BounceCount        int16     `json:"bounceCount"        bun:"bounce_count,type:SMALLINT,notnull,default:0" keel:"bounce_count"`
	LastBouncedAt      *int64    `json:"lastBouncedAt"      bun:"last_bounced_at,type:BIGINT,nullzero" keel:"last_bounced_at"`
	EscalatedFromID    *pulid.ID `json:"escalatedFromId"    bun:"escalated_from_tender_id,type:VARCHAR(100),nullzero" keel:"escalated_from_tender_id"`
	EscalationRound    int16     `json:"escalationRound"    bun:"escalation_round,type:SMALLINT,notnull,default:0" keel:"escalation_round"`
	Version            int64     `json:"version"            bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt          int64     `json:"createdAt"          bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt          int64     `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Shipment     *shipment.Shipment     `json:"shipment,omitempty"     bun:"rel:belongs-to,join:shipment_id=id" keel:"belongs:shipment_id"`
	ShipmentMove *shipment.ShipmentMove `json:"shipmentMove,omitempty" bun:"rel:belongs-to,join:shipment_move_id=id" keel:"belongs:shipment_move_id"`
	RoutingGuide *RoutingGuide          `json:"routingGuide,omitempty" bun:"rel:belongs-to,join:routing_guide_id=id" keel:"belongs:routing_guide_id"`
	Offers       []*TenderOffer         `json:"offers,omitempty"       bun:"rel:has-many,join:id=tender_id" keel:"many:tender_id"`
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

func (t *Tender) BeforeCreate(_ context.Context) error {
	if t.Status == "" {
		t.Status = StatusActive
	}

	return nil
}
