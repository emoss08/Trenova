package costingcontrol

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type CostCategoryGLAccount struct {
	keel.Table `keel:"cost_category_gl_accounts"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:ccga_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	CostCategoryID pulid.ID `json:"costCategoryId" keel:"cost_category_id"`
	GLAccountID    pulid.ID `json:"glAccountId" keel:"gl_account_id"`
	CreatedAt      int64    `json:"createdAt" keel:"created_at,autocreate"`

	// Relationships
	GLAccount *glaccount.GLAccount `json:"glAccount,omitempty" keel:"belongs:gl_account_id"`
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
