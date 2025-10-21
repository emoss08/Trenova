package equipmenttype

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
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*EquipmentType)(nil)
	_ domaintypes.PostgresSearchable = (*EquipmentType)(nil)
	_ domain.Validatable             = (*EquipmentType)(nil)
	_ framework.TenantedEntity       = (*EquipmentType)(nil)
)

type EquipmentType struct {
	bun.BaseModel `bun:"table:equipment_types,alias:et" json:"-"`

	ID             pulid.ID      `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID      `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	OrganizationID pulid.ID      `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	Status         domain.Status `json:"status"         bun:"status,type:status_enum,notnull,default:'Active'"`
	Code           string        `json:"code"           bun:"code,type:VARCHAR(10),notnull"`
	Description    string        `json:"description"    bun:"description,type:TEXT,nullzero"`
	Class          Class         `json:"class"          bun:"class,type:equipment_class_enum,notnull"`
	Color          string        `json:"color"          bun:"color,type:VARCHAR(10),nullzero"`
	Version        int64         `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64         `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64         `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector   string        `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank           string        `json:"-"              bun:"rank,type:VARCHAR(100),scanonly"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (et *EquipmentType) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(et,
		validation.Field(&et.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 100).Error("Code must be between 1 and 100 characters"),
		),
		validation.Field(
			&et.Class,
			validation.Required.Error("Class is required"),
			validation.In(ClassTractor, ClassTrailer, ClassContainer, ClassOther).
				Error("Class must be a valid class"),
		),
		validation.Field(&et.Color,
			is.HexColor.Error("Color must be a valid hex color. Please try again."),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (et *EquipmentType) GetID() string {
	return et.ID.String()
}

func (et *EquipmentType) GetTableName() string {
	return "equipment_types"
}

func (et *EquipmentType) GetOrganizationID() pulid.ID {
	return et.OrganizationID
}

func (et *EquipmentType) GetBusinessUnitID() pulid.ID {
	return et.BusinessUnitID
}

func (et *EquipmentType) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "et",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{Name: "class", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightB},
		},
	}
}

func (et *EquipmentType) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if et.ID.IsNil() {
			et.ID = pulid.MustNew("et_")
		}

		et.CreatedAt = now
	case *bun.UpdateQuery:
		et.UpdatedAt = now
	}

	return nil
}
