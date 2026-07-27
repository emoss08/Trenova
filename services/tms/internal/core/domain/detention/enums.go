package detention

import "errors"

type PolicyStatus string

const (
	PolicyStatusActive   = PolicyStatus("Active")
	PolicyStatusInactive = PolicyStatus("Inactive")
	PolicyStatusDraft    = PolicyStatus("Draft")
)

func (s PolicyStatus) String() string { return string(s) }

func PolicyStatusFromString(v string) (PolicyStatus, error) {
	switch v {
	case "Active":
		return PolicyStatusActive, nil
	case "Inactive":
		return PolicyStatusInactive, nil
	case "Draft":
		return PolicyStatusDraft, nil
	default:
		return "", errors.New("invalid detention policy status")
	}
}

type ClockStartBasis string

const (
	ClockStartBasisArrival                       = ClockStartBasis("Arrival")
	ClockStartBasisAppointment                   = ClockStartBasis("Appointment")
	ClockStartBasisLaterOfArrivalOrAppointment   = ClockStartBasis("LaterOfArrivalOrAppointment")
	ClockStartBasisEarlierOfArrivalOrAppointment = ClockStartBasis("EarlierOfArrivalOrAppointment")
)

func (b ClockStartBasis) String() string { return string(b) }

func ClockStartBasisFromString(v string) (ClockStartBasis, error) {
	switch v {
	case "Arrival":
		return ClockStartBasisArrival, nil
	case "Appointment":
		return ClockStartBasisAppointment, nil
	case "LaterOfArrivalOrAppointment":
		return ClockStartBasisLaterOfArrivalOrAppointment, nil
	case "EarlierOfArrivalOrAppointment":
		return ClockStartBasisEarlierOfArrivalOrAppointment, nil
	default:
		return "", errors.New("invalid detention clock start basis")
	}
}

type LateArrivalRule string

const (
	LateArrivalRuleNoEffect             = LateArrivalRule("NoEffect")
	LateArrivalRuleForfeit              = LateArrivalRule("Forfeit")
	LateArrivalRuleClockFromAppointment = LateArrivalRule("ClockFromAppointment")
	LateArrivalRuleReduceFreeTime       = LateArrivalRule("ReduceFreeTime")
)

func (r LateArrivalRule) String() string { return string(r) }

func LateArrivalRuleFromString(v string) (LateArrivalRule, error) {
	switch v {
	case "NoEffect":
		return LateArrivalRuleNoEffect, nil
	case "Forfeit":
		return LateArrivalRuleForfeit, nil
	case "ClockFromAppointment":
		return LateArrivalRuleClockFromAppointment, nil
	case "ReduceFreeTime":
		return LateArrivalRuleReduceFreeTime, nil
	default:
		return "", errors.New("invalid detention late arrival rule")
	}
}

type RoundingMode string

const (
	RoundingModeUp      = RoundingMode("Up")
	RoundingModeDown    = RoundingMode("Down")
	RoundingModeNearest = RoundingMode("Nearest")
	RoundingModeExact   = RoundingMode("Exact")
)

func (m RoundingMode) String() string { return string(m) }

func RoundingModeFromString(v string) (RoundingMode, error) {
	switch v {
	case "Up":
		return RoundingModeUp, nil
	case "Down":
		return RoundingModeDown, nil
	case "Nearest":
		return RoundingModeNearest, nil
	case "Exact":
		return RoundingModeExact, nil
	default:
		return "", errors.New("invalid detention rounding mode")
	}
}

type RateSource string

const (
	RateSourceAccessorial = RateSource("Accessorial")
	RateSourceTiers       = RateSource("Tiers")
)

func (s RateSource) String() string { return string(s) }

func RateSourceFromString(v string) (RateSource, error) {
	switch v {
	case "Accessorial":
		return RateSourceAccessorial, nil
	case "Tiers":
		return RateSourceTiers, nil
	default:
		return "", errors.New("invalid detention rate source")
	}
}

type TierRateUnit string

const (
	TierRateUnitHour = TierRateUnit("Hour")
	TierRateUnitDay  = TierRateUnit("Day")
	TierRateUnitFlat = TierRateUnit("Flat")
)

func (u TierRateUnit) String() string { return string(u) }

func TierRateUnitFromString(v string) (TierRateUnit, error) {
	switch v {
	case "Hour":
		return TierRateUnitHour, nil
	case "Day":
		return TierRateUnitDay, nil
	case "Flat":
		return TierRateUnitFlat, nil
	default:
		return "", errors.New("invalid detention tier rate unit")
	}
}

type NotificationRequirement string

