package integration

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type Integration struct {
	keel.Table `keel:"integrations"`

	ID             pulid.ID       `json:"id"             keel:"id,pk,autoid:intg_"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID       `json:"organizationId" keel:"organization_id,pk,tenant"`
	Type           Type           `json:"type"           keel:"type"`
	Name           string         `json:"name"           keel:"name"`
	Description    string         `json:"description"    keel:"description,nullzero"`
	Enabled        bool           `json:"enabled"        keel:"enabled"`
	BuiltBy        string         `json:"builtBy"        keel:"built_by,nullzero"`
	Category       Category       `json:"category"       keel:"category"`
	Configuration  map[string]any `json:"configuration"  keel:"configuration,jsonb,nullzero"`
	DocsURL        string         `json:"docsUrl"        keel:"docs_url,nullzero"`
	Featured       bool           `json:"featured"       keel:"featured"`
	LogoURL        string         `json:"logoUrl"        keel:"logo_url,nullzero"`
	WebsiteURL     string         `json:"websiteUrl"     keel:"website_url,nullzero"`
	EnabledByID    pulid.ID       `json:"enabledById"    keel:"enabled_by_id,nullzero"`
	Version        int64          `json:"version"        keel:"version,version"`
	CreatedAt      int64          `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64          `json:"updatedAt"      keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization *tenant.Organization `json:"organization,omitempty"`
	EnabledBy    *tenant.User         `json:"enabledBy,omitempty"    keel:"belongs:enabled_by_id"`
}

func (i *Integration) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(i,
		validation.Field(&i.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&i.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&i.Type,
			validation.Required.Error("Type is required"),
			domainvalidation.ValidEnum[Type]("Type is not an integration this system supports"),
		),
		validation.Field(&i.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxIntegrationNameLength).
				Error("Name cannot be longer than 100 characters"),
		),
		validation.Field(&i.Category,
			validation.Required.Error("Category is required"),
			domainvalidation.ValidEnum[Category]("Category is invalid"),
		),
		validation.Field(&i.BuiltBy,
			validation.Length(0, maxIntegrationNameLength).
				Error("Built by cannot be longer than 100 characters"),
		),
		// These are rendered as links on the integrations page, so a relative or
		// malformed one would send an administrator nowhere.
		validation.Field(&i.DocsURL, is.URL.Error("Docs URL must be a valid URL")),
		validation.Field(&i.LogoURL, is.URL.Error("Logo URL must be a valid URL")),
		validation.Field(&i.WebsiteURL, is.URL.Error("Website URL must be a valid URL")),
		// Turning an integration on is an accountable act: it starts calling a
		// third party with the organization's credentials.
		validation.Field(&i.EnabledByID,
			validation.When(
				i.Enabled,
				validation.Required.Error("An enabled integration must record who enabled it"),
			),
		),
	))
}

const maxIntegrationNameLength = 100
