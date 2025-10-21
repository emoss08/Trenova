package pagefavorite

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*PageFavorite)(nil)
	_ domain.Validatable        = (*PageFavorite)(nil)
)

type PageFavorite struct {
	bun.BaseModel `bun:"table:page_favorites,alias:pf" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	PageURL        string   `json:"pageUrl"        bun:"page_url,type:VARCHAR(500),notnull"`
	PageTitle      string   `json:"pageTitle"      bun:"page_title,type:VARCHAR(255),notnull"`
	Version        int64    `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `json:"-" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-" bun:"rel:belongs-to,join:organization_id=id"`
	User         *tenant.User         `json:"-" bun:"rel:belongs-to,join:user_id=id"`
}

func (f *PageFavorite) Validate(multiErr *errortypes.MultiError) {
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
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (f *PageFavorite) GetID() string {
	return f.ID.String()
}

func (f *PageFavorite) GetTableName() string {
	return "favorites"
}

func (f *PageFavorite) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := utils.NowUnix()

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
