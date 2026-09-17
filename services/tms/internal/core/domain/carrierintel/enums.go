package carrierintel

type Section string

const (
	SectionIdentity      = Section("Identity")
	SectionAuthority     = Section("Authority")
	SectionInsurance     = Section("Insurance")
	SectionSafety        = Section("Safety")
	SectionBasics        = Section("Basics")
	SectionInspections   = Section("Inspections")
	SectionCrashes       = Section("Crashes")
	SectionFleet         = Section("Fleet")
	SectionEquipment     = Section("Equipment")
	SectionContacts      = Section("Contacts")
	SectionOperations    = Section("Operations")
	SectionChangeHistory = Section("ChangeHistory")
	SectionNetwork       = Section("Network")
	SectionLanes         = Section("Lanes")
	SectionBenchmarks    = Section("Benchmarks")
)

func (s Section) String() string { return string(s) }

func (s Section) IsValid() bool {
	switch s {
	case SectionIdentity, SectionAuthority, SectionInsurance, SectionSafety, SectionBasics,
		SectionInspections, SectionCrashes, SectionFleet, SectionEquipment, SectionContacts,
		SectionOperations, SectionChangeHistory, SectionNetwork, SectionLanes,
		SectionBenchmarks:
		return true
	default:
		return false
	}
}

func AllSections() []Section {
	return []Section{
		SectionIdentity,
		SectionAuthority,
		SectionInsurance,
		SectionSafety,
		SectionBasics,
		SectionInspections,
		SectionCrashes,
		SectionFleet,
		SectionEquipment,
		SectionContacts,
		SectionOperations,
		SectionChangeHistory,
		SectionNetwork,
		SectionLanes,
		SectionBenchmarks,
	}
}

type RuleAction string

const (
	RuleActionBlock  = RuleAction("Block")
	RuleActionWarn   = RuleAction("Warn")
	RuleActionNotify = RuleAction("Notify")
	RuleActionOff    = RuleAction("Off")
)

func (a RuleAction) String() string { return string(a) }

func (a RuleAction) IsValid() bool {
	switch a {
	case RuleActionBlock, RuleActionWarn, RuleActionNotify, RuleActionOff:
		return true
	default:
		return false
	}
}

func (a RuleAction) Severity() Severity {
	switch a {
	case RuleActionBlock:
		return SeverityCritical
	case RuleActionWarn:
		return SeverityMedium
	case RuleActionNotify:
		return SeverityLow
	default:
		return SeverityInfo
	}
}

type EnrollmentPolicy string

const (
	EnrollmentPolicyAllActive    = EnrollmentPolicy("AllActive")
	EnrollmentPolicyRecentlyUsed = EnrollmentPolicy("RecentlyUsed")
	EnrollmentPolicyManual       = EnrollmentPolicy("Manual")
)

func (p EnrollmentPolicy) String() string { return string(p) }

func (p EnrollmentPolicy) IsValid() bool {
	switch p {
	case EnrollmentPolicyAllActive, EnrollmentPolicyRecentlyUsed, EnrollmentPolicyManual:
		return true
	default:
		return false
	}
}

type SubjectType string

const (
	SubjectTypeCarrier      = SubjectType("Carrier")
	SubjectTypeOrganization = SubjectType("Organization")
	SubjectTypeCustomer     = SubjectType("Customer")
	SubjectTypeProspect     = SubjectType("Prospect")
)

func (t SubjectType) String() string { return string(t) }

func (t SubjectType) IsValid() bool {
	switch t {
	case SubjectTypeCarrier, SubjectTypeOrganization, SubjectTypeCustomer, SubjectTypeProspect:
		return true
	default:
		return false
	}
}

type LookupDepth string

const (
	LookupDepthFMCSA = LookupDepth("FMCSA")
	LookupDepthLite  = LookupDepth("Lite")
	LookupDepthFull  = LookupDepth("Full")
)

func (d LookupDepth) String() string { return string(d) }

func (d LookupDepth) IsValid() bool {
	switch d {
	case LookupDepthFMCSA, LookupDepthLite, LookupDepthFull:
		return true
	default:
		return false
	}
}

