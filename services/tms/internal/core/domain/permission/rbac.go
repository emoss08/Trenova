package permission

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*RoleHierarchyEdge)(nil)
	_ bun.BeforeAppendModelHook = (*RoleConstraint)(nil)
	_ bun.BeforeAppendModelHook = (*RoleConstraintRole)(nil)
)

type RoleConstraintType string

const (
	RoleConstraintTypeSSD RoleConstraintType = "ssd"
	RoleConstraintTypeDSD RoleConstraintType = "dsd"
)

type RoleHierarchyEdge struct {
	bun.BaseModel `bun:"table:role_hierarchy_edges,alias:rhe" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	SeniorRoleID   pulid.ID `json:"seniorRoleId"   bun:"senior_role_id,type:VARCHAR(100),notnull"`
	JuniorRoleID   pulid.ID `json:"juniorRoleId"   bun:"junior_role_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	CreatedBy      pulid.ID `json:"createdBy"      bun:"created_by,type:VARCHAR(100)"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,notnull"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,notnull"`

	SeniorRole *Role `json:"seniorRole,omitempty" bun:"rel:belongs-to,join:senior_role_id=id"`
	JuniorRole *Role `json:"juniorRole,omitempty" bun:"rel:belongs-to,join:junior_role_id=id"`
}

func (e *RoleHierarchyEdge) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()
	switch q.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("rhe_")
		}
		e.CreatedAt = now
		e.UpdatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
	}
	return nil
}

type RoleConstraint struct {
	bun.BaseModel `bun:"table:role_constraints,alias:rc" json:"-"`

	ID             pulid.ID           `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	Name           string             `json:"name"           bun:"name,type:VARCHAR(255),notnull"`
	Description    string             `json:"description"    bun:"description,type:TEXT"`
	Type           RoleConstraintType `json:"type"           bun:"type,type:VARCHAR(20),notnull"`
	MaxRoles       int                `json:"maxRoles"       bun:"max_roles,notnull"`
	Enabled        bool               `json:"enabled"        bun:"enabled,notnull,default:true"`
	CreatedBy      pulid.ID           `json:"createdBy"      bun:"created_by,type:VARCHAR(100)"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,notnull"`
	UpdatedAt      int64              `json:"updatedAt"      bun:"updated_at,notnull"`

	Roles []*Role `json:"roles,omitempty" bun:"m2m:role_constraint_roles,join:RoleConstraint=Role"`
}

func (c *RoleConstraint) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()
	switch q.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("rco_")
		}
		c.CreatedAt = now
		c.UpdatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}
	return nil
}

type RoleConstraintRole struct {
	bun.BaseModel `bun:"table:role_constraint_roles,alias:rcr" json:"-"`

	ID               pulid.ID        `json:"id"               bun:"id,pk,type:VARCHAR(100)"`
	RoleConstraintID pulid.ID        `json:"roleConstraintId" bun:"role_constraint_id,type:VARCHAR(100),notnull"`
	RoleID           pulid.ID        `json:"roleId"           bun:"role_id,type:VARCHAR(100),notnull"`
	OrganizationID   pulid.ID        `json:"organizationId"   bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID   pulid.ID        `json:"businessUnitId"   bun:"business_unit_id,type:VARCHAR(100),notnull"`
	CreatedAt        int64           `json:"createdAt"        bun:"created_at,notnull"`
	RoleConstraint   *RoleConstraint `json:"-"                bun:"rel:belongs-to,join:role_constraint_id=id"`
	Role             *Role           `json:"-"                bun:"rel:belongs-to,join:role_id=id"`
}

func (r *RoleConstraintRole) BeforeAppendModel(_ context.Context, q bun.Query) error {
	if _, ok := q.(*bun.InsertQuery); ok {
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("rcr_")
		}
		r.CreatedAt = timeutils.NowUnix()
	}
	return nil
}
