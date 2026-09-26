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
			"list_shipments", "search_shipments",
		),
		ask(
			"Who should take the next uncovered move?",
			"Rank the best available drivers for the next uncovered move and explain the top pick.",
			"rank_move_candidates",
		),
		ask(
			"What is picking up today?",
			"Which shipments are scheduled to pick up today?",
			"list_shipments",
		),
		ask(
			"Who is off this week?",
			"Which drivers have time off booked this week?",
			"list_time_off",
		),
	},
	TemplateBillingAssistant: {
		ask(
			"What is ready to transfer to billing?",
			"Which delivered shipments are ready to transfer to billing, and which cannot go yet?",
			"list_billing_transfer_candidates",
		),
		ask(
			"Which billing items are stuck the longest?",
			"Which billing queue items have been blocked the longest, and why?",
			"list_billing_queue_items", "transition_item_to_in_review", "request_missing_docs",
		),
		ask(
			"Which deliveries are missing paperwork?",
			"Which delivered shipments are still missing the documents billing needs?",
			"request_missing_docs",
		),
		ask(
			"Which billing reports can I run?",
			"Which reports can I run for billing, and what does each one show?",
			"list_reports",
		),
	},
	TemplateComplianceAssistant: {
		ask(
			"Whose credentials expire soon?",
			"Which drivers have a credential expiring in the next 30 days?",
			"list_expiring_credentials",
		),
		ask(
			"Who is off this week?",
			"Which drivers have time off booked this week?",
			"list_time_off",
		),
		ask(
			"Which equipment is out of service?",
			"Which tractors and trailers are out of service right now?",
			"list_tractors", "list_trailers",
		),
		ask(
			"Which compliance reports can I run?",
			"Which reports can I run for safety and compliance?",
			"list_reports",
		),
	},
	TemplateCustomerAssistant: {
		ask(
			"What delivers today?",
			"Which shipments are scheduled to deliver today?",
			"list_shipments",
		),
		ask(
			"Which shipments are running late?",
			"Which shipments are running late right now, and for which customers?",
			"list_shipments", "search_shipments",
		),
		ask(
			"Which invoices are still open?",
			"Which invoices are still open, oldest first?",
			"list_invoices",
		),
		ask(
			"Look up a customer",
			"Find a customer and summarize their account for me.",
			"list_customers",
		),
	},
	TemplateGeneralAssistant: howToStarters,
	TemplateBillingException: {
		ask(
			"Which billing items are blocked, and why?",
			"Which billing items are blocked right now, and what is blocking each one?",
			"list_billing_queue_items", "transition_item_to_in_review", "search_shipments",
		),
		ask(
			"Which shipments are missing documents?",
			"Which shipments are held from billing for missing documents?",
			"request_missing_docs",
		),
		ask(
			"Which charge codes need correcting?",
			"Which shipments have a charge code that looks wrong, and what should it be?",
			"correct_charge_code",
		),
	},
	TemplateDispatchAssignment: {
		ask(
			"Which moves are unassigned today?",
			"Which moves on the dispatch board are still unassigned for today?",
			"get_dispatch_board",
		),
		ask(
			"Who should take the next uncovered move?",
			"Rank the best available drivers for the next uncovered move and explain the top pick.",
			"rank_move_candidates",
		),
		ask(
			"Which carriers could cover an open load?",
			"Which carriers could cover today's open loads, and at what rate?",
			"shop_carriers",
		),
	},
	TemplateImportAssistant: {
		ask(
			"Finish this shipment",
			"Help me finish this shipment from the document: what is still missing?",
			"set_required_field", "set_stop_location",
		),
		ask(
			"Match the stops to locations",
			"Match each stop on the document to one of our locations.",
			"list_locations", "set_stop_location",
		),
		ask(
			"Which rating method should I use?",
			"Which rating methods can this shipment be priced with?",
			"list_formula_templates",
		),
	},
	TemplateLoadMonitor: {
		ask(
			"Which loads are running late?",
			"Which loads on the board are running late right now?",
			"get_dispatch_board", "get_shipment_tracking",
		),
		ask(
			"Who is low on hours?",
			"Which drivers are running low on hours of service today?",
			"get_worker_hos",
		),
		ask(
			"Is weather affecting any loads?",
			"Are any weather alerts affecting loads that are moving today?",
			"list_weather_alerts",
		),
		ask(
			"Which service failures are open?",
			"Which service failures are still open, and what caused them?",
			"list_service_failures",
		),
	},
	TemplateShipmentIntake: {
		ask(
			"Help me enter a shipment",
			"Help me enter a new shipment from a customer's document.",
			"create_shipment",
		),
		ask(
			"Quote a shipment",
			"Quote a shipment for me before I enter it.",
			"quote_shipment",
		),
		ask(
			"Which equipment types can I use?",
			"Which equipment types can I put on a shipment?",
			"list_equipment_types",
		),
	},
	TemplateCashApplication: {
		ask(
			"Which receipts still need matching?",
			"Which bank receipts still need to be matched to a payment?",
			"list_bank_receipt_exceptions",
		),
		ask(
			"Which invoices are past due?",
			"Which invoices are past due, and for which customers?",
			"list_invoices",
		),
		ask(
			"Which payments came in this week?",
			"Which customer payments were recorded this week?",
			"list_customer_payments",
		),
	},
	TemplateDetentionDesk: {
		ask(
			"Where is detention building up?",
			"Which stops are running up detention right now?",
			"list_detention_desk",
		),
		ask(
			"Which notices still need to go out?",
			"Which detention cases still need a notice sent to the customer?",
			"list_detention_desk",
		),
		ask(
			"What should be escalated?",
			"Which detention cases should be escalated, and why?",
			"escalate_detention",
		),
		ask(
			"Which charges need approval?",
			"Which detention charges are waiting on approval, and does the evidence support billing them?",
			"approve_detention",
		),
	},
	TemplateCredentialDesk: {
		ask(
			"Whose credentials expire soon?",
			"Which drivers have a credential expiring in the next 30 days?",
			"list_expiring_credentials",
		),
		ask(
			"Who needs a renewal request?",
			"Which drivers still need a renewal request for papers that are coming due?",
			"list_expiring_credentials", "request_credential_renewal",
		),
		ask(
			"Look up a driver's papers",
			"Find a driver and tell me which of their credentials are current.",
			"get_worker_credential", "list_workers",
		),
	},
	TemplateCustomerUpdateDesk: {
		ask(
			"Draft a shipment update",
			"Draft an update to a customer about where their shipment is.",
			"email_customer",
		),
		ask(
			"Look up a shipment's progress",
			"Find a shipment and tell me where it is and when it should arrive.",
			"get_shipment_tracking", "get_shipment",
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
			"list_carriers", "get_carrier_intel_event",
		),
		ask(
			"Look up a carrier",
			"Find a carrier and tell me whether they can still be given freight.",
			"get_carrier", "list_carriers",
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
			"list_inbound_messages",
		),
		ask(
			"Which messages need a reply?",
			"Which inbound messages are still waiting on a reply?",
			"list_inbound_messages",
		),
		ask(
			"Find a shipment",
			"Find a shipment by its reference and tell me where it stands.",
			"search_shipments",
		),
	},
	TemplateLoadEntryCheck: {
		ask(
			"Is this shipment a duplicate?",
			"Look for another shipment with the same BOL or PRO as this one, and tell me "+
				"whether it is the same load.",
			"search_shipments",
		),
		ask(
			"Does the rate fit the lane?",
			"Compare this shipment's rate with a fresh quote for the same customer and stops.",
			"explain_rate", "quote_shipment",
		),
		ask(
			"Which hold reasons can I use?",
			"Which hold reasons are set up, and what does each one block?",
			"list_hold_reasons",
		),
	},
	TemplateServiceFailureDesk: {
		ask(
			"Which service failures are open?",
			"Which service failures are still open, and what caused them?",
			"list_service_failures",
		),
		ask(
			"Why was this stop late?",
			"Look into this shipment's late stop and tell me what caused it.",
			"get_service_failure", "get_shipment_tracking",
		),
		ask(
			"Which reason codes can I use?",
			"Which service failure reason codes are set up, and when does each one apply?",
			"list_service_failure_reason_codes",
		),
	},
	TemplateInsightAnalyst: {
		ask(
			"What stands out today?",
			"Which open insights need my attention today?",
			"list_insights",
		),
		ask(
			"Does this finding hold up?",
			"Check this insight against the records behind it and tell me whether it holds up.",
			"get_insight",
		),
		ask(
			"Which reports can I run?",
			"Which reports can I run, and what does each one show?",
			"list_reports",
		),
	},
	TemplateEDIDesk: {
		ask(
			"Which EDI files are quarantined?",
			"Which inbound EDI files are in quarantine, and why did each one fail?",
			"list_edi_inbound_files",
		),
		ask(
			"Why did this EDI file fail?",
			"Tell me why this EDI file could not be processed and what would fix it.",
			"get_edi_inbound_file",
		),
		ask(
			"Is a partner ready for EDI?",
			"Is this trading partner set up to send us EDI, and what is still missing?",
			"get_edi_partner",
		),
		ask(
			"Which tenders were rejected?",
			"Which EDI load tenders were rejected or failed, and why?",
			"list_edi_transfers",
		),
	},
	TemplateBooksKeeper: {
		ask(
			"What did not reach the books?",
			"Which documents did not reach the accounting system, and why?",
			"list_accounting_sync_records",
		),
		ask(
			"Did this invoice sync?",
			"Did this invoice reach the accounting system, and if not, what is holding it?",
			"get_record_accounting_sync_state",
		),
		ask(
			"Which mappings are missing?",
			"Which mappings are still missing, and which documents are waiting on them?",
			"list_accounting_mapping_gaps",
		),
		ask(
			"Is the accounting connection healthy?",
			"Is the accounting system connected and answering, and how many documents are queued?",
			"get_accounting_sync_status",
		),
	},
	TemplateFormulaAssistant: {
		ask(
			"Write a formula",
			"Write a formula that charges per mile with a fuel surcharge and a minimum charge.",
			"propose_formula", "describe_formula_schema",
		),
		ask(
			"Explain this formula",
			"Explain what this formula charges, term by term.",
			"describe_formula_schema",
		),
		ask(
			"What would this charge?",
			"What would this formula charge for a 500 mile load?",
			"test_formula_expression",
		),
		ask(
			"How was this rate worked out?",
			"Explain how this shipment's rate was worked out, charge by charge.",
			"explain_rate",
		),
	},
}