func (d LookupDepth) Rank() int {
	switch d {
	case LookupDepthFMCSA:
		return 1
	case LookupDepthLite:
		return 2
	case LookupDepthFull:
		return 3
	default:
		return 0
	}
}

func (d LookupDepth) Satisfies(minimum LookupDepth) bool {
	return d.Rank() >= minimum.Rank()
}

type SnapshotSource string

const (
	SnapshotSourcePrimary         = SnapshotSource("Primary")
	SnapshotSourceFallback        = SnapshotSource("Fallback")
	SnapshotSourceChangeFeedPatch = SnapshotSource("ChangeFeedPatch")
)

func (s SnapshotSource) String() string { return string(s) }

func (s SnapshotSource) IsValid() bool {
	switch s {
	case SnapshotSourcePrimary, SnapshotSourceFallback, SnapshotSourceChangeFeedPatch:
		return true
	default:
		return false
	}
}

type EventSource string

const (
	EventSourceNativeChangeFeed      = EventSource("NativeChangeFeed")
	EventSourceSnapshotDiff          = EventSource("SnapshotDiff")
	EventSourceRuleEvaluation        = EventSource("RuleEvaluation")
	EventSourceEquipmentVerification = EventSource("EquipmentVerification")
	EventSourceEnrollment            = EventSource("Enrollment")
	EventSourceProviderError         = EventSource("ProviderError")
	EventSourceOverride              = EventSource("Override")
)

func (s EventSource) String() string { return string(s) }

func (s EventSource) IsValid() bool {
	switch s {
	case EventSourceNativeChangeFeed, EventSourceSnapshotDiff, EventSourceRuleEvaluation,
		EventSourceEquipmentVerification, EventSourceEnrollment, EventSourceProviderError,
		EventSourceOverride:
		return true
	default:
		return false
	}
}

type EventStatus string

const (
	EventStatusOpen         = EventStatus("Open")
	EventStatusAcknowledged = EventStatus("Acknowledged")
	EventStatusResolved     = EventStatus("Resolved")
	EventStatusDismissed    = EventStatus("Dismissed")
)

func (s EventStatus) String() string { return string(s) }

func (s EventStatus) IsValid() bool {
	switch s {
	case EventStatusOpen, EventStatusAcknowledged, EventStatusResolved, EventStatusDismissed:
		return true
	default:
		return false
	}
}

func (s EventStatus) IsClosed() bool {
	return s == EventStatusResolved || s == EventStatusDismissed
}

type EventResolution string

const (
	EventResolutionCarrierUpdated   = EventResolution("CarrierUpdated")
	EventResolutionCarrierBlocked   = EventResolution("CarrierBlocked")
	EventResolutionOverrideGranted  = EventResolution("OverrideGranted")
	EventResolutionNoActionRequired = EventResolution("NoActionRequired")
	EventResolutionFalsePositive    = EventResolution("FalsePositive")
)

func (r EventResolution) String() string { return string(r) }

func (r EventResolution) IsValid() bool {
	switch r {
	case EventResolutionCarrierUpdated, EventResolutionCarrierBlocked,
		EventResolutionOverrideGranted, EventResolutionNoActionRequired,
		EventResolutionFalsePositive:
		return true
	default:
		return false
	}
}

type Severity string

const (
	SeverityCritical = Severity("Critical")
	SeverityHigh     = Severity("High")
	SeverityMedium   = Severity("Medium")
	SeverityLow      = Severity("Low")
	SeverityInfo     = Severity("Info")
)

func (s Severity) String() string { return string(s) }

func (s Severity) IsValid() bool {
	switch s {
	case SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo:
		return true
	default:
		return false
	}
}