const (
	NotificationRequirementNone     = NotificationRequirement("None")
	NotificationRequirementAdvisory = NotificationRequirement("Advisory")
	NotificationRequirementRequired = NotificationRequirement("Required")
)

func (r NotificationRequirement) String() string { return string(r) }

func NotificationRequirementFromString(v string) (NotificationRequirement, error) {
	switch v {
	case "None":
		return NotificationRequirementNone, nil
	case "Advisory":
		return NotificationRequirementAdvisory, nil
	case "Required":
		return NotificationRequirementRequired, nil
	default:
		return "", errors.New("invalid detention notification requirement")
	}
}

type UnnotifiedBehavior string

const (
	UnnotifiedBehaviorBill     = UnnotifiedBehavior("Bill")
	UnnotifiedBehaviorFlag     = UnnotifiedBehavior("Flag")
	UnnotifiedBehaviorSuppress = UnnotifiedBehavior("Suppress")
)

func (b UnnotifiedBehavior) String() string { return string(b) }

func UnnotifiedBehaviorFromString(v string) (UnnotifiedBehavior, error) {
	switch v {
	case "Bill":
		return UnnotifiedBehaviorBill, nil
	case "Flag":
		return UnnotifiedBehaviorFlag, nil
	case "Suppress":
		return UnnotifiedBehaviorSuppress, nil
	default:
		return "", errors.New("invalid detention unnotified behavior")
	}
}

type CapScope string

const (
	CapScopePerStop     = CapScope("PerStop")
	CapScopePerDay      = CapScope("PerDay")
	CapScopePerShipment = CapScope("PerShipment")
)

func (s CapScope) String() string { return string(s) }

func CapScopeFromString(v string) (CapScope, error) {
	switch v {
	case "PerStop":
		return CapScopePerStop, nil
	case "PerDay":
		return CapScopePerDay, nil
	case "PerShipment":
		return CapScopePerShipment, nil
	default:
		return "", errors.New("invalid detention cap scope")
	}
}

type OccurrenceStatus string

const (
	OccurrenceStatusAccruing    = OccurrenceStatus("Accruing")
	OccurrenceStatusPending     = OccurrenceStatus("Pending")
	OccurrenceStatusApproved    = OccurrenceStatus("Approved")
	OccurrenceStatusBilled      = OccurrenceStatus("Billed")
	OccurrenceStatusWaived      = OccurrenceStatus("Waived")
	OccurrenceStatusDisputed    = OccurrenceStatus("Disputed")
	OccurrenceStatusNotBillable = OccurrenceStatus("NotBillable")
)

func (s OccurrenceStatus) String() string { return string(s) }

func (s OccurrenceStatus) IsTerminal() bool {
	switch s {
	case OccurrenceStatusBilled, OccurrenceStatusWaived, OccurrenceStatusDisputed:
		return true
	case OccurrenceStatusAccruing, OccurrenceStatusPending, OccurrenceStatusApproved,
		OccurrenceStatusNotBillable:
		return false
	default:
		return false
	}
}

func (s OccurrenceStatus) IsBillable() bool {
	switch s {
	case OccurrenceStatusPending, OccurrenceStatusApproved, OccurrenceStatusBilled:
		return true
	case OccurrenceStatusAccruing, OccurrenceStatusWaived, OccurrenceStatusDisputed,
		OccurrenceStatusNotBillable:
		return false
	default:
		return false
	}
}

func OccurrenceStatusFromString(v string) (OccurrenceStatus, error) {
	switch v {
	case "Accruing":
		return OccurrenceStatusAccruing, nil
	case "Pending":
		return OccurrenceStatusPending, nil
	case "Approved":
		return OccurrenceStatusApproved, nil
	case "Billed":
		return OccurrenceStatusBilled, nil
	case "Waived":
		return OccurrenceStatusWaived, nil
	case "Disputed":
		return OccurrenceStatusDisputed, nil
	case "NotBillable":
		return OccurrenceStatusNotBillable, nil
	default:
		return "", errors.New("invalid detention occurrence status")
	}
}

type NotificationStatus string

const (
	NotificationStatusNotRequired = NotificationStatus("NotRequired")
	NotificationStatusPending     = NotificationStatus("Pending")
	NotificationStatusSent        = NotificationStatus("Sent")
	NotificationStatusLate        = NotificationStatus("Late")
	NotificationStatusMissed      = NotificationStatus("Missed")
	NotificationStatusFailed      = NotificationStatus("Failed")
)

func (s NotificationStatus) String() string { return string(s) }

func (s NotificationStatus) SatisfiesRequirement() bool {
	switch s {
	case NotificationStatusNotRequired, NotificationStatusSent:
		return true
	case NotificationStatusPending, NotificationStatusLate, NotificationStatusMissed,
		NotificationStatusFailed:
		return false
	default:
		return false
	}
}

