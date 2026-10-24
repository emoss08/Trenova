package shipmenttype

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

type ShipmentType struct {
	bun.BaseModel `bun:"table:shipment_types,alias:sht" json:"-"`
	keel.Table    `keel:"shipment_types"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID           `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull" keel:"id,pk,autoid:sht_"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	Status         domaintypes.Status `json:"status"         bun:"status,type:status_enum,notnull,default:'Active'" keel:"status"`
	Code           string             `json:"code"           bun:"code,type:VARCHAR(10),notnull" keel:"code"`
	Description    string             `json:"description"    bun:"description,type:TEXT,nullzero" keel:"description,nullzero"`
	Color          string             `json:"color"          bun:"color,type:VARCHAR(10)" keel:"color,nullzero"`
	Version        int64              `json:"version"        bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt      int64              `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
	SearchVector   string             `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly" keel:"search_vector,tsvector,readonly,noselect"`
	Rank           string             `json:"-"              bun:"rank,type:VARCHAR(100),scanonly" keel:"rank,computed"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (sht *ShipmentType) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(sht,
		validation.Field(&sht.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 100).Error("Code must be between 1 and 100 characters"),
		),
		validation.Field(&sht.Color,
			is.HexColor.Error("Color must be a valid hex color. Please try again."),
		),
	))
}

func (sht *ShipmentType) GetID() pulid.ID {
	return sht.ID
}

func (sht *ShipmentType) GetCreatedAt() int64 {
	return sht.CreatedAt
}

func (sht *ShipmentType) GetTableName() string {
	return "shipment_types"
}

func (sht *ShipmentType) GetOrganizationID() pulid.ID {
	return sht.OrganizationID
}

func (sht *ShipmentType) GetBusinessUnitID() pulid.ID {
	return sht.BusinessUnitID
}

func (sht *ShipmentType) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "sht",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Weight: domaintypes.SearchWeightA, Type: domaintypes.FieldTypeText},
			{
				Name:   "description",
				Weight: domaintypes.SearchWeightB,
				Type:   domaintypes.FieldTypeText,
			},
		},
	}
}

func (sht *ShipmentType) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if sht.ID.IsNil() {
			sht.ID = pulid.MustNew("sht_")
		}

		sht.CreatedAt = now
	case *bun.UpdateQuery:
		sht.UpdatedAt = now
	}

	return nil
}

func (sht *ShipmentType) BeforeCreate(ctx context.Context) error {
	return sht.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}
