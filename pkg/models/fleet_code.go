package models

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"

	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type FleetCodePermission string

const (
	// PermissionFleetCodeView is the permission to view fleet code details
	PermissionFleetCodeView = FleetCodePermission("fleetcode.view")

	// PermissionFleetCodeEdit is the permission to edit fleet code details
	PermissionFleetCodeEdit = FleetCodePermission("fleetcode.edit")

	// PermissionFleetCodeAdd is the permission to add a new fleet code
	PermissionFleetCodeAdd = FleetCodePermission("fleetcode.add")

	// PermissionFleetCodeDelete is the permission to delete an fleet code
	PermissionFleetCodeDelete = FleetCodePermission("fleetcode.delete")
)

// String returns the string representation of the FleetCodePermission
func (p FleetCodePermission) String() string {
	return string(p)
}

type FleetCode struct {
	bun.BaseModel  `bun:"table:fleet_codes,alias:fl" json:"-"`
	CreatedAt      time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID             uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status         property.Status `bun:"status,type:status" json:"status"`
	Code           string          `bun:"type:VARCHAR(10),notnull" json:"code" queryField:"true"`
	Description    string          `bun:"type:VARCHAR(100)" json:"description"`
	RevenueGoal    decimal.Decimal `bun:"type:numeric(10,2),nullzero" json:"revenueGoal"`
	DeadheadGoal   decimal.Decimal `bun:"type:numeric(10,2),nullzero" json:"deadheadGoal"`
	MileageGoal    decimal.Decimal `bun:"type:numeric(10,2),nullzero" json:"mileageGoal"`
	Color          string          `bun:"type:VARCHAR(10)" json:"color"`
	ManagerID      *uuid.UUID      `bun:"type:uuid" json:"managerId"`
	Manager        *User           `bun:"rel:belongs-to,join:manager_id=id" json:"manager"`
	BusinessUnitID uuid.UUID       `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID       `bun:"type:uuid,notnull" json:"organizationId"`

	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
}

func (f FleetCode) Validate() error {
	return validation.ValidateStruct(
		&f,
		validation.Field(&f.Code, validation.Required),
		validation.Field(&f.Color, is.HexColor),
		validation.Field(&f.BusinessUnitID, validation.Required),
		validation.Field(&f.OrganizationID, validation.Required),
	)
}

var _ bun.BeforeAppendModelHook = (*FleetCode)(nil)

func (f *FleetCode) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		f.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		f.UpdatedAt = time.Now()
	}
	return nil
}
