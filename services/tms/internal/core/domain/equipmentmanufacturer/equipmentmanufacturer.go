package equipmentmanufacturer

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*EquipmentManufacturer)(nil)
	_ domaintypes.PostgresSearchable = (*EquipmentManufacturer)(nil)
	_ domain.Validatable             = (*EquipmentManufacturer)(nil)
	_ framework.TenantedEntity       = (*EquipmentManufacturer)(nil)
)

type EquipmentManufacturer struct {
	bun.BaseModel `bun:"table:equipment_manufacturers,alias:em" json:"-"`

	ID             pulid.ID      `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID      `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	OrganizationID pulid.ID      `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	Status         domain.Status `json:"status"         bun:"status,type:status_enum,notnull,default:'Active'"`
	Name           string        `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
	Description    string        `json:"description"    bun:"description,type:TEXT,nullzero"`
	SearchVector   string        `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank           string        `json:"-"              bun:"rank,type:VARCHAR(100),scanonly"`
	Version        int64         `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64         `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64         `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (em *EquipmentManufacturer) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(em,
		validation.Field(&em.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (em *EquipmentManufacturer) GetID() string {
	return em.ID.String()
}

func (em *EquipmentManufacturer) GetTableName() string {
	return "equipment_manufacturers"
}

func (em *EquipmentManufacturer) GetOrganizationID() pulid.ID {
	return em.OrganizationID
}

func (em *EquipmentManufacturer) GetBusinessUnitID() pulid.ID {
	return em.BusinessUnitID
}

func (em *EquipmentManufacturer) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "em",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (em *EquipmentManufacturer) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

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
