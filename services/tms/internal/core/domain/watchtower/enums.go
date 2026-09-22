package watchtower

import "github.com/emoss08/trenova/internal/core/domain/permission"

// SourceKind names the record an item stands in for. The source stays
// authoritative: the item is a projection of it, keyed by kind and id, and
// resolves when the source does.
type SourceKind string

const (
	SourceInsight               = SourceKind("Insight")
	SourceAgentProposal         = SourceKind("AgentProposal")
	SourceAgentPlan             = SourceKind("AgentPlan")
	SourceAgentRunFailed        = SourceKind("AgentRunFailed")
	SourceAgentException        = SourceKind("AgentException")
	SourceServiceFailure        = SourceKind("ServiceFailure")
	SourceCarrierIntelEvent     = SourceKind("CarrierIntelEvent")
	SourceHOSViolation          = SourceKind("HOSViolation")
	SourceWeatherAlert          = SourceKind("WeatherAlert")
	SourceEDIInboundQuarantined = SourceKind("EDIInboundQuarantined")
	SourceBillingException      = SourceKind("BillingException")
	SourceDetentionOccurrence   = SourceKind("DetentionOccurrence")
	SourceInboundMessage        = SourceKind("InboundMessage")
	SourceWorkerCredential      = SourceKind("WorkerCredential")
	SourceMoveCoverage          = SourceKind("MoveCoverage")
)

// readResources is the permission a reader needs to be shown items of each
// kind: the same read that opens the source record, so the watchtower never
// shows someone a headline about a record they could not open.
var readResources = map[SourceKind]permission.Resource{
	SourceInsight:               permission.ResourceInsight,
	SourceAgentProposal:         permission.ResourceAgentProposal,
	SourceAgentPlan:             permission.ResourceAgentProposal,
	SourceAgentRunFailed:        permission.ResourceAgentRun,
	SourceAgentException:        permission.ResourceAgentException,
	SourceServiceFailure:        permission.ResourceServiceFailure,
	SourceCarrierIntelEvent:     permission.ResourceCarrierIntelligence,
	SourceHOSViolation:          permission.ResourceWorker,
	SourceWeatherAlert:          permission.ResourceShipment,
	SourceEDIInboundQuarantined: permission.ResourceEDI,
	SourceBillingException:      permission.ResourceBillingQueue,
	SourceDetentionOccurrence:   permission.ResourceDetentionPolicy,
	SourceInboundMessage:        permission.ResourceInboundMessage,
	SourceWorkerCredential:      permission.ResourceWorker,
	SourceMoveCoverage:          permission.ResourceShipmentMove,
}

func (k SourceKind) IsValid() bool {
	_, ok := readResources[k]
	return ok
}

func (k SourceKind) String() string { return string(k) }

// ReadResource is the permission resource a reader must hold read on to be
// shown this kind.
func (k SourceKind) ReadResource() permission.Resource {
	return readResources[k]
}

// Label is the kind in the reader's words, for a filter chip.
func (k SourceKind) Label() string {
	switch k {
	case SourceInsight:
		return "Insight"
	case SourceAgentProposal:
		return "Proposal"
	case SourceAgentPlan:
		return "Plan"
	case SourceAgentRunFailed:
		return "Agent run failed"
	case SourceAgentException:
		return "Agent exception"
	case SourceServiceFailure:
		return "Service failure"
	case SourceCarrierIntelEvent:
		return "Carrier change"
	case SourceHOSViolation:
		return "Hours of service"
	case SourceWeatherAlert:
		return "Weather"
	case SourceEDIInboundQuarantined:
		return "EDI quarantined"
	case SourceBillingException:
		return "Billing exception"
	case SourceDetentionOccurrence:
		return "Detention"
	case SourceInboundMessage:
		return "Inbound message"
	case SourceWorkerCredential:
		return "Credential expiring"
	case SourceMoveCoverage:
		return "Coverage at risk"
	default:
		return string(k)
	}
}

func AllSourceKinds() []SourceKind {
	return []SourceKind{
		SourceInsight,
		SourceAgentProposal,
		SourceAgentPlan,
		SourceAgentRunFailed,
		SourceAgentException,
		SourceServiceFailure,
		SourceCarrierIntelEvent,
		SourceHOSViolation,
		SourceWeatherAlert,
		SourceEDIInboundQuarantined,
		SourceBillingException,
		SourceDetentionOccurrence,
		SourceInboundMessage,
		SourceWorkerCredential,
		SourceMoveCoverage,
	}
}

// Severity is how loudly an item presents itself. The source decides it
// from its own thresholds; the watchtower only orders by it.
type Severity string

const (
	SeverityInfo     = Severity("Info")
	SeverityWarning  = Severity("Warning")
	SeverityCritical = Severity("Critical")
)

func (s Severity) IsValid() bool {
	switch s {
	case SeverityInfo, SeverityWarning, SeverityCritical:
		return true
	default:
		return false
	}
}

func (s Severity) String() string { return string(s) }

// Rank orders severities, highest first, so a sort does not read the
// strings alphabetically and put Critical after Info.
func (s Severity) Rank() int {
	switch s {
	case SeverityCritical:
		return 3
	case SeverityWarning:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}
