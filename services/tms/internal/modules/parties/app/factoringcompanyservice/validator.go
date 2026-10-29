package factoringcompanyservice

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany/factoringcompanydb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/keelcheck"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB   *postgres.Connection
	Repo factoringcompanyports.FactoringCompanyRepository
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*factoringcompany.FactoringCompany]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := keelcheck.NewChecker(
		p.DB,
		factoringcompanydb.FactoringCompaniesSchema(),
	)
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*factoringcompany.FactoringCompany]().
			WithModelName("FactoringCompany").
			WithUniquenessChecker(checker).
			WithUniqueField(
				"code",
				"code",
				"A factoring company with this code already exists in your organization",
				func(e *factoringcompany.FactoringCompany) any { return e.Code },
			).
			WithCustomRule(
				validationframework.
					NewTenantedRule[*factoringcompany.FactoringCompany]("inactive_with_customers").
					OnUpdate().
					WithValidation(refuseDeactivatingAssigned(p.Repo)),
			).
			Build(),
	}
}

// refuseDeactivatingAssigned keeps a factor active while customers' invoices
// are assigned to it: their next invoice could not post.
func refuseDeactivatingAssigned(
	repo factoringcompanyports.FactoringCompanyRepository,
) func(
	context.Context,
	*factoringcompany.FactoringCompany,
	*validationframework.TenantedValidationContext,
	*errortypes.MultiError,
) error {
	return func(
		ctx context.Context,
		entity *factoringcompany.FactoringCompany,
		_ *validationframework.TenantedValidationContext,
		multiErr *errortypes.MultiError,
	) error {
		if entity.Status != domaintypes.StatusInactive || entity.ID.IsNil() {
			return nil
		}
		count, err := repo.CountAssignedCustomers(
			ctx,
			factoringcompanyports.CountAssignedCustomersRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				},
				FactoringCompanyID: entity.ID,
			},
		)
		if err != nil {
			return err
		}
		if count == 0 {
			return nil
		}
		message := "1 customer's invoices are assigned to this factor. " +
			"Move that customer to another factor or stop factoring it before making this one inactive."
		if count > 1 {
			message = strconv.Itoa(count) + " customers' invoices are assigned to this factor. " +
				"Move them to another factor or stop factoring them before making this one inactive."
		}
		multiErr.Add("status", errortypes.ErrInvalid, message)
		return nil
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *factoringcompany.FactoringCompany,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *factoringcompany.FactoringCompany,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
