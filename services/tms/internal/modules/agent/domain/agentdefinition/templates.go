package agentdefinition

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
		TemplateReportAnalyst:
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
	}
}

func (t Template) Label() string {
	switch t {
	case TemplateDispatchAssistant:
		return "Dispatch assistant"
	case TemplateBillingAssistant:
		return "Billing assistant"
	case TemplateComplianceAssistant:
		return "Compliance assistant"
	case TemplateCustomerAssistant:
		return "Customer assistant"
	case TemplateGeneralAssistant:
		return "General assistant"
	case TemplateBillingException:
		return "Billing exception agent"
	case TemplateDispatchAssignment:
		return "Dispatch coverage agent"
	case TemplateImportAssistant:
		return "Shipment import assistant"
	case TemplateLoadMonitor:
		return "Load monitor"
	case TemplateShipmentIntake:
		return "Shipment intake agent"
	case TemplateCashApplication:
		return "Cash application agent"
	case TemplateDetentionDesk:
		return "Detention desk"
	case TemplateCredentialDesk:
		return "Credential desk"
	case TemplateCustomerUpdateDesk:
		return "Customer update desk"
	case TemplateCarrierRiskDesk:
		return "Carrier risk desk"
	case TemplateIntakeDesk:
		return "Intake desk"
	case TemplateLoadEntryCheck:
		return "Load entry check"
	case TemplateServiceFailureDesk:
		return "Service failure desk"
	case TemplateInsightAnalyst:
		return "Insight analyst"
	case TemplateEDIDesk:
		return "EDI desk"
	case TemplateFormulaAssistant:
		return "Formula assistant"
	case TemplateBooksKeeper:
		return "Books keeper"
	case TemplateSettlementsClerk:
		return "Settlements clerk"
	case TemplateReceivables:
		return "Receivables assistant"
	case TemplateMasterDataSteward:
		return "Master data steward"
	case TemplateWorkforceCoordinator:
		return "Workforce coordinator"
	case TemplateFuelTaxClerk:
		return "Fuel and IFTA clerk"
	case TemplateReportAnalyst:
		return "Report analyst"
	default:
		return string(t)
	}
}

func (t Template) Description() string {
	switch t {
	case TemplateDispatchAssistant:
		return "Answers questions and acts on shipments, moves, drivers, and equipment."
	case TemplateBillingAssistant:
		return "Works the billing queue, charges, and invoices."
	case TemplateComplianceAssistant:
		return "Covers worker qualification, hours of service, and hazmat requirements."
	case TemplateCustomerAssistant:
		return "Handles customer records, quotes, and shipment status enquiries."
	case TemplateGeneralAssistant:
		return "Answers questions across the system without changing anything."
	case TemplateBillingException:
		return "Diagnoses blocked billing items as they appear and proposes how to clear them."
	case TemplateDispatchAssignment:
		return "Reviews uncovered moves on a schedule and proposes driver assignments."
	case TemplateImportAssistant:
		return "Turns a customer's shipment document into a reviewed, ready-to-save shipment."
	case TemplateLoadMonitor:
		return "Watches the board every quarter hour for late, stalled and uncovered loads, " +
			"and proposes what to do about each."
	case TemplateShipmentIntake:
		return "Turns each document as it is read into a quoted, ready-to-enter shipment, " +
			"and asks a person only about what it could not resolve."
	case TemplateCashApplication:
		return "Works each bank receipt that could not be matched on its own: finds the payment " +
			"or the invoices it pays and proposes the match, so the morning's reconciliation " +
			"is a review rather than a search."
	case TemplateDetentionDesk:
		return "Works each detention clock from the moment it starts: gets the notice out " +
			"while the window is open, says what the stay is going to cost, and puts a " +
			"held charge the evidence supports in front of a person for approval."
	case TemplateCredentialDesk:
		return "Takes a driver's papers in hand before they expire, chasing the renewal " +
			"and saying when somebody has to come off dispatch."
	case TemplateCustomerUpdateDesk:
		return "Tells each customer their freight arrived or left, as they asked to be " +
			"told, and stays quiet with the ones who did not."
	case TemplateCarrierRiskDesk:
		return "Reads every change on a carrier's authority, insurance and safety record " +
			"and says whether they can still be tendered freight."
	case TemplateIntakeDesk:
		return "Works the inbox: files what arrives against the right load, attaches the " +
			"paperwork, answers the status questions and puts a tender, emailed or sent " +
			"over EDI, in front of a person as a ready shipment."
	case TemplateLoadEntryCheck:
		return "Reads each new shipment as it is entered and flags what looks wrong — a " +
			"duplicate, a rate that disagrees with the lane, a stop out of order — before " +
			"anyone dispatches it."
	case TemplateServiceFailureDesk:
		return "Works every open service failure on a shipment as it is detected: finds the " +
			"cause, records it, and tells the customer when they asked to be told."
	case TemplateInsightAnalyst:
		return "Looks into each new insight as it is found: checks the figures against the " +
			"records, names the likely cause and says what a person should do about it."
	case TemplateEDIDesk:
		return "Diagnoses each EDI file that lands in quarantine: reads what the partner sent " +
			"and why it failed, says what has to change for it to process, and proposes " +
			"running it again once that is fixed."
	case TemplateFormulaAssistant:
		return "Helps write and explain the rating formulas that price freight, drafts and " +
			"revises rate agreements, and proposes rate changes, rate sheets and fuel index " +
			"prices for a person to approve."
	case TemplateBooksKeeper:
		return "Works out why a document did not reach the accounting system and what fixes " +
			"it, and retries it once it is fixed."
	case TemplateSettlementsClerk:
		return "Runs driver and carrier settlements: drafts the period's pay, sorts out what " +
			"is missing or held, matches carrier invoices, and puts every payment in front of " +
			"a person to approve."
	case TemplateReceivables:
		return "Works what customers owe once an invoice is out: applies payments and credit, " +
			"handles disputes and late charges, and says who to chase first."
	case TemplateMasterDataSteward:
		return "Keeps carriers, customers, commodities, hazardous materials, locations and " +
			"equipment right, files scanned paperwork and clears the attention feed, " +
			"proposing every change for a person to approve."
	case TemplateWorkforceCoordinator:
		return "Handles time off, leave, injuries, reviews, safety records, random testing " +
			"rounds, checklists and shipment permits, and proposes each change for a person " +
			"to approve."
	case TemplateFuelTaxClerk:
		return "Keeps the fuel tax record: records and corrects fuel purchases and card " +
			"statements, fills in state miles, and drafts the quarter's IFTA return for a " +
			"person to file."
	case TemplateReportAnalyst:
		return "Builds, runs and explains reports and dashboards, compares runs to say what " +
			"moved, and proposes schedules that email a report for a person to approve."
	default:
		return ""
	}
}
