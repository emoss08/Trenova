package agentdefinition

import "github.com/emoss08/trenova/internal/core/domain/agent"

type Template string

const (
	TemplateDispatchAssistant   = Template("DispatchAssistant")
	TemplateBillingAssistant    = Template("BillingAssistant")
	TemplateComplianceAssistant = Template("ComplianceAssistant")
	TemplateCustomerAssistant   = Template("CustomerAssistant")
	TemplateGeneralAssistant    = Template("GeneralAssistant")
	TemplateBillingException    = Template("BillingException")
	TemplateDispatchAssignment  = Template("DispatchAssignment")
	TemplateImportAssistant     = Template("ImportAssistant")
	TemplateLoadMonitor         = Template("LoadMonitor")
	TemplateShipmentIntake      = Template("ShipmentIntake")
	TemplateCashApplication     = Template("CashApplication")
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
		TemplateCashApplication:
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
	default:
		return ""
	}
}

func (t Template) StarterInstructions() string {
	switch t {
	case TemplateDispatchAssistant:
		return "You support the dispatch desk. Prioritise keeping freight moving: when a driver is " +
			"asked about, check hours of service and current assignment before anything else. Cite " +
			"pro numbers and driver names in every answer. When a change to a move is needed, propose " +
			"it with the reason and let a dispatcher confirm."
	case TemplateBillingAssistant:
		return "You support the billing team. Your job is to get shipments invoiced correctly and on " +
			"time. Explain what blocks an item in plain language, name the missing document or the " +
			"charge in question, and prefer proposing a specific fix over describing the problem. " +
			"Never invent a rate or a charge code."
	case TemplateComplianceAssistant:
		return "You support safety and compliance. Focus on driver qualification: medical cards, " +
			"licence class and endorsements, hours of service, and expiring documents. When something " +
			"is about to lapse, say when and what is needed to renew it."
	case TemplateCustomerAssistant:
		return "You support the customer-facing team. Give clear shipment status, expected dates and " +
			"the next stop. Do not disclose internal cost or margin. When a customer promise would be " +
			"needed, say what you can confirm and what a person must decide."
	case TemplateGeneralAssistant:
		return "You explain how Trenova works and where to find things. Answer from what you can look " +
			"up and say plainly when something needs a person with the right access."
	case TemplateBillingException:
		return "You are the billing exception analyst. For each blocked billing item, work out why " +
			"it is blocked from the shipment, its documents and its readiness checks. Prefer proposing " +
			"an action a registered tool can carry out. When the blocker cannot be resolved with the " +
			"tools you have, or your confidence is low, raise an exception for a person with the " +
			"evidence you used."
	case TemplateDispatchAssignment:
		return "You review moves that have no driver. For each uncovered move, weigh the candidates " +
			"on hours available, proximity, equipment fit and customer requirements, then propose one " +
			"assignment with your reasoning. If nothing fits, raise an exception saying what is missing."
	case TemplateImportAssistant:
		return "You help a person turn a shipment document into a shipment record. Reconcile every " +
			"extracted field against the records you can look up, accept what matches with high " +
			"confidence, and ask about what does not. Never save a shipment without the person's " +
			"confirmation."
	case TemplateLoadMonitor:
		return "You watch loads in progress so the desk does not have to. Each run, read the " +
			"dispatch board and work through what is Late or Now: for each, read the shipment's " +
			"tracking, decide from the stops, the position and the driver's hours whether the " +
			"next stop will be made, and act. A stop already late with no failure on record " +
			"gets evaluate_service_failures. A delivery that will miss its window gets the " +
			"customer told with email_customer and the driver told with notify_driver, each " +
			"with the new expected time and nothing internal. A detention notice that is due " +
			"gets sent. Anything you cannot resolve — an uncovered move, a truck with no " +
			"position for hours, a weather alert on the route — gets flag_for_manual_review " +
			"with the evidence. Never estimate an arrival as a promise; say it is an estimate. " +
			"Do not repeat an action the shipment's comments show was taken in the last hour. " +
			"Finish with a short report: what is at risk, what you did, what needs a person."
	case TemplateShipmentIntake:
		return "You enter shipments from customer documents. A run starts when document " +
			"intelligence has read a document; read its draft with get_shipment_draft first. " +
			"Resolve every name on the draft to a record: the customer with list_customers, " +
			"each stop's address to a location with list_locations, the service and shipment " +
			"types from their lists. Use a field only when its confidence is high or you " +
			"confirmed it against another field; a low-confidence rate or date is not a " +
			"guess to fill in. Price the lane with quote_shipment and compare it with the " +
			"rate on the document. Then propose create_shipment with sourceDocumentId set, " +
			"the stops in travel order, and the document's BOL as the reference. If a " +
			"customer, location or type cannot be matched, if the draft needs review, or " +
			"if the document's rate and the quote disagree by more than a little, raise an " +
			"exception with what you found instead of creating the shipment. Never create " +
			"a shipment twice for one document."
	case TemplateCashApplication:
		return "You apply cash. A run starts when an imported bank receipt could not be matched " +
			"to a customer payment automatically; read it with get_bank_receipt first, which " +
			"gives you the receipt, the scored candidate payments and the work item. Take the " +
			"top candidate when its reference and amount both agree; when the candidates " +
			"disagree or are absent, search list_customer_payments by the reference, the " +
			"memo and the amount, and identify the customer from the reference, the memo or " +
			"the amount against their open invoices with list_invoices. Propose " +
			"match_bank_receipt when a posted payment of the same amount is the one; propose " +
			"post_customer_payment with bankReceiptId when no payment has been recorded yet " +
			"and you can name the customer and the invoices the money pays, applying the " +
			"receipt's full amount and leaving any remainder unapplied rather than " +
			"short-paying. Never post a payment against a customer you inferred from the " +
			"amount alone. When the receipt is not a customer payment at all, or the " +
			"customer cannot be identified from the records, resolve the work item with " +
			"RequiresExternalFollowUp or MarkedFalsePositive and say why. Report what you " +
			"matched, what you posted and what you left for a person."
	default:
		return ""
	}
}

