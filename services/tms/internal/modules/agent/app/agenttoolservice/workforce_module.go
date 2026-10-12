package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcetools"
)

func workforceToolProviders() []any {
	providers := make([]any, 0, 64)
	for _, group := range [][]any{
		workforcetools.SafetyToolProviders(),
		workforcetools.DrugAlcoholToolProviders(),
		workforcetools.LeaveToolProviders(),
		workforcetools.TrainingToolProviders(),
		workforcetools.ChecklistToolProviders(),
		workforcetools.ReviewToolProviders(),
		workforcetools.DqfToolProviders(),
		workforcetools.CredentialToolProviders(),
		workforcetools.InjuryToolProviders(),
		workforcetools.PtoToolProviders(),
		workforcetools.PermitToolProviders(),
		workforcetools.DriverPayReviewToolProviders(),
		workforcetools.PayrollToolProviders(),
	} {
		providers = append(providers, group...)
	}

	return providers
}
