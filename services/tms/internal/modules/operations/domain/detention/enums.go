package detention

import "errors"

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

type BillingHoldReason string

const (
	BillingHoldReasonOverApprovalThreshold = BillingHoldReason("OverApprovalThreshold")
	BillingHoldReasonNoticeNotSent         = BillingHoldReason("NoticeNotSent")
	BillingHoldReasonEscalated             = BillingHoldReason("Escalated")
)

func (r BillingHoldReason) String() string { return string(r) }

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
