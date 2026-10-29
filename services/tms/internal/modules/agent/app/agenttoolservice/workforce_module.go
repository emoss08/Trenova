package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/drugalcoholtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/payrolltools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/permittools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/qualificationtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/reviewtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/safetytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/timeofftools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/trainingtools"
)

func workforceToolProviders() []any {
	providers := make([]any, 0, 64)
	for _, group := range [][]any{
		safetytools.SafetyToolProviders(),
		drugalcoholtools.DrugAlcoholToolProviders(),
		timeofftools.LeaveToolProviders(),
		trainingtools.TrainingToolProviders(),
		trainingtools.ChecklistToolProviders(),
		reviewtools.ReviewToolProviders(),
		qualificationtools.DqfToolProviders(),
		qualificationtools.CredentialToolProviders(),
		safetytools.InjuryToolProviders(),
		timeofftools.PtoToolProviders(),
		permittools.PermitToolProviders(),
		payrolltools.DriverPayReviewToolProviders(),
		payrolltools.PayrollToolProviders(),
	} {
		providers = append(providers, group...)
	}

	return providers
}
