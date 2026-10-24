package report

import (
	"context"

	"github.com/emoss08/keel/analytics/ir"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	maxDefinitionNameLength     = 255
	maxDefinitionCategoryLength = 100
	maxCannedKeyLength          = 100
)

var (
	_ validationframework.TenantedEntity = (*ReportDefinition)(nil)
)

// Saved report definitions
type ReportDefinition struct {
	keel.Table `                                                   keel:"report_definitions,report=Reporting,label=Report"`

	ID              pulid.ID         `json:"id" keel:"id,pk,autoid:rd_"`
	BusinessUnitID  pulid.ID         `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID         `json:"organizationId" keel:"organization_id,pk,tenant"`
	Name            string           `json:"name" keel:"name"`
	Description     string           `json:"description" keel:"description,nullzero"`
	Category        string           `json:"category" keel:"category,nullzero"`
	Tags            []string         `json:"tags" keel:"tags,nullzero,noreport"`
	Kind            DefinitionKind   `json:"kind" keel:"kind"`
	CannedKey       string           `json:"cannedKey" keel:"canned_key,nullzero,label=Canned Report Key"`
	CannedVersion   string           `json:"cannedVersion" keel:"canned_version,nullzero,noreport"`
	OwnerID         pulid.ID         `json:"ownerId" keel:"owner_id"`
	Visibility      Visibility       `json:"visibility" keel:"visibility"`
	Status          DefinitionStatus `json:"status" keel:"status"`
	Diagnostics     []string         `json:"diagnostics" keel:"diagnostics,nullzero,noreport"`
	CatalogVersion  string           `json:"catalogVersion" keel:"catalog_version,noreport"`
	Definition      *ir.Definition   `json:"definition" keel:"definition,jsonb,noreport"`
	DefaultFormat   Format           `json:"defaultFormat" keel:"default_format"`
	CurrentRevision int64            `json:"currentRevision" keel:"current_revision,label=Revision,format=count,dimension"`
	LastRunAt       int64            `json:"lastRunAt" keel:"last_run_at,nullzero"`
	Version         int64            `json:"version" keel:"version,version,noreport"`
	CreatedAt       int64            `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt       int64            `json:"updatedAt" keel:"updated_at,autoupdate,noreport"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Owner        *tenant.User         `json:"owner,omitempty" keel:"belongs:owner_id,noreport"`
}

func (rd *ReportDefinition) BeforeCreate(_ context.Context) error {
	if rd.Kind == "" {
		rd.Kind = DefinitionKindCustom
	}
	if rd.Visibility == "" {
		rd.Visibility = VisibilityPrivate
	}
	if rd.Status == "" {
		rd.Status = DefinitionStatusDraft
	}
	if rd.DefaultFormat == "" {
		rd.DefaultFormat = FormatCSV
	}
	if rd.CurrentRevision == 0 {
		rd.CurrentRevision = 1
	}

	return nil
}

func (rd *ReportDefinition) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(rd,
		validation.Field(&rd.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&rd.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&rd.OwnerID, validation.Required.Error("Owner is required")),
		validation.Field(&rd.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxDefinitionNameLength).
				Error("Name cannot be longer than 255 characters"),
		),
		validation.Field(&rd.Category,
			validation.Length(0, maxDefinitionCategoryLength).
				Error("Category cannot be longer than 100 characters"),
		),
		validation.Field(&rd.Kind,
			validation.Required.Error("Kind must be custom or canned_fork"),
			domainvalidation.ValidEnum[DefinitionKind]("Kind must be custom or canned_fork"),
		),
		// A fork with no key has nothing to fork from, so it would resolve to an
		// empty report the first time it ran.
		validation.Field(&rd.CannedKey,
			validation.When(
				rd.Kind == DefinitionKindCannedFork,
				validation.Required.Error("Canned key is required for canned forks"),
			),
			validation.Length(0, maxCannedKeyLength).
				Error("Canned key cannot be longer than 100 characters"),
		),
		validation.Field(&rd.Visibility,
			validation.Required.Error("Visibility must be private or shared"),
			domainvalidation.ValidEnum[Visibility]("Visibility must be private or shared"),
		),
		validation.Field(&rd.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[DefinitionStatus]("Status is invalid"),
		),
		validation.Field(&rd.DefaultFormat,
			validation.Required.Error("Default format is required"),
			domainvalidation.ValidEnum[Format]("Default format is invalid"),
		),
		validation.Field(&rd.Definition, validation.Required.Error("Definition is required")),
		// Only once it is stored. The revision is the repository's to assign —
		// 1 on insert, incremented on update, with the matching revision row
		// written in the same transaction — so an unsaved definition has
		// nothing to put here and a caller has no way to supply it. Demanding
		// one refused every create, which report_create hit as a failed
		// execution after its proposal had already been approved.
		validation.Field(&rd.CurrentRevision,
			validation.When(rd.ID.IsNotNil(),
				validation.Min(int64(1)).Error("Revision must be at least one"),
			),
		),
	))
}

func (rd *ReportDefinition) GetCreatedAt() int64 { return rd.CreatedAt }

func (rd *ReportDefinition) GetID() pulid.ID { return rd.ID }

func (rd *ReportDefinition) GetOrganizationID() pulid.ID { return rd.OrganizationID }

func (rd *ReportDefinition) GetBusinessUnitID() pulid.ID { return rd.BusinessUnitID }

func (rd *ReportDefinition) GetTableName() string { return "report_definitions" }

func (rd *ReportDefinition) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "rdef",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "description", Type: domaintypes.FieldTypeText},
			{Name: "category", Type: domaintypes.FieldTypeText},
		},
	}
}
