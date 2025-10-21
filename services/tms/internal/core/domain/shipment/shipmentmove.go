package shipment

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*ShipmentMove)(nil)
	_ domain.Validatable        = (*ShipmentMove)(nil)
	_ framework.TenantedEntity  = (*ShipmentMove)(nil)
)

//nolint:revive // valid struct name
type ShipmentMove struct {
	bun.BaseModel `bun:"table:shipment_moves,alias:sm" json:"-"`

	ID             pulid.ID   `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID   `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID   `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	ShipmentID     pulid.ID   `json:"shipmentId"     bun:"shipment_id,type:VARCHAR(100),notnull"`
	Status         MoveStatus `json:"status"         bun:"status,type:move_status_enum,notnull,default:'New'"`
	Loaded         bool       `json:"loaded"         bun:"loaded,type:BOOLEAN,notnull,default:true"`
	Sequence       int        `json:"sequence"       bun:"sequence,type:INTEGER,notnull,default:0"`
	Distance       *float64   `json:"distance"       bun:"distance,type:FLOAT,nullzero"`
	Version        int64      `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64      `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64      `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
	Shipment     *Shipment            `bun:"rel:belongs-to,join:shipment_id=id"      json:"shipment,omitempty"`
	Assignment   *Assignment          `bun:"rel:has-one,join:id=shipment_move_id"    json:"assignment,omitempty"`
	Stops        []*Stop              `bun:"rel:has-many,join:id=shipment_move_id"   json:"stops,omitempty"`
}

func (sm *ShipmentMove) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(sm,
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
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (sm *ShipmentMove) GetID() string {
	return sm.ID.String()
}

func (sm *ShipmentMove) GetOrganizationID() pulid.ID {
	return sm.OrganizationID
}

func (sm *ShipmentMove) GetBusinessUnitID() pulid.ID {
	return sm.BusinessUnitID
}

func (sm *ShipmentMove) GetTableName() string {
	return "shipment_moves"
}

func (sm *ShipmentMove) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	if _, ok := query.(*bun.InsertQuery); ok {
		if sm.ID.IsNil() {
			sm.ID = pulid.MustNew("smv_")
		}

		sm.CreatedAt = now
	}

	return nil
}

func (sm *ShipmentMove) IsCompleted() bool {
	return sm.Status == MoveStatusCompleted
}

func (sm *ShipmentMove) IsInTransit() bool {
	return sm.Status == MoveStatusInTransit
}

func (sm *ShipmentMove) IsAssigned() bool {
	return sm.Status == MoveStatusAssigned
}

func (sm *ShipmentMove) IsNew() bool {
	return sm.Status == MoveStatusNew
}

func (sm *ShipmentMove) IsCanceled() bool {
	return sm.Status == MoveStatusCanceled
}
