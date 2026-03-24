package formulatemplate

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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

	ID                  pulid.ID                           `json:"id"                  bun:"id,pk,type:VARCHAR(100)"`
	TemplateID          pulid.ID                           `json:"templateId"          bun:"template_id,type:VARCHAR(100),notnull"`
	OrganizationID      pulid.ID                           `json:"organizationId"      bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID      pulid.ID                           `json:"businessUnitId"      bun:"business_unit_id,type:VARCHAR(100),notnull"`
	VersionNumber       int64                              `json:"versionNumber"       bun:"version_number,type:BIGINT,notnull"`
	Name                string                             `json:"name"                bun:"name,type:VARCHAR(100),notnull"`
	Description         string                             `json:"description"         bun:"description,type:TEXT"`
	Type                TemplateType                       `json:"type"                bun:"type,type:formula_template_type_enum,notnull"`
	Expression          string                             `json:"expression"          bun:"expression,type:TEXT,notnull"`
	Status              Status                             `json:"status"              bun:"status,type:formula_template_status_enum,notnull"`
	SchemaID            string                             `json:"schemaId"            bun:"schema_id,type:VARCHAR(100),notnull"`
	VariableDefinitions []*formulatypes.VariableDefinition `json:"variableDefinitions" bun:"variable_definitions,type:JSONB,notnull"`
	Metadata            map[string]any                     `json:"metadata"            bun:"metadata,type:JSONB"`
	ChangeMessage       string                             `json:"changeMessage"       bun:"change_message,type:TEXT"`
	ChangeSummary       map[string]jsonutils.FieldChange   `json:"changeSummary"       bun:"change_summary,type:JSONB"`
	Tags                []VersionTag                       `json:"tags"                bun:"tags,type:TEXT[],array"`
	CreatedByID         pulid.ID                           `json:"createdById"         bun:"created_by_id,type:VARCHAR(100),notnull"`
	CreatedAt           int64                              `json:"createdAt"           bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	CreatedBy *tenant.User `json:"createdBy,omitempty" bun:"rel:belongs-to,join:created_by_id=id"`
}

func (ftv *FormulaTemplateVersion) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if ftv.ID.IsNil() {
			ftv.ID = pulid.MustNew("ftv_")
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
		TemplateID:          ft.ID,
		OrganizationID:      ft.OrganizationID,
		BusinessUnitID:      ft.BusinessUnitID,
		VersionNumber:       versionNumber,
		Name:                ft.Name,
		Description:         ft.Description,
		Type:                ft.Type,
		Expression:          ft.Expression,
		Status:              ft.Status,
		SchemaID:            ft.SchemaID,
		VariableDefinitions: ft.VariableDefinitions,
		Metadata:            ft.Metadata,
		ChangeMessage:       changeMessage,
		ChangeSummary:       changeSummary,
		CreatedByID:         createdByID,
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