func (s Severity) Rank() int {
	switch s {
	case SeverityCritical:
		return 5
	case SeverityHigh:
		return 4
	case SeverityMedium:
		return 3
	case SeverityLow:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}

type RiskLevel string

const (
	RiskLevelLow      = RiskLevel("Low")
	RiskLevelModerate = RiskLevel("Moderate")
	RiskLevelElevated = RiskLevel("Elevated")
	RiskLevelHigh     = RiskLevel("High")
	RiskLevelVeryHigh = RiskLevel("VeryHigh")
	RiskLevelUnknown  = RiskLevel("Unknown")
)

func (l RiskLevel) String() string { return string(l) }

func (l RiskLevel) IsValid() bool {
	switch l {
	case RiskLevelLow, RiskLevelModerate, RiskLevelElevated, RiskLevelHigh, RiskLevelVeryHigh,
		RiskLevelUnknown:
		return true
	default:
		return false
	}
}

func (l RiskLevel) Rank() int {
	switch l {
	case RiskLevelLow:
		return 1
	case RiskLevelModerate:
		return 2
	case RiskLevelElevated:
		return 3
	case RiskLevelHigh:
		return 4
	case RiskLevelVeryHigh:
		return 5
	default:
		return 0
	}
}

type ReviewState string

const (
	ReviewStateNone        = ReviewState("None")
	ReviewStateNeedsReview = ReviewState("NeedsReview")
	ReviewStateReviewed    = ReviewState("Reviewed")
)

func (s ReviewState) String() string { return string(s) }

func (s ReviewState) IsValid() bool {
	switch s {
	case ReviewStateNone, ReviewStateNeedsReview, ReviewStateReviewed:
		return true
	default:
		return false
	}
}

type VerificationResult string

const (
	VerificationResultMatch         = VerificationResult("Match")
	VerificationResultMismatch      = VerificationResult("Mismatch")
	VerificationResultNotFound      = VerificationResult("NotFound")
	VerificationResultUnverifiable  = VerificationResult("Unverifiable")
	VerificationResultProviderError = VerificationResult("ProviderError")
)

func (r VerificationResult) String() string { return string(r) }

func (r VerificationResult) IsValid() bool {
	switch r {
	case VerificationResultMatch, VerificationResultMismatch, VerificationResultNotFound,
		VerificationResultUnverifiable, VerificationResultProviderError:
		return true
	default:
		return false
	}
}

type OutagePolicy string

const (
	OutagePolicyFailOpen   = OutagePolicy("FailOpen")
	OutagePolicyFailClosed = OutagePolicy("FailClosed")
)

func (p OutagePolicy) String() string { return string(p) }

func (p OutagePolicy) IsValid() bool {
	return p == OutagePolicyFailOpen || p == OutagePolicyFailClosed
}

type Purpose string

const (
	PurposeVet         = Purpose("Vet")
	PurposeRefresh     = Purpose("Refresh")
	PurposePreTender   = Purpose("PreTender")
	PurposeMonitor     = Purpose("Monitor")
	PurposeSourcing    = Purpose("Sourcing")
	PurposeVerify      = Purpose("Verify")
	PurposeSelfMonitor = Purpose("SelfMonitor")
	PurposeReconcile   = Purpose("Reconcile")
	PurposeTest        = Purpose("Test")
)

func (p Purpose) String() string { return string(p) }

func (p Purpose) IsValid() bool {
	switch p {
	case PurposeVet, PurposeRefresh, PurposePreTender, PurposeMonitor, PurposeSourcing,
		PurposeVerify, PurposeSelfMonitor, PurposeReconcile, PurposeTest:
		return true
	default:
		return false
	}
}

func (p Purpose) IsInteractive() bool {
	switch p {
	case PurposeVet, PurposePreTender, PurposeSourcing, PurposeVerify, PurposeTest:
		return true
	default:
		return false
	}
}

func (p Purpose) DefaultDepth() LookupDepth {
	switch p {
	case PurposeVet, PurposeSelfMonitor:
		return LookupDepthFull
	case PurposeSourcing, PurposeVerify:
		return LookupDepthLite
	default:
		return LookupDepthFMCSA
	}
}

type EnrollmentMode string

const (
	EnrollmentModeNative       = EnrollmentMode("Native")
	EnrollmentModeSnapshotDiff = EnrollmentMode("SnapshotDiff")
)

func (m EnrollmentMode) String() string { return string(m) }

func (m EnrollmentMode) IsValid() bool {
	return m == EnrollmentModeNative || m == EnrollmentModeSnapshotDiff
}

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

type UnitType string

const (
	UnitTypeTractor  = UnitType("Tractor")
	UnitTypeTrailer  = UnitType("Trailer")
	UnitTypeStraight = UnitType("Straight")
)

func (t UnitType) String() string { return string(t) }

func (t UnitType) IsValid() bool {
	switch t {
	case UnitTypeTractor, UnitTypeTrailer, UnitTypeStraight:
		return true
	default:
		return false
	}
}

type UsageOutcome string

const (
	UsageOutcomeSuccess     = UsageOutcome("Success")
	UsageOutcomeNotFound    = UsageOutcome("NotFound")
	UsageOutcomeRateLimited = UsageOutcome("RateLimited")
	UsageOutcomeClientError = UsageOutcome("ClientError")
	UsageOutcomeServerError = UsageOutcome("ServerError")
	UsageOutcomeTransport   = UsageOutcome("TransportError")
	UsageOutcomeDenied      = UsageOutcome("Denied")
)

func (o UsageOutcome) String() string { return string(o) }

func (o UsageOutcome) IsValid() bool {
	switch o {
	case UsageOutcomeSuccess, UsageOutcomeNotFound, UsageOutcomeRateLimited,
		UsageOutcomeClientError, UsageOutcomeServerError, UsageOutcomeTransport,
		UsageOutcomeDenied:
		return true
	default:
		return false
	}
}

type AuthorityStatus string

const (
	AuthorityStatusActive   = AuthorityStatus("Active")
	AuthorityStatusInactive = AuthorityStatus("Inactive")
	AuthorityStatusRevoked  = AuthorityStatus("Revoked")
	AuthorityStatusNone     = AuthorityStatus("None")
	AuthorityStatusUnknown  = AuthorityStatus("Unknown")
)

func (s AuthorityStatus) String() string { return string(s) }

func (s AuthorityStatus) IsValid() bool {
	switch s {
	case AuthorityStatusActive, AuthorityStatusInactive, AuthorityStatusRevoked,
		AuthorityStatusNone, AuthorityStatusUnknown:
		return true
	default:
		return false
	}
}

type SafetyRating string

const (
	SafetyRatingSatisfactory   = SafetyRating("Satisfactory")
	SafetyRatingConditional    = SafetyRating("Conditional")
	SafetyRatingUnsatisfactory = SafetyRating("Unsatisfactory")
	SafetyRatingNotRated       = SafetyRating("NotRated")
)

func (r SafetyRating) String() string { return string(r) }

func (r SafetyRating) IsValid() bool {
	switch r {
	case SafetyRatingSatisfactory, SafetyRatingConditional, SafetyRatingUnsatisfactory,
		SafetyRatingNotRated:
		return true
	default:
		return false
	}
}

type NetworkKind string

const (
	NetworkKindAddress   = NetworkKind("Address")
	NetworkKindPhone     = NetworkKind("Phone")
	NetworkKindEmail     = NetworkKind("Email")
	NetworkKindEIN       = NetworkKind("EIN")
	NetworkKindEquipment = NetworkKind("Equipment")
)

func (k NetworkKind) String() string { return string(k) }

func (k NetworkKind) IsValid() bool {
	switch k {
	case NetworkKindAddress, NetworkKindPhone, NetworkKindEmail, NetworkKindEIN,
		NetworkKindEquipment:
		return true
	default:
		return false
	}
}

type InsuranceFilingType string

const (
	InsuranceFilingTypeBIPD  = InsuranceFilingType("BIPD")
	InsuranceFilingTypeCargo = InsuranceFilingType("Cargo")
	InsuranceFilingTypeBond  = InsuranceFilingType("Bond")
	InsuranceFilingTypeOther = InsuranceFilingType("Other")
)

func (t InsuranceFilingType) String() string { return string(t) }

func (t InsuranceFilingType) IsValid() bool {
	switch t {
	case InsuranceFilingTypeBIPD, InsuranceFilingTypeCargo, InsuranceFilingTypeBond,
		InsuranceFilingTypeOther:
		return true
	default:
		return false
	}
}
