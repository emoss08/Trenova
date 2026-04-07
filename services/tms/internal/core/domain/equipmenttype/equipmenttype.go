package equipmenttype

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*EquipmentType)(nil)
	_ validationframework.TenantedEntity = (*EquipmentType)(nil)
	_ domaintypes.PostgresSearchable     = (*EquipmentType)(nil)
)

type EquipmentType struct {
	bun.BaseModel `bun:"table:equipment_types,alias:et" json:"-"`

	ID             pulid.ID           `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	Status         domaintypes.Status `json:"status"         bun:"status,type:status_enum,notnull,default:'Active'"`
	Code           string             `json:"code"           bun:"code,type:VARCHAR(10),notnull"`
	Description    string             `json:"description"    bun:"description,type:TEXT,nullzero"`
	Class          Class              `json:"class"          bun:"class,type:equipment_class_enum,notnull"`
	Color          string             `json:"color"          bun:"color,type:VARCHAR(10),nullzero"`
	InteriorLength *float64           `json:"interiorLength" bun:"interior_length,type:NUMERIC(10,2),nullzero"`
	SearchVector   string             `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank           string             `json:"-"              bun:"rank,type:VARCHAR(100),scanonly"`
	Version        int64              `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64              `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (et *EquipmentType) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		et,
		validation.Field(&et.Code, validation.Required),
		validation.Field(
			&et.Code,
			validation.Length(1, 10).Error("Code must be between 1 and 10 characters"),
		),
		validation.Field(&et.Class, validation.Required, validation.By(func(value any) error {
			c, ok := value.(Class)
			if !ok {
				return errors.New("invalid class type")
			}
			if !c.IsValid() {
				return errors.New("Class must be one of: Tractor, Trailer, Container, Other")
			}
			return nil
		})),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (et *EquipmentType) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

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

func (et *EquipmentType) GetID() pulid.ID {
	return et.ID
}

func (et *EquipmentType) GetOrganizationID() pulid.ID {
	return et.OrganizationID
}

func (et *EquipmentType) GetBusinessUnitID() pulid.ID {
	return et.BusinessUnitID
}

func (et *EquipmentType) GetTableName() string {
	return "equipment_types"
}

func (et *EquipmentType) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "et",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText},
			{Name: "description", Type: domaintypes.FieldTypeText},
		},
	}
}
