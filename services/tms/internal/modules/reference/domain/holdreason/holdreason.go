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
	keel.Table               `                                           keel:"hold_reasons"`
	ID                       pulid.ID       `json:"id" keel:"id,pk,autoid:hr_"`
	BusinessUnitID           pulid.ID       `json:"businessUnitId" keel:"business_unit_id,tenant"`
	OrganizationID           pulid.ID       `json:"organizationId" keel:"organization_id,pk,tenant"`
	Type                     HoldType       `json:"type" keel:"type"`
	Code                     string         `json:"code" keel:"code"`
	Label                    string         `json:"label" keel:"label"`
	Description              string         `json:"description" keel:"description,nullzero"`
	Active                   bool           `json:"active" keel:"active"`
	DefaultSeverity          HoldSeverity   `json:"defaultSeverity" keel:"default_severity"`
	DefaultBlocksDispatch    bool           `json:"defaultBlocksDispatch" keel:"default_blocks_dispatch"`
	DefaultBlocksDelivery    bool           `json:"defaultBlocksDelivery" keel:"default_blocks_delivery"`
	DefaultBlocksBilling     bool           `json:"defaultBlocksBilling" keel:"default_blocks_billing"`
	DefaultVisibleToCustomer bool           `json:"defaultVisibleToCustomer" keel:"default_visible_to_customer"`
	SortOrder                int32          `json:"sortOrder" keel:"sort_order"`
	ExternalMap              map[string]any `json:"externalMap" keel:"external_map,jsonb,nullzero"`
	Version                  int64          `json:"version" keel:"version,version"`
	CreatedAt                int64          `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt                int64          `json:"updatedAt" keel:"updated_at,autoupdate"`
	SearchVector             string         `json:"-" keel:"search_vector,tsvector,readonly,noselect"`

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
