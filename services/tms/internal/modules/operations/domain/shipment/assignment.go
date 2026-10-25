package shipment

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// Equipment and driver assignments for a move
type Assignment struct {
	keel.Table `keel:"assignments,report=Operations"`

	ID                pulid.ID         `json:"id"                          keel:"id,pk,autoid:asn_"`
	BusinessUnitID    pulid.ID         `json:"businessUnitId"              keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID         `json:"organizationId"              keel:"organization_id,pk,tenant"`
	ShipmentMoveID    pulid.ID         `json:"shipmentMoveId"              keel:"shipment_move_id"`
	PrimaryWorkerID   *pulid.ID        `json:"primaryWorkerId"             keel:"primary_worker_id"`
	TractorID         *pulid.ID        `json:"tractorId"                   keel:"tractor_id"`
	TrailerID         *pulid.ID        `json:"trailerId,omitempty"         keel:"trailer_id"`
	SecondaryWorkerID *pulid.ID        `json:"secondaryWorkerId,omitempty" keel:"secondary_worker_id"`
	Status            AssignmentStatus `json:"status"                      keel:"status"`
	AckStatus         AssignmentAck    `json:"ackStatus"                   keel:"ack_status"`
	AckAt             *int64           `json:"ackAt,omitempty"             keel:"ack_at"`
	AckReason         string           `json:"ackReason,omitempty"         keel:"ack_reason,nullzero"`
	ArchivedAt        *int64           `json:"archivedAt,omitempty"        keel:"archived_at"`
	Version           int64            `json:"version"                     keel:"version,version"`
	CreatedAt         int64            `json:"createdAt"                   keel:"created_at,autocreate"`
	UpdatedAt         int64            `json:"updatedAt"                   keel:"updated_at,autoupdate"`

	ShipmentMove    *ShipmentMove    `json:"shipmentMove,omitempty"    keel:"belongs:shipment_move_id"`
	Tractor         *tractor.Tractor `json:"tractor,omitempty"         keel:"belongs:tractor_id"`
	Trailer         *trailer.Trailer `json:"trailer,omitempty"         keel:"belongs:trailer_id"`
	PrimaryWorker   *worker.Worker   `json:"primaryWorker,omitempty"   keel:"belongs:primary_worker_id"`
	SecondaryWorker *worker.Worker   `json:"secondaryWorker,omitempty" keel:"belongs:secondary_worker_id"`
}

func (a *Assignment) Validate(multiErr *errortypes.MultiError) {
	a.applyDefaults()

	var primaryWorkerID pulid.ID
	if a.PrimaryWorkerID != nil {
		primaryWorkerID = *a.PrimaryWorkerID
	}

	// Tenancy and the owning move are stamped by the repository at write time.
	multiErr.AddOzzoError(validation.ValidateStruct(
		a,
		validation.Field(
			&a.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[AssignmentStatus]("Status is invalid"),
		),
		validation.Field(
			&a.AckStatus,
			validation.Required.Error("Acknowledgment status is required"),
			domainvalidation.ValidEnum[AssignmentAck]("Acknowledgment status is invalid"),
		),
		// An assignment with no driver puts equipment on a move nobody is
		// scheduled to run.
		validation.Field(
			&a.PrimaryWorkerID,
			validation.Required.Error("Primary worker is required"),
		),
		// The same person twice on one move would show as a team assignment
		// while leaving the second seat empty. Read the primary into a local:
		// the rule is built before When decides whether to run it, so the
		// dereference cannot sit inside the condition.
		validation.Field(
			&a.SecondaryWorkerID,
			validation.When(
				a.PrimaryWorkerID != nil && a.SecondaryWorkerID != nil,
				validation.NotIn(primaryWorkerID).
					Error("The secondary worker cannot be the primary worker"),
			),
		),
		validation.Field(
			&a.AckAt,
			validation.When(
				a.AckStatus != AssignmentAckPending,
				validation.Required.Error(
					"An answered assignment must record when it was answered",
				),
			),
		),
		// A decline that records no reason leaves dispatch reassigning blind.
		validation.Field(
			&a.AckReason,
			validation.When(
				a.AckStatus == AssignmentAckDeclined,
				validation.Required.Error("A declined assignment must record its reason"),
			),
		),
	))
}

func (a *Assignment) applyDefaults() {
	if a.Status == "" {
		a.Status = AssignmentStatusNew
	}
	if a.AckStatus == "" {
		a.AckStatus = AssignmentAckPending
	}
}

func (a *Assignment) BeforeCreate(_ context.Context) error {
	if a.Status == "" {
		a.Status = AssignmentStatusNew
	}
	if a.AckStatus == "" {
		a.AckStatus = AssignmentAckPending
	}
	a.applyDefaults()

	return nil
}

func (a *Assignment) GetID() pulid.ID {
	return a.ID
}
