package models

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type DelayCodePermission string

const (
	// PermissionDelayCodeView is the permission to view delay code details
	PermissionDelayCodeView = DelayCodePermission("delaycode.view")

	// PermissionDelayCodeEdit is the permission to edit delay code details
	PermissionDelayCodeEdit = DelayCodePermission("delaycode.edit")

	// PermissionDelayCodeAdd is the permission to add a new delay code
	PermissionDelayCodeAdd = DelayCodePermission("delaycode.add")

	// PermissionDelayCodeDelete is the permission to delete an delay code
	PermissionDelayCodeDelete = DelayCodePermission("delaycode.delete")
)

// String returns the string representation of the DelayCodePermission
func (p DelayCodePermission) String() string {
	return string(p)
}

type DelayCode struct {
	bun.BaseModel    `bun:"table:delay_codes,alias:dc" json:"-"`
	CreatedAt        time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt        time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID               uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status           property.Status `bun:"status,type:status" json:"status"`
	Code             string          `bun:"type:VARCHAR(10),notnull" json:"code" queryField:"true"`
	Description      string          `bun:"type:TEXT,notnull" json:"description"`
	FCarrierOrDriver bool            `bun:"type:BOOLEAN,default:false" json:"fCarrierOrDriver"`
	Color            string          `bun:"type:VARCHAR(10)" json:"color"`
	BusinessUnitID   uuid.UUID       `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID   uuid.UUID       `bun:"type:uuid,notnull" json:"organizationId"`

	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
}

func (d DelayCode) Validate() error {
	return validation.ValidateStruct(
		&d,
		validation.Field(&d.Code, validation.Required),
		validation.Field(&d.Color, is.HexColor),
		validation.Field(&d.BusinessUnitID, validation.Required),
		validation.Field(&d.OrganizationID, validation.Required),
	)
}

var _ bun.BeforeAppendModelHook = (*DelayCode)(nil)

func (d *DelayCode) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		d.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		d.UpdatedAt = time.Now()
	}
	return nil
}
