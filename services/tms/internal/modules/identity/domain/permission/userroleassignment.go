package permission

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*UserRoleAssignment)(nil)

type UserRoleAssignment struct {
	bun.BaseModel `bun:"table:user_role_assignments,alias:ura" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	RoleID         pulid.ID `json:"roleId"         bun:"role_id,type:VARCHAR(100),notnull"`
	ExpiresAt      *int64   `json:"expiresAt"      bun:"expires_at"`
	AssignedBy     pulid.ID `json:"assignedBy"     bun:"assigned_by,type:VARCHAR(100)"`
	AssignedAt     int64    `json:"assignedAt"     bun:"assigned_at,notnull"`

	Role *Role `json:"role,omitempty" bun:"rel:belongs-to,join:role_id=id"`
}

func (ura *UserRoleAssignment) BeforeAppendModel(_ context.Context, q bun.Query) error {
	switch q.(type) { //nolint:gocritic // this is fine
	case *bun.InsertQuery:
		if ura.ID.IsNil() {
			ura.ID = pulid.MustNew("ura_")
		}
		ura.AssignedAt = timeutils.NowUnix()
	}

	return nil
}

func (ura *UserRoleAssignment) GetCreatedAt() int64 {
	return ura.AssignedAt
}

func (ura *UserRoleAssignment) IsExpired() bool {
	if ura.ExpiresAt == nil {
		return false
	}
	return *ura.ExpiresAt < timeutils.NowUnix()
}

func (ura *UserRoleAssignment) GetID() pulid.ID {
	return ura.ID
}

func (a *UserRoleAssignment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&a.UserID, validation.Required.Error("User is required")),
		validation.Field(&a.RoleID, validation.Required.Error("Role is required")),
		// An assignment that expires before it was made grants nothing, which
		// reads to the user as a permission that silently never applied.
		validation.Field(&a.ExpiresAt,
			validation.Min(a.AssignedAt+1).
				Error("Expiry must be after the assignment was made"),
		),
	))
}
