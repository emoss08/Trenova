package rateconfirmation

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type Status string

const (
	StatusGenerated = Status("Generated")
	StatusSent      = Status("Sent")
	StatusConfirmed = Status("Confirmed")
	StatusVoided    = Status("Voided")
)

func (s Status) String() string {
	return string(s)
}

func (s Status) IsValid() bool {
	switch s {
	case StatusGenerated, StatusSent, StatusConfirmed, StatusVoided:
		return true
	}
	return false
}

func (s Status) IsActive() bool {
	return s != StatusVoided
}

type Via string

const (
	ViaDispatcher       = Via("Dispatcher")
	ViaTenderAcceptance = Via("TenderAcceptance")
	ViaPublicSignature  = Via("PublicSignature")
)

func (v Via) String() string {
	return string(v)
}

func (v Via) IsValid() bool {
	switch v {
	case ViaDispatcher, ViaTenderAcceptance, ViaPublicSignature:
		return true
	}
	return false
}

var _ bun.BeforeAppendModelHook = (*RateConfirmation)(nil)

type RateConfirmation struct {
	bun.BaseModel `bun:"table:rate_confirmations,alias:ratecon" json:"-"`

	ID                  pulid.ID `json:"id"                  bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID      pulid.ID `json:"businessUnitId"      bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID      pulid.ID `json:"organizationId"      bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	CarrierAssignmentID pulid.ID `json:"carrierAssignmentId" bun:"carrier_assignment_id,type:VARCHAR(100),notnull"`
	CarrierID           pulid.ID `json:"carrierId"           bun:"carrier_id,type:VARCHAR(100),notnull"`
	ShipmentID          pulid.ID `json:"shipmentId"          bun:"shipment_id,type:VARCHAR(100),notnull"`
	ShipmentMoveID      pulid.ID `json:"shipmentMoveId"      bun:"shipment_move_id,type:VARCHAR(100),notnull"`

	Revision        int64          `json:"revision"        bun:"revision,type:INTEGER,notnull,default:1"`
	Status          Status         `json:"status"          bun:"status,type:VARCHAR(50),notnull,default:'Generated'"`
	DocumentID      *pulid.ID      `json:"documentId"      bun:"document_id,type:VARCHAR(100),nullzero"`
	SentAt          *int64         `json:"sentAt"          bun:"sent_at,type:BIGINT,nullzero"`
	SentToEmails    string         `json:"sentToEmails"    bun:"sent_to_emails,type:TEXT,nullzero"`
	ConfirmedAt     *int64         `json:"confirmedAt"     bun:"confirmed_at,type:BIGINT,nullzero"`
	ConfirmedByName string         `json:"confirmedByName" bun:"confirmed_by_name,type:VARCHAR(255),nullzero"`
	VoidedAt        *int64         `json:"voidedAt"        bun:"voided_at,type:BIGINT,nullzero"`
	VoidReason      string         `json:"voidReason"      bun:"void_reason,type:TEXT,nullzero"`
	PayloadSnapshot map[string]any `json:"payloadSnapshot" bun:"payload_snapshot,type:JSONB,nullzero"`
	GeneratedByID   *pulid.ID      `json:"generatedById"   bun:"generated_by_id,type:VARCHAR(100),nullzero"`

	GeneratedVia        Via       `json:"generatedVia"        bun:"generated_via,type:VARCHAR(50),notnull,default:'Dispatcher'"`
	SourceTenderOfferID *pulid.ID `json:"sourceTenderOfferId" bun:"source_tender_offer_id,type:VARCHAR(100),nullzero"`
	ConfirmedVia        Via       `json:"confirmedVia"        bun:"confirmed_via,type:VARCHAR(50),nullzero"`
	ConfirmedByTitle    string    `json:"confirmedByTitle"    bun:"confirmed_by_title,type:VARCHAR(255),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Carrier           *carrier.Carrier            `json:"carrier,omitempty"           bun:"rel:belongs-to,join:carrier_id=id"`
	CarrierAssignment *shipment.CarrierAssignment `json:"carrierAssignment,omitempty" bun:"rel:belongs-to,join:carrier_assignment_id=id"`
	Shipment          *shipment.Shipment          `json:"shipment,omitempty"          bun:"rel:belongs-to,join:shipment_id=id"`
}

func (rc *RateConfirmation) applyDefaults() {
	if rc.Status == "" {
		rc.Status = StatusGenerated
	}
	if rc.Revision == 0 {
		rc.Revision = 1
	}
	if rc.GeneratedVia == "" {
		rc.GeneratedVia = ViaDispatcher
	}
}

func (rc *RateConfirmation) IsActive() bool {
	return rc != nil && rc.Status.IsActive()
}

// CanSend includes Confirmed so the executed record copy can still be emailed;
// the post-send status write preserves Confirmed rather than demoting it.
func (rc *RateConfirmation) CanSend() bool {
	return rc.Status == StatusGenerated || rc.Status == StatusSent ||
		rc.Status == StatusConfirmed
}

func (rc *RateConfirmation) CanConfirm() bool {
	return rc.Status == StatusGenerated || rc.Status == StatusSent
}

func (rc *RateConfirmation) Validate(multiErr *errortypes.MultiError) {
	rc.applyDefaults()

	multiErr.AddOzzoError(validation.ValidateStruct(rc,
		validation.Field(&rc.CarrierAssignmentID,
			validation.Required.Error("Carrier assignment is required"),
		),
		validation.Field(&rc.CarrierID,
			validation.Required.Error("Carrier is required"),
		),
		validation.Field(&rc.ShipmentID,
			validation.Required.Error("Shipment is required"),
		),
		validation.Field(&rc.ShipmentMoveID,
			validation.Required.Error("Shipment move is required"),
		),
		validation.Field(&rc.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[Status]("Status is invalid"),
		),
		validation.Field(&rc.Revision,
			validation.Min(int64(1)).Error("Revision must be at least 1"),
		),
		validation.Field(&rc.VoidReason,
			validation.When(
				rc.Status == StatusVoided,
				validation.Required.Error("A voided rate confirmation must record its reason"),
			),
		),
		validation.Field(&rc.GeneratedVia,
			validation.Required.Error("Generated via is required"),
			domainvalidation.ValidEnum[Via]("Generated via is invalid"),
		),
		validation.Field(&rc.ConfirmedVia,
			validation.When(
				rc.ConfirmedVia != "",
				domainvalidation.ValidEnum[Via]("Confirmed via is invalid"),
			),
		),
	))
}

func (rc *RateConfirmation) GetID() pulid.ID {
	return rc.ID
}

func (rc *RateConfirmation) GetCreatedAt() int64 {
	return rc.CreatedAt
}

func (rc *RateConfirmation) GetTableName() string {
	return "rate_confirmations"
}

func (rc *RateConfirmation) GetOrganizationID() pulid.ID {
	return rc.OrganizationID
}

func (rc *RateConfirmation) GetBusinessUnitID() pulid.ID {
	return rc.BusinessUnitID
}

func (rc *RateConfirmation) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if rc.ID.IsNil() {
			rc.ID = pulid.MustNew("ratecon_")
		}
		rc.CreatedAt = now
	case *bun.UpdateQuery:
		rc.UpdatedAt = now
	}

	return nil
}