func NotificationStatusFromString(v string) (NotificationStatus, error) {
	switch v {
	case "NotRequired":
		return NotificationStatusNotRequired, nil
	case "Pending":
		return NotificationStatusPending, nil
	case "Sent":
		return NotificationStatusSent, nil
	case "Late":
		return NotificationStatusLate, nil
	case "Missed":
		return NotificationStatusMissed, nil
	case "Failed":
		return NotificationStatusFailed, nil
	default:
		return "", errors.New("invalid detention notification status")
	}
}

type WaiverReason string

const (
	WaiverReasonWeather          = WaiverReason("Weather")
	WaiverReasonFacilityClosure  = WaiverReason("FacilityClosure")
	WaiverReasonCarrierFault     = WaiverReason("CarrierFault")
	WaiverReasonEquipmentIssue   = WaiverReason("EquipmentIssue")
	WaiverReasonCustomerGoodwill = WaiverReason("CustomerGoodwill")
	WaiverReasonDataCorrection   = WaiverReason("DataCorrection")
	WaiverReasonForceMajeure     = WaiverReason("ForceMajeure")
	WaiverReasonOther            = WaiverReason("Other")
)

func (r WaiverReason) String() string { return string(r) }

func WaiverReasonFromString(v string) (WaiverReason, error) {
	switch v {
	case "Weather":
		return WaiverReasonWeather, nil
	case "FacilityClosure":
		return WaiverReasonFacilityClosure, nil
	case "CarrierFault":
		return WaiverReasonCarrierFault, nil
	case "EquipmentIssue":
		return WaiverReasonEquipmentIssue, nil
	case "CustomerGoodwill":
		return WaiverReasonCustomerGoodwill, nil
	case "DataCorrection":
		return WaiverReasonDataCorrection, nil
	case "ForceMajeure":
		return WaiverReasonForceMajeure, nil
	case "Other":
		return WaiverReasonOther, nil
	default:
		return "", errors.New("invalid detention waiver reason")
	}
}

type CapKind string

const (
	CapKindNone            = CapKind("None")
	CapKindMaxMinutes      = CapKind("MaxBillableMinutes")
	CapKindPerStopAmount   = CapKind("MaxChargePerStop")
	CapKindPerDayAmount    = CapKind("MaxChargePerDay")
	CapKindPerShipment     = CapKind("MaxChargePerShipment")
	CapKindLayoverBoundary = CapKind("LayoverBoundary")
)

func (k CapKind) String() string { return string(k) }

type EvidenceKind string

const (
	EvidenceKindArrival      = EvidenceKind("Arrival")
	EvidenceKindDeparture    = EvidenceKind("Departure")
	EvidenceKindAppointment  = EvidenceKind("Appointment")
	EvidenceKindNotice       = EvidenceKind("Notice")
	EvidenceKindDocument     = EvidenceKind("Document")
	EvidenceKindStatusChange = EvidenceKind("StatusChange")
	EvidenceKindRecalculated = EvidenceKind("Recalculated")
	EvidenceKindWaiver       = EvidenceKind("Waiver")
	EvidenceKindDispute      = EvidenceKind("Dispute")
)

func (k EvidenceKind) String() string { return string(k) }

func EvidenceKindFromString(v string) (EvidenceKind, error) {
	switch v {
	case "Arrival":
		return EvidenceKindArrival, nil
	case "Departure":
		return EvidenceKindDeparture, nil
	case "Appointment":
		return EvidenceKindAppointment, nil
	case "Notice":
		return EvidenceKindNotice, nil
	case "Document":
		return EvidenceKindDocument, nil
	case "StatusChange":
		return EvidenceKindStatusChange, nil
	case "Recalculated":
		return EvidenceKindRecalculated, nil
	case "Waiver":
		return EvidenceKindWaiver, nil
	case "Dispute":
		return EvidenceKindDispute, nil
	default:
		return "", errors.New("invalid detention evidence kind")
	}
}

// EvidenceSource ranks how defensible a captured fact is in a dispute. The
// ordering drives the collectability score: a geofence crossing is testimony a
// customer rarely argues with, a dispatcher's keystroke is not.
type EvidenceSource string

const (
	EvidenceSourceTelematics = EvidenceSource("Telematics")
	EvidenceSourceGeofence   = EvidenceSource("Geofence")
	EvidenceSourceDriverApp  = EvidenceSource("DriverApp")
	EvidenceSourceEDI        = EvidenceSource("EDI")
	EvidenceSourceDocument   = EvidenceSource("Document")
	EvidenceSourceManual     = EvidenceSource("Manual")
	EvidenceSourceSystem     = EvidenceSource("System")
)

