package holdreason

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

type HoldType string

const (
	HoldTypeOperational HoldType = "OperationalHold"
	HoldTypeCompliance  HoldType = "ComplianceHold"
	HoldTypeCustomer    HoldType = "CustomerHold"
	HoldTypeFinance     HoldType = "FinanceHold"
)

type HoldSeverity string

const (
	HoldSeverityInformational HoldSeverity = "Informational"
	HoldSeverityAdvisory      HoldSeverity = "Advisory"
	HoldSeverityBlocking      HoldSeverity = "Blocking"
)

var (
	_ bun.BeforeAppendModelHook          = (*HoldReason)(nil)
	_ validationframework.TenantedEntity = (*HoldReason)(nil)
	_ domaintypes.PostgresSearchable     = (*HoldReason)(nil)
)

type HoldReason struct {
	bun.BaseModel `bun:"table:hold_reasons,alias:hr" json:"-"`

	ID                       pulid.ID       `json:"id"                       bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID           pulid.ID       `json:"businessUnitId"           bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID           pulid.ID       `json:"organizationId"           bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Type                     HoldType       `json:"type"                     bun:"type,type:hold_type_enum,notnull"`
	Code                     string         `json:"code"                     bun:"code,type:VARCHAR(64),notnull"`
	Label                    string         `json:"label"                    bun:"label,type:VARCHAR(100),notnull"`
	Description              string         `json:"description"              bun:"description,type:TEXT,nullzero"`
	Active                   bool           `json:"active"                   bun:"active,type:BOOLEAN,notnull,default:true"`
	DefaultSeverity          HoldSeverity   `json:"defaultSeverity"          bun:"default_severity,type:hold_severity_enum,notnull"`
	DefaultBlocksDispatch    bool           `json:"defaultBlocksDispatch"    bun:"default_blocks_dispatch,type:BOOLEAN,notnull,default:false"`
	DefaultBlocksDelivery    bool           `json:"defaultBlocksDelivery"    bun:"default_blocks_delivery,type:BOOLEAN,notnull,default:false"`
	DefaultBlocksBilling     bool           `json:"defaultBlocksBilling"     bun:"default_blocks_billing,type:BOOLEAN,notnull,default:false"`
	DefaultVisibleToCustomer bool           `json:"defaultVisibleToCustomer" bun:"default_visible_to_customer,type:BOOLEAN,notnull,default:false"`
	SortOrder                int32          `json:"sortOrder"                bun:"sort_order,type:INTEGER,notnull,default:100"`
	ExternalMap              map[string]any `json:"externalMap"              bun:"external_map,type:JSONB,nullzero"`
	Version                  int64          `json:"version"                  bun:"version,type:BIGINT"`
	CreatedAt                int64          `json:"createdAt"                bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                int64          `json:"updatedAt"                bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector             string         `json:"-"                        bun:"search_vector,type:TSVECTOR,scanonly"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"-" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-" bun:"rel:belongs-to,join:organization_id=id"`
}

func (hr *HoldReason) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(hr,
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
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (hr *HoldReason) GetID() pulid.ID {
	return hr.ID
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

func (hr *HoldReason) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if hr.ID.IsNil() {
			hr.ID = pulid.MustNew("hr_")
		}
		hr.CreatedAt = now
	case *bun.UpdateQuery:
		hr.UpdatedAt = now
	}

	return nil
}
