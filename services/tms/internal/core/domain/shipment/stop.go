/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package shipment

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*Stop)(nil)

type Stop struct {
	bun.BaseModel `bun:"table:stops,alias:stp" json:"-"`

	ID               pulid.ID   `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID   pulid.ID   `json:"businessUnitId"   bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID   pulid.ID   `json:"organizationId"   bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	ShipmentMoveID   pulid.ID   `json:"shipmentMoveId"   bun:"shipment_move_id,notnull,type:VARCHAR(100)"`
	LocationID       pulid.ID   `json:"locationId"       bun:"location_id,notnull,type:VARCHAR(100)"`
	Status           StopStatus `json:"status"           bun:"status,notnull,type:VARCHAR(100)"`
	Type             StopType   `json:"type"             bun:"type,notnull,type:VARCHAR(100)"`
	AddressLine      string     `json:"addressLine"      bun:"address_line,type:VARCHAR(200),nullzero"`
	Sequence         int        `json:"sequence"         bun:"sequence,notnull,type:INT"`
	Pieces           *int       `json:"pieces"           bun:"pieces,type:INTEGER,nullzero"`
	Weight           *int       `json:"weight"           bun:"weight,type:INTEGER,nullzero"`
	ActualArrival    *int64     `json:"actualArrival"    bun:"actual_arrival,type:BIGINT,nullzero"`
	ActualDeparture  *int64     `json:"actualDeparture"  bun:"actual_departure,type:BIGINT,nullzero"`
	PlannedArrival   int64      `json:"plannedArrival"   bun:"planned_arrival,type:BIGINT "`
	PlannedDeparture int64      `json:"plannedDeparture" bun:"planned_departure,type:BIGINT"`
	Version          int64      `json:"version"          bun:"version,type:BIGINT"`
	CreatedAt        int64      `json:"createdAt"        bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64      `json:"updatedAt"        bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	ShipmentMove *ShipmentMove              `json:"shipmentMove,omitempty" bun:"rel:belongs-to,join:shipment_move_id=id"`
	Location     *location.Location         `json:"location,omitempty"     bun:"rel:belongs-to,join:location_id=id"`
}

func (s *Stop) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, s,
		// Type is required and must be a valid stop type
		validation.Field(&s.Type,
			validation.Required.Error("Type is required"),
			validation.In(
				StopTypePickup,
				StopTypeDelivery,
				StopTypeSplitPickup,
				StopTypeSplitDelivery,
			).Error("Type must be a valid stop type"),
		),

		// Status is required and must be a valid stop status
		validation.Field(&s.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				StopStatusNew,
				StopStatusInTransit,
				StopStatusCompleted,
				StopStatusCanceled,
			).Error("Status must be a valid stop status"),
		),

		// Planned arrival is required
		validation.Field(&s.PlannedArrival,
			validation.Required.Error("Planned arrival is required"),
		),

		// Planned departure is required
		validation.Field(&s.PlannedDeparture,
			validation.Required.Error("Planned departure is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (s *Stop) GetID() string {
	return s.ID.String()
}

func (s *Stop) GetTableName() string {
	return "stops"
}

func (s *Stop) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := time.Now().Unix()

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID == "" {
			s.ID = pulid.MustNew("stp_")
		}

		s.CreatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}

	return nil
}

func (s *Stop) IsOriginStop() bool {
	return s.Type == StopTypePickup || s.Type == StopTypeSplitPickup
}

func (s *Stop) StatusEquals(status StopStatus) bool {
	return s.Status == status
}

func (s *Stop) IsDestinationStop() bool {
	return s.Type == StopTypeDelivery || s.Type == StopTypeSplitDelivery
}
