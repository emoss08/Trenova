package pagefavorite

import (
	"regexp"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	maxPageURLLength   = 500
	maxPageTitleLength = 255
)

// pageURLPattern keeps a favorite pointing inside the application: an absolute
// URL here would be a redirect target rendered as if it were a page of the TMS.
var pageURLPattern = regexp.MustCompile(`^/\S*$`)

var (
	_ validationframework.TenantedEntity = (*PageFavorite)(nil)
)

type PageFavorite struct {
	keel.Table `keel:"page_favorites"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:pf_"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,tenant"`
	UserID         pulid.ID `json:"userId"         keel:"user_id"`
	PageURL        string   `json:"pageUrl"        keel:"page_url"`
	PageTitle      string   `json:"pageTitle"      keel:"page_title"`
	Version        int64    `json:"version"        keel:"version,version"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	User         *tenant.User         `json:"user,omitempty"         keel:"belongs:user_id"`
}

func (pf *PageFavorite) GetID() pulid.ID {
	return pf.ID
}

func (pf *PageFavorite) GetOrganizationID() pulid.ID {
	return pf.OrganizationID
}

func (pf *PageFavorite) GetBusinessUnitID() pulid.ID {
	return pf.BusinessUnitID
}

func (pf *PageFavorite) GetTableName() string {
	return "page_favorites"
}

func (pf *PageFavorite) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(pf,
		validation.Field(&pf.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&pf.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&pf.UserID, validation.Required.Error("User is required")),
		// A favorite that points nowhere would render a dead entry in the
		// sidebar, so the path is checked for shape as well as presence.
		validation.Field(&pf.PageURL,
			validation.Required.Error("Page URL is required"),
			validation.Length(1, maxPageURLLength).
				Error("Page URL cannot be longer than 500 characters"),
			validation.Match(pageURLPattern).
				Error("Page URL must be an application path, such as /shipments"),
		),
		validation.Field(&pf.PageTitle,
			validation.Required.Error("Page title is required"),
			validation.Length(1, maxPageTitleLength).
				Error("Page title cannot be longer than 255 characters"),
		),
	))
}
