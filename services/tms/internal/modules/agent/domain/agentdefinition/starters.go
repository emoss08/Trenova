package agentdefinition

import "slices"

const (
	MaxStarters = 4
	minStarters = 3
)

type Starter struct {
	Label  string `json:"label"`
	Prompt string `json:"prompt"`
}

type starterQuestion struct {
	starter Starter
	tools   []string
}

func ask(label, prompt string, tools ...string) starterQuestion {
	return starterQuestion{starter: Starter{Label: label, Prompt: prompt}, tools: tools}
}

func (q starterQuestion) answerableWith(tools []string) bool {
	if len(q.tools) == 0 {
		return true
	}

	for _, tool := range q.tools {
		if IsCoreTool(tool) || slices.Contains(tools, tool) {
			return true
		}
	}

	return false
}

var howToStarters = []starterQuestion{
	ask(
		"How do I create a shipment?",
		"How do I create a new shipment, step by step?",
	),
	ask(
		"Where do I approve an agent's change?",
		"Where do I review and approve a change an agent has proposed?",
	),
	ask(
		"How do I add a new driver?",
		"How do I add a new driver and record their credentials?",
	),
	ask(
		"How do I set up customer rates?",
		"How do I set up a rate matrix for a customer?",
	),
}

var templateStarters = map[Template][]starterQuestion{
	TemplateDispatchAssistant: {
		ask(
			"Which moves are unassigned today?",
			"Which moves are still unassigned for today, and what is each one waiting on?",
			"shipment_list",
		),
		ask(
			"Who should take the next uncovered move?",
			"Rank the best available drivers for the next uncovered move and explain the top pick.",
			"dispatch_rank_candidates",
		),
		ask(
			"What is picking up today?",
			"Which shipments are scheduled to pick up today?",
			"shipment_list",
		),
		ask(
			"Who is off this week?",
			"Which drivers have time off booked this week?",
			"worker_list_time_off",
		),
	},
	TemplateBillingAssistant: {
		ask(
			"What is ready to transfer to billing?",
			"Which delivered shipments are ready to transfer to billing, and which cannot go yet?",
			"billing_list_transfer_candidates",
		),
		ask(
			"Which billing items are stuck the longest?",
			"Which billing queue items have been blocked the longest, and why?",
			"billing_list_queue_items",
			"billing_start_review",
			"shipment_request_documents",
		),
		ask(
			"Which deliveries are missing paperwork?",
			"Which delivered shipments are still missing the documents billing needs?",
			"shipment_request_documents",
		),
		ask(
			"Which billing reports can I run?",
			"Which reports can I run for billing, and what does each one show?",
			"report_list",
		),
	},
	TemplateComplianceAssistant: {
		ask(
			"Whose credentials expire soon?",
			"Which drivers have a credential expiring in the next 30 days?",
			toolListExpiringQualifications,
		),
		ask(
			"Who is off this week?",
			"Which drivers have time off booked this week?",
			"worker_list_time_off",
		),
		ask(
			"Which equipment is out of service?",
			"Which tractors and trailers are out of service right now?",
			"equipment_list_tractors", "equipment_list_trailers",
		),
		ask(
			"Which compliance reports can I run?",
			"Which reports can I run for safety and compliance?",
			"report_list",
		),
	},
	TemplateCustomerAssistant: {
		ask(
			"What delivers today?",
			"Which shipments are scheduled to deliver today?",
			"shipment_list",
		),
		ask(
			"Which shipments are running late?",
			"Which shipments are running late right now, and for which customers?",
			"shipment_list",
		),
		ask(
			"Which invoices are still open?",
			"Which invoices are still open, oldest first?",
			"invoice_list",
		),
		ask(
			"Look up a customer",
			"Find a customer and summarize their account for me.",
			toolListCustomers,
		),
	},
	TemplateGeneralAssistant: howToStarters,
	TemplateBillingException: {
		ask(
			"Which billing items are blocked, and why?",
			"Which billing items are blocked right now, and what is blocking each one?",
			"billing_list_queue_items", "billing_start_review", "shipment_list",
		),
		ask(
			"Which shipments are missing documents?",
			"Which shipments are held from billing for missing documents?",
			"shipment_request_documents",
		),
		ask(
			"Which charge codes need correcting?",
			"Which shipments have a charge code that looks wrong, and what should it be?",
			"billing_correct_charge_codes",
		),
	},
	TemplateDispatchAssignment: {
		ask(
			"Which moves are unassigned today?",
			"Which moves on the dispatch board are still unassigned for today?",
			"dispatch_get_board",
		),
		ask(
			"Who should take the next uncovered move?",
			"Rank the best available drivers for the next uncovered move and explain the top pick.",
			"dispatch_rank_candidates",
		),
		ask(
			"Which carriers could cover an open load?",
			"Which carriers could cover today's open loads, and at what rate?",
			"carrier_shop",
		),
	},
	TemplateImportAssistant: {
		ask(
			"Finish this shipment",
			"Help me finish this shipment from the document: what is still missing?",
			"intake_set_required_field", "intake_set_stop_location",
		),
		ask(
			"Match the stops to locations",
			"Match each stop on the document to one of our locations.",
			"location_list", "intake_set_stop_location",
		),
		ask(
			"Which rating method should I use?",
			"Which rating methods can this shipment be priced with?",
			toolListFormulaTemplates,
		),
	},
	TemplateLoadMonitor: {
		ask(
			"Which loads are running late?",
			"Which loads on the board are running late right now?",
			"dispatch_get_board", "shipment_get_tracking",
		),
		ask(
			"Who is low on hours?",
			"Which drivers are running low on hours of service today?",
			toolGetWorkerHOS,
		),
		ask(
			"Is weather affecting any loads?",
			"Are any weather alerts affecting loads that are moving today?",
			"insight_list_weather_alerts",
		),
		ask(
			"Which service failures are open?",
			"Which service failures are still open, and what caused them?",
			"exception_list_service_failures",
		),
	},
	TemplateShipmentIntake: {
		ask(
			"Help me enter a shipment",
			"Help me enter a new shipment from a customer's document.",
			"shipment_create",
		),
		ask(
			"Quote a shipment",
			"Quote a shipment for me before I enter it.",
			toolRateQuote,
		),
		ask(
			"Which equipment types can I use?",
			"Which equipment types can I put on a shipment?",
			"equipment_list_types",
		),
	},
	TemplateCashApplication: {
		ask(
			"Which receipts still need matching?",
			"Which bank receipts still need to be matched to a payment?",
			"receivable_list_bank_receipt_exceptions",
		),
		ask(
			"Which invoices are past due?",
			"Which invoices are past due, and for which customers?",
			"invoice_list",
		),
		ask(
			"Which payments came in this week?",
			"Which customer payments were recorded this week?",
			"receivable_list_customer_payments",
		),
	},
	TemplateDetentionDesk: {
		ask(
			"Where is detention building up?",
			"Which stops are running up detention right now?",
			"detention_list_occurrences",
		),
		ask(
			"Which notices still need to go out?",
			"Which detention cases still need a notice sent to the customer?",
			"detention_list_occurrences",
		),
		ask(
			"What should be escalated?",
			"Which detention cases should be escalated, and why?",
			"detention_escalate",
		),
		ask(
			"Which charges need approval?",
			"Which detention charges are waiting on approval, and does the evidence support billing them?",
			"detention_approve",
		),
	},
	TemplateCredentialDesk: {
		ask(
			"Whose credentials expire soon?",
			"Which drivers have a credential expiring in the next 30 days?",
			toolListExpiringQualifications,
		),
		ask(
			"Who needs a renewal request?",
			"Which drivers still need a renewal request for papers that are coming due?",
			toolListExpiringQualifications, "worker_request_credential_renewal",
		),
		ask(
			"Look up a driver's papers",
			"Find a driver and tell me which of their credentials are current.",
			"worker_get_credential", toolListWorkers,
		),
	},
	TemplateCustomerUpdateDesk: {
		ask(
			"Draft a shipment update",
			"Draft an update to a customer about where their shipment is.",
			toolSendCustomerEmail,
		),
		ask(
			"Look up a shipment's progress",
			"Find a shipment and tell me where it is and when it should arrive.",
			"shipment_get_tracking", "shipment_get",
		),
		ask(
			"How do customers choose their updates?",
			"How does a customer choose which shipment updates they receive?",
		),
	},
	TemplateCarrierRiskDesk: {
		ask(
			"Which carriers should I watch?",
			"Which carriers have a recent change to their authority, insurance or safety record?",
			toolListCarrierIntelEvents, "carrier_get_intelligence_event", toolListCarriers,
		),
		ask(
			"Look up a carrier",
			"Find a carrier and tell me whether they can still be given freight.",
			"carrier_get", toolListCarriers,
		),
		ask(
			"How is carrier risk monitored?",
			"How does Trenova monitor a carrier's authority, insurance and safety record?",
		),
	},
	TemplateIntakeDesk: {
		ask(
			"What in the inbox is unfiled?",
			"Which messages in the inbox have not been filed against a load yet?",
			"inbox_list_inbound_messages",
		),
		ask(
			"Which messages need a reply?",
			"Which inbound messages are still waiting on a reply?",
			"inbox_list_inbound_messages",
		),
		ask(
			"Find a shipment",
			"Find a shipment by its reference and tell me where it stands.",
			"shipment_list",
		),
	},
	TemplateLoadEntryCheck: {
		ask(
			"Is this shipment a duplicate?",
			"Look for another shipment with the same BOL or PRO as this one, and tell me "+
				"whether it is the same load.",
			"shipment_list",
		),
		ask(
			"Does the rate fit the lane?",
			"Compare this shipment's rate with a fresh quote for the same customer and stops.",
			toolExplainRate, toolRateQuote,
		),
		ask(
			"Which hold reasons can I use?",
			"Which hold reasons are set up, and what does each one block?",
			"masterdata_list_hold_reasons",
		),
	},
	TemplateServiceFailureDesk: {
		ask(
			"Which service failures are open?",
			"Which service failures are still open, and what caused them?",
			"exception_list_service_failures",
		),
		ask(
			"Why was this stop late?",
			"Look into this shipment's late stop and tell me what caused it.",
			"exception_get_service_failure", "shipment_get_tracking",
		),
		ask(
			"Which reason codes can I use?",
			"Which service failure reason codes are set up, and when does each one apply?",
			"exception_list_service_failure_reason_codes",
		),
	},
	TemplateInsightAnalyst: {
		ask(
			"What stands out today?",
			"Which open insights need my attention today?",
			toolListInsights,
		),
		ask(
			"Does this finding hold up?",
			"Check this insight against the records behind it and tell me whether it holds up.",
			toolGetInsight,
		),
		ask(
			"Which reports can I run?",
			"Which reports can I run, and what does each one show?",
			"report_list",
		),
	},
	TemplateEDIDesk: {
		ask(
			"Which EDI files are quarantined?",
			"Which inbound EDI files are in quarantine, and why did each one fail?",
			"edi_list_inbound_files",
		),
		ask(
			"Why did this EDI file fail?",
			"Tell me why this EDI file could not be processed and what would fix it.",
			"edi_get_inbound_file",
		),
		ask(
			"Is a partner ready for EDI?",
			"Is this trading partner set up to send us EDI, and what is still missing?",
			"edi_get_partner",
		),
		ask(
			"Which tenders were rejected?",
			"Which EDI load tenders were rejected or failed, and why?",
			"edi_list_transfers",
		),
	},
	TemplateFactoringClerk: {
		ask(
			"Which invoices are ready to factor?",
			"Which invoices are ready to sell to the factor, and how much is ready in all?",
			"factoring_list_readiness",
		),
		ask(
			"What holds an invoice back?",
			"Why is this invoice not ready to factor, and what clears it?",
			"factoring_get_readiness",
		),
		ask(
			"Which customers have no notice?",
			"Which factored customers have no acknowledged notice of assignment?",
			"factoring_list_noas",
		),
		ask(
			"What are the factor's terms?",
			"What are the advance, reserve and fee terms of the factoring agreement in force?",
			"factoring_get_agreement",
		),
		ask(
			"Which schedules failed?",
			"Which factoring schedules failed to send, and why?",
			"factoring_list_submissions",
		),
		ask(
			"What is each factor holding?",
			"How much reserve does each factor still hold, and what fees have we paid?",
			"factoring_get_reserve_balance",
		),
	},
	TemplateBooksKeeper: {
		ask(
			"What did not reach the books?",
			"Which documents did not reach the accounting system, and why?",
			"accounting_list_sync_records",
		),
		ask(
			"Did this invoice sync?",
			"Did this invoice reach the accounting system, and if not, what is holding it?",
			"accounting_get_sync_state",
		),
		ask(
			"Which mappings are missing?",
			"Which mappings are still missing, and which documents are waiting on them?",
			"accounting_list_mapping_gaps",
		),
		ask(
			"Is the accounting connection healthy?",
			"Is the accounting system connected and answering, and how many documents are queued?",
			"accounting_get_connection",
		),
		ask(
			"Which payments from the books wait?",
			"Which payments recorded in the accounting system wait to be applied in Trenova, and why?",
			"accounting_list_inbound_changes",
		),
		ask(
			"Where do the books differ?",
			"Which documents differ between the accounting system and Trenova, and how should each be fixed?",
			"accounting_list_drift_findings",
		),
	},
	TemplateSettlementsClerk: {
		ask(
			"Which settlements need attention?",
			"Which driver settlements have exceptions or open disputes, and what has to be "+
				"settled before they can be paid?",
			"settlement_list_driver",
		),
		ask(
			"What has a driver earned?",
			"What has this driver earned that is not on a settlement yet, and what advances "+
				"do they still owe?",
			"settlement_get_driver_earnings",
		),
		ask(
			"Why is a load missing from pay?",
			"Why is this load missing from the driver's settlement?",
			"settlement_list_pay_events",
		),
		ask(
			"Which carrier invoices do not match?",
			"Which carrier invoices differ from what the load was expected to cost, and by "+
				"how much?",
			"carrier_list_invoice_matches",
		),
	},
	TemplateReceivables: {
		ask(
			"Who owes us the most?",
			"Which customers owe us the most, and how late is it?",
			"receivable_get_aging",
		),
		ask(
			"Who should I chase today?",
			"Which overdue invoices should I chase today, most urgent first?",
			"receivable_list_collection_worklist",
		),
		ask(
			"Which invoices are disputed?",
			"Which invoices are in dispute, and what is each customer's reason?",
			"invoice_list_disputes",
		),
		ask(
			"Which payments are unapplied?",
			"Which customer payments still have cash that is not applied to an invoice?",
			"receivable_list_customer_payments",
		),
	},
	TemplateMasterDataSteward: {
		ask(
			"Is this carrier already on file?",
			"Is this carrier already on file under its DOT or MC number, and is it active?",
			toolListCarriers,
		),
		ask(
			"What paperwork needs filing?",
			"Which scanned documents are waiting to be filed, and which record does each "+
				"belong on?",
			"inbox_list_capture_batches",
		),
		ask(
			"What is on the attention feed?",
			"Which watchtower items are open, and which are already dealt with?",
			"exception_list_watchtower_items",
		),
		ask(
			"Add a hazardous material",
			"Add a hazardous material from its UN number and safety data sheet.",
			"masterdata_create_hazardous_material",
		),
	},
	TemplateWorkforceCoordinator: {
		ask(
			"Who is asking for time off?",
			"Which time-off requests are waiting for a decision, and what is each worker "+
				"covering on those days?",
			"worker_list_time_off",
		),
		ask(
			"Which leave cases are open?",
			"Which leave cases are open, and what does each one still need?",
			"worker_list_leave_cases",
		),
		ask(
			"Draw a random testing round",
			"Draw this period's random drug and alcohol testing round from our pool.",
			"worker_run_dot_random_draw",
		),
		ask(
			"Which permits are missing?",
			"Which permit requirements on this shipment are not covered by an active permit?",
			"shipment_list_permits",
		),
	},
	TemplateFuelTaxClerk: {
		ask(
			"Is the IFTA return ready?",
			"Is last quarter's IFTA return drafted, and what still blocks finalizing it?",
			"fuel_list_ifta_returns",
		),
		ask(
			"Which fuel imports are held?",
			"Which fuel card statements have rows held back, and why?",
			"fuel_list_purchase_imports",
		),
		ask(
			"Which fuel cards are unassigned?",
			"Which fuel cards are not assigned to a tractor or a driver?",
			"fuel_list_cards",
		),
		ask(
			"Record a fuel receipt",
			"Record this fuel receipt against the tractor that bought it.",
			"fuel_record_purchase",
		),
	},
	TemplateReportAnalyst: {
		ask(
			"Which report answers this?",
			"Which of our reports shows on-time delivery by customer, and what does it contain?",
			"report_list",
		),
		ask(
			"What changed since last week?",
			"Run this report again and tell me what moved since last week's run.",
			"report_compare_runs",
		),
		ask(
			"Build a report",
			"Build a report of delivered shipments by customer for last month.",
			"report_create",
		),
		ask(
			"Email a report every week",
			"Email this report to the operations team every Monday morning.",
			"report_create_schedule",
		),
	},
	TemplateFormulaAssistant: {
		ask(
			"Write a formula",
			"Write a formula that charges per mile with a fuel surcharge and a minimum charge.",
			"formula_propose", "formula_describe_schema",
		),
		ask(
			"Explain this formula",
			"Explain what this formula charges, term by term.",
			"formula_describe_schema",
		),
		ask(
			"What would this charge?",
			"What would this formula charge for a 500 mile load?",
			"formula_test_expression",
		),
		ask(
			"How was this rate worked out?",
			"Explain how this shipment's rate was worked out, charge by charge.",
			toolExplainRate,
		),
		ask(
			"Draft a rate agreement",
			"Draft next year's rate agreement for this customer from its current one.",
			"rate_duplicate_agreement", "rate_update_agreement_draft",
		),
	},
}

