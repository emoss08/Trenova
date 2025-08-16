package shipment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*HoldReason)(nil)
	_ infra.PostgresSearchable  = (*HoldReason)(nil)
)

type HoldReason struct {
	bun.BaseModel `bun:"table:hold_reasons,alias:hr" json:"-"`

	ID                       pulid.ID     `json:"id"                       bun:"id,type:VARCHAR(100),pk,notnull"`
	OrganizationID           pulid.ID     `json:"organizationId"           bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID           pulid.ID     `json:"businessUnitId"           bun:"business_unit_id,type:VARCHAR(100),notnull"`
	Type                     HoldType     `json:"type"                     bun:"type,type:hold_type_enum,notnull"`
	Code                     string       `json:"code"                     bun:"code,type:VARCHAR(64),notnull"`   // stable key (slug)
	Label                    string       `json:"label"                    bun:"label,type:VARCHAR(100),notnull"` // human name
	Description              string       `json:"description"              bun:"description,type:TEXT,nullzero"`
	DefaultSeverity          HoldSeverity `json:"defaultSeverity"          bun:"default_severity,type:hold_severity_enum,notnull,default:'Advisory'"`
	DefaultBlocksDispatch    bool         `json:"defaultBlocksDispatch"    bun:"default_blocks_dispatch,type:BOOLEAN,notnull,default:false"`
	DefaultBlocksDelivery    bool         `json:"defaultBlocksDelivery"    bun:"default_blocks_delivery,type:BOOLEAN,notnull,default:false"`
	DefaultBlocksBilling     bool         `json:"defaultBlocksBilling"     bun:"default_blocks_billing,type:BOOLEAN,notnull,default:false"`
	DefaultVisibleToCustomer bool         `json:"defaultVisibleToCustomer" bun:"default_visible_to_customer,type:BOOLEAN,notnull,default:false"`

	Active      bool           `json:"active"      bun:"active,type:BOOLEAN,notnull,default:true"`
	SortOrder   int32          `json:"sortOrder"   bun:"sort_order,type:INTEGER,notnull,default:100"`
	ExternalMap map[string]any `json:"externalMap" bun:"external_map,type:JSONB,nullzero"` // e.g., {"edi214":"AD","eld":"OOS"}
	Version     int64          `json:"version"     bun:"version,type:BIGINT"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (hr *HoldReason) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, hr,
		validation.Field(&hr.Type,
			validation.Required.Error("Hold Type is required"),
			validation.In(
				HoldOperational,
				HoldCompliance,
				HoldCustomer,
				HoldFinance,
			).Error("Hold Type must be a valid value"),
		),
		validation.Field(&hr.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 64).Error("Code must be between 1 and 64 characters"),
		),
		validation.Field(&hr.Label,
			validation.Required.Error("Label is required"),
			validation.Length(1, 100).Error("Label must be between 1 and 100 characters"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (hr *HoldReason) GetID() string {
	return hr.ID.String()
}

func (hr *HoldReason) GetTableName() string {
	return "hold_reasons"
}

func (hr *HoldReason) GetPostgresSearchConfig() infra.PostgresSearchConfig {
	return infra.PostgresSearchConfig{
		TableAlias: "hr",
		Fields: []infra.PostgresSearchableField{
			{
				Name:   "code",
				Weight: "A",
				Type:   infra.PostgresSearchTypeComposite,
			},
			{
				Name:   "label",
				Weight: "A",
				Type:   infra.PostgresSearchTypeComposite,
			},
			{
				Name:       "type",
				Weight:     "B",
				Type:       infra.PostgresSearchTypeEnum,
				Dictionary: "english",
			},
			{
				Name:       "default_severity",
				Weight:     "B",
				Type:       infra.PostgresSearchTypeEnum,
				Dictionary: "english",
			},
		},
		MinLength:       2,
		MaxTerms:        6,
		UsePartialMatch: true,
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
