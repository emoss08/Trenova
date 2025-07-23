package permissiongrant

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/uptrace/bun"
)

type Grant struct {
	bun.BaseModel `bun:"table:permission_grants,alias:pg"`

	ID             pulid.ID                      `json:"id"                       bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID                      `json:"organizationId"           bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID                      `json:"businessUnitId"           bun:"business_unit_id,type:VARCHAR(100),notnull"`
	UserID         pulid.ID                      `json:"userId"                   bun:"user_id,type:VARCHAR(100),notnull"`
	PermissionID   pulid.ID                      `json:"permissionId"             bun:"permission_id,type:VARCHAR(100),notnull"`
	GrantedBy      pulid.ID                      `json:"grantedBy"                bun:"granted_by,type:VARCHAR(100),notnull"`
	RevokedBy      *pulid.ID                     `json:"revokedBy,omitempty"      bun:"revoked_by,type:VARCHAR(100),nullzero"`
	Status         permission.Status             `json:"status"                   bun:"status,type:permission_status_enum,notnull,default:'Active'"`
	ExpiresAt      *int64                        `json:"expiresAt,omitempty"      bun:"expires_at,nullzero"`
	RevokedAt      *int64                        `json:"revokedAt,omitempty"      bun:"revoked_at,nullzero"`
	CreatedAt      int64                         `json:"createdAt"                bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	Reason         string                        `json:"reason,omitempty"         bun:"reason,type:TEXT"`
	FieldOverrides []*permission.FieldPermission `json:"fieldOverrides,omitempty" bun:"field_overrides,type:JSONB,nullzero"`
	Conditions     []*permission.Condition       `json:"conditions,omitempty"     bun:"conditions,type:JSONB,nullzero"`
	ResourceID     *pulid.ID                     `json:"resourceId,omitempty"     bun:"resource_id,type:VARCHAR(100),nullzero"`
	AuditTrail     map[string]any                `json:"auditTrail,omitempty"     bun:"audit_trail,type:JSONB"`

	// Relationships
	User       *user.User             `json:"-" bun:"rel:belongs-to,join:user_id=id"`
	Permission *permission.Permission `json:"-" bun:"rel:belongs-to,join:permission_id=id"`
	Grantor    *user.User             `json:"-" bun:"rel:belongs-to,join:granted_by=id"`
	Revoker    *user.User             `json:"-" bun:"rel:belongs-to,join:revoked_by=id"`
}

var _ bun.BeforeAppendModelHook = (*Grant)(nil)

func (g *Grant) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if g.ID == "" {
			g.ID = pulid.MustNew("pg_")
		}
	}
	return nil
}
