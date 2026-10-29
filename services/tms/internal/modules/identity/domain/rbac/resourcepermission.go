package rbac

import (
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type ResourcePermission struct {
	keel.Table `keel:"resource_permissions"`

	ID         pulid.ID               `json:"id"         keel:"id,pk,autoid:rp_"`
	RoleID     pulid.ID               `json:"roleId"     keel:"role_id"`
	Resource   string                 `json:"resource"   keel:"resource"`
	Operations []permission.Operation `json:"operations" keel:"operations"`
	DataScope  permission.DataScope   `json:"dataScope"  keel:"data_scope"`
	CreatedAt  int64                  `json:"createdAt"  keel:"created_at,autocreate"`
	UpdatedAt  int64                  `json:"updatedAt"  keel:"updated_at,autoupdate"`
}

func (rp *ResourcePermission) BeforeCreate(ctx context.Context) error {
	if rp.DataScope == "" {
		rp.DataScope = permission.DataScopeOrganization
	}
	rp.Operations = permission.Normalize(rp.Operations)

	return nil
}

func (rp *ResourcePermission) BeforeUpdate(_ context.Context) error {
	rp.Operations = permission.Normalize(rp.Operations)
	rp.Operations = permission.Normalize(rp.Operations)

	return nil
}

func (rp *ResourcePermission) HasOperation(op permission.Operation) bool {
	return slices.Contains(permission.Implied(rp.Operations), op)
}

func (rp *ResourcePermission) GetOperationSet() permission.OperationSet {
	return permission.ExpandWithDependencies(permission.NewOperationSet(rp.Operations...))
}

func (p *ResourcePermission) Validate(multiErr *errortypes.MultiError) {
	// RoleID is stamped when the role is written, so it is not the caller's to
	// supply on a permission that arrives as part of one.
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.Resource,
			validation.Required.Error("Resource is required"),
			validation.Length(1, maxPermissionResourceLength).
				Error("Resource cannot be longer than 100 characters"),
		),
		// A grant with no operations authorizes nothing while still appearing in
		// the role, and one naming an undefined operation can never be evaluated.
		validation.Field(&p.Operations,
			validation.Required.Error("At least one operation is required"),
			validation.Each(domainvalidation.ValidEnum[permission.Operation](
				"Operation is not one this system defines",
			)),
		),
		validation.Field(&p.DataScope,
			validation.Required.Error("Data scope is required"),
			domainvalidation.ValidEnum[permission.DataScope]("Data scope is invalid"),
		),
	))
}

const maxPermissionResourceLength = 100
