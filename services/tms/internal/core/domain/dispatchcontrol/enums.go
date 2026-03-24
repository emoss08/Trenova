package dispatchcontrol

import "errors"

var (
	ErrInvalidServiceIncidentType        = errors.New("invalid service incident type")
	ErrInvalidAutoAssignmentStrategy     = errors.New("invalid auto assignment strategy")
	ErrInvalidComplianceEnforcementLevel = errors.New("invalid compliance enforcement level")
)

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

func ServiceIncidentTypeFromString(s string) (ServiceIncidentType, error) {
	switch s {
	case "Never":
		return ServiceIncidentTypeNever, nil
	case "Pickup":
		return ServiceIncidentTypePickup, nil
	case "Delivery":
		return ServiceIncidentTypeDelivery, nil
	case "PickupDelivery":
		return ServiceIncidentTypePickupDelivery, nil
	case "AllExceptShipper":
		return ServiceIncidentTypeAllExceptShipper, nil
	default:
		return "", ErrInvalidServiceIncidentType
	}
}

type AutoAssignmentStrategy string

const (
	AutoAssignmentStrategyProximity     = AutoAssignmentStrategy("Proximity")
	AutoAssignmentStrategyAvailability  = AutoAssignmentStrategy("Availability")
	AutoAssignmentStrategyLoadBalancing = AutoAssignmentStrategy("LoadBalancing")
)

func (a AutoAssignmentStrategy) String() string {
	return string(a)
}

func (a AutoAssignmentStrategy) IsValid() bool {
	switch a {
	case AutoAssignmentStrategyProximity,
		AutoAssignmentStrategyAvailability,
		AutoAssignmentStrategyLoadBalancing:
		return true
	default:
		return false
	}
}

func AutoAssignmentStrategyFromString(s string) (AutoAssignmentStrategy, error) {
	switch s {
	case "Proximity":
		return AutoAssignmentStrategyProximity, nil
	case "Availability":
		return AutoAssignmentStrategyAvailability, nil
	case "LoadBalancing":
		return AutoAssignmentStrategyLoadBalancing, nil
	default:
		return "", ErrInvalidAutoAssignmentStrategy
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

func ComplianceEnforcementLevelFromString(s string) (ComplianceEnforcementLevel, error) {
	switch s {
	case "Warning":
		return ComplianceEnforcementLevelWarning, nil
	case "Block":
		return ComplianceEnforcementLevelBlock, nil
	case "Audit":
		return ComplianceEnforcementLevelAudit, nil
	default:
		return "", ErrInvalidComplianceEnforcementLevel
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
