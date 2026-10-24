package formulalibrary

import (
	"context"
	"regexp"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
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
	PathPrefix    = "org."
	MaxNameLength = 64
)

var (
	_ bun.BeforeAppendModelHook          = (*FormulaLibrary)(nil)
	_ validationframework.TenantedEntity = (*FormulaLibrary)(nil)
	_ domaintypes.PostgresSearchable     = (*FormulaLibrary)(nil)
)

var namePattern = regexp.MustCompile(`^[a-z][A-Za-z0-9]*$`)

var keywords = []string{
	"tariff", "library", "condition", "use", "input", "let", "var", "const", "fn", "type",
	"enum", "private", "charge", "note", "if", "else", "match", "for", "in", "where", "break",
	"continue", "return", "minimum", "maximum", "round", "to", "up", "down", "half", "even",
	"require", "fail", "fails", "test", "given", "expect", "and", "or", "not", "true", "false",
	"none",
}

type FormulaLibrary struct {
	bun.BaseModel `bun:"table:formula_libraries,alias:flib" json:"-"`
	keel.Table    `keel:"formula_libraries"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID                   pulid.ID               `json:"id"                   bun:"id,pk,type:VARCHAR(100)" keel:"id,pk,autoid:fl_"`
	OrganizationID       pulid.ID               `json:"organizationId"       bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID       pulid.ID               `json:"businessUnitId"       bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	Name                 string                 `json:"name"                 bun:"name,type:VARCHAR(64),notnull" keel:"name"`
	Description          string                 `json:"description"          bun:"description,type:TEXT" keel:"description,nullzero"`
	Source               string                 `json:"source"               bun:"source,type:TEXT,notnull" keel:"source"`
	Imports              []tarifftypes.Import   `json:"imports"              bun:"imports,type:JSONB,notnull,default:'[]'" keel:"imports,jsonb"`
	TestCount            int32                  `json:"testCount"            bun:"test_count,type:INTEGER,notnull,default:0" keel:"test_count"`
	Status               formulatemplate.Status `json:"status"               bun:"status,type:formula_template_status_enum,notnull,default:'Draft'" keel:"status"`
	SubmittedByID        *pulid.ID              `json:"submittedById"        bun:"submitted_by_id,type:VARCHAR(100)" keel:"submitted_by_id"`
	SubmittedAt          *int64                 `json:"submittedAt"          bun:"submitted_at,type:BIGINT" keel:"submitted_at"`
	ApprovedByID         *pulid.ID              `json:"approvedById"         bun:"approved_by_id,type:VARCHAR(100)" keel:"approved_by_id"`
	ApprovedAt           *int64                 `json:"approvedAt"           bun:"approved_at,type:BIGINT" keel:"approved_at"`
	ReviewComment        string                 `json:"reviewComment"        bun:"review_comment,type:TEXT" keel:"review_comment,nullzero"`
	SearchVector         string                 `json:"-"                    bun:"search_vector,type:TSVECTOR,scanonly" keel:"search_vector,tsvector,readonly,noselect"`
	Rank                 string                 `json:"-"                    bun:"rank,type:VARCHAR(100),scanonly" keel:"rank,computed"`
	Version              int64                  `json:"version"              bun:"version,type:BIGINT" keel:"version,version"`
	SourceLibraryID      *pulid.ID              `json:"sourceLibraryId"      bun:"source_library_id,type:VARCHAR(100)" keel:"source_library_id"`
	SourceVersionNumber  *int64                 `json:"sourceVersionNumber"  bun:"source_version_number,type:BIGINT" keel:"source_version_number"`
	CurrentVersionNumber int64                  `json:"currentVersionNumber" bun:"current_version_number,type:BIGINT,notnull,default:1" keel:"current_version_number"`
	CreatedAt            int64                  `json:"createdAt"            bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt            int64                  `json:"updatedAt"            bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
}

func Path(name string) string {
	return PathPrefix + name
}

func (fl *FormulaLibrary) Path() string {
	return Path(fl.Name)
}

func (fl *FormulaLibrary) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(fl,
		validation.Field(&fl.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, MaxNameLength).Error("Name cannot be longer than 64 characters"),
			validation.Match(namePattern).Error(
				"Name must be one camelCase word, such as accessorials or zoneRates, so it can be imported as org.<name>",
			),
		),
		validation.Field(&fl.Source, validation.Required.Error("Source is required")),
		validation.Field(&fl.Status, validation.Required, validation.In(
			formulatemplate.StatusActive,
			formulatemplate.StatusInactive,
			formulatemplate.StatusDraft,
			formulatemplate.StatusInReview,
		)),
	))

	if slices.Contains(keywords, fl.Name) {
		multiErr.Add(
			"name",
			errortypes.ErrInvalid,
			"{0} is a keyword of the formula language and cannot name a library",
			fl.Name,
		)
	}

	formulatemplate.ValidateContentSizes(multiErr, fl.Source, fl.Description, nil)
}

func (fl *FormulaLibrary) ApplyVersion(version *FormulaLibraryVersion) *FormulaLibrary {
	resolved := *fl
	resolved.Source = version.Source
	resolved.Imports = version.Imports
	resolved.TestCount = version.TestCount
	resolved.Description = version.Description
	resolved.CurrentVersionNumber = version.VersionNumber
	return &resolved
}

func (fl *FormulaLibrary) HasMaterialChange(other *FormulaLibrary) bool {
	if other == nil {
		return true
	}

	return fl.Source != other.Source ||
		fl.Name != other.Name ||
		!tarifftypes.ImportsEqual(fl.Imports, other.Imports)
}

func (fl *FormulaLibrary) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if fl.ID.IsNil() {
			fl.ID = pulid.MustNew("fl_")
		}
		if fl.Imports == nil {
			fl.Imports = []tarifftypes.Import{}
		}
		fl.CreatedAt = now
	case *bun.UpdateQuery:
		if fl.Imports == nil {
			fl.Imports = []tarifftypes.Import{}
		}
		fl.UpdatedAt = now
	}

	return nil
}

func (fl *FormulaLibrary) BeforeCreate(ctx context.Context) error {
	return fl.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}

func (fl *FormulaLibrary) GetID() pulid.ID {
	return fl.ID
}

func (fl *FormulaLibrary) GetCreatedAt() int64 {
	return fl.CreatedAt
}

func (fl *FormulaLibrary) GetOrganizationID() pulid.ID {
	return fl.OrganizationID
}

func (fl *FormulaLibrary) GetBusinessUnitID() pulid.ID {
	return fl.BusinessUnitID
}

func (fl *FormulaLibrary) GetTableName() string {
	return "formula_libraries"
}

func (fl *FormulaLibrary) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "flib",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "description", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}
}
