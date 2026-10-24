package report

import (
	"context"

	"github.com/emoss08/keel/analytics/ir"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	maxDefinitionNameLength     = 255
	maxDefinitionCategoryLength = 100
	maxCannedKeyLength          = 100
)

var (
	_ bun.BeforeAppendModelHook          = (*ReportDefinition)(nil)
	_ validationframework.TenantedEntity = (*ReportDefinition)(nil)
)

// Saved report definitions
type ReportDefinition struct {
	bun.BaseModel             `bun:"table:report_definitions,alias:rdef" json:"-"`
	keel.Table                `                                                   keel:"report_definitions,report=Reporting,label=Report"`
	pagination.CursorValueSet `bun:",embed"                              json:"-"`

	ID              pulid.ID         `json:"id"              bun:"id,pk,type:VARCHAR(100)"                                                           keel:"id,pk,autoid:rd_"`
	BusinessUnitID  pulid.ID         `json:"businessUnitId"  bun:"business_unit_id,pk,type:VARCHAR(100),notnull"                                     keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID         `json:"organizationId"  bun:"organization_id,pk,type:VARCHAR(100),notnull"                                      keel:"organization_id,pk,tenant"`
	Name            string           `json:"name"            bun:"name,type:VARCHAR(255),notnull"                                                    keel:"name"`
	Description     string           `json:"description"     bun:"description,type:TEXT,nullzero"                                                    keel:"description,nullzero"`
	Category        string           `json:"category"        bun:"category,type:VARCHAR(100),nullzero"                                               keel:"category,nullzero"`
	Tags            []string         `json:"tags"            bun:"tags,type:TEXT[],array,nullzero"                                                   keel:"tags,nullzero,noreport"`
	Kind            DefinitionKind   `json:"kind"            bun:"kind,type:VARCHAR(20),notnull,default:'custom'"                                    keel:"kind"`
	CannedKey       string           `json:"cannedKey"       bun:"canned_key,type:VARCHAR(100),nullzero"                                             keel:"canned_key,nullzero,label=Canned Report Key"`
	CannedVersion   string           `json:"cannedVersion"   bun:"canned_version,type:VARCHAR(20),nullzero"                                          keel:"canned_version,nullzero,noreport"`
	OwnerID         pulid.ID         `json:"ownerId"         bun:"owner_id,type:VARCHAR(100),notnull"                                                keel:"owner_id"`
	Visibility      Visibility       `json:"visibility"      bun:"visibility,type:VARCHAR(20),notnull,default:'private'"                             keel:"visibility"`
	Status          DefinitionStatus `json:"status"          bun:"status,type:VARCHAR(20),notnull,default:'draft'"                                   keel:"status"`
	Diagnostics     []string         `json:"diagnostics"     bun:"diagnostics,type:TEXT[],array,nullzero"                                            keel:"diagnostics,nullzero,noreport"`
	CatalogVersion  string           `json:"catalogVersion"  bun:"catalog_version,type:VARCHAR(80),notnull"                                          keel:"catalog_version,noreport"`
	Definition      *ir.Definition   `json:"definition"      bun:"definition,type:JSONB,notnull"                                                     keel:"definition,jsonb,noreport"`
	DefaultFormat   Format           `json:"defaultFormat"   bun:"default_format,type:VARCHAR(10),notnull,default:'csv'"                             keel:"default_format"`
	CurrentRevision int64            `json:"currentRevision" bun:"current_revision,type:BIGINT,notnull,default:1"                                    keel:"current_revision,label=Revision,format=count,dimension"`
	LastRunAt       int64            `json:"lastRunAt"       bun:"last_run_at,type:BIGINT,nullzero"                                                  keel:"last_run_at,nullzero"`
	Version         int64            `json:"version"         bun:"version,type:BIGINT"                                                               keel:"version,version,noreport"`
	CreatedAt       int64            `json:"createdAt"       bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt       int64            `json:"updatedAt"       bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate,noreport"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Owner        *tenant.User         `json:"owner,omitempty"        bun:"rel:belongs-to,join:owner_id=id"         keel:"belongs:owner_id,noreport"`
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

func (rd *ReportDefinition) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if rd.ID.IsNil() {
			rd.ID = pulid.MustNew("rd_")
		}
		rd.CreatedAt = now
	case *bun.UpdateQuery:
		rd.UpdatedAt = now
	}

	return nil
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
