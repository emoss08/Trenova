package carrierintelenrollment

type DesiredState string

const (
	DesiredStateEnrolled    = DesiredState("Enrolled")
	DesiredStateNotEnrolled = DesiredState("NotEnrolled")
)

func (s DesiredState) String() string { return string(s) }

func (s DesiredState) IsValid() bool {
	return s == DesiredStateEnrolled || s == DesiredStateNotEnrolled
}

type VendorState string

const (
	VendorStateUnknown       = VendorState("Unknown")
	VendorStatePendingAdd    = VendorState("PendingAdd")
	VendorStateActive        = VendorState("Active")
	VendorStatePendingRemove = VendorState("PendingRemove")
	VendorStateRemoved       = VendorState("Removed")
	VendorStateFailed        = VendorState("Failed")
)

func (s VendorState) String() string { return string(s) }

func (s VendorState) IsValid() bool {
	switch s {
	case VendorStateUnknown, VendorStatePendingAdd, VendorStateActive, VendorStatePendingRemove,
		VendorStateRemoved, VendorStateFailed:
		return true
	default:
		return false
	}
}

type EnrollmentReason string

const (
	EnrollmentReasonPolicyAllActive    = EnrollmentReason("PolicyAllActive")
	EnrollmentReasonPolicyRecentUse    = EnrollmentReason("PolicyRecentUse")
	EnrollmentReasonAssignedOrTendered = EnrollmentReason("AssignedOrTendered")
	EnrollmentReasonManual             = EnrollmentReason("Manual")
	EnrollmentReasonSelfMonitor        = EnrollmentReason("SelfMonitor")
	EnrollmentReasonCustomerBroker     = EnrollmentReason("CustomerBroker")
)

func (r EnrollmentReason) String() string { return string(r) }

func (r EnrollmentReason) IsValid() bool {
	switch r {
	case EnrollmentReasonPolicyAllActive, EnrollmentReasonPolicyRecentUse,
		EnrollmentReasonAssignedOrTendered, EnrollmentReasonManual,
		EnrollmentReasonSelfMonitor, EnrollmentReasonCustomerBroker:
		return true
	default:
		return false
	}
}

type FeedType string

const (
	FeedTypeChangeFeed      = FeedType("ChangeFeed")
	FeedTypeSnapshotRefresh = FeedType("SnapshotRefresh")
)

func (t FeedType) String() string { return string(t) }

func (t FeedType) IsValid() bool {
	return t == FeedTypeChangeFeed || t == FeedTypeSnapshotRefresh
}

type FeedPauseReason string

const (
	FeedPauseReasonNone            = FeedPauseReason("")
	FeedPauseReasonPaymentRequired = FeedPauseReason("PaymentRequired")
	FeedPauseReasonSpendCap        = FeedPauseReason("SpendCap")
	FeedPauseReasonUnauthorized    = FeedPauseReason("Unauthorized")
	FeedPauseReasonDisabled        = FeedPauseReason("Disabled")
)

func (r FeedPauseReason) String() string { return string(r) }

func (r FeedPauseReason) IsValid() bool {
	switch r {
	case FeedPauseReasonNone, FeedPauseReasonPaymentRequired, FeedPauseReasonSpendCap,
		FeedPauseReasonUnauthorized, FeedPauseReasonDisabled:
		return true
	default:
		return false
	}
}
