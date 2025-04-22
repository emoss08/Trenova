package shipment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*Assignment)(nil)

type Assignment struct {
	bun.BaseModel `bun:"table:assignments,alias:a" json:"-"`

	// Primary identifiers
	// TODO(wolfred): We need to change the ID to a generated ID so it is searchable
	ID             pulid.ID `bun:",pk,type:VARCHAR(100),notnull" json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,pk,notnull,type:VARCHAR(100)" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,pk,notnull,type:VARCHAR(100)" json:"organizationId"`

	// Relationship identifiers (Non-Primary-Keys)
	ShipmentMoveID    pulid.ID  `bun:"shipment_move_id,type:VARCHAR(100),notnull" json:"shipmentMoveId"`
	PrimaryWorkerID   pulid.ID  `bun:"primary_worker_id,type:VARCHAR(100),notnull" json:"primaryWorkerId"`
	TrailerID         pulid.ID  `bun:"trailer_id,type:VARCHAR(100),nullzero" json:"trailerId"`
	TractorID         pulid.ID  `bun:"tractor_id,type:VARCHAR(100),nullzero" json:"tractorId"`
	SecondaryWorkerID *pulid.ID `bun:"secondary_worker_id,type:VARCHAR(100),nullzero" json:"secondaryWorkerId"`

	// Core Fields
	Status AssignmentStatus `json:"status" bun:"status,type:assignment_status_enum,notnull,default:'New'"`

	// Metadata
	Version   int64 `bun:"version,type:BIGINT" json:"version"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	Tractor         *tractor.Tractor `bun:"rel:belongs-to,join:tractor_id=id" json:"tractor,omitempty"`
	Trailer         *trailer.Trailer `bun:"rel:belongs-to,join:trailer_id=id" json:"trailer,omitempty"`
	PrimaryWorker   *worker.Worker   `bun:"rel:belongs-to,join:primary_worker_id=id" json:"primaryWorker,omitempty"`
	SecondaryWorker *worker.Worker   `bun:"rel:belongs-to,join:secondary_worker_id=id" json:"secondaryWorker,omitempty"`
	ShipmentMove    *ShipmentMove    `bun:"rel:belongs-to,join:shipment_move_id=id" json:"shipmentMove,omitempty"`
}

// Pagination Configuration
func (a *Assignment) GetID() string {
	return a.ID.String()
}

func (a *Assignment) GetTableName() string {
	return "assignments"
}

func (a *Assignment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("a_")
		}

		a.CreatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}

	return nil
}
