package homelayout

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Preference)(nil)
	_ validationframework.TenantedEntity = (*Preference)(nil)
)

// Preference is one user's home screen document. A row exists only once the
// user has changed something; until then they resolve against a preset.
type Preference struct {
	bun.BaseModel `bun:"table:home_preferences,alias:hpf" json:"-"`

	ID             pulid.ID  `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID  `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	UserID         pulid.ID  `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	Preferences    *Document `json:"preferences"    bun:"preferences,type:JSONB,notnull"`
	Version        int64     `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64     `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64     `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	User         *tenant.User         `json:"user,omitempty"         bun:"rel:belongs-to,join:user_id=id"`
}

func (p *Preference) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&p.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&p.UserID, validation.Required.Error("User is required")),
		validation.Field(&p.Preferences,
			validation.Required.Error("Preferences document is required"),
		),
	))

	if p.Preferences != nil {
		p.Preferences.Validate(multiErr)
	}
}

func (p *Preference) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("hpf_")
		}
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}

func (p *Preference) GetID() pulid.ID { return p.ID }

func (p *Preference) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *Preference) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *Preference) GetTableName() string { return "home_preferences" }
