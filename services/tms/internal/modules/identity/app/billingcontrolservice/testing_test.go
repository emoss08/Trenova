package billingcontrolservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*billingcontrol.BillingControl]().
			WithModelName("BillingControl").
			WithCustomRule(createTransferAutomationRule()).
			WithCustomRule(createInvoiceAutomationRule()).
			WithCustomRule(createRequirementEnforcementRule()).
			WithCustomRule(createRateValidationRule()).
			Build(),
	}
}