var toolFamilies = [][]starterQuestion{
	{
		ask(
			"Build a report",
			"Build a report of in-transit shipments by customer.",
			"create_report",
		),
		ask(
			"Which reports can I run?",
			"Which reports can I run, and what does each one show?",
			"list_reports",
		),
		ask(
			"What can I report on?",
			"Which datasets can I build a report from, and what does each one hold?",
			"list_report_datasets", "describe_report_dataset",
		),
		ask(
			"Change one of my reports",
			"Add a column for delivery date to one of my reports.",
			"update_report",
		),
	},
	{
		ask(
			"Add a widget to my home page",
			"Add a widget to my home page that shows today's late shipments.",
			"add_home_widget",
		),
		ask(
			"What is on my home page?",
			"What widgets are on my home page right now?",
			"list_home_widgets", "get_my_home_layout",
		),
	},
	{
		ask(
			"Build a dashboard",
			"Build a dashboard of this month's on-time delivery: its headline figures "+
				"first, as KPI tiles if the report gives them, then the detail table.",
			"create_dashboard", "add_dashboard_tile",
		),
		ask(
			"Which dashboards do we have?",
			"Which dashboards do we have, and what does each one track?",
			"list_dashboards",
		),
	},
	{
		ask(
			"Which moves are unassigned today?",
			"Which moves on the dispatch board are still unassigned for today?",
			"get_dispatch_board",
		),
		ask(
			"Who should take the next uncovered move?",
			"Rank the best available drivers for the next uncovered move and explain the top pick.",
			"rank_move_candidates",
		),
		ask(
			"Plan today's dispatch",
			"Plan driver assignments for today's uncovered moves.",
			"plan_dispatch",
		),
	},
	{
		ask(
			"Which loads are running late?",
			"Which loads are running late right now, based on tracking?",
			"get_shipment_tracking", "list_vehicle_positions",
		),
		ask(
			"Is weather affecting any loads?",
			"Are any weather alerts affecting loads that are moving today?",
			"list_weather_alerts",
		),
	},
	{
		ask(
			"What is picking up today?",
			"Which shipments are scheduled to pick up today?",
			"list_shipments",
		),
		ask(
			"Find a shipment",
			"Find a shipment by its reference and tell me where it stands.",
			"search_shipments", "get_shipment",
		),
	},
	{
		ask(
			"Which carriers could cover an open load?",
			"Which carriers could cover today's open loads, and at what rate?",
			"shop_carriers",
		),
		ask(
			"Which carriers should I watch?",
			"Which carriers have a recent change to their authority, insurance or safety record?",
			"get_carrier_intel_event",
		),
		ask(
			"Look up a carrier",
			"Find a carrier and tell me whether they can still be given freight.",
			"list_carriers", "get_carrier",
		),
	},
	{
		ask(
			"Whose credentials expire soon?",
			"Which drivers have a credential expiring in the next 30 days?",
			"list_expiring_credentials",
		),
		ask(
			"Who is low on hours?",
			"Which drivers are running low on hours of service today?",
			"get_worker_hos",
		),
		ask(
			"Who is off this week?",
			"Which drivers have time off booked this week?",
			"list_time_off",
		),
		ask(
			"Look up a driver",
			"Find a driver and summarize where they are and what they are qualified for.",
			"search_worker", "list_workers", "get_worker",
		),
	},
	{
		ask(
			"Which equipment is out of service?",
			"Which tractors and trailers are out of service right now?",
			"list_tractors", "list_trailers",
		),
	},
	{
		ask(
			"Which billing items are stuck the longest?",
			"Which billing queue items have been blocked the longest, and why?",
			"transition_item_to_in_review", "correct_charge_code",
		),
		ask(
			"Which deliveries are missing paperwork?",
			"Which delivered shipments are still missing the documents billing needs?",
			"request_missing_docs",
		),
	},
	{
		ask(
			"Which invoices are past due?",
			"Which invoices are past due, and for which customers?",
			"list_invoices", "get_invoice",
		),
		ask(
			"Which receipts still need matching?",
			"Which bank receipts still need to be matched to a payment?",
			"list_bank_receipt_exceptions",
		),
		ask(
			"Which payments came in this week?",
			"Which customer payments were recorded this week?",
			"list_customer_payments",
		),
	},
	{
		ask(
			"Where is detention building up?",
			"Which stops are running up detention right now?",
			"list_detention_desk",
		),
		ask(
			"Which service failures are open?",
			"Which service failures are still open, and what caused them?",
			"list_service_failures",
		),
	},
	{
		ask(
			"What in the inbox is unfiled?",
			"Which messages in the inbox have not been filed against a load yet?",
			"list_inbound_messages",
		),
	},
	{
		ask(
			"Help me enter a shipment",
			"Help me enter a new shipment from a customer's document.",
			"create_shipment",
		),
		ask(
			"Quote a shipment",
			"Quote a shipment for me before I enter it.",
			"quote_shipment",
		),
	},
	{
		ask(
			"Look up a customer",
			"Find a customer and summarize their account for me.",
			"list_customers", "get_customer",
		),
	},
	{
		ask(
			"What stands out today?",
			"Which open insights need my attention today?",
			"list_insights",
		),
		ask(
			"Alert me to a change",
			"Alert me whenever a shipment is put on hold.",
			"create_table_change_alert",
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
