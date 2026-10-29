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
	case agenttemplate.TemplateDispatchAssistant,
		agenttemplate.TemplateBillingAssistant,
		agenttemplate.TemplateComplianceAssistant,
		agenttemplate.TemplateCustomerAssistant,
		agenttemplate.TemplateGeneralAssistant,
		agenttemplate.TemplateBillingException,
		agenttemplate.TemplateDispatchAssignment,
		agenttemplate.TemplateImportAssistant,
		agenttemplate.TemplateLoadMonitor,
		agenttemplate.TemplateShipmentIntake,
		agenttemplate.TemplateDetentionDesk,
		agenttemplate.TemplateCredentialDesk,
		agenttemplate.TemplateCustomerUpdateDesk,
		agenttemplate.TemplateCarrierRiskDesk,
		agenttemplate.TemplateIntakeDesk,
		agenttemplate.TemplateServiceFailureDesk,
		agenttemplate.TemplateEDIDesk,
		agenttemplate.TemplateFormulaAssistant,
		agenttemplate.TemplateBooksKeeper,
		agenttemplate.TemplateSettlementsClerk,
		agenttemplate.TemplateReceivables,
		agenttemplate.TemplateWorkforceCoordinator:
		return starterDataAccessForTrigger(t)
	default:
		return starterDataAccessForTrigger(t)
	}
}

func starterDataAccessForTrigger(t agenttemplate.Template) agentdefinition.DataAccessCeiling {
	if StarterTrigger(t) == agentdefinition.TriggerChat {
		return agentdefinition.DataAccessRestricted
	}

	return agentdefinition.DataAccessInternal
}
