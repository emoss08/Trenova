package homelayout

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*Preset)(nil)
)

// Preset is a home screen an administrator authors once and assigns to the
// roles that should see it. Users with similar jobs get the same starting
// point, and a locked preset stays exactly as authored.
type Preset struct {
	keel.Table `keel:"home_layout_presets"`

	ID                 pulid.ID                `json:"id"                 keel:"id,pk,autoid:hlp_"`
	BusinessUnitID     pulid.ID                `json:"businessUnitId"     keel:"business_unit_id,pk,tenant"`
	OrganizationID     pulid.ID                `json:"organizationId"     keel:"organization_id,pk,tenant"`
	Name               string                  `json:"name"               keel:"name"`
	Description        string                  `json:"description"        keel:"description,nullzero"`
	Layout             *Layout                 `json:"layout"             keel:"layout,jsonb"`
	RoleIDs            []pulid.ID              `json:"roleIds"            keel:"role_ids,nullzero"`
	CoreResponsibility rbac.CoreResponsibility `json:"coreResponsibility" keel:"core_responsibility,nullzero"`
	IsOrgDefault       bool                    `json:"isOrgDefault"       keel:"is_org_default"`
	Locked             bool                    `json:"locked"             keel:"locked"`
	Priority           int                     `json:"priority"           keel:"priority"`
	CreatedBy          pulid.ID                `json:"createdBy"          keel:"created_by,nullzero"`
	Version            int64                   `json:"version"            keel:"version,version"`
	CreatedAt          int64                   `json:"createdAt"          keel:"created_at,autocreate"`
	UpdatedAt          int64                   `json:"updatedAt"          keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
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
				rbac.CoreResponsibilityBilling,
				rbac.CoreResponsibilityOperations,
				rbac.CoreResponsibilityFinance,
				rbac.CoreResponsibilityLeadership,
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
func (p *Preset) AppliesToRole(roleID pulid.ID, responsibility rbac.CoreResponsibility) bool {
	for _, assigned := range p.RoleIDs {
		if assigned == roleID {
			return true
		}
	}

	return p.CoreResponsibility != "" && p.CoreResponsibility == responsibility
}

func (p *Preset) GetID() pulid.ID { return p.ID }

func (p *Preset) GetCreatedAt() int64 { return p.CreatedAt }

func (p *Preset) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *Preset) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *Preset) GetTableName() string { return "home_layout_presets" }
