package equipmentmanufacturer

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*EquipmentManufacturer)(nil)
	_ domaintypes.PostgresSearchable     = (*EquipmentManufacturer)(nil)
)

// Equipment manufacturers
type EquipmentManufacturer struct {
	keel.Table `                                                      keel:"equipment_manufacturers,report=Equipment"`

	ID             pulid.ID           `json:"id" keel:"id,pk,autoid:em_"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	Status         domaintypes.Status `json:"status" keel:"status"`
	Name           string             `json:"name" keel:"name"`
	Description    string             `json:"description" keel:"description,nullzero"`
	SearchVector   string             `json:"-"`
	Rank           string             `json:"-" keel:"rank,computed"`
	Version        int64              `json:"version" keel:"version,version"`
	CreatedAt      int64              `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64              `json:"updatedAt" keel:"updated_at,autoupdate"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" keel:"belongs:business_unit_id,noreport"`
	Organization *tenant.Organization `json:"organization,omitempty" keel:"belongs:organization_id,noreport"`
}

func (em *EquipmentManufacturer) BeforeCreate(_ context.Context) error {
	if em.Status == "" {
		em.Status = domaintypes.StatusActive
	}

	return nil
}

func (em *EquipmentManufacturer) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		em,
		validation.Field(&em.Name, validation.Required),
		validation.Field(
			&em.Name,
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
	))
}

func (em *EquipmentManufacturer) GetID() pulid.ID {
	return em.ID
}

func (em *EquipmentManufacturer) GetOrganizationID() pulid.ID {
	return em.OrganizationID
}

func (em *EquipmentManufacturer) GetBusinessUnitID() pulid.ID {
	return em.BusinessUnitID
}

func (em *EquipmentManufacturer) GetCreatedAt() int64 {
	return em.CreatedAt
}

func (em *EquipmentManufacturer) GetTableName() string {
	return "equipment_manufacturers"
}

func (em *EquipmentManufacturer) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "em",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
		},
	}
}
