package customerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB      *postgres.Connection
	Factors factoringcompanyports.FactoringCompanyRepository `optional:"true"`
}

type Validator struct {
	validator validationframework.TenantedEntityValidator[*customer.Customer]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := validationframework.NewKeelChecker(p.DB, customerdb.CustomersSchema())
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*customer.Customer]().
			WithModelName("Customer").
			WithUniquenessChecker(checker).
			WithReferenceChecker(checker).
			WithUniqueField(
				"code",
				"code",
				"Customer with this code already exists in your organization",
				func(c *customer.Customer) any { return c.Code },
			).
			WithCustomReferenceCheck(
				"stateId",
				"State does not exist",
				func(c *customer.Customer) pulid.ID { return c.StateID },
				validationframework.KeelReferenceCheck(p.DB, usstatedb.UsStatesSchema()),
			).
			WithCustomRule(
				validationframework.
					NewTenantedRule[*customer.Customer]("active_factoring_company").
					OnBoth().
					WithValidation(requireActiveFactor(p.Factors)),
			).
			Build(),
	}
}

// requireActiveFactor accepts a customer assigned to a factoring company
// only when that factor exists and is active: its invoices would otherwise
// name a payee that cannot be used.
func requireActiveFactor(
	factors factoringcompanyports.FactoringCompanyRepository,
) func(
	context.Context,
	*customer.Customer,
	*validationframework.TenantedValidationContext,
	*errortypes.MultiError,
) error {
	return func(
		ctx context.Context,
		entity *customer.Customer,
		_ *validationframework.TenantedValidationContext,
		multiErr *errortypes.MultiError,
	) error {
		if factors == nil || entity.BillingProfile == nil ||
			entity.BillingProfile.FactoringCompanyID == nil ||
			entity.BillingProfile.FactoringCompanyID.IsNil() {
			return nil
		}
		factor, err := factors.GetByID(ctx, factoringcompanyports.GetFactoringCompanyByIDRequest{
			ID: *entity.BillingProfile.FactoringCompanyID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		})
		switch {
		case errortypes.IsNotFoundError(err):
			multiErr.Add("billingProfile.factoringCompanyId", errortypes.ErrInvalid,
				"That factoring company does not exist")
			return nil
		case err != nil:
			return err
		case !factor.IsActive():
			multiErr.Add("billingProfile.factoringCompanyId", errortypes.ErrInvalid,
				"That factoring company is inactive. Choose an active one")
		}
		return nil
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *customer.Customer,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *customer.Customer,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
