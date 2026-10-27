package formulatemplate

import (
	"context"
	"unicode/utf8"

	"github.com/bytedance/sonic"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	MaxDescriptionLength = 2_000
	MaxMetadataBytes     = 16 * 1024
)

var (
	_ validationframework.TenantedEntity = (*FormulaTemplate)(nil)
	_ domaintypes.PostgresSearchable     = (*FormulaTemplate)(nil)
)

type BillingVersion struct {
	VersionNumber int64 `json:"versionNumber"`
	EffectiveFrom int64 `json:"effectiveFrom"`
	Pending       bool  `json:"pending,omitempty"`
}

type FormulaTemplate struct {
	keel.Table `keel:"formula_templates"`

	ID                   pulid.ID                `json:"id"                   keel:"id,pk,autoid:ft_"`
	OrganizationID       pulid.ID                `json:"organizationId"       keel:"organization_id,pk,tenant"`
	BusinessUnitID       pulid.ID                `json:"businessUnitId"       keel:"business_unit_id,pk,tenant"`
	Name                 string                  `json:"name"                 keel:"name"`
	Description          string                  `json:"description"          keel:"description,nullzero"`
	Source               string                  `json:"source"               keel:"source"`
	Inputs               []tarifftypes.InputDecl `json:"inputs"               keel:"inputs,jsonb"`
	Imports              []tarifftypes.Import    `json:"imports"              keel:"imports,jsonb"`
	Tables               []tarifftypes.TablePin  `json:"tables"               keel:"tables,jsonb"`
	TestCount            int32                   `json:"testCount"            keel:"test_count"`
	Status               Status                  `json:"status"               keel:"status"`
	SubmittedByID        *pulid.ID               `json:"submittedById"        keel:"submitted_by_id"`
	SubmittedAt          *int64                  `json:"submittedAt"          keel:"submitted_at"`
	ApprovedByID         *pulid.ID               `json:"approvedById"         keel:"approved_by_id"`
	ApprovedAt           *int64                  `json:"approvedAt"           keel:"approved_at"`
	ReviewComment        string                  `json:"reviewComment"        keel:"review_comment,nullzero"`
	SearchVector         string                  `json:"-"                    keel:"search_vector,tsvector,readonly,noselect"`
	Rank                 string                  `json:"-"                    keel:"rank,computed"`
	Metadata             map[string]any          `json:"metadata"             keel:"metadata,jsonb"`
	Version              int64                   `json:"version"              keel:"version,version"`
	SourceTemplateID     *pulid.ID               `json:"sourceTemplateId"     keel:"source_template_id"`
	SourceVersionNumber  *int64                  `json:"sourceVersionNumber"  keel:"source_version_number"`
	CurrentVersionNumber int64                   `json:"currentVersionNumber" keel:"current_version_number"`
	CreatedAt            int64                   `json:"createdAt"            keel:"created_at,autocreate"`
	UpdatedAt            int64                   `json:"updatedAt"            keel:"updated_at,autoupdate"`

	Billing *BillingVersion `json:"billing,omitempty"`

	Organization   *tenant.Organization      `json:"organization,omitempty"`
	BusinessUnit   *tenant.BusinessUnit      `json:"businessUnit,omitempty"`
	SourceTemplate *FormulaTemplate          `json:"sourceTemplate,omitempty" keel:"belongs:source_template_id"`
	Versions       []*FormulaTemplateVersion `json:"versions,omitempty"       keel:"many:template_id"`
}

func (ft *FormulaTemplate) BeforeUpdate(_ context.Context) error {
	ft.normalizeCollections()

	return nil
}

func (ft *FormulaTemplate) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(ft,
		validation.Field(&ft.Name, validation.Required, validation.Length(1, 100)),
		validation.Field(&ft.Source, validation.Required.Error("Source is required")),
		validation.Field(&ft.Status, validation.Required, validation.In(
			StatusActive,
			StatusInactive,
			StatusDraft,
			StatusInReview,
		)),
	))

	ValidateContentSizes(multiErr, ft.Source, ft.Description, ft.Metadata)
}

func ValidateContentSizes(
	multiErr *errortypes.MultiError,
	source, description string,
	metadata map[string]any,
) {
	if len(source) > tarifftypes.MaxSourceLength {
		multiErr.Add("source", errortypes.ErrInvalid, "Source cannot exceed 64 KiB")
	}

	if utf8.RuneCountInString(description) > MaxDescriptionLength {
		multiErr.Add(
			"description",
			errortypes.ErrInvalid,
			"Description cannot exceed 2,000 characters",
		)
	}

	if len(metadata) == 0 {
		return
	}
	encoded, err := sonic.Marshal(metadata)
	if err != nil {
		multiErr.Add("metadata", errortypes.ErrInvalid, "Metadata must be plain JSON values")
		return
	}
	if len(encoded) > MaxMetadataBytes {
		multiErr.Add("metadata", errortypes.ErrInvalid, "Metadata cannot exceed 16 KiB")
	}
}

func (ft *FormulaTemplate) ApplyVersion(version *FormulaTemplateVersion) *FormulaTemplate {
	resolved := *ft
	resolved.Source = version.Source
	resolved.Inputs = version.Inputs
	resolved.Imports = version.Imports
	resolved.Tables = version.Tables
	resolved.TestCount = version.TestCount
	resolved.CurrentVersionNumber = version.VersionNumber
	return &resolved
}

func (ft *FormulaTemplate) ApplyVersionFull(version *FormulaTemplateVersion) *FormulaTemplate {
	resolved := ft.ApplyVersion(version)
	resolved.Name = version.Name
	resolved.Description = version.Description
	resolved.Metadata = version.Metadata
	return resolved
}

func (ft *FormulaTemplate) HasMaterialChange(other *FormulaTemplate) bool {
	if other == nil {
		return true
	}

	return ft.Source != other.Source ||
		!tarifftypes.ImportsEqual(ft.Imports, other.Imports) ||
		!tarifftypes.TablePinsEqual(ft.Tables, other.Tables)
}

func (ft *FormulaTemplate) BeforeCreate(ctx context.Context) error {
	if ft.Status == "" {
		ft.Status = StatusDraft
	}
	if ft.CurrentVersionNumber == 0 {
		ft.CurrentVersionNumber = 1
	}
	ft.normalizeCollections()

	return nil
}

func (ft *FormulaTemplate) BeforeSave(_ context.Context) error {
	ft.normalizeCollections()
	return nil
}

func (ft *FormulaTemplate) normalizeCollections() {
	if ft.Inputs == nil {
		ft.Inputs = []tarifftypes.InputDecl{}
	}
	if ft.Imports == nil {
		ft.Imports = []tarifftypes.Import{}
	}
	if ft.Tables == nil {
		ft.Tables = []tarifftypes.TablePin{}
	}
}

func (ft *FormulaTemplate) GetID() pulid.ID {
	return ft.ID
}

func (ft *FormulaTemplate) GetCreatedAt() int64 {
	return ft.CreatedAt
}

func (ft *FormulaTemplate) GetOrganizationID() pulid.ID {
	return ft.OrganizationID
}

func (ft *FormulaTemplate) GetBusinessUnitID() pulid.ID {
	return ft.BusinessUnitID
}

func (ft *FormulaTemplate) GetTableName() string {
	return "formula_templates"
}

func (ft *FormulaTemplate) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "ft",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "description", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}
}
