package formulatemplate

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
	keel.Table    `keel:"formula_template_versions"`

	ID             pulid.ID                         `json:"id"             bun:"id,pk,type:VARCHAR(100)" keel:"id,pk,autoid:ftv_"`
	TemplateID     pulid.ID                         `json:"templateId"     bun:"template_id,type:VARCHAR(100),notnull" keel:"template_id"`
	OrganizationID pulid.ID                         `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull" keel:"organization_id,tenant"`
	BusinessUnitID pulid.ID                         `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull" keel:"business_unit_id,tenant"`
	VersionNumber  int64                            `json:"versionNumber"  bun:"version_number,type:BIGINT,notnull" keel:"version_number"`
	Name           string                           `json:"name"           bun:"name,type:VARCHAR(100),notnull" keel:"name"`
	Description    string                           `json:"description"    bun:"description,type:TEXT" keel:"description,nullzero"`
	Source         string                           `json:"source"         bun:"source,type:TEXT,notnull" keel:"source"`
	Inputs         []tarifftypes.InputDecl          `json:"inputs"         bun:"inputs,type:JSONB,notnull,default:'[]'" keel:"inputs,jsonb"`
	Imports        []tarifftypes.Import             `json:"imports"        bun:"imports,type:JSONB,notnull,default:'[]'" keel:"imports,jsonb"`
	Tables         []tarifftypes.TablePin           `json:"tables"         bun:"tables,type:JSONB,notnull,default:'[]'" keel:"tables,jsonb"`
	TestCount      int32                            `json:"testCount"      bun:"test_count,type:INTEGER,notnull,default:0" keel:"test_count"`
	Status         Status                           `json:"status"         bun:"status,type:formula_template_status_enum,notnull" keel:"status"`
	EffectiveFrom  *int64                           `json:"effectiveFrom"  bun:"effective_from,type:BIGINT" keel:"effective_from"`
	Metadata       map[string]any                   `json:"metadata"       bun:"metadata,type:JSONB" keel:"metadata,jsonb"`
	ChangeMessage  string                           `json:"changeMessage"  bun:"change_message,type:TEXT" keel:"change_message,nullzero"`
	ChangeSummary  map[string]jsonutils.FieldChange `json:"changeSummary"  bun:"change_summary,type:JSONB" keel:"change_summary,jsonb"`
	Tags           []VersionTag                     `json:"tags"           bun:"tags,type:TEXT[],array" keel:"tags"`
	CreatedByID    pulid.ID                         `json:"createdById"    bun:"created_by_id,type:VARCHAR(100),notnull" keel:"created_by_id"`
	CreatedAt      int64                            `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`

	CreatedBy *tenant.User `json:"createdBy,omitempty" bun:"rel:belongs-to,join:created_by_id=id" keel:"belongs:created_by_id"`
}

func (ftv *FormulaTemplateVersion) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if ftv.ID.IsNil() {
			ftv.ID = pulid.MustNew("ftv_")
		}
		if ftv.Inputs == nil {
			ftv.Inputs = []tarifftypes.InputDecl{}
		}
		if ftv.Imports == nil {
			ftv.Imports = []tarifftypes.Import{}
		}
		if ftv.Tables == nil {
			ftv.Tables = []tarifftypes.TablePin{}
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
		TemplateID:     ft.ID,
		OrganizationID: ft.OrganizationID,
		BusinessUnitID: ft.BusinessUnitID,
		VersionNumber:  versionNumber,
		Name:           ft.Name,
		Description:    ft.Description,
		Source:         ft.Source,
		Inputs:         ft.Inputs,
		Imports:        ft.Imports,
		Tables:         ft.Tables,
		TestCount:      ft.TestCount,
		Status:         ft.Status,
		Metadata:       ft.Metadata,
		ChangeMessage:  changeMessage,
		ChangeSummary:  changeSummary,
		CreatedByID:    createdByID,
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
		validation.Field(&v.Source, validation.Required.Error("Source is required")),
		validation.Field(&v.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[Status]("Status is invalid"),
		),
		validation.Field(&v.CreatedByID, validation.Required.Error("Created by is required")),
		validation.Field(&v.Tags,
			validation.Each(domainvalidation.ValidEnum[VersionTag]("Tag is invalid")),
		),
	))
}

const maxVersionNameLength = 100

func (v *FormulaTemplateVersion) EffectiveStart() int64 {
	if v.EffectiveFrom != nil {
		return *v.EffectiveFrom
	}
	return v.CreatedAt
}
