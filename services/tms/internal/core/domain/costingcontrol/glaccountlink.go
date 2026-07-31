package costingcontrol

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*CostCategoryGLAccount)(nil)

type CostCategoryGLAccount struct {
	bun.BaseModel `bun:"table:cost_category_gl_accounts,alias:ccga" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	CostCategoryID pulid.ID `json:"costCategoryId" bun:"cost_category_id,type:VARCHAR(100),notnull"`
	GLAccountID    pulid.ID `json:"glAccountId"    bun:"gl_account_id,type:VARCHAR(100),notnull"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	GLAccount *glaccount.GLAccount `json:"glAccount,omitempty" bun:"rel:belongs-to,join:gl_account_id=id"`
}

func (cga *CostCategoryGLAccount) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if cga.ID.IsNil() {
			cga.ID = pulid.MustNew("ccga_")
		}

		cga.CreatedAt = timeutils.NowUnix()
	}

	return nil
}

func (l *CostCategoryGLAccount) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(l,
		validation.Field(&l.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&l.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&l.CostCategoryID,
			validation.Required.Error("Cost category is required"),
		),
		validation.Field(&l.GLAccountID, validation.Required.Error("GL account is required")),
	))
}
