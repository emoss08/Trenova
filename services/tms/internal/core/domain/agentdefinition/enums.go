package agentdefinition

import "github.com/emoss08/trenova/internal/core/domain/permission"

// Kind is a Trenova-owned agent template. An organization picks one and narrows
// it; it cannot invent a kind, because a kind carries the system prompt and the
// outer bound on which resources the agent may touch.
type Kind string

const (
	KindDispatchAssistant   = Kind("DispatchAssistant")
	KindBillingAssistant    = Kind("BillingAssistant")
	KindComplianceAssistant = Kind("ComplianceAssistant")
	KindCustomerAssistant   = Kind("CustomerAssistant")
	KindGeneralAssistant    = Kind("GeneralAssistant")
)

func (k Kind) IsValid() bool {
	switch k {
	case KindDispatchAssistant,
		KindBillingAssistant,
		KindComplianceAssistant,
		KindCustomerAssistant,
		KindGeneralAssistant:
		return true
	default:
		return false
	}
}

// AllKinds lists every template, for building configuration UIs.
func AllKinds() []Kind {
	return []Kind{
		KindDispatchAssistant,
		KindBillingAssistant,
		KindComplianceAssistant,
		KindCustomerAssistant,
		KindGeneralAssistant,
	}
}

func (k Kind) Label() string {
	switch k {
	case KindDispatchAssistant:
		return "Dispatch assistant"
	case KindBillingAssistant:
		return "Billing assistant"
	case KindComplianceAssistant:
		return "Compliance assistant"
	case KindCustomerAssistant:
		return "Customer assistant"
	case KindGeneralAssistant:
		return "General assistant"
	default:
		return string(k)
	}
}

func (k Kind) Description() string {
	switch k {
	case KindDispatchAssistant:
		return "Answers questions and acts on shipments, moves, drivers, and equipment."
	case KindBillingAssistant:
		return "Works the billing queue, charges, and invoices."
	case KindComplianceAssistant:
		return "Covers worker qualification, hours of service, and hazmat requirements."
	case KindCustomerAssistant:
		return "Handles customer records, quotes, and shipment status enquiries."
	case KindGeneralAssistant:
		return "Answers questions across the system without changing anything."
	default:
		return ""
	}
}

// AllowedResources is the outer bound on what a kind may touch. Every tool an
// organization selects must fall inside this set, which is how narrowing stays
// narrowing: an organization choosing a customer assistant cannot hand it
// billing tools, whatever it writes in the focus field.
//
// The general assistant is deliberately absent — it is read-only and gets no
// mutating tools at all, which `MutatingAllowed` enforces separately.
func (k Kind) AllowedResources() []permission.Resource {
	switch k {
	case KindDispatchAssistant:
		return []permission.Resource{
			permission.ResourceShipment,
			permission.ResourceShipmentMove,
			permission.ResourceWorker,
			permission.ResourceTractor,
			permission.ResourceTrailer,
			permission.ResourceAgentException,
		}
	case KindBillingAssistant:
		return []permission.Resource{
			permission.ResourceBillingQueue,
			permission.ResourceShipment,
			permission.ResourceDocument,
			permission.ResourceAgentException,
		}
	case KindComplianceAssistant:
		return []permission.Resource{
			permission.ResourceWorker,
			permission.ResourceDocument,
			permission.ResourceAgentException,
		}
	case KindCustomerAssistant:
		return []permission.Resource{
			permission.ResourceCustomer,
			permission.ResourceShipment,
			permission.ResourceAgentException,
		}
	case KindGeneralAssistant:
		return nil
	default:
		return nil
	}
}

// MutatingAllowed reports whether a kind may hold tools that change data. A
// general assistant answers questions and nothing more, so no configuration can
// give it a write tool.
func (k Kind) MutatingAllowed() bool {
	return k != KindGeneralAssistant
}
