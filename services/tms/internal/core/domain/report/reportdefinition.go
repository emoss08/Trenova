package report

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*ReportDefinition)(nil)
	_ validationframework.TenantedEntity = (*ReportDefinition)(nil)
)

type ReportDefinition struct {
	bun.BaseModel             `bun:"table:report_definitions,alias:rdef" json:"-"`
	pagination.CursorValueSet `bun:",embed"                              json:"-"`

	ID              pulid.ID         `json:"id"              bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID  pulid.ID         `json:"businessUnitId"  bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID  pulid.ID         `json:"organizationId"  bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Name            string           `json:"name"            bun:"name,type:VARCHAR(255),notnull"`
	Description     string           `json:"description"     bun:"description,type:TEXT,nullzero"`
	Category        string           `json:"category"        bun:"category,type:VARCHAR(100),nullzero"`
	Tags            []string         `json:"tags"            bun:"tags,type:TEXT[],array,nullzero"`
	Kind            DefinitionKind   `json:"kind"            bun:"kind,type:VARCHAR(20),notnull,default:'custom'"`
	CannedKey       string           `json:"cannedKey"       bun:"canned_key,type:VARCHAR(100),nullzero"`
	CannedVersion   string           `json:"cannedVersion"   bun:"canned_version,type:VARCHAR(20),nullzero"`
	OwnerID         pulid.ID         `json:"ownerId"         bun:"owner_id,type:VARCHAR(100),notnull"`
	Visibility      Visibility       `json:"visibility"      bun:"visibility,type:VARCHAR(20),notnull,default:'private'"`
	Status          DefinitionStatus `json:"status"          bun:"status,type:VARCHAR(20),notnull,default:'draft'"`
	Diagnostics     []string         `json:"diagnostics"     bun:"diagnostics,type:TEXT[],array,nullzero"`
	CatalogVersion  string           `json:"catalogVersion"  bun:"catalog_version,type:VARCHAR(80),notnull"`
	Definition      *Definition      `json:"definition"      bun:"definition,type:JSONB,notnull"`
	DefaultFormat   Format           `json:"defaultFormat"   bun:"default_format,type:VARCHAR(10),notnull,default:'csv'"`
	CurrentRevision int64            `json:"currentRevision" bun:"current_revision,type:BIGINT,notnull,default:1"`
	LastRunAt       int64            `json:"lastRunAt"       bun:"last_run_at,type:BIGINT,nullzero"`
	Version         int64            `json:"version"         bun:"version,type:BIGINT"`
	CreatedAt       int64            `json:"createdAt"       bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt       int64            `json:"updatedAt"       bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Owner        *tenant.User         `json:"owner,omitempty"        bun:"rel:belongs-to,join:owner_id=id"`
}

func (rd *ReportDefinition) Validate(multiErr *errortypes.MultiError) {
	if rd.Name == "" {
		multiErr.Add("name", errortypes.ErrRequired, "Name is required")
	}
	if !rd.Kind.IsValid() {
		multiErr.Add("kind", errortypes.ErrInvalid, "Kind must be custom or canned_fork")
	}
	if rd.Kind == DefinitionKindCannedFork && rd.CannedKey == "" {
		multiErr.Add("cannedKey", errortypes.ErrRequired, "Canned key is required for canned forks")
	}
	if !rd.Visibility.IsValid() {
		multiErr.Add("visibility", errortypes.ErrInvalid, "Visibility must be private or shared")
	}
	if !rd.Status.IsValid() {
		multiErr.Add("status", errortypes.ErrInvalid, "Status is invalid")
	}
	if !rd.DefaultFormat.IsValid() {
		multiErr.Add("defaultFormat", errortypes.ErrInvalid, "Default format is invalid")
	}
	if rd.Definition == nil {
		multiErr.Add("definition", errortypes.ErrRequired, "Definition is required")
	}
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
