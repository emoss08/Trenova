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
		TemplateImportAssistant:
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
			"list_hold_reasons",
			"assign_move",
			"add_shipment_comment",
			"place_shipment_hold",
			"release_shipment_hold",
		}
	case TemplateBillingAssistant:
		return []string{
			"get_shipment",
			"search_shipments",
			"list_shipments",
			"list_customers",
			"list_reports",
			"run_report",
			"get_report_run",
			"request_missing_docs",
			"flag_for_manual_review",
			"transition_item_to_in_review",
			"list_accessorial_charges",
			"add_shipment_comment",
		}
	case TemplateComplianceAssistant:
		return []string{
			"get_worker",
			"search_worker",
			"list_workers",
			"list_expiring_credentials",
			"list_tractors",
			"list_trailers",
			"list_reports",
			"run_report",
			"get_report_run",
			"flag_for_manual_review",
		}
	case TemplateCustomerAssistant:
		return []string{
			"get_shipment",
			"search_shipments",
			"list_shipments",
			"list_customers",
			"list_invoices",
			"add_shipment_comment",
		}
	case TemplateGeneralAssistant:
		return nil
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
		return []string{"get_shipment", "search_shipments", "get_worker", "search_worker", "assign_move", "raise_exception"}
	case TemplateImportAssistant:
		return []string{"search_shipments", "search_worker"}
	default:
		return nil
	}
}

func (t Template) StarterTrigger() TriggerMode {
	switch t {
	case TemplateBillingException:
		return TriggerEvent
	case TemplateDispatchAssignment:
		return TriggerScheduled
	default:
		return TriggerChat
	}
}

func (t Template) StarterEvents() []agent.EventKind {
	switch t {
	case TemplateBillingException:
		return []agent.EventKind{agent.EventBillingQueueItemException}
	default:
		return nil
	}
}

func (t Template) StarterCron() string {
	if t == TemplateDispatchAssignment {
		return "*/30 * * * *"
	}

	return ""
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
)

func (p ContextProvider) IsValid() bool {
	switch p {
	case ContextOrganization, ContextClock, ContextUser, ContextPage, ContextTools:
		return true
	default:
		return false
	}
}

func AllContextProviders() []ContextProvider {
	return []ContextProvider{ContextOrganization, ContextClock, ContextUser, ContextPage, ContextTools}
}