func (t Template) StarterTools() []string {
	switch t {
	case TemplateDispatchAssistant:
		return []string{
			"get_shipment",
			"search_shipments",
			"list_shipments",
			"get_worker",
			"search_worker",
			"list_workers",
			"list_tractors",
			"list_trailers",
			"list_expiring_credentials",
			"list_time_off",
			"list_hold_reasons",
			"rank_move_candidates",
			"shop_carriers",
			"assign_move",
			"add_shipment_comment",
			"place_shipment_hold",
			"release_shipment_hold",
			"update_tractor_status",
			"update_trailer_status",
			"recall_memory",
			"remember",
			"list_insights",
			"get_insight",
		}
	case TemplateBillingAssistant:
		return []string{
			"get_shipment",
			"search_shipments",
			"list_shipments",
			"list_customers",
			"list_reports",
			"describe_report",
			"preview_report",
			"run_report",
			"get_report_run",
			"list_email_profiles",
			"request_missing_docs",
			"flag_for_manual_review",
			"transition_item_to_in_review",
			"list_accessorial_charges",
			"add_shipment_comment",
			"recall_memory",
			"remember",
			"list_insights",
			"get_insight",
		}
	case TemplateComplianceAssistant:
		return []string{
			"get_worker",
			"search_worker",
			"list_workers",
			"list_expiring_credentials",
			"list_time_off",
			"list_tractors",
			"list_trailers",
			"list_reports",
			"describe_report",
			"preview_report",
			"run_report",
			"get_report_run",
			"flag_for_manual_review",
			"recall_memory",
			"remember",
			"list_insights",
			"get_insight",
		}
	case TemplateCustomerAssistant:
		return []string{
			"get_shipment",
			"search_shipments",
			"list_shipments",
			"list_customers",
			"list_invoices",
			"add_shipment_comment",
			"recall_memory",
			"remember",
			"list_insights",
			"get_insight",
		}
	case TemplateLoadMonitor:
		return []string{
			"get_dispatch_board",
			"get_shipment_tracking",
			"get_shipment",
			"list_vehicle_positions",
			"get_worker_hos",
			"list_service_failures",
			"list_service_failure_reason_codes",
			"list_detention_desk",
			"list_weather_alerts",
			"evaluate_service_failures",
			"resolve_service_failure",
			"add_shipment_comment",
			"notify_driver",
			"list_email_profiles",
			"email_customer",
			"send_detention_notice",
			"flag_for_manual_review",
		}
	case TemplateGeneralAssistant:
		return nil
	case TemplateCashApplication:
		return []string{
			"get_bank_receipt",
			"list_bank_receipt_exceptions",
			"list_customer_payments",
			"list_customers",
			"get_customer",
			"list_invoices",
			"get_invoice",
			"match_bank_receipt",
			"post_customer_payment",
			"resolve_bank_receipt_work_item",
			"recall_memory",
			"remember",
		}
	case TemplateBillingException:
		return []string{
			"get_shipment",
			"search_shipments",
			"transition_item_to_in_review",
			"correct_charge_code",
			"request_missing_docs",
			"attach_document_to_bqi",
			"flag_for_manual_review",
			"raise_exception",
		}
	case TemplateDispatchAssignment:
		return []string{
			"get_shipment",
			"search_shipments",
			"get_worker",
			"search_worker",
			"get_dispatch_board",
			"rank_move_candidates",
			"plan_dispatch",
			"list_carriers",
			"shop_carriers",
			"assign_move",
			"tender_move_to_routing_guide",
			"tender_move_to_carriers",
			"raise_exception",
		}
	case TemplateImportAssistant:
		return []string{
			"search_shipments",
			"search_worker",
			"get_shipment_draft",
			"list_customers",
			"list_locations",
			"list_service_types",
			"list_shipment_types",
			"quote_shipment",
			"create_shipment",
		}
	case TemplateShipmentIntake:
		return []string{
			"get_shipment_draft",
			"list_customers",
			"list_locations",
			"list_service_types",
			"list_shipment_types",
			"list_equipment_types",
			"search_shipments",
			"quote_shipment",
			"create_shipment",
			"raise_exception",
		}
	default:
		return nil
	}
}

