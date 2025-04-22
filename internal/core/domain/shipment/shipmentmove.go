package shipment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*ShipmentMove)(nil)

//nolint:revive // valid struct name
type ShipmentMove struct {
	bun.BaseModel `bun:"table:shipment_moves,alias:sm" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:",pk,type:VARCHAR(100),notnull" json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,pk,notnull,type:VARCHAR(100)" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,pk,notnull,type:VARCHAR(100)" json:"organizationId"`

	// Relationship identifiers (Non-Primary-Keys)
	ShipmentID pulid.ID `bun:"shipment_id,type:VARCHAR(100),notnull" json:"shipmentId"`

	// Core Fields
	Status   MoveStatus `json:"status" bun:"status,type:move_status_enum,notnull,default:'New'"`
	Loaded   bool       `json:"loaded" bun:"loaded,type:BOOLEAN,notnull,default:true"`
	Sequence int        `json:"sequence" bun:"sequence,type:INTEGER,notnull,default:0"`
	Distance *float64   `json:"distance" bun:"distance,type:FLOAT,nullzero"`

	// Metadata
	Version   int64 `bun:"version,type:BIGINT" json:"version"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *organization.Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	Shipment     *Shipment                  `bun:"rel:belongs-to,join:shipment_id=id" json:"shipment,omitempty"`
	Assignment   *Assignment                `bun:"rel:has-one,join:id=shipment_move_id" json:"assignment,omitempty"`
	Stops        []*Stop                    `bun:"rel:has-many,join:id=shipment_move_id" json:"stops,omitempty"`
}

func (sm *ShipmentMove) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, sm,
		// Status is required and must be a valid move status
		validation.Field(&sm.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				MoveStatusNew,
				MoveStatusAssigned,
				MoveStatusInTransit,
				MoveStatusCompleted,
				MoveStatusCanceled,
			).Error("Status must be a valid move status"),
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

	if _, ok := query.(*bun.InsertQuery); ok {
		if sm.ID.IsNil() {
			sm.ID = pulid.MustNew("smv_")
		}

		sm.CreatedAt = now
	}

	return nil
}
