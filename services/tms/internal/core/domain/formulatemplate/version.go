package formulatemplate

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type VersionTag string

const (
	VersionTagStable     VersionTag = "Stable"
	VersionTagProduction VersionTag = "Production"
	VersionTagDraft      VersionTag = "Draft"
	VersionTagTesting    VersionTag = "Testing"
	VersionTagDeprecated VersionTag = "Deprecated"
)

func (vt VersionTag) String() string {
	return string(vt)
}

func (vt VersionTag) IsValid() bool {
	switch vt {
	case VersionTagStable,
		VersionTagProduction,
		VersionTagDraft,
		VersionTagTesting,
		VersionTagDeprecated:
		return true
	default:
		return false
	}
}

var _ bun.BeforeAppendModelHook = (*FormulaTemplateVersion)(nil)

type FormulaTemplateVersion struct {
	bun.BaseModel `bun:"table:formula_template_versions,alias:ftv" json:"-"`

	ID                   pulid.ID                            `json:"id"                   bun:"id,pk,type:VARCHAR(100)"`
	TemplateID           pulid.ID                            `json:"templateId"           bun:"template_id,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID                            `json:"organizationId"       bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID                            `json:"businessUnitId"       bun:"business_unit_id,type:VARCHAR(100),notnull"`
	VersionNumber        int64                               `json:"versionNumber"        bun:"version_number,type:BIGINT,notnull"`
	Name                 string                              `json:"name"                 bun:"name,type:VARCHAR(100),notnull"`
	Description          string                              `json:"description"          bun:"description,type:TEXT"`
	Type                 TemplateType                        `json:"type"                 bun:"type,type:formula_template_type_enum,notnull"`
	Expression           string                              `json:"expression"           bun:"expression,type:TEXT,notnull"`
	Status               Status                              `json:"status"               bun:"status,type:formula_template_status_enum,notnull"`
	SchemaID             string                              `json:"schemaId"             bun:"schema_id,type:VARCHAR(100),notnull"`
	VariableDefinitions  []*formulatypes.VariableDefinition  `json:"variableDefinitions"  bun:"variable_definitions,type:JSONB,notnull"`
	BreakdownDefinitions []*formulatypes.BreakdownDefinition `json:"breakdownDefinitions" bun:"breakdown_definitions,type:JSONB,notnull,default:'[]'"`
	MinCharge            decimal.NullDecimal                 `json:"minCharge"            bun:"min_charge,type:NUMERIC(19,4)"`
	MaxCharge            decimal.NullDecimal                 `json:"maxCharge"            bun:"max_charge,type:NUMERIC(19,4)"`
	RoundingMode         ratetypes.RoundingMode              `json:"roundingMode"         bun:"rounding_mode,type:rate_rounding_mode_enum,notnull,default:'HalfUp'"`
	RoundingPrecision    int32                               `json:"roundingPrecision"    bun:"rounding_precision,type:SMALLINT,notnull,default:2"`
	EffectiveFrom        *int64                              `json:"effectiveFrom"        bun:"effective_from,type:BIGINT"`
	Metadata             map[string]any                      `json:"metadata"             bun:"metadata,type:JSONB"`
	ChangeMessage        string                              `json:"changeMessage"        bun:"change_message,type:TEXT"`
	ChangeSummary        map[string]jsonutils.FieldChange    `json:"changeSummary"        bun:"change_summary,type:JSONB"`
	Tags                 []VersionTag                        `json:"tags"                 bun:"tags,type:TEXT[],array"`
	CreatedByID          pulid.ID                            `json:"createdById"          bun:"created_by_id,type:VARCHAR(100),notnull"`
	CreatedAt            int64                               `json:"createdAt"            bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	CreatedBy *tenant.User `json:"createdBy,omitempty" bun:"rel:belongs-to,join:created_by_id=id"`
}

func (ftv *FormulaTemplateVersion) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if ftv.ID.IsNil() {
			ftv.ID = pulid.MustNew("ftv_")
		}
		if ftv.RoundingMode == "" {
			ftv.RoundingMode = ratetypes.RoundingModeHalfUp
			ftv.RoundingPrecision = formulatypes.DefaultRoundingPrecision
		}
		ftv.CreatedAt = timeutils.NowUnix()
	}

	return nil
}

func (ftv *FormulaTemplateVersion) GetID() pulid.ID {
	return ftv.ID
}

func (ftv *FormulaTemplateVersion) GetOrganizationID() pulid.ID {
	return ftv.OrganizationID
}

func (ftv *FormulaTemplateVersion) GetBusinessUnitID() pulid.ID {
	return ftv.BusinessUnitID
}

func NewVersionFromTemplate(
	ft *FormulaTemplate,
	versionNumber int64,
	createdByID pulid.ID,
	changeMessage string,
	changeSummary map[string]jsonutils.FieldChange,
) *FormulaTemplateVersion {
	return &FormulaTemplateVersion{
		TemplateID:           ft.ID,
		OrganizationID:       ft.OrganizationID,
		BusinessUnitID:       ft.BusinessUnitID,
		VersionNumber:        versionNumber,
		Name:                 ft.Name,
		Description:          ft.Description,
		Type:                 ft.Type,
		Expression:           ft.Expression,
		Status:               ft.Status,
		SchemaID:             ft.SchemaID,
		VariableDefinitions:  ft.VariableDefinitions,
		BreakdownDefinitions: ft.BreakdownDefinitions,
		MinCharge:            ft.MinCharge,
		MaxCharge:            ft.MaxCharge,
		RoundingMode:         ft.RoundingMode,
		RoundingPrecision:    ft.RoundingPrecision,
		Metadata:             ft.Metadata,
		ChangeMessage:        changeMessage,
		ChangeSummary:        changeSummary,
		CreatedByID:          createdByID,
	}
}

func (ftv *FormulaTemplateVersion) GetTableName() string {
	return "formula_template_versions"
}

func (ftv *FormulaTemplateVersion) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "ftv",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "description", Type: domaintypes.FieldTypeText},
			{Name: "type", Type: domaintypes.FieldTypeEnum},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
		Relationships: []*domaintypes.RelationshipDefintion{
			{
				Field:        "Template",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*FormulaTemplate)(nil),
				TargetTable:  "formula_templates",
			},
		},
	}
}

func (v *FormulaTemplateVersion) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(v,
		validation.Field(&v.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&v.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&v.TemplateID, validation.Required.Error("Template is required")),
		validation.Field(&v.VersionNumber,
			validation.Required.Error("Version number is required"),
			validation.Min(int64(1)).Error("Version number must be at least one"),
		),
		validation.Field(&v.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxVersionNameLength).
				Error("Name cannot be longer than 100 characters"),
		),
		validation.Field(&v.Type,
			validation.Required.Error("Type is required"),
			domainvalidation.ValidEnum[TemplateType]("Type is invalid"),
		),
		// The expression is the whole point of the version; a blank one would
		// price every shipment it is applied to at nothing.
		validation.Field(&v.Expression, validation.Required.Error("Expression is required")),
		validation.Field(&v.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[Status]("Status is invalid"),
		),
		validation.Field(&v.SchemaID,
			validation.Required.Error("Schema is required"),
			validation.Length(1, maxVersionSchemaLength).
				Error("Schema cannot be longer than 100 characters"),
		),
		validation.Field(&v.CreatedByID, validation.Required.Error("Created by is required")),
		validation.Field(&v.Tags,
			validation.Each(domainvalidation.ValidEnum[VersionTag]("Tag is invalid")),
		),
	))

	// A ceiling below the floor makes every charge fail the bounds it is
	// clamped to, which surfaces as a priced shipment nobody can explain.
	if v.MinCharge.Valid && v.MaxCharge.Valid &&
		v.MaxCharge.Decimal.LessThan(v.MinCharge.Decimal) {
		multiErr.Add("maxCharge", errortypes.ErrInvalid,
			"Maximum charge cannot be less than the minimum charge")
	}
}

const (
	maxVersionNameLength   = 100
	maxVersionSchemaLength = 100
)
