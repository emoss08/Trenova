package models

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ReasonCodePermission string

const (
	// PermissionReasonCodeView is the permission to view reason code details
	PermissionReasonCodeView = ReasonCodePermission("reasoncode.view")

	// PermissionReasonCodeEdit is the permission to edit reason code details
	PermissionReasonCodeEdit = ReasonCodePermission("reasoncode.edit")

	// PermissionReasonCodeAdd is the permission to add a necw reason code
	PermissionReasonCodeAdd = ReasonCodePermission("reasoncode.add")

	// PermissionReasonCodeDelete is the permission to delete an reason code
	PermissionReasonCodeDelete = ReasonCodePermission("reasoncode.delete")
)

// String returns the string representation of the ReasonCodePermission
func (p ReasonCodePermission) String() string {
	return string(p)
}

type ReasonCode struct {
	bun.BaseModel  `bun:"table:reason_codes,alias:rc" json:"-"`
	CreatedAt      time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID             uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status         property.Status `bun:"status,type:status" json:"status"`
	Code           string          `bun:"type:VARCHAR(10),notnull" json:"code" queryField:"true"`
	CodeType       string          `bun:"type:VARCHAR(10)" json:"codeType"`
	Description    string          `bun:"type:TEXT" json:"description"`
	BusinessUnitID uuid.UUID       `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID       `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (c ReasonCode) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.Code, validation.Required, validation.Length(4, 4).Error("Code must be 4 characters")),
		validation.Field(&c.BusinessUnitID, validation.Required),
		validation.Field(&c.OrganizationID, validation.Required),
	)
}

var _ bun.BeforeAppendModelHook = (*ReasonCode)(nil)

func (c *ReasonCode) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
