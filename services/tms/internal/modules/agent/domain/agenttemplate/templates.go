package agenttemplate

type Template string

const (
	TemplateDispatchAssistant    = Template("DispatchAssistant")
	TemplateBillingAssistant     = Template("BillingAssistant")
	TemplateComplianceAssistant  = Template("ComplianceAssistant")
	TemplateCustomerAssistant    = Template("CustomerAssistant")
	TemplateGeneralAssistant     = Template("GeneralAssistant")
	TemplateBillingException     = Template("BillingException")
	TemplateDispatchAssignment   = Template("DispatchAssignment")
	TemplateImportAssistant      = Template("ImportAssistant")
	TemplateLoadMonitor          = Template("LoadMonitor")
	TemplateShipmentIntake       = Template("ShipmentIntake")
	TemplateCashApplication      = Template("CashApplication")
	TemplateDetentionDesk        = Template("DetentionDesk")
	TemplateCredentialDesk       = Template("CredentialDesk")
	TemplateCustomerUpdateDesk   = Template("CustomerUpdateDesk")
	TemplateCarrierRiskDesk      = Template("CarrierRiskDesk")
	TemplateIntakeDesk           = Template("IntakeDesk")
	TemplateLoadEntryCheck       = Template("LoadEntryCheck")
	TemplateServiceFailureDesk   = Template("ServiceFailureDesk")
	TemplateInsightAnalyst       = Template("InsightAnalyst")
	TemplateEDIDesk              = Template("EDIDesk")
	TemplateFormulaAssistant     = Template("FormulaAssistant")
	TemplateBooksKeeper          = Template("BooksKeeper")
	TemplateSettlementsClerk     = Template("SettlementsClerk")
	TemplateReceivables          = Template("Receivables")
	TemplateMasterDataSteward    = Template("MasterDataSteward")
	TemplateWorkforceCoordinator = Template("WorkforceCoordinator")
	TemplateFuelTaxClerk         = Template("FuelTaxClerk")
	TemplateReportAnalyst        = Template("ReportAnalyst")
	TemplateFactoringClerk       = Template("FactoringClerk")
)

func (t Template) IsValid() bool {
	switch t {
	case TemplateDispatchAssistant,
		TemplateBillingAssistant,
		TemplateComplianceAssistant,
		TemplateCustomerAssistant,
		TemplateGeneralAssistant,
		TemplateBillingException,
		TemplateDispatchAssignment,
		TemplateImportAssistant,
		TemplateLoadMonitor,
		TemplateShipmentIntake,
		TemplateCashApplication,
		TemplateDetentionDesk,
		TemplateCredentialDesk,
		TemplateCustomerUpdateDesk,
		TemplateCarrierRiskDesk,
		TemplateIntakeDesk,
		TemplateLoadEntryCheck,
		TemplateServiceFailureDesk,
		TemplateInsightAnalyst,
		TemplateEDIDesk,
		TemplateFormulaAssistant,
		TemplateBooksKeeper,
		TemplateSettlementsClerk,
		TemplateReceivables,
		TemplateMasterDataSteward,
		TemplateWorkforceCoordinator,
		TemplateFuelTaxClerk,
		TemplateReportAnalyst,
		TemplateFactoringClerk:
		return true
	default:
		return false
	}
}

func AllTemplates() []Template {
	return []Template{
		TemplateDispatchAssistant,
		TemplateBillingAssistant,
		TemplateComplianceAssistant,
		TemplateCustomerAssistant,
		TemplateGeneralAssistant,
		TemplateBillingException,
		TemplateDispatchAssignment,
		TemplateImportAssistant,
		TemplateLoadMonitor,
		TemplateShipmentIntake,
		TemplateCashApplication,
		TemplateDetentionDesk,
		TemplateCredentialDesk,
		TemplateCustomerUpdateDesk,
		TemplateCarrierRiskDesk,
		TemplateIntakeDesk,
		TemplateLoadEntryCheck,
		TemplateServiceFailureDesk,
		TemplateInsightAnalyst,
		TemplateEDIDesk,
		TemplateFormulaAssistant,
		TemplateBooksKeeper,
		TemplateSettlementsClerk,
		TemplateReceivables,
		TemplateMasterDataSteward,
		TemplateWorkforceCoordinator,
		TemplateFuelTaxClerk,
		TemplateReportAnalyst,
		TemplateFactoringClerk,
	}
}
