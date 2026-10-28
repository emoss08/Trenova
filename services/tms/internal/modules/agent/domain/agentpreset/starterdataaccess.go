package agentpreset

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
)

func StarterDataAccess(t agenttemplate.Template) agentdefinition.DataAccessCeiling {
	switch t {
	case agenttemplate.TemplateCashApplication,
		agenttemplate.TemplateLoadEntryCheck,
		agenttemplate.TemplateInsightAnalyst,
		agenttemplate.TemplateReportAnalyst:
		return agentdefinition.DataAccessRestricted
	case agenttemplate.TemplateMasterDataSteward,
		agenttemplate.TemplateFuelTaxClerk,
		agenttemplate.TemplateFactoringClerk:
		return agentdefinition.DataAccessInternal
	default:
		if StarterTrigger(t) == agentdefinition.TriggerChat {
			return agentdefinition.DataAccessRestricted
		}

		return agentdefinition.DataAccessInternal
	}
}
