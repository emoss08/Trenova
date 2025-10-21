package permission

import (
	"context"

	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/uptrace/bun"
)

type RoleScope struct {
	Type          ScopeType  `json:"type"`
	Organizations []pulid.ID `json:"organizations"`
	Inheritable   bool       `json:"inheritable"`
}

type AutoAssignRule struct {
	Enabled    bool              `json:"enabled"`
	Conditions []PolicyCondition `json:"conditions"`
	Attributes map[string]any    `json:"attributes"`
}

type RoleAssignment struct {
	bun.BaseModel `bun:"table:user_organization_roles,alias:uor" json:"-"`

	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),pk"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk"`
	RoleID         pulid.ID `json:"roleId"         bun:"role_id,type:VARCHAR(100),pk"`
	AssignedAt     int64    `json:"assignedAt"     bun:"assigned_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	AssignedBy     pulid.ID `json:"assignedBy"     bun:"assigned_by,type:VARCHAR(100)"`
	ExpiresAt      *int64   `json:"expiresAt"      bun:"expires_at"`

	Overrides []PolicyOverride `json:"overrides" bun:"overrides,type:JSONB"`
}

type PolicyOverride struct {
	PolicyID     pulid.ID   `json:"policyId"`
	ResourceType string     `json:"resourceType"`
	Actions      *ActionSet `json:"actions,omitempty"`
	DataScope    *DataScope `json:"dataScope,omitempty"`
}

type PolicyTemplate struct {
	bun.BaseModel `bun:"table:policy_templates,alias:pt" json:"-"`

	ID            pulid.ID           `json:"id"            bun:"id,pk,type:VARCHAR(100)"`
	Name          string             `json:"name"          bun:"name,type:VARCHAR(255),notnull"`
	Description   string             `json:"description"   bun:"description,type:TEXT"`
	Industry      string             `json:"industry"      bun:"industry,type:VARCHAR(100)"`
	Category      string             `json:"category"      bun:"category,type:VARCHAR(100)"`
	Policies      []PolicyDefinition `json:"policies"      bun:"policies,type:JSONB"`
	RoleStructure []RoleTemplate     `json:"roleStructure" bun:"role_structure,type:JSONB"`
	IsActive      bool               `json:"isActive"      bun:"is_active,default:true"`
	CreatedBy     pulid.ID           `json:"createdBy"     bun:"created_by,type:VARCHAR(100)"`
	CreatedAt     int64              `json:"createdAt"     bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

type PolicyDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Resources   []ResourceRule `json:"resources"`
	Effect      Effect         `json:"effect"`
	Priority    int            `json:"priority"`
}

type RoleTemplate struct {
	Name        string    `json:"name"`
	Level       RoleLevel `json:"level"`
	PolicyNames []string  `json:"policyNames"`
}

var _ bun.BeforeAppendModelHook = (*Role)(nil)

type Role struct {
	bun.BaseModel `bun:"table:roles,alias:r" json:"-"`

	ID             pulid.ID        `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	Name           string          `json:"name"           bun:"name,type:VARCHAR(255),notnull"`
	Description    string          `json:"description"    bun:"description,type:TEXT"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	Level          RoleLevel       `json:"level"          bun:"level,type:VARCHAR(20),notnull"`
	ParentRoles    []pulid.ID      `json:"parentRoles"    bun:"parent_roles,type:TEXT[]"`
	Scope          RoleScope       `json:"scope"          bun:"scope,type:JSONB"`
	PolicyIDs      []pulid.ID      `json:"policyIds"      bun:"policy_ids,type:TEXT[]"`
	AutoAssign     *AutoAssignRule `json:"autoAssign"     bun:"auto_assign,type:JSONB"`
	IsAdmin        bool            `json:"isAdmin"        bun:"is_admin,default:false"`
	IsSystem       bool            `json:"isSystem"       bun:"is_system,default:false"`
	CreatedBy      pulid.ID        `json:"createdBy"      bun:"created_by,type:VARCHAR(100)"`
	CreatedAt      int64           `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64           `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (r *Role) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := utils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("rol_")
		}
		r.CreatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}
