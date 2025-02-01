package shipment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

type ShipmentMove struct {
	bun.BaseModel `bun:"table:shipment_moves,alias:sm" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:",pk,type:VARCHAR(100),notnull" json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,pk,notnull,type:VARCHAR(100)" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,pk,notnull,type:VARCHAR(100)" json:"organizationId"`

	// Relationship identifiers (Non-Primary-Keys)
	ShipmentID        pulid.ID  `bun:"shipment_id,type:VARCHAR(100),notnull" json:"shipmentId"`
	PrimaryWorkerID   pulid.ID  `bun:"primary_worker_id,type:VARCHAR(100),notnull" json:"primaryWorkerId"`
	SecondaryWorkerID *pulid.ID `bun:"secondary_worker_id,type:VARCHAR(100),nullzero" json:"secondaryWorkerId"`
	TrailerID         pulid.ID  `bun:"trailer_id,type:VARCHAR(100),nullzero" json:"trailerId"`
	TractorID         pulid.ID  `bun:"tractor_id,type:VARCHAR(100),nullzero" json:"tractorId"`

	// Core Fields
	Status   StopStatus `json:"status" bun:"status,type:stop_status_enum,notnull,default:'New'"`
	Loaded   bool       `json:"loaded" bun:"loaded,type:BOOLEAN,notnull,default:true"`
	Sequence int        `json:"sequence" bun:"sequence,type:INTEGER,notnull,default:0"`
	Distance *float64   `json:"distance" bun:"distance,type:FLOAT,nullzero"`

	// Metadata
	Version   int64 `bun:"version,type:BIGINT" json:"version"`
	CreatedAt int64 `bun:"created_at,type:BIGINT,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" json:"createdAt"`
	UpdatedAt int64 `bun:"updated_at,type:BIGINT,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" json:"updatedAt"`

	// Relationships
	BusinessUnit    *businessunit.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization    *organization.Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	Shipment        *Shipment                  `bun:"rel:belongs-to,join:shipment_id=id" json:"shipment,omitempty"`
	Tractor         *tractor.Tractor           `bun:"rel:belongs-to,join:tractor_id=id" json:"tractor,omitempty"`
	Trailer         *trailer.Trailer           `bun:"rel:belongs-to,join:trailer_id=id" json:"trailer,omitempty"`
	PrimaryWorker   *worker.Worker             `bun:"rel:belongs-to,join:primary_worker_id=id" json:"primaryWorker,omitempty"`
	SecondaryWorker *worker.Worker             `bun:"rel:belongs-to,join:secondary_worker_id=id" json:"secondaryWorker,omitempty"`
	Stops           []*Stop                    `bun:"rel:has-many,join:id=shipment_move_id" json:"stops,omitempty"`
}

func (sm *ShipmentMove) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, sm,
		// Status is required and must be a valid stop status
		validation.Field(&sm.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				StopStatusNew,
				StopStatusInTransit,
				StopStatusCompleted,
				StopStatusCanceled,
			).Error("Status must be a valid stop status"),
		),

		// Tractor ID is requird
		validation.Field(&sm.TractorID,
			validation.Required.Error("Tractor is required"),
		),
		// Trailer ID is requird
		validation.Field(&sm.TrailerID,
			validation.Required.Error("Trailer is required"),
		),
		// Primary Worker ID is requird
		validation.Field(&sm.PrimaryWorkerID,
			validation.Required.Error("Primary Worker is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// Pagination Configuration
func (sm *ShipmentMove) GetID() string {
	return sm.ID.String()
}

func (sm *ShipmentMove) GetTableName() string {
	return "shipment_moves"
}

func (sm *ShipmentMove) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if sm.ID.IsNil() {
			sm.ID = pulid.MustNew("smv_")
		}

		sm.CreatedAt = now
	case *bun.UpdateQuery:
		sm.UpdatedAt = now
	}

	return nil
}
