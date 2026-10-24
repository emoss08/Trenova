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
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	PathPrefix    = "org."
	MaxNameLength = 64
)

var (
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
	keel.Table `                                                  keel:"formula_libraries"`

	ID                   pulid.ID               `json:"id" keel:"id,pk,autoid:fl_"`
	OrganizationID       pulid.ID               `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID       pulid.ID               `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	Name                 string                 `json:"name" keel:"name"`
	Description          string                 `json:"description" keel:"description,nullzero"`
	Source               string                 `json:"source" keel:"source"`
	Imports              []tarifftypes.Import   `json:"imports" keel:"imports,jsonb"`
	TestCount            int32                  `json:"testCount" keel:"test_count"`
	Status               formulatemplate.Status `json:"status" keel:"status"`
	SubmittedByID        *pulid.ID              `json:"submittedById" keel:"submitted_by_id"`
	SubmittedAt          *int64                 `json:"submittedAt" keel:"submitted_at"`
	ApprovedByID         *pulid.ID              `json:"approvedById" keel:"approved_by_id"`
	ApprovedAt           *int64                 `json:"approvedAt" keel:"approved_at"`
	ReviewComment        string                 `json:"reviewComment" keel:"review_comment,nullzero"`
	SearchVector         string                 `json:"-" keel:"search_vector,tsvector,readonly,noselect"`
	Rank                 string                 `json:"-" keel:"rank,computed"`
	Version              int64                  `json:"version" keel:"version,version"`
	SourceLibraryID      *pulid.ID              `json:"sourceLibraryId" keel:"source_library_id"`
	SourceVersionNumber  *int64                 `json:"sourceVersionNumber" keel:"source_version_number"`
	CurrentVersionNumber int64                  `json:"currentVersionNumber" keel:"current_version_number"`
	CreatedAt            int64                  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt            int64                  `json:"updatedAt" keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
}

func (fl *FormulaLibrary) BeforeUpdate(_ context.Context) error {
	if fl.Imports == nil {
		fl.Imports = []tarifftypes.Import{}
	}

	return nil
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

func (fl *FormulaLibrary) BeforeCreate(ctx context.Context) error {
	if fl.Status == "" {
		fl.Status = formulatemplate.StatusDraft
	}
	if fl.CurrentVersionNumber == 0 {
		fl.CurrentVersionNumber = 1
	}
	if fl.Imports == nil {
		fl.Imports = []tarifftypes.Import{}
	}

	return nil
}

func (fl *FormulaLibrary) BeforeSave(_ context.Context) error {
	if fl.Imports == nil {
		fl.Imports = []tarifftypes.Import{}
	}
	return nil
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
