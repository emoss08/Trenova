package shipmenttype

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// Shipment classification
type ShipmentType struct {
	keel.Table `keel:"shipment_types,report=Configuration"`

	ID             pulid.ID           `json:"id"             keel:"id,pk,autoid:sht_"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	Status         domaintypes.Status `json:"status"         keel:"status"`
	Code           string             `json:"code"           keel:"code"`
	Description    string             `json:"description"    keel:"description,nullzero"`
	Color          string             `json:"color"          keel:"color,nullzero"`
	Version        int64              `json:"version"        keel:"version,version"`
	CreatedAt      int64              `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64              `json:"updatedAt"      keel:"updated_at,autoupdate"`
	SearchVector   string             `json:"-"              keel:"search_vector,tsvector,readonly,noselect"`
	Rank           string             `json:"-"              keel:"rank,computed"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization *tenant.Organization `json:"organization,omitempty"`
}

func (sht *ShipmentType) BeforeCreate(_ context.Context) error {
	if sht.Status == "" {
		sht.Status = domaintypes.StatusActive
	}

	return nil
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

func (ShipmentType) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "idx_shipment_types_code",
			Message: "Shipment type with this code already exists in your organization",
		},
	}
}
