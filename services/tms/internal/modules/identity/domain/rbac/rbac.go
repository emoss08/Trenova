package rbac

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type RoleConstraintType string

const (
	RoleConstraintTypeSSD RoleConstraintType = "ssd"
	RoleConstraintTypeDSD RoleConstraintType = "dsd"
)

type RoleHierarchyEdge struct {
	keel.Table `keel:"role_hierarchy_edges"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:rhe_"`
	SeniorRoleID   pulid.ID `json:"seniorRoleId"   keel:"senior_role_id"`
	JuniorRoleID   pulid.ID `json:"juniorRoleId"   keel:"junior_role_id"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,tenant"`
	CreatedBy      pulid.ID `json:"createdBy"      keel:"created_by,nullzero"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      keel:"updated_at,autoupdate"`

	SeniorRole *Role `json:"seniorRole,omitempty" keel:"belongs:senior_role_id"`
	JuniorRole *Role `json:"juniorRole,omitempty" keel:"belongs:junior_role_id"`
}

type RoleConstraint struct {
	keel.Table `keel:"role_constraints"`

	ID             pulid.ID           `json:"id"             keel:"id,pk,autoid:rco_"`
	OrganizationID pulid.ID           `json:"organizationId" keel:"organization_id,tenant"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" keel:"business_unit_id,tenant"`
	Name           string             `json:"name"           keel:"name"`
	Description    string             `json:"description"    keel:"description,nullzero"`
	Type           RoleConstraintType `json:"type"           keel:"type"`
	MaxRoles       int                `json:"maxRoles"       keel:"max_roles"`
	Enabled        bool               `json:"enabled"        keel:"enabled"`
	CreatedBy      pulid.ID           `json:"createdBy"      keel:"created_by,nullzero"`
	CreatedAt      int64              `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64              `json:"updatedAt"      keel:"updated_at,autoupdate"`

	Roles []*Role `json:"roles,omitempty" keel:"m2m:role_constraint_roles.role_constraint_id.role_id"`
}

type RoleConstraintRole struct {
	keel.Table `keel:"role_constraint_roles"`

	ID               pulid.ID        `json:"id"               keel:"id,pk,autoid:rcr_"`
	RoleConstraintID pulid.ID        `json:"roleConstraintId" keel:"role_constraint_id"`
	RoleID           pulid.ID        `json:"roleId"           keel:"role_id"`
	OrganizationID   pulid.ID        `json:"organizationId"   keel:"organization_id,tenant"`
	BusinessUnitID   pulid.ID        `json:"businessUnitId"   keel:"business_unit_id,tenant"`
	CreatedAt        int64           `json:"createdAt"        keel:"created_at,autocreate"`
	RoleConstraint   *RoleConstraint `json:"-"                keel:"belongs:role_constraint_id"`
	Role             *Role           `json:"-"                keel:"belongs:role_id"`
}

func (e *RoleHierarchyEdge) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&e.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&e.SeniorRoleID, validation.Required.Error("Senior role is required")),
		// A role senior to itself is a one-node cycle, and inheritance resolution
		// walks these edges.
		validation.Field(&e.JuniorRoleID,
			validation.Required.Error("Junior role is required"),
			validation.NotIn(e.SeniorRoleID).Error("A role cannot inherit from itself"),
		),
	))
}

func (c *RoleConstraint) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&c.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&c.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxConstraintNameLength).
				Error("Name cannot be longer than 255 characters"),
		),
		validation.Field(&c.Type,
			validation.Required.Error("Type is required"),
			validation.In(RoleConstraintTypeSSD, RoleConstraintTypeDSD).
				Error("Type must be static or dynamic separation of duty"),
		),
		// A separation of duty rule permitting fewer than one role would deny
		// every assignment, and one permitting none of the set is not a
		// separation rule at all.
		validation.Field(&c.MaxRoles,
			validation.Required.Error("Maximum roles is required"),
			validation.Min(1).Error("Maximum roles must be at least one"),
		),
	))
}

func (r *RoleConstraintRole) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&r.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&r.RoleConstraintID,
			validation.Required.Error("Role constraint is required"),
		),
		validation.Field(&r.RoleID, validation.Required.Error("Role is required")),
	))
}

const maxConstraintNameLength = 255
