package pagefavorite

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*PageFavorite)(nil)
var _ domain.Validatable = (*PageFavorite)(nil)

type PageFavorite struct {
	bun.BaseModel `bun:"table:page_favorites,alias:pf" json:"-"`

	// Primary identifiers
	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`

	// Core fields
	PageURL   string `json:"pageUrl"   bun:"page_url,type:VARCHAR(500),notnull"`
	PageTitle string `json:"pageTitle" bun:"page_title,type:VARCHAR(255),notnull"`
	Version   int64  `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64  `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64  `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"-" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"-" bun:"rel:belongs-to,join:organization_id=id"`
	User         *user.User                 `json:"-" bun:"rel:belongs-to,join:user_id=id"`
}

// Validate validates the favorite entity
func (f *PageFavorite) Validate(_ context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStruct(f,
		validation.Field(&f.PageURL,
			validation.Required.Error("Page URL is required"),
			validation.Length(1, 500).Error("Page URL must be between 1 and 500 characters"),
			is.URL.Error("Page URL must be a valid URL"),
		),
		validation.Field(&f.PageTitle,
			validation.Required.Error("Page title is required"),
			validation.Length(1, 255).Error("Page title must be between 1 and 255 characters"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// GetID returns the ID for pagination purposes
func (f *PageFavorite) GetID() string {
	return f.ID.String()
}

// GetTableName returns the table name for pagination purposes
func (f *PageFavorite) GetTableName() string {
	return "favorites"
}

// BeforeAppendModel implements the bun.BeforeAppendModelHook interface.
func (f *PageFavorite) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if f.ID.IsNil() {
			f.ID = pulid.MustNew("pf_")
		}

		f.CreatedAt = now
	case *bun.UpdateQuery:
		f.UpdatedAt = now
	}

	return nil
}
