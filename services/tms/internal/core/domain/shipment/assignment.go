package shipment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*Assignment)(nil)

type Assignment struct {
	bun.BaseModel `json:"-" bun:"table:assignments,alias:a"`

	ID                pulid.ID         `json:"id"                          bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID         `json:"businessUnitId"              bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID    pulid.ID         `json:"organizationId"              bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ShipmentMoveID    pulid.ID         `json:"shipmentMoveId"              bun:"shipment_move_id,type:VARCHAR(100),notnull"`
	PrimaryWorkerID   *pulid.ID        `json:"primaryWorkerId"             bun:"primary_worker_id,type:VARCHAR(100),nullzero"`
	TractorID         *pulid.ID        `json:"tractorId"                   bun:"tractor_id,type:VARCHAR(100),nullzero"`
	TrailerID         *pulid.ID        `json:"trailerId,omitempty"         bun:"trailer_id,type:VARCHAR(100),nullzero"`
	SecondaryWorkerID *pulid.ID        `json:"secondaryWorkerId,omitempty" bun:"secondary_worker_id,type:VARCHAR(100),nullzero"`
	Status            AssignmentStatus `json:"status"                      bun:"status,type:assignment_status_enum,notnull,default:'New'"`
	AckStatus         AssignmentAck    `json:"ackStatus"                   bun:"ack_status,type:VARCHAR(20),notnull,default:'Pending'"`
	AckAt             *int64           `json:"ackAt,omitempty"             bun:"ack_at,type:BIGINT,nullzero"`
	AckReason         string           `json:"ackReason,omitempty"         bun:"ack_reason,type:TEXT,nullzero"`
	ArchivedAt        *int64           `json:"archivedAt,omitempty"        bun:"archived_at,type:BIGINT,nullzero"`
	Version           int64            `json:"version"                     bun:"version,type:BIGINT"`
	CreatedAt         int64            `json:"createdAt"                   bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64            `json:"updatedAt"                   bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	ShipmentMove    *ShipmentMove    `json:"shipmentMove,omitempty"    bun:"rel:belongs-to,join:shipment_move_id=id"`
	Tractor         *tractor.Tractor `json:"tractor,omitempty"         bun:"rel:belongs-to,join:tractor_id=id"`
	Trailer         *trailer.Trailer `json:"trailer,omitempty"         bun:"rel:belongs-to,join:trailer_id=id"`
	PrimaryWorker   *worker.Worker   `json:"primaryWorker,omitempty"   bun:"rel:belongs-to,join:primary_worker_id=id"`
	SecondaryWorker *worker.Worker   `json:"secondaryWorker,omitempty" bun:"rel:belongs-to,join:secondary_worker_id=id"`
}

// applyDefaults mirrors the column defaults so validation judges the row the
// insert would actually write, rather than failing a payload that leaves the
// lifecycle fields to the database.
func (a *Assignment) applyDefaults() {
	if a.Status == "" {
		a.Status = AssignmentStatusNew
	}
	if a.AckStatus == "" {
		a.AckStatus = AssignmentAckPending
	}
}

func (a *Assignment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("asn_")
		}
		a.CreatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}

	return nil
}

func (a *Assignment) GetID() pulid.ID {
	return a.ID
}

func (a *Assignment) Validate(multiErr *errortypes.MultiError) {
	a.applyDefaults()

	var primaryWorkerID pulid.ID
	if a.PrimaryWorkerID != nil {
		primaryWorkerID = *a.PrimaryWorkerID
	}

	// Tenancy and the owning move are stamped by the repository at write time.
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[AssignmentStatus]("Status is invalid"),
		),
		validation.Field(&a.AckStatus,
			validation.Required.Error("Acknowledgment status is required"),
			domainvalidation.ValidEnum[AssignmentAck]("Acknowledgment status is invalid"),
		),
		// An assignment with no driver puts equipment on a move nobody is
		// scheduled to run.
		validation.Field(&a.PrimaryWorkerID,
			validation.Required.Error("Primary worker is required"),
		),
		// The same person twice on one move would show as a team assignment
		// while leaving the second seat empty. Read the primary into a local:
		// the rule is built before When decides whether to run it, so the
		// dereference cannot sit inside the condition.
		validation.Field(&a.SecondaryWorkerID,
			validation.When(
				a.PrimaryWorkerID != nil && a.SecondaryWorkerID != nil,
				validation.NotIn(primaryWorkerID).
					Error("The secondary worker cannot be the primary worker"),
			),
		),
		validation.Field(&a.AckAt,
			validation.When(
				a.AckStatus != AssignmentAckPending,
				validation.Required.Error(
					"An answered assignment must record when it was answered",
				),
			),
		),
		// A decline that records no reason leaves dispatch reassigning blind.
		validation.Field(&a.AckReason,
			validation.When(
				a.AckStatus == AssignmentAckDeclined,
				validation.Required.Error("A declined assignment must record its reason"),
			),
		),
	))
}
