package shipment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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