func (t Template) StarterTrigger() TriggerMode {
	switch t {
	case TemplateBillingException, TemplateShipmentIntake, TemplateCashApplication:
		return TriggerEvent
	case TemplateDispatchAssignment, TemplateLoadMonitor:
		return TriggerScheduled
	default:
		return TriggerChat
	}
}

func (t Template) StarterEvents() []agent.EventKind {
	switch t {
	case TemplateBillingException:
		return []agent.EventKind{agent.EventBillingQueueItemException}
	case TemplateShipmentIntake:
		return []agent.EventKind{agent.EventDocumentExtracted}
	case TemplateCashApplication:
		return []agent.EventKind{agent.EventBankReceiptException}
	default:
		return nil
	}
}

func (t Template) StarterCron() string {
	switch t {
	case TemplateDispatchAssignment:
		return "*/30 * * * *"
	case TemplateLoadMonitor:
		return "*/15 * * * *"
	default:
		return ""
	}
}

func (t Template) StarterCeiling() agent.AutonomyTier {
	switch t {
	case TemplateGeneralAssistant, TemplateCustomerAssistant:
		return agent.TierPropose
	default:
		return agent.TierActWithApproval
	}
}

type TriggerMode string

const (
	TriggerChat       = TriggerMode("Chat")
	TriggerScheduled  = TriggerMode("Scheduled")
	TriggerEvent      = TriggerMode("Event")
	TriggerContinuous = TriggerMode("Continuous")
)

func (m TriggerMode) IsValid() bool {
	switch m {
	case TriggerChat, TriggerScheduled, TriggerEvent, TriggerContinuous:
		return true
	default:
		return false
	}
}

func (m TriggerMode) RunTrigger() agent.RunTrigger {
	switch m {
	case TriggerScheduled:
		return agent.RunTriggerScheduled
	case TriggerEvent:
		return agent.RunTriggerEvent
	case TriggerContinuous:
		return agent.RunTriggerContinuous
	default:
		return agent.RunTriggerChat
	}
}

type OutputMode string

const (
	OutputConversational = OutputMode("Conversational")
	OutputReport         = OutputMode("Report")
)

func (m OutputMode) IsValid() bool {
	switch m {
	case OutputConversational, OutputReport:
		return true
	default:
		return false
	}
}

type ContextProvider string

const (
	ContextOrganization = ContextProvider("Organization")
	ContextClock        = ContextProvider("Clock")
	ContextUser         = ContextProvider("User")
	ContextPage         = ContextProvider("Page")
	ContextTools        = ContextProvider("Tools")
	ContextMemory       = ContextProvider("Memory")
)

func (p ContextProvider) IsValid() bool {
	switch p {
	case ContextOrganization, ContextClock, ContextUser, ContextPage, ContextTools, ContextMemory:
		return true
	default:
		return false
	}
}

func AllContextProviders() []ContextProvider {
	return []ContextProvider{
		ContextOrganization,
		ContextClock,
		ContextUser,
		ContextPage,
		ContextTools,
		ContextMemory,
	}
}
