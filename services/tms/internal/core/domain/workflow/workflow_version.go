package workflow

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*WorkflowVersion)(nil)
	_ domain.Validatable        = (*WorkflowVersion)(nil)
	_ framework.TenantedEntity  = (*WorkflowVersion)(nil)
)

// WorkflowVersion represents a specific version of a workflow
type WorkflowVersion struct {
	bun.BaseModel `bun:"table:workflow_versions,alias:wfv" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	WorkflowID     pulid.ID `json:"workflowId"     bun:"workflow_id,notnull,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,notnull,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,notnull,pk,type:VARCHAR(100)"`

	// Version Info
	VersionNumber int    `json:"versionNumber" bun:"version_number,notnull"`
	VersionName   string `json:"versionName"   bun:"version_name,type:VARCHAR(255)"`
	Changelog     string `json:"changelog"     bun:"changelog,type:TEXT"`

	// Workflow Definition (stored as JSON)
	WorkflowDefinition map[string]any `json:"workflowDefinition" bun:"workflow_definition,type:jsonb,notnull"`

	// Status
	IsPublished *bool     `json:"isPublished" bun:"is_published,default:false"`
	PublishedAt *int64    `json:"publishedAt" bun:"published_at,type:BIGINT,nullzero"`
	PublishedBy *pulid.ID `json:"publishedBy" bun:"published_by,type:VARCHAR(100),nullzero"`

	// Metadata
	CreatedBy pulid.ID `json:"createdBy" bun:"created_by,notnull,type:VARCHAR(100)"`
	CreatedAt int64    `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id"  json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"   json:"-"`
	Workflow     *Workflow            `bun:"rel:belongs-to,join:workflow_id=id"       json:"-"`
	Nodes        []*WorkflowNode      `bun:"rel:has-many,join:id=workflow_version_id" json:"nodes,omitempty"`
	Edges        []*WorkflowEdge      `bun:"rel:has-many,join:id=workflow_version_id" json:"edges,omitempty"`
}

func (wv *WorkflowVersion) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(wv,
		validation.Field(&wv.WorkflowID,
			validation.Required.Error("Workflow ID is required"),
		),
		validation.Field(&wv.VersionNumber,
			validation.Required.Error("Version number is required"),
			validation.Min(1).Error("Version number must be at least 1"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (wv *WorkflowVersion) GetID() string {
	return wv.ID.String()
}

func (wv *WorkflowVersion) GetTableName() string {
	return "workflow_versions"
}

func (wv *WorkflowVersion) GetOrganizationID() pulid.ID {
	return wv.OrganizationID
}

func (wv *WorkflowVersion) GetBusinessUnitID() pulid.ID {
	return wv.BusinessUnitID
}

func (wv *WorkflowVersion) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		if wv.ID.IsNil() {
			wv.ID = pulid.MustNew("wfv_")
		}
	}
	return nil
}
