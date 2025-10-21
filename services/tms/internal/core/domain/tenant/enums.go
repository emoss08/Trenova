package tenant

type Type string

const (
	TypeBrokerage        = Type("Brokerage")
	TypeCarrier          = Type("Carrier")
	TypeBrokerageCarrier = Type("BrokerageCarrier")
)

type ExceptionHandling string

const (
	BillingExceptionQueue       = ExceptionHandling("Queue")
	BillingExceptionNotify      = ExceptionHandling("Notify")
	BillingExceptionAutoResolve = ExceptionHandling("AutoResolve")
	BillingExceptionReject      = ExceptionHandling("Reject")
)

type PaymentTerm string

const (
	PaymentTermNet15        = PaymentTerm("Net15")
	PaymentTermNet30        = PaymentTerm("Net30")
	PaymentTermNet45        = PaymentTerm("Net45")
	PaymentTermNet60        = PaymentTerm("Net60")
	PaymentTermNet90        = PaymentTerm("Net90")
	PaymentTermDueOnReceipt = PaymentTerm("DueOnReceipt")
)

type TransferSchedule string

const (
	TransferScheduleContinuous = TransferSchedule("Continuous")
	TransferScheduleHourly     = TransferSchedule("Hourly")
	TransferScheduleDaily      = TransferSchedule("Daily")
	TransferScheduleWeekly     = TransferSchedule("Weekly")
)

type AutoAssignmentStrategy string

const (
	AutoAssignmentStrategyProximity     = AutoAssignmentStrategy("Proximity")
	AutoAssignmentStrategyAvailability  = AutoAssignmentStrategy("Availability")
	AutoAssignmentStrategyLoadBalancing = AutoAssignmentStrategy("LoadBalancing")
)

type ComplianceEnforcementLevel string

const (
	ComplianceEnforcementLevelWarning = ComplianceEnforcementLevel("Warning")
	ComplianceEnforcementLevelBlock   = ComplianceEnforcementLevel("Block")
	ComplianceEnforcementLevelAudit   = ComplianceEnforcementLevel("Audit")
)

type ServiceIncidentType string

const (
	ServiceIncidentTypeNever            = ServiceIncidentType("Never")
	ServiceIncidentTypePickup           = ServiceIncidentType("Pickup")
	ServiceIncidentTypeDelivery         = ServiceIncidentType("Delivery")
	ServiceIncidentTypePickupDelivery   = ServiceIncidentType("PickupDelivery")
	ServiceIncidentTypeAllExceptShipper = ServiceIncidentType("AllExceptShipper")
)

func (s ServiceIncidentType) NotEqual(value ServiceIncidentType) bool {
	return s != value
}
