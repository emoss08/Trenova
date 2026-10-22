package formulatemplate

import (
	"context"
	"unicode/utf8"

	"github.com/bytedance/sonic"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	MaxDescriptionLength = 2_000
	MaxMetadataBytes     = 16 * 1024
)

var (
	_ bun.BeforeAppendModelHook          = (*FormulaTemplate)(nil)
	_ validationframework.TenantedEntity = (*FormulaTemplate)(nil)
	_ domaintypes.PostgresSearchable     = (*FormulaTemplate)(nil)
)

type BillingVersion struct {
	VersionNumber int64 `json:"versionNumber"`
	EffectiveFrom int64 `json:"effectiveFrom"`
	Pending       bool  `json:"pending,omitempty"`
}

type FormulaTemplate struct {
	bun.BaseModel `bun:"table:formula_templates,alias:ft" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID                   pulid.ID                `json:"id"                   bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID       pulid.ID                `json:"organizationId"       bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID       pulid.ID                `json:"businessUnitId"       bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	Name                 string                  `json:"name"                 bun:"name,type:VARCHAR(100),notnull"`
	Description          string                  `json:"description"          bun:"description,type:TEXT"`
	Source               string                  `json:"source"               bun:"source,type:TEXT,notnull"`
	Inputs               []tarifftypes.InputDecl `json:"inputs"               bun:"inputs,type:JSONB,notnull,default:'[]'"`
	Imports              []tarifftypes.Import    `json:"imports"              bun:"imports,type:JSONB,notnull,default:'[]'"`
	TestCount            int32                   `json:"testCount"            bun:"test_count,type:INTEGER,notnull,default:0"`
	Status               Status                  `json:"status"               bun:"status,type:formula_template_status_enum,notnull,default:'Draft'"`
	SubmittedByID        *pulid.ID               `json:"submittedById"        bun:"submitted_by_id,type:VARCHAR(100)"`
	SubmittedAt          *int64                  `json:"submittedAt"          bun:"submitted_at,type:BIGINT"`
	ApprovedByID         *pulid.ID               `json:"approvedById"         bun:"approved_by_id,type:VARCHAR(100)"`
	ApprovedAt           *int64                  `json:"approvedAt"           bun:"approved_at,type:BIGINT"`
	ReviewComment        string                  `json:"reviewComment"        bun:"review_comment,type:TEXT"`
	SearchVector         string                  `json:"-"                    bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                 string                  `json:"-"                    bun:"rank,type:VARCHAR(100),scanonly"`
	Metadata             map[string]any          `json:"metadata"             bun:"metadata,type:JSONB"`
	Version              int64                   `json:"version"              bun:"version,type:BIGINT"`
	SourceTemplateID     *pulid.ID               `json:"sourceTemplateId"     bun:"source_template_id,type:VARCHAR(100)"`
	SourceVersionNumber  *int64                  `json:"sourceVersionNumber"  bun:"source_version_number,type:BIGINT"`
	CurrentVersionNumber int64                   `json:"currentVersionNumber" bun:"current_version_number,type:BIGINT,notnull,default:1"`
	CreatedAt            int64                   `json:"createdAt"            bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64                   `json:"updatedAt"            bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Billing *BillingVersion `json:"billing,omitempty" bun:"-"`

	Organization   *tenant.Organization      `json:"organization,omitempty"   bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit   *tenant.BusinessUnit      `json:"businessUnit,omitempty"   bun:"rel:belongs-to,join:business_unit_id=id"`
	SourceTemplate *FormulaTemplate          `json:"sourceTemplate,omitempty" bun:"rel:belongs-to,join:source_template_id=id"`
	Versions       []*FormulaTemplateVersion `json:"versions,omitempty"       bun:"rel:has-many,join:id=template_id"`
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

	return ft.Source != other.Source || !tarifftypes.ImportsEqual(ft.Imports, other.Imports)
}

func (ft *FormulaTemplate) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if ft.ID.IsNil() {
			ft.ID = pulid.MustNew("ft_")
		}
		ft.normalizeCollections()
		ft.CreatedAt = now
	case *bun.UpdateQuery:
		ft.normalizeCollections()
		ft.UpdatedAt = now
	}

	return nil
}

func (ft *FormulaTemplate) normalizeCollections() {
	if ft.Inputs == nil {
		ft.Inputs = []tarifftypes.InputDecl{}
	}
	if ft.Imports == nil {
		ft.Imports = []tarifftypes.Import{}
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