func (s EvidenceSource) String() string { return string(s) }

// Weight returns the defensibility contribution of a source, 0-100.
func (s EvidenceSource) Weight() int {
	switch s {
	case EvidenceSourceGeofence, EvidenceSourceTelematics:
		return 100
	case EvidenceSourceEDI:
		return 85
	case EvidenceSourceDocument:
		return 80
	case EvidenceSourceDriverApp:
		return 70
	case EvidenceSourceManual:
		return 35
	case EvidenceSourceSystem:
		return 50
	default:
		return 0
	}
}

func EvidenceSourceFromString(v string) (EvidenceSource, error) {
	switch v {
	case "Telematics":
		return EvidenceSourceTelematics, nil
	case "Geofence":
		return EvidenceSourceGeofence, nil
	case "DriverApp":
		return EvidenceSourceDriverApp, nil
	case "EDI":
		return EvidenceSourceEDI, nil
	case "Document":
		return EvidenceSourceDocument, nil
	case "Manual":
		return EvidenceSourceManual, nil
	case "System":
		return EvidenceSourceSystem, nil
	default:
		return "", errors.New("invalid detention evidence source")
	}
}

type NoticeChannel string

const (
	NoticeChannelEmail  = NoticeChannel("Email")
	NoticeChannelEDI    = NoticeChannel("EDI")
	NoticeChannelPortal = NoticeChannel("Portal")
	NoticeChannelManual = NoticeChannel("Manual")
)

func (c NoticeChannel) String() string { return string(c) }

func NoticeChannelFromString(v string) (NoticeChannel, error) {
	switch v {
	case "Email":
		return NoticeChannelEmail, nil
	case "EDI":
		return NoticeChannelEDI, nil
	case "Portal":
		return NoticeChannelPortal, nil
	case "Manual":
		return NoticeChannelManual, nil
	default:
		return "", errors.New("invalid detention notice channel")
	}
}

type NoticeKind string

const (
	NoticeKindWarning = NoticeKind("Warning")
	NoticeKindStarted = NoticeKind("Started")
	NoticeKindUpdate  = NoticeKind("Update")
	NoticeKindFinal   = NoticeKind("Final")
)

func (k NoticeKind) String() string { return string(k) }

func NoticeKindFromString(v string) (NoticeKind, error) {
	switch v {
	case "Warning":
		return NoticeKindWarning, nil
	case "Started":
		return NoticeKindStarted, nil
	case "Update":
		return NoticeKindUpdate, nil
	case "Final":
		return NoticeKindFinal, nil
	default:
		return "", errors.New("invalid detention notice kind")
	}
}

type NoticeDeliveryStatus string

const (
	NoticeDeliveryStatusQueued    = NoticeDeliveryStatus("Queued")
	NoticeDeliveryStatusSent      = NoticeDeliveryStatus("Sent")
	NoticeDeliveryStatusDelivered = NoticeDeliveryStatus("Delivered")
	NoticeDeliveryStatusOpened    = NoticeDeliveryStatus("Opened")
	NoticeDeliveryStatusBounced   = NoticeDeliveryStatus("Bounced")
	NoticeDeliveryStatusFailed    = NoticeDeliveryStatus("Failed")
)

func (s NoticeDeliveryStatus) String() string { return string(s) }

func (s NoticeDeliveryStatus) IsDelivered() bool {
	switch s {
	case NoticeDeliveryStatusDelivered, NoticeDeliveryStatusOpened:
		return true
	case NoticeDeliveryStatusQueued, NoticeDeliveryStatusSent,
		NoticeDeliveryStatusBounced, NoticeDeliveryStatusFailed:
		return false
	default:
		return false
	}
}

func (s NoticeDeliveryStatus) IsFailure() bool {
	switch s {
	case NoticeDeliveryStatusBounced, NoticeDeliveryStatusFailed:
		return true
	case NoticeDeliveryStatusQueued, NoticeDeliveryStatusSent,
		NoticeDeliveryStatusDelivered, NoticeDeliveryStatusOpened:
		return false
	default:
		return false
	}
}

func NoticeDeliveryStatusFromString(v string) (NoticeDeliveryStatus, error) {
	switch v {
	case "Queued":
		return NoticeDeliveryStatusQueued, nil
	case "Sent":
		return NoticeDeliveryStatusSent, nil
	case "Delivered":
		return NoticeDeliveryStatusDelivered, nil
	case "Opened":
		return NoticeDeliveryStatusOpened, nil
	case "Bounced":
		return NoticeDeliveryStatusBounced, nil
	case "Failed":
		return NoticeDeliveryStatusFailed, nil
	default:
		return "", errors.New("invalid detention notice delivery status")
	}
}
