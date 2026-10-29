package billingcontrolservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/pkg/validationframework"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*billingcontrol.BillingControl]().
			WithModelName("BillingControl").
			WithCustomRule(createTransferAutomationRule()).
			WithCustomRule(createInvoiceAutomationRule()).
			WithCustomRule(createRequirementEnforcementRule()).
			WithCustomRule(createRateValidationRule()).
			Build(),
	}
}
