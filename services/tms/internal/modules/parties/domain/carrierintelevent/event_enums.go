package carrierintelevent

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
