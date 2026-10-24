package formulatemplate

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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

type FormulaTemplateVersion struct {
	keel.Table `                                                         keel:"formula_template_versions"`

	ID             pulid.ID                         `json:"id" keel:"id,pk,autoid:ftv_"`
	TemplateID     pulid.ID                         `json:"templateId" keel:"template_id"`
	OrganizationID pulid.ID                         `json:"organizationId" keel:"organization_id,tenant"`
	BusinessUnitID pulid.ID                         `json:"businessUnitId" keel:"business_unit_id,tenant"`
	VersionNumber  int64                            `json:"versionNumber" keel:"version_number"`
	Name           string                           `json:"name" keel:"name"`
	Description    string                           `json:"description" keel:"description,nullzero"`
	Source         string                           `json:"source" keel:"source"`
	Inputs         []tarifftypes.InputDecl          `json:"inputs" keel:"inputs,jsonb"`
	Imports        []tarifftypes.Import             `json:"imports" keel:"imports,jsonb"`
	Tables         []tarifftypes.TablePin           `json:"tables" keel:"tables,jsonb"`
	TestCount      int32                            `json:"testCount" keel:"test_count"`
	Status         Status                           `json:"status" keel:"status"`
	EffectiveFrom  *int64                           `json:"effectiveFrom" keel:"effective_from"`
	Metadata       map[string]any                   `json:"metadata" keel:"metadata,jsonb"`
	ChangeMessage  string                           `json:"changeMessage" keel:"change_message,nullzero"`
	ChangeSummary  map[string]jsonutils.FieldChange `json:"changeSummary" keel:"change_summary,jsonb"`
	Tags           []VersionTag                     `json:"tags" keel:"tags"`
	CreatedByID    pulid.ID                         `json:"createdById" keel:"created_by_id"`
	CreatedAt      int64                            `json:"createdAt" keel:"created_at,autocreate"`

	CreatedBy *tenant.User `json:"createdBy,omitempty" keel:"belongs:created_by_id"`
}

func (ftv *FormulaTemplateVersion) BeforeCreate(_ context.Context) error {
	ftv.normalizeCollections()
	ftv.normalizeCollections()

	return nil
}

func (ftv *FormulaTemplateVersion) normalizeCollections() {
	if ftv.Inputs == nil {
		ftv.Inputs = []tarifftypes.InputDecl{}
	}
	if ftv.Imports == nil {
		ftv.Imports = []tarifftypes.Import{}
	}
	if ftv.Tables == nil {
		ftv.Tables = []tarifftypes.TablePin{}
	}
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
