package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/pkg/gen"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type CustomerPermission string

const (
	// PermissionCustomerView is the permission to view customer details
	PermissionCustomerView = CustomerPermission("customer.view")

	// PermissionCustomerEdit is the permission to edit customer details
	PermissionCustomerEdit = CustomerPermission("customer.edit")

	// PermissionCustomerAdd is the permission to add a new customer
	PermissionCustomerAdd = CustomerPermission("customer.add")

	// PermissionCustomerDelete is the permission to delete an customer
	PermissionCustomerDelete = CustomerPermission("customer.delete")
)

// String returns the string representation of the CustomerPermission
func (p CustomerPermission) String() string {
	return string(p)
}

type Customer struct {
	bun.BaseModel `bun:"table:customers,alias:cu" json:"-"`

	ID                  uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status              property.Status `bun:"status,type:status" json:"status"`
	Code                string          `bun:"type:VARCHAR(10),notnull" json:"code" queryField:"true"`
	Name                string          `bun:"type:VARCHAR(150),notnull" json:"name"`
	AddressLine1        string          `bun:"address_line_1,type:VARCHAR(150),notnull" json:"addressLine1"`
	AddressLine2        string          `bun:"address_line_2,type:VARCHAR(150),notnull" json:"addressLine2"`
	City                string          `bun:"type:VARCHAR(150),notnull" json:"city"`
	AutoMarkReadyToBill bool            `bun:"type:boolean,notnull,default:false" json:"autoMarkReadyToBill"`
	HasCustomerPortal   bool            `bun:"type:boolean,notnull,default:false" json:"hasCustomerPortal"`
	PostalCode          string          `bun:"type:VARCHAR(10),notnull" json:"postalCode"`
	Version             int64           `bun:"type:BIGINT" json:"version"`
	CreatedAt           time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt           time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	StateID        uuid.UUID `bun:"type:uuid,notnull" json:"stateId"`
	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	State        *UsState      `bun:"rel:belongs-to,join:state_id=id" json:"state"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
}

func (c Customer) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.BusinessUnitID, validation.Required),
		validation.Field(&c.OrganizationID, validation.Required),
		validation.Field(&c.Code, validation.Required, validation.Length(1, 10)),
		validation.Field(&c.Name, validation.Required, validation.Length(1, 150)),
		validation.Field(&c.AddressLine1, validation.Required, validation.Length(0, 150)),
		validation.Field(&c.AddressLine2, validation.Length(0, 150)),
		validation.Field(&c.City, validation.Required, validation.Length(0, 150)),
		validation.Field(&c.PostalCode, validation.Required, validation.Length(0, 10)),
	)
}

func (c Customer) TableName() string {
	return "customers"
}

func (c Customer) GetCodePrefix(pattern string) string {
	switch pattern {
	case "NAME-COUNTER":
		return utils.TruncateString(strings.ToUpper(c.Name), 4)
	case "CITY-COUNTER":
		return utils.TruncateString(strings.ToUpper(c.City), 4)
	default:
		return utils.TruncateString(strings.ToUpper(c.Name), 4)
	}
}

func (c Customer) GenerateCode(pattern string, counter int) string {
	switch pattern {
	case "NAME-COUNTER":
		return fmt.Sprintf("%s%04d", utils.TruncateString(strings.ToUpper(c.Name), 4), counter)
	case "CITY-COUNTER":
		return fmt.Sprintf("%s%04d", utils.TruncateString(strings.ToUpper(c.City), 4), counter)
	default:
		return fmt.Sprintf("%s%04d", utils.TruncateString(strings.ToUpper(c.Name), 4), counter)
	}
}

func (c *Customer) BeforeUpdate(_ context.Context) error {
	c.Version++

	return nil
}

func (c *Customer) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
	ov := c.Version

	if err := c.BeforeUpdate(ctx); err != nil {
		return err
	}

	result, err := tx.NewUpdate().
		Model(c).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return &validator.BusinessLogicError{
			Message: fmt.Sprintf("Version mismatch. The Customer (ID: %s) has been updated by another user. Please refresh and try again.", c.ID),
		}
	}

	return nil
}

var _ bun.BeforeAppendModelHook = (*Customer)(nil)

func (c *Customer) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}

func (c *Customer) InsertCustomer(ctx context.Context, tx bun.Tx, codeGen *gen.CodeGenerator, pattern string) error {
	code, err := codeGen.GenerateUniqueCode(ctx, c, pattern, c.OrganizationID)
	if err != nil {
		return err
	}
	c.Code = code

	_, err = tx.NewInsert().Model(c).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
