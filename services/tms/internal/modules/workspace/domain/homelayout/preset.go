package homelayout

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Preset)(nil)
	_ validationframework.TenantedEntity = (*Preset)(nil)
)

// Preset is a home screen an administrator authors once and assigns to the
// roles that should see it. Users with similar jobs get the same starting
// point, and a locked preset stays exactly as authored.
type Preset struct {
	bun.BaseModel             `bun:"table:home_layout_presets,alias:hlp" json:"-"`
	keel.Table                `keel:"home_layout_presets"`
	pagination.CursorValueSet `bun:",embed"                              json:"-"`

	ID                 pulid.ID                      `json:"id"                 bun:"id,pk,type:VARCHAR(100)" keel:"id,pk,autoid:hlp_"`
	BusinessUnitID     pulid.ID                      `json:"businessUnitId"     bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID     pulid.ID                      `json:"organizationId"     bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	Name               string                        `json:"name"               bun:"name,type:VARCHAR(255),notnull" keel:"name"`
	Description        string                        `json:"description"        bun:"description,type:TEXT,nullzero" keel:"description,nullzero"`
	Layout             *Layout                       `json:"layout"             bun:"layout,type:JSONB,notnull" keel:"layout,jsonb"`
	RoleIDs            []pulid.ID                    `json:"roleIds"            bun:"role_ids,type:TEXT[],array,nullzero" keel:"role_ids,nullzero"`
	CoreResponsibility permission.CoreResponsibility `json:"coreResponsibility" bun:"core_responsibility,type:VARCHAR(50),nullzero" keel:"core_responsibility,nullzero"`
	IsOrgDefault       bool                          `json:"isOrgDefault"       bun:"is_org_default,type:BOOLEAN,notnull" keel:"is_org_default"`
	Locked             bool                          `json:"locked"             bun:"locked,type:BOOLEAN,notnull" keel:"locked"`
	Priority           int                           `json:"priority"           bun:"priority,type:INTEGER,notnull" keel:"priority"`
	CreatedBy          pulid.ID                      `json:"createdBy"          bun:"created_by,type:VARCHAR(100),nullzero" keel:"created_by,nullzero"`
	Version            int64                         `json:"version"            bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt          int64                         `json:"createdAt"          bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt          int64                         `json:"updatedAt"          bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
}

func (p *Preset) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&p.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&p.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxPresetNameLength).
				Error("Name cannot be longer than 255 characters"),
		),
		validation.Field(&p.Priority,
			validation.Min(0).Error("Priority must be between 0 and 1000"),
			validation.Max(MaxPresetPriority).Error("Priority must be between 0 and 1000"),
		),
		validation.Field(&p.CoreResponsibility,
			validation.In(
				permission.CoreResponsibilityBilling,
				permission.CoreResponsibilityOperations,
				permission.CoreResponsibilityFinance,
				permission.CoreResponsibilityLeadership,
			).Error("Unknown core responsibility"),
		),
		validation.Field(&p.Layout, validation.Required.Error("Layout is required")),
	))

	if p.Layout != nil {
		p.Layout.Validate(multiErr, "layout")
	}
}

// AppliesToRole reports whether this preset was assigned to the given role,
// either directly or through the role's core responsibility.
func (p *Preset) AppliesToRole(roleID pulid.ID, responsibility permission.CoreResponsibility) bool {
	for _, assigned := range p.RoleIDs {
		if assigned == roleID {
			return true
		}
	}

	return p.CoreResponsibility != "" && p.CoreResponsibility == responsibility
}

func (p *Preset) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("hlp_")
		}
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}

func (p *Preset) GetID() pulid.ID { return p.ID }

func (p *Preset) GetCreatedAt() int64 { return p.CreatedAt }

func (p *Preset) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *Preset) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *Preset) GetTableName() string { return "home_layout_presets" }
