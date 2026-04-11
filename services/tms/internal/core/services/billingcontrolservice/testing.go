package billingcontrolservice

import (
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/validationframework"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*tenant.BillingControl]().
			WithModelName("BillingControl").
			WithCustomRule(createTransferAutomationRule()).
			WithCustomRule(createInvoiceAutomationRule()).
			WithCustomRule(createRequirementEnforcementRule()).
			WithCustomRule(createRateValidationRule()).
			Build(),
	}
}
