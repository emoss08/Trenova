package models

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/uptrace/bun"
)

type TableChangeAlertPermission string

const (
	// PermissionTableChangeAlertView is the permission to view table change alert details
	PermissionTableChangeAlertView = TableChangeAlertPermission("tablechangealert.view")

	// PermissionTableChangeAlertEdit is the permission to edit table change alert details
	PermissionTableChangeAlertEdit = TableChangeAlertPermission("tablechangealert.edit")

	// PermissionTableChangeAlertAdd is the permission to add a new table change alert
	PermissionTableChangeAlertAdd = TableChangeAlertPermission("tablechangealert.add")

	// PermissionTableChangeAlertDelete is the permission to delete an table change alert
	PermissionTableChangeAlertDelete = TableChangeAlertPermission("tablechangealert.delete")
)

// String returns the string representation of the TableChangeAlertPermission
func (p TableChangeAlertPermission) String() string {
	return string(p)
}

type TableChangeAlert struct {
	bun.BaseModel   `bun:"table:table_change_alerts,alias:tca" json:"-"`
	CreatedAt       time.Time               `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt       time.Time               `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID              uuid.UUID               `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status          property.Status         `bun:"status,type:status" json:"status"`
	Name            string                  `bun:"type:VARCHAR(50),notnull" json:"name" queryField:"true"`
	DatabaseAction  property.DatabaseAction `bun:"type:database_action_enum,notnull" json:"databaseAction"`
	TopicName       string                  `bun:"type:VARCHAR(200),notnull" json:"topicName"`
	Description     string                  `bun:"type:TEXT" json:"description"`
	CustomSubject   string                  `bun:"type:VARCHAR" json:"customSubject"`
	DeliveryMethod  property.DeliveryMethod `bun:"type:delivery_method_enum,notnull" json:"deliveryMethod"`
	EmailRecipients string                  `bun:"type:TEXT" json:"emailRecipients"`
	EffectiveDate   *pgtype.Date            `bun:"type:date" json:"effectiveDate"`
	ExpirationDate  *pgtype.Date            `bun:"type:date" json:"expirationDate"`
	BusinessUnitID  uuid.UUID               `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID  uuid.UUID               `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (f TableChangeAlert) Validate() error {
	return validation.ValidateStruct(
		&f,
		validation.Field(&f.Name, validation.Required),
		validation.Field(&f.DatabaseAction, validation.Required),
		validation.Field(&f.TopicName, validation.Required),
		validation.Field(&f.DeliveryMethod, validation.Required),
		validation.Field(&f.BusinessUnitID, validation.Required),
		validation.Field(&f.OrganizationID, validation.Required),
	)
}

var _ bun.BeforeAppendModelHook = (*TableChangeAlert)(nil)

func (f *TableChangeAlert) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		f.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		f.UpdatedAt = time.Now()
	}
	return nil
}
