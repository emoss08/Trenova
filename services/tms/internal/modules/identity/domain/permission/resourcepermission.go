package permission

import (
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*ResourcePermission)(nil)

type ResourcePermission struct {
	bun.BaseModel `bun:"table:resource_permissions,alias:rp" json:"-"`
	keel.Table    `keel:"resource_permissions"`

	ID         pulid.ID    `json:"id"         bun:"id,pk,type:VARCHAR(100)" keel:"id,pk,autoid:rp_"`
	RoleID     pulid.ID    `json:"roleId"     bun:"role_id,type:VARCHAR(100),notnull" keel:"role_id"`
	Resource   string      `json:"resource"   bun:"resource,type:VARCHAR(100),notnull" keel:"resource"`
	Operations []Operation `json:"operations" bun:"operations,type:TEXT[],array,notnull" keel:"operations"`
	DataScope  DataScope   `json:"dataScope"  bun:"data_scope,type:VARCHAR(20),notnull,default:'organization'" keel:"data_scope"`
	CreatedAt  int64       `json:"createdAt"  bun:"created_at,notnull" keel:"created_at,autocreate"`
	UpdatedAt  int64       `json:"updatedAt"  bun:"updated_at,notnull" keel:"updated_at,autoupdate"`
}

func (rp *ResourcePermission) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if rp.ID.IsNil() {
			rp.ID = pulid.MustNew("rp_")
		}
		rp.CreatedAt = now
		rp.UpdatedAt = now
		rp.Operations = Normalize(rp.Operations)
	case *bun.UpdateQuery:
		rp.UpdatedAt = now
		rp.Operations = Normalize(rp.Operations)
	}

	return nil
}

func (rp *ResourcePermission) BeforeCreate(ctx context.Context) error {
	return rp.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}

func (rp *ResourcePermission) HasOperation(op Operation) bool {
	return slices.Contains(Implied(rp.Operations), op)
}

func (rp *ResourcePermission) GetOperationSet() OperationSet {
	return ExpandWithDependencies(NewOperationSet(rp.Operations...))
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
			validation.Each(domainvalidation.ValidEnum[Operation](
				"Operation is not one this system defines",
			)),
		),
		validation.Field(&p.DataScope,
			validation.Required.Error("Data scope is required"),
			domainvalidation.ValidEnum[DataScope]("Data scope is invalid"),
		),
	))
}

const maxPermissionResourceLength = 100