var toolFamilies = [][]starterQuestion{
	{
		ask(
			"Build a report",
			"Build a report of in-transit shipments by customer.",
			"report_create",
		),
		ask(
			"Which reports can I run?",
			"Which reports can I run, and what does each one show?",
			"report_list",
		),
		ask(
			"What can I report on?",
			"Which datasets can I build a report from, and what does each one hold?",
			"report_list_datasets", "report_get_dataset",
		),
		ask(
			"Change one of my reports",
			"Add a column for delivery date to one of my reports.",
			"report_update",
		),
	},
	{
		ask(
			"Add a widget to my home page",
			"Add a widget to my home page that shows today's late shipments.",
			"home_add_widget",
		),
		ask(
			"What is on my home page?",
			"What widgets are on my home page right now?",
			"home_list_widgets", "home_get_layout",
		),
	},
	{
		ask(
			"Build a dashboard",
			"Build a dashboard of this month's on-time delivery: its headline figures "+
				"first, as KPI tiles if the report gives them, then the detail table.",
			"report_create_dashboard", "report_add_dashboard_tile",
		),
		ask(
			"Which dashboards do we have?",
			"Which dashboards do we have, and what does each one track?",
			"report_list_dashboards",
		),
	},
	{
		ask(
			"Which moves are unassigned today?",
			"Which moves on the dispatch board are still unassigned for today?",
			"dispatch_get_board",
		),
		ask(
			"Who should take the next uncovered move?",
			"Rank the best available drivers for the next uncovered move and explain the top pick.",
			"dispatch_rank_candidates",
		),
		ask(
			"Plan today's dispatch",
			"Plan driver assignments for today's uncovered moves.",
			"dispatch_plan",
		),
	},
	{
		ask(
			"Which loads are running late?",
			"Which loads are running late right now, based on tracking?",
			"shipment_get_tracking", "equipment_list_positions",
		),
		ask(
			"Is weather affecting any loads?",
			"Are any weather alerts affecting loads that are moving today?",
			"insight_list_weather_alerts",
		),
	},
	{
		ask(
			"What is picking up today?",
			"Which shipments are scheduled to pick up today?",
			"shipment_list",
		),
		ask(
			"Find a shipment",
			"Find a shipment by its reference and tell me where it stands.",
			"shipment_list", "shipment_get",
		),
	},
	{
		ask(
			"Which carriers could cover an open load?",
			"Which carriers could cover today's open loads, and at what rate?",
			"carrier_shop",
		),
		ask(
			"Which carriers should I watch?",
			"Which carriers have a recent change to their authority, insurance or safety record?",
			toolListCarrierIntelEvents, "carrier_get_intelligence_event",
		),
		ask(
			"Look up a carrier",
			"Find a carrier and tell me whether they can still be given freight.",
			toolListCarriers, "carrier_get",
		),
	},
	{
		ask(
			"Whose credentials expire soon?",
			"Which drivers have a credential expiring in the next 30 days?",
			toolListExpiringQualifications,
		),
		ask(
			"Who is low on hours?",
			"Which drivers are running low on hours of service today?",
			toolGetWorkerHOS,
		),
		ask(
			"Who is off this week?",
			"Which drivers have time off booked this week?",
			"worker_list_time_off",
		),
		ask(
			"Look up a driver",
			"Find a driver and summarize where they are and what they are qualified for.",
			toolSearchWorker, toolListWorkers, toolGetWorker,
		),
	},
	{
		ask(
			"Which equipment is out of service?",
			"Which tractors and trailers are out of service right now?",
			"equipment_list_tractors", "equipment_list_trailers",
		),
	},
	{
		ask(
			"Which billing items are stuck the longest?",
			"Which billing queue items have been blocked the longest, and why?",
			"billing_start_review", "billing_correct_charge_codes",
		),
		ask(
			"Which deliveries are missing paperwork?",
			"Which delivered shipments are still missing the documents billing needs?",
			"shipment_request_documents",
		),
	},
	{
		ask(
			"Which invoices are past due?",
			"Which invoices are past due, and for which customers?",
			"invoice_list", toolInvoiceGet,
		),
		ask(
			"Which receipts still need matching?",
			"Which bank receipts still need to be matched to a payment?",
			"receivable_list_bank_receipt_exceptions",
		),
		ask(
			"Which payments came in this week?",
			"Which customer payments were recorded this week?",
			"receivable_list_customer_payments",
		),
		ask(
			"Who should I chase today?",
			"Which overdue invoices should I chase today, most urgent first?",
			"receivable_list_collection_worklist",
		),
	},
	{
		ask(
			"Which settlements need attention?",
			"Which driver settlements have exceptions or open disputes, and what has to be "+
				"settled before they can be paid?",
			"settlement_list_driver",
		),
		ask(
			"Which carrier invoices do not match?",
			"Which carrier invoices differ from what the load was expected to cost, and by "+
				"how much?",
			"carrier_list_invoice_matches",
		),
	},
	{
		ask(
			"Where is detention building up?",
			"Which stops are running up detention right now?",
			"detention_list_occurrences",
		),
		ask(
			"Which service failures are open?",
			"Which service failures are still open, and what caused them?",
			"exception_list_service_failures",
		),
	},
	{
		ask(
			"What in the inbox is unfiled?",
			"Which messages in the inbox have not been filed against a load yet?",
			"inbox_list_inbound_messages",
		),
	},
	{
		ask(
			"Help me enter a shipment",
			"Help me enter a new shipment from a customer's document.",
			"shipment_create",
		),
		ask(
			"Quote a shipment",
			"Quote a shipment for me before I enter it.",
			toolRateQuote,
		),
	},
	{
		ask(
			"Look up a customer",
			"Find a customer and summarize their account for me.",
			toolListCustomers, toolGetCustomer,
		),
	},
	{
		ask(
			"What stands out today?",
			"Which open insights need my attention today?",
			toolListInsights,
		),
		ask(
			"Alert me to a change",
			"Alert me whenever a shipment is put on hold.",
			"report_create_table_change_alert",
		),
	},
}

