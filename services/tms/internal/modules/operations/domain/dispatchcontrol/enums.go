package dispatchcontrol

type PlanningMode string

const (
	PlanningModeImmediate = PlanningMode("Immediate")
	PlanningModeHorizon   = PlanningMode("Horizon")
)

func (p PlanningMode) String() string {
	return string(p)
}

func (p PlanningMode) IsValid() bool {
	switch p {
	case PlanningModeImmediate, PlanningModeHorizon:
		return true
	default:
		return false
	}
}

type ServiceIncidentType string

const (
	ServiceIncidentTypeNever            = ServiceIncidentType("Never")
	ServiceIncidentTypePickup           = ServiceIncidentType("Pickup")
	ServiceIncidentTypeDelivery         = ServiceIncidentType("Delivery")
	ServiceIncidentTypePickupDelivery   = ServiceIncidentType("PickupDelivery")
	ServiceIncidentTypeAllExceptShipper = ServiceIncidentType("AllExceptShipper")
)

func (s ServiceIncidentType) String() string {
	return string(s)
}

func (s ServiceIncidentType) IsValid() bool {
	switch s {
	case ServiceIncidentTypeNever, ServiceIncidentTypePickup, ServiceIncidentTypeDelivery,
		ServiceIncidentTypePickupDelivery, ServiceIncidentTypeAllExceptShipper:
		return true
	default:
		return false
	}
}

type ComplianceEnforcementLevel string

const (
	ComplianceEnforcementLevelWarning = ComplianceEnforcementLevel("Warning")
	ComplianceEnforcementLevelBlock   = ComplianceEnforcementLevel("Block")
	ComplianceEnforcementLevelAudit   = ComplianceEnforcementLevel("Audit")
)

func (c ComplianceEnforcementLevel) String() string {
	return string(c)
}

func (c ComplianceEnforcementLevel) IsValid() bool {
	switch c {
	case ComplianceEnforcementLevelWarning,
		ComplianceEnforcementLevelBlock,
		ComplianceEnforcementLevelAudit:
		return true
	default:
		return false
	}
}

func (c ComplianceEnforcementLevel) ShouldBlock() bool {
	return c == ComplianceEnforcementLevelBlock
}

func (c ComplianceEnforcementLevel) ShouldWarn() bool {
	return c == ComplianceEnforcementLevelWarning
}

func (c ComplianceEnforcementLevel) IsAuditOnly() bool {
	return c == ComplianceEnforcementLevelAudit
}

// TenderExhaustedAction is what happens to a move once every carrier on its
// tender has declined, let the offer lapse or failed to receive it.
type TenderExhaustedAction string

const (
	// TenderExhaustedActionNotify tells dispatch and stops there.
	TenderExhaustedActionNotify = TenderExhaustedAction("Notify")
	// TenderExhaustedActionRebroadcast offers the move to every carrier on the
	// exhausted tender at once, at the escalation rate.
	TenderExhaustedActionRebroadcast = TenderExhaustedAction("Rebroadcast")
	// TenderExhaustedActionRewaterfall walks the same carriers again in rank
	// order, at the escalation rate.
	TenderExhaustedActionRewaterfall = TenderExhaustedAction("Rewaterfall")
)

func (a TenderExhaustedAction) String() string {
	return string(a)
}

func (a TenderExhaustedAction) IsValid() bool {
	switch a {
	case TenderExhaustedActionNotify,
		TenderExhaustedActionRebroadcast,
		TenderExhaustedActionRewaterfall:
		return true
	default:
		return false
	}
}

func (a TenderExhaustedAction) Escalates() bool {
	return a == TenderExhaustedActionRebroadcast || a == TenderExhaustedActionRewaterfall
}
