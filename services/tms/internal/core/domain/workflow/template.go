package workflow

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*Template)(nil)
	_ domain.Validatable             = (*Template)(nil)
	_ framework.TenantedEntity       = (*Template)(nil)
	_ domaintypes.PostgresSearchable = (*Template)(nil)
)

type Template struct {
	bun.BaseModel `bun:"table:workflow_templates,alias:wft" json:"-"`

	ID                 pulid.ID  `json:"id"                 bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID     pulid.ID  `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID     pulid.ID  `json:"organizationId"     bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Name               string    `json:"name"               bun:"name,type:VARCHAR(255),notnull"`
	Description        string    `json:"description"        bun:"description,type:TEXT,nullzero"`
	IsTemplate         bool      `json:"isTemplate"         bun:"is_template,type:BOOLEAN,notnull,default:false"`
	PublishedVersionID *pulid.ID `json:"publishedVersionId" bun:"published_version_id,type:VARCHAR(100),nullzero"`
	SearchVector       string    `json:"-"                  bun:"search_vector,type:TSVECTOR,scanonly"`
	Version            int64     `json:"version"            bun:"version,type:BIGINT,notnull,default:0"`
	CreatedByID        pulid.ID  `json:"createdById"        bun:"created_by_id,type:VARCHAR(100),notnull"`
	UpdatedByID        pulid.ID  `json:"updatedById"        bun:"updated_by_id,type:VARCHAR(100),notnull"`
	CreatedAt          int64     `json:"createdAt"          bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64     `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relations
	BusinessUnit     *tenant.BusinessUnit `json:"businessUnit,omitempty"     bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization     *tenant.Organization `json:"organization,omitempty"     bun:"rel:belongs-to,join:organization_id=id"`
	CreatedBy        *tenant.User         `json:"createdBy,omitempty"        bun:"rel:belongs-to,join:created_by_id=id"`
	UpdatedBy        *tenant.User         `json:"updatedBy,omitempty"        bun:"rel:belongs-to,join:updated_by_id=id"`
	PublishedVersion *Version             `json:"publishedVersion,omitempty" bun:"rel:belongs-to,join:published_version_id=id"`
	Versions         []*Version           `json:"versions,omitempty"         bun:"rel:has-many,join:id=workflow_template_id"`
	Instances        []*Instance          `json:"instances,omitempty"        bun:"rel:has-many,join:id=workflow_template_id"`
}

func (t *Template) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(t,
		validation.Field(&t.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (t *Template) GetID() string {
	return t.ID.String()
}

func (t *Template) GetOrganizationID() pulid.ID {
	return t.OrganizationID
}

func (t *Template) GetBusinessUnitID() pulid.ID {
	return t.BusinessUnitID
}

func (t *Template) GetTableName() string {
	return "workflow_templates"
}

func (t *Template) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("wft_")
		}
		t.CreatedAt = now
		t.UpdatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}

	return nil
}

func (t *Template) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "wft",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (t *Template) HasPublishedVersion() bool {
	return t.PublishedVersionID != nil && !t.PublishedVersionID.IsNil()
}

func (t *Template) HasVersions() bool {
	return len(t.Versions) > 0
}