func (d *Definition) Starters() []Starter {
	return StartersFor(d.Template, d.ToolNames)
}

func StartersFor(template Template, tools []string) []Starter {
	picked := make([]Starter, 0, MaxStarters)
	picked = appendAnswerable(picked, templateStarters[template], tools, MaxStarters)
	if len(picked) < minStarters {
		picked = appendFromFamilies(picked, tools, MaxStarters)
	}
	if len(picked) == 0 {
		picked = appendAnswerable(picked, howToStarters, tools, minStarters)
	}

	return picked
}

func appendAnswerable(
	picked []Starter,
	questions []starterQuestion,
	tools []string,
	limit int,
) []Starter {
	for _, question := range questions {
		if len(picked) >= limit {
			break
		}
		if question.answerableWith(tools) {
			picked = appendUnique(picked, question.starter)
		}
	}

	return picked
}

func appendFromFamilies(picked []Starter, tools []string, limit int) []Starter {
	for round := 0; len(picked) < limit; round++ {
		progressed := false
		for _, family := range toolFamilies {
			if len(picked) >= limit {
				break
			}
			question, ok := nthAnswerable(family, tools, round)
			if !ok {
				continue
			}
			progressed = true
			picked = appendUnique(picked, question.starter)
		}
		if !progressed {
			break
		}
	}

	return picked
}

func nthAnswerable(family []starterQuestion, tools []string, n int) (starterQuestion, bool) {
	seen := 0
	for _, question := range family {
		if !question.answerableWith(tools) {
			continue
		}
		if seen == n {
			return question, true
		}
		seen++
	}

	return starterQuestion{}, false
}

func appendUnique(picked []Starter, starter Starter) []Starter {
	for _, existing := range picked {
		if existing.Prompt == starter.Prompt || existing.Label == starter.Label {
			return picked
		}
	}

	return append(picked, starter)
}
