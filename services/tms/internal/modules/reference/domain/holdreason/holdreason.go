package holdreason

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

func HoldSeverityValues() []HoldSeverity {
	return []HoldSeverity{
		HoldSeverityInformational,
		HoldSeverityAdvisory,
		HoldSeverityBlocking,
	}
}

var (
	_ validationframework.TenantedEntity = (*HoldReason)(nil)
	_ domaintypes.PostgresSearchable     = (*HoldReason)(nil)
	_ pagination.CursorEntity            = (*HoldReason)(nil)
)

type HoldReason struct {
	keel.Table               `keel:"hold_reasons"`
	ID                       pulid.ID       `keel:"id,pk,autoid:hr_"                         json:"id"`
	BusinessUnitID           pulid.ID       `keel:"business_unit_id,tenant"                  json:"businessUnitId"`
	OrganizationID           pulid.ID       `keel:"organization_id,pk,tenant"                json:"organizationId"`
	Type                     HoldType       `keel:"type"                                     json:"type"`
	Code                     string         `keel:"code"                                     json:"code"`
	Label                    string         `keel:"label"                                    json:"label"`
	Description              string         `keel:"description,nullzero"                     json:"description"`
	Active                   bool           `keel:"active"                                   json:"active"`
	DefaultSeverity          HoldSeverity   `keel:"default_severity"                         json:"defaultSeverity"`
	DefaultBlocksDispatch    bool           `keel:"default_blocks_dispatch"                  json:"defaultBlocksDispatch"`
	DefaultBlocksDelivery    bool           `keel:"default_blocks_delivery"                  json:"defaultBlocksDelivery"`
	DefaultBlocksBilling     bool           `keel:"default_blocks_billing"                   json:"defaultBlocksBilling"`
	DefaultVisibleToCustomer bool           `keel:"default_visible_to_customer"              json:"defaultVisibleToCustomer"`
	SortOrder                int32          `keel:"sort_order"                               json:"sortOrder"`
	ExternalMap              map[string]any `keel:"external_map,jsonb,nullzero"              json:"externalMap"`
	Version                  int64          `keel:"version,version"                          json:"version"`
	CreatedAt                int64          `keel:"created_at,autocreate"                    json:"createdAt"`
	UpdatedAt                int64          `keel:"updated_at,autoupdate"                    json:"updatedAt"`
	SearchVector             string         `keel:"search_vector,tsvector,readonly,noselect" json:"-"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (hr *HoldReason) BeforeCreate(_ context.Context) error {
	if hr.SortOrder == 0 {
		hr.SortOrder = 100
	}

	return nil
}

func (hr *HoldReason) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(hr,
		validation.Field(
			&hr.Type,
			validation.Required.Error("Hold type is required"),
			validation.In(HoldTypeOperational, HoldTypeCompliance, HoldTypeCustomer, HoldTypeFinance).
				Error("Invalid hold type"),
		),
		validation.Field(&hr.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 64).Error("Code must be between 1 and 64 characters"),
		),
		validation.Field(&hr.Label,
			validation.Required.Error("Label is required"),
			validation.Length(1, 100).Error("Label must be between 1 and 100 characters"),
		),
		validation.Field(
			&hr.DefaultSeverity,
			validation.Required.Error("Default severity is required"),
			validation.In(HoldSeverityInformational, HoldSeverityAdvisory, HoldSeverityBlocking).
				Error("Invalid hold severity"),
		),
	))
}

func (hr *HoldReason) GetID() pulid.ID {
	return hr.ID
}

func (hr *HoldReason) GetCreatedAt() int64 {
	return hr.CreatedAt
}

func (hr *HoldReason) GetTableName() string {
	return "hold_reasons"
}

func (hr *HoldReason) GetOrganizationID() pulid.ID {
	return hr.OrganizationID
}

func (hr *HoldReason) GetBusinessUnitID() pulid.ID {
	return hr.BusinessUnitID
}

func (hr *HoldReason) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "hr",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "label", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "type", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
			{
				Name:   "default_severity",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (HoldReason) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "ux_hr_org_bu_type_code",
			Field:   "code",
			Message: "Hold reason with this code already exists in your organization",
		},
	}
}
