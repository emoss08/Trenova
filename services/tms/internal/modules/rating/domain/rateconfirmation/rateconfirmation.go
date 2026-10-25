package rateconfirmation

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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

type RateConfirmation struct {
	keel.Table `keel:"rate_confirmations"`

	ID                  pulid.ID `json:"id"                  keel:"id,pk,autoid:ratecon_"`
	BusinessUnitID      pulid.ID `json:"businessUnitId"      keel:"business_unit_id,pk,tenant"`
	OrganizationID      pulid.ID `json:"organizationId"      keel:"organization_id,pk,tenant"`
	CarrierAssignmentID pulid.ID `json:"carrierAssignmentId" keel:"carrier_assignment_id"`
	CarrierID           pulid.ID `json:"carrierId"           keel:"carrier_id"`
	ShipmentID          pulid.ID `json:"shipmentId"          keel:"shipment_id"`
	ShipmentMoveID      pulid.ID `json:"shipmentMoveId"      keel:"shipment_move_id"`

	Revision        int64          `json:"revision"        keel:"revision"`
	Status          Status         `json:"status"          keel:"status"`
	DocumentID      *pulid.ID      `json:"documentId"      keel:"document_id"`
	SentAt          *int64         `json:"sentAt"          keel:"sent_at"`
	SentToEmails    string         `json:"sentToEmails"    keel:"sent_to_emails,nullzero"`
	ConfirmedAt     *int64         `json:"confirmedAt"     keel:"confirmed_at"`
	ConfirmedByName string         `json:"confirmedByName" keel:"confirmed_by_name,nullzero"`
	VoidedAt        *int64         `json:"voidedAt"        keel:"voided_at"`
	VoidReason      string         `json:"voidReason"      keel:"void_reason,nullzero"`
	PayloadSnapshot map[string]any `json:"payloadSnapshot" keel:"payload_snapshot,jsonb,nullzero"`
	GeneratedByID   *pulid.ID      `json:"generatedById"   keel:"generated_by_id"`

	GeneratedVia        Via       `json:"generatedVia"        keel:"generated_via"`
	SourceTenderOfferID *pulid.ID `json:"sourceTenderOfferId" keel:"source_tender_offer_id"`
	ConfirmedVia        Via       `json:"confirmedVia"        keel:"confirmed_via,nullzero"`
	ConfirmedByTitle    string    `json:"confirmedByTitle"    keel:"confirmed_by_title,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Carrier           *carrier.Carrier            `json:"carrier,omitempty"           keel:"belongs:carrier_id"`
	CarrierAssignment *shipment.CarrierAssignment `json:"carrierAssignment,omitempty" keel:"belongs:carrier_assignment_id"`
	Shipment          *shipment.Shipment          `json:"shipment,omitempty"          keel:"belongs:shipment_id"`
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

// Confirmation is who executed an agreement, how, and when.
type Confirmation struct {
	At    int64
	Name  string
	Title string
	Via   Via
}

// Confirm records the agreement as executed.
func (rc *RateConfirmation) Confirm(c Confirmation) {
	at := c.At
	rc.Status = StatusConfirmed
	rc.ConfirmedAt = &at
	rc.ConfirmedByName = c.Name
	rc.ConfirmedByTitle = c.Title
	rc.ConfirmedVia = c.Via
}

// Void retires the agreement, recording when and why.
func (rc *RateConfirmation) Void(at int64, reason string) {
	rc.Status = StatusVoided
	rc.VoidedAt = &at
	rc.VoidReason = reason
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

func (rc *RateConfirmation) BeforeCreate(ctx context.Context) error {
	if rc.Revision == 0 {
		rc.Revision = 1
	}
	if rc.Status == "" {
		rc.Status = StatusGenerated
	}
	if rc.GeneratedVia == "" {
		rc.GeneratedVia = ViaDispatcher
	}

	return nil
}
