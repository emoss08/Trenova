package equipmentmanufacturer

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*EquipmentManufacturer)(nil)
	_ domain.Validatable        = (*EquipmentManufacturer)(nil)
)

type EquipmentManufacturer struct {
	bun.BaseModel `bun:"table:equipment_manufacturers,alias:em" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:"id,type:VARCHAR(100),pk,notnull" json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,type:VARCHAR(100),notnull,pk" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,type:VARCHAR(100),notnull,pk" json:"organizationId"`

	// Core Fields
	Status      domain.Status `json:"status" bun:"status,type:status_enum,notnull,default:'Active'"`
	Name        string        `json:"name" bun:"name,type:VARCHAR(100),notnull"`
	Description string        `json:"description" bun:"description,type:VARCHAR(255)"`

	// Metadata
	Version   int64 `json:"version" bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (em *EquipmentManufacturer) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, em,
		// Name is required and must be between 1 and 100 characters
		validation.Field(&em.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromValidationErrors(validationErrs, multiErr, "")
		}
	}
}

// Pagination Configuration
func (em *EquipmentManufacturer) GetID() string {
	return em.ID.String()
}

func (em *EquipmentManufacturer) GetTableName() string {
	return "equipment_manufacturers"
}

func (em *EquipmentManufacturer) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if em.ID.IsNil() {
			em.ID = pulid.MustNew("em_")
		}

		em.CreatedAt = now
	case *bun.UpdateQuery:
		em.UpdatedAt = now
	}

	return nil
}
