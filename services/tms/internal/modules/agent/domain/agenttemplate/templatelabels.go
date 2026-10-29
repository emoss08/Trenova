package agenttemplate

var templateLabels = map[Template]string{
	TemplateDispatchAssistant:    "Dispatch assistant",
	TemplateBillingAssistant:     "Billing assistant",
	TemplateComplianceAssistant:  "Compliance assistant",
	TemplateCustomerAssistant:    "Customer assistant",
	TemplateGeneralAssistant:     "General assistant",
	TemplateBillingException:     "Billing exception agent",
	TemplateDispatchAssignment:   "Dispatch coverage agent",
	TemplateImportAssistant:      "Shipment import assistant",
	TemplateLoadMonitor:          "Load monitor",
	TemplateShipmentIntake:       "Shipment intake agent",
	TemplateCashApplication:      "Cash application agent",
	TemplateDetentionDesk:        "Detention desk",
	TemplateCredentialDesk:       "Credential desk",
	TemplateCustomerUpdateDesk:   "Customer update desk",
	TemplateCarrierRiskDesk:      "Carrier risk desk",
	TemplateIntakeDesk:           "Intake desk",
	TemplateLoadEntryCheck:       "Load entry check",
	TemplateServiceFailureDesk:   "Service failure desk",
	TemplateInsightAnalyst:       "Insight analyst",
	TemplateEDIDesk:              "EDI desk",
	TemplateFormulaAssistant:     "Formula assistant",
	TemplateBooksKeeper:          "Books keeper",
	TemplateSettlementsClerk:     "Settlements clerk",
	TemplateReceivables:          "Receivables assistant",
	TemplateMasterDataSteward:    "Master data steward",
	TemplateWorkforceCoordinator: "Workforce coordinator",
	TemplateFuelTaxClerk:         "Fuel and IFTA clerk",
	TemplateReportAnalyst:        "Report analyst",
	TemplateFactoringClerk:       "Factoring clerk",
}

var templateDescriptions = map[Template]string{
	TemplateDispatchAssistant: "Answers questions and acts on shipments, moves, drivers, and " +
		"equipment.",
	TemplateBillingAssistant: "Works the billing queue, charges, and invoices.",
	TemplateComplianceAssistant: "Covers worker qualification, hours of service, and hazmat " +
		"requirements.",
	TemplateCustomerAssistant: "Handles customer records, quotes, and shipment status " +
		"enquiries.",
	TemplateGeneralAssistant: "Answers questions across the system without changing " +
		"anything.",
	TemplateBillingException: "Diagnoses blocked billing items as they appear and proposes " +
		"how to clear them.",
	TemplateDispatchAssignment: "Reviews uncovered moves on a schedule and proposes driver " +
		"assignments.",
	TemplateImportAssistant: "Turns a customer's shipment document into a reviewed, " +
		"ready-to-save shipment.",
	TemplateLoadMonitor: "Watches the board every quarter hour for late, stalled and " +
		"uncovered loads, and proposes what to do about each.",
	TemplateShipmentIntake: "Turns each document as it is read into a quoted, " +
		"ready-to-enter shipment, and asks a person only about what it could not resolve.",
	TemplateCashApplication: "Works each bank receipt that could not be matched on its own: " +
		"finds the payment or the invoices it pays and proposes the match, so the morning's " +
		"reconciliation is a review rather than a search.",
	TemplateDetentionDesk: "Works each detention clock from the moment it starts: gets " +
		"the notice out while the window is open, says what the stay is going to cost, and puts " +
		"a held charge the evidence supports in front of a person for approval.",
	TemplateCredentialDesk: "Takes a driver's papers in hand before they expire, chasing " +
		"the renewal and saying when somebody has to come off dispatch.",
	TemplateCustomerUpdateDesk: "Tells each customer their freight arrived or left, as they " +
		"asked to be told, and stays quiet with the ones who did not.",
	TemplateCarrierRiskDesk: "Reads every change on a carrier's authority, insurance and " +
		"safety record and says whether they can still be tendered freight.",
	TemplateIntakeDesk: "Works the inbox: files what arrives against the right load, " +
		"attaches the paperwork, answers the status questions and puts a tender, emailed or sent " +
		"over EDI, in front of a person as a ready shipment.",
	TemplateLoadEntryCheck: "Reads each new shipment as it is entered and flags what looks " +
		"wrong — a duplicate, a rate that disagrees with the lane, a stop out of order — before " +
		"anyone dispatches it.",
	TemplateServiceFailureDesk: "Works every open service failure on a shipment as it is " +
		"detected: finds the cause, records it, and tells the customer when they asked to be " +
		"told.",
	TemplateInsightAnalyst: "Looks into each new insight as it is found: checks the " +
		"figures against the records, names the likely cause and says what a person should do " +
		"about it.",
	TemplateEDIDesk: "Diagnoses each EDI file that lands in quarantine: reads what " +
		"the partner sent and why it failed, says what has to change for it to process, and " +
		"proposes running it again once that is fixed. Asked, it reports which partners have not " +
		"acknowledged what was sent, previews a 204, 214 or 990 before it goes, and reads pasted " +
		"X12.",
	TemplateFormulaAssistant: "Helps write and explain the rating formulas that price " +
		"freight, drafts and revises rate agreements, and proposes rate changes, rate sheets and " +
		"fuel index prices for a person to approve.",
	TemplateBooksKeeper: "Works out why a document did not reach the accounting system " +
		"and what fixes it, and retries it once it is fixed.",
	TemplateSettlementsClerk: "Runs driver and carrier settlements: drafts the period's pay, " +
		"sorts out what is missing or held, matches carrier invoices, and puts every payment in " +
		"front of a person to approve.",
	TemplateReceivables: "Works what customers owe once an invoice is out: applies " +
		"payments and credit, handles disputes and late charges, and says who to chase first.",
	TemplateMasterDataSteward: "Keeps carriers, customers, commodities, hazardous materials, " +
		"locations and equipment right, files scanned paperwork and clears the attention feed, " +
		"proposing every change for a person to approve.",
	TemplateWorkforceCoordinator: "Handles time off, leave, injuries, reviews, safety records, " +
		"random testing rounds, checklists and shipment permits, and proposes each change for a " +
		"person to approve.",
	TemplateFuelTaxClerk: "Keeps the fuel tax record: records and corrects fuel " +
		"purchases and card statements, fills in state miles, and drafts the quarter's IFTA " +
		"return for a person to file.",
	TemplateReportAnalyst: "Builds, runs and explains reports and dashboards, compares " +
		"runs to say what moved, and proposes schedules that email a report for a person to " +
		"approve.",
	TemplateFactoringClerk: "Gets factored invoices ready to sell and sends them: chases " +
		"the notices of assignment and documents the factor needs, proposes one schedule per " +
		"factor for the invoices that are ready, follows up schedules that fail to send or go " +
		"unfunded, proposes booking the factor's funding and reserve releases, and warns before " +
		"a chargeback.",
}

func (t Template) Label() string {
	if label, ok := templateLabels[t]; ok {
		return label
	}
	return string(t)
}

func (t Template) Description() string {
	return templateDescriptions[t]
}
