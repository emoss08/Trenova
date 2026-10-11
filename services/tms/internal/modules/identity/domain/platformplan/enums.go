package platformplan

import "github.com/emoss08/trenova/pkg/errortypes"

type PlanKey string

const (
	PlanKeyFreeDemo  = PlanKey("free_demo")
	PlanKeyUnlimited = PlanKey("unlimited")
)

func AllPlanKeys() []PlanKey {
	return []PlanKey{PlanKeyFreeDemo, PlanKeyUnlimited}
}

func (k PlanKey) String() string {
	return string(k)
}

func (k PlanKey) IsValid() bool {
	switch k {
	case PlanKeyFreeDemo, PlanKeyUnlimited:
		return true
	default:
		return false
	}
}

type Window string

const (
	WindowLifetime = Window("lifetime")
	WindowMonthly  = Window("monthly")
	WindowPerItem  = Window("per_item")
)

func AllWindows() []Window {
	return []Window{WindowLifetime, WindowMonthly, WindowPerItem}
}

func (w Window) String() string {
	return string(w)
}

func (w Window) IsValid() bool {
	switch w {
	case WindowLifetime, WindowMonthly, WindowPerItem:
		return true
	default:
		return false
	}
}

type Capability string

const (
	CapabilityEmailOutbound           = Capability("email.outbound")
	CapabilityIntegrations            = Capability("integrations")
	CapabilityAPIKeys                 = Capability("api_keys")
	CapabilityAgentAutomation         = Capability("agent.automation")
	CapabilityAgentWebSearch          = Capability("agent.web_search")
	CapabilityCarrierIntelligencePaid = Capability("carrier_intelligence.paid")
	CapabilityDocumentIntelligence    = Capability("document_intelligence")
	CapabilitySMS                     = Capability("sms")
	CapabilitySSO                     = Capability("sso")
)

func AllCapabilities() []Capability {
	return []Capability{
		CapabilityEmailOutbound,
		CapabilityIntegrations,
		CapabilityAPIKeys,
		CapabilityAgentAutomation,
		CapabilityAgentWebSearch,
		CapabilityCarrierIntelligencePaid,
		CapabilityDocumentIntelligence,
		CapabilitySMS,
		CapabilitySSO,
	}
}

func (c Capability) String() string {
	return string(c)
}

func (c Capability) IsValid() bool {
	switch c {
	case CapabilityEmailOutbound,
		CapabilityIntegrations,
		CapabilityAPIKeys,
		CapabilityAgentAutomation,
		CapabilityAgentWebSearch,
		CapabilityCarrierIntelligencePaid,
		CapabilityDocumentIntelligence,
		CapabilitySMS,
		CapabilitySSO:
		return true
	default:
		return false
	}
}

type Origin string

const (
	OriginSelfHosted   = Origin("self_hosted")
	OriginInternal     = Origin("internal")
	OriginSubscription = Origin("subscription")
)

const (
	ReasonPlanRestricted       = errortypes.PlanRestrictionReasonPlan
	ReasonSubscriptionReadOnly = errortypes.PlanRestrictionReasonReadOnly
	ReasonSubscriptionExpired  = errortypes.PlanRestrictionReasonExpired
	ReasonSignupsPaused        = errortypes.PlanRestrictionReasonSignupsPaused
)
