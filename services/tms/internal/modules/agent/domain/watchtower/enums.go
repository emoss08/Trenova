package watchtower

import "github.com/emoss08/trenova/internal/modules/identity/domain/permission"

// SourceKind names the record an item stands in for. The source stays
// authoritative: the item is a projection of it, keyed by kind and id, and
// resolves when the source does.
type SourceKind string

const (
	SourceKindInsight                  = SourceKind("Insight")
	SourceKindAgentProposal            = SourceKind("AgentProposal")
	SourceKindAgentPlan                = SourceKind("AgentPlan")
	SourceKindAgentRunFailed           = SourceKind("AgentRunFailed")
	SourceKindAgentException           = SourceKind("AgentException")
	SourceKindServiceFailure           = SourceKind("ServiceFailure")
	SourceKindCarrierIntelEvent        = SourceKind("CarrierIntelEvent")
	SourceKindHOSViolation             = SourceKind("HOSViolation")
	SourceKindEDIInboundQuarantined    = SourceKind("EDIInboundQuarantined")
	SourceKindBillingException         = SourceKind("BillingException")
	SourceKindDetentionOccurrence      = SourceKind("DetentionOccurrence")
	SourceKindInboundMessage           = SourceKind("InboundMessage")
	SourceKindWorkerCredential         = SourceKind("WorkerCredential")
	SourceKindMoveCoverage             = SourceKind("MoveCoverage")
	SourceKindTelematicsStopVisit      = SourceKind("TelematicsStopVisit")
	SourceKindAgentQualityRegression   = SourceKind("AgentQualityRegression")
	SourceKindAccountingSync           = SourceKind("AccountingSync")
	SourceKindAccountingReconciliation = SourceKind("AccountingReconciliation")
	SourceKindNoCheckCall              = SourceKind("NoCheckCall")
	SourceKindEtaAtRisk                = SourceKind("EtaAtRisk")
	SourceKindMissedPickup             = SourceKind("MissedPickup")
	SourceKindMissedDelivery           = SourceKind("MissedDelivery")
	SourceKindPODMissing               = SourceKind("PodMissing")
	SourceKindTenderExhausted          = SourceKind("TenderExhausted")
	SourceKindFactoring                = SourceKind("Factoring")
)

// readResources is the permission a reader needs to be shown items of each
// kind: the same read that opens the source record, so the watchtower never
// shows someone a headline about a record they could not open.
var readResources = map[SourceKind]permission.Resource{
	SourceKindInsight:                  permission.ResourceInsight,
	SourceKindAgentProposal:            permission.ResourceAgentProposal,
	SourceKindAgentPlan:                permission.ResourceAgentProposal,
	SourceKindAgentRunFailed:           permission.ResourceAgentRun,
	SourceKindAgentException:           permission.ResourceAgentException,
	SourceKindServiceFailure:           permission.ResourceServiceFailure,
	SourceKindCarrierIntelEvent:        permission.ResourceCarrierIntelligence,
	SourceKindHOSViolation:             permission.ResourceWorker,
	SourceKindEDIInboundQuarantined:    permission.ResourceEDI,
	SourceKindBillingException:         permission.ResourceBillingQueue,
	SourceKindDetentionOccurrence:      permission.ResourceDetention,
	SourceKindInboundMessage:           permission.ResourceInboundMessage,
	SourceKindWorkerCredential:         permission.ResourceWorker,
	SourceKindMoveCoverage:             permission.ResourceShipmentMove,
	SourceKindTelematicsStopVisit:      permission.ResourceShipmentMove,
	SourceKindAgentQualityRegression:   permission.ResourceAgentEvalSuite,
	SourceKindAccountingSync:           permission.ResourceAccountingSync,
	SourceKindAccountingReconciliation: permission.ResourceAccountingReconciliation,
	SourceKindNoCheckCall:              permission.ResourceShipmentMove,
	SourceKindEtaAtRisk:                permission.ResourceShipment,
	SourceKindMissedPickup:             permission.ResourceShipment,
	SourceKindMissedDelivery:           permission.ResourceShipment,
	SourceKindPODMissing:               permission.ResourceShipment,
	SourceKindTenderExhausted:          permission.ResourceTender,
	SourceKindFactoring:                permission.ResourceInvoice,
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
	case SourceKindInsight:
		return "Insight"
	case SourceKindAgentProposal:
		return "Proposal"
	case SourceKindAgentPlan:
		return "Plan"
	case SourceKindAgentRunFailed:
		return "Agent run failed"
	case SourceKindAgentException:
		return "Agent exception"
	case SourceKindServiceFailure:
		return "Service failure"
	case SourceKindCarrierIntelEvent:
		return "Carrier change"
	case SourceKindHOSViolation:
		return "Hours of service"
	case SourceKindEDIInboundQuarantined:
		return "EDI quarantined"
	case SourceKindBillingException:
		return "Billing exception"
	case SourceKindDetentionOccurrence:
		return "Detention"
	case SourceKindInboundMessage:
		return "Inbound message"
	case SourceKindWorkerCredential:
		return "Credential expiring"
	case SourceKindMoveCoverage:
		return "Coverage at risk"
	case SourceKindTelematicsStopVisit:
		return "Stop visit not recorded"
	case SourceKindAgentQualityRegression:
		return "Agent quality regressed"
	case SourceKindAccountingSync:
		return "Accounting sync"
	case SourceKindAccountingReconciliation:
		return "Accounting reconciliation"
	case SourceKindNoCheckCall:
		return "No check call"
	case SourceKindEtaAtRisk:
		return "ETA at risk"
	case SourceKindMissedPickup:
		return "Missed pickup"
	case SourceKindMissedDelivery:
		return "Missed delivery"
	case SourceKindPODMissing:
		return "POD missing"
	case SourceKindTenderExhausted:
		return "Tender exhausted"
	case SourceKindFactoring:
		return "Factoring"
	default:
		return string(k)
	}
}

func AllSourceKinds() []SourceKind {
	return []SourceKind{
		SourceKindInsight,
		SourceKindAgentProposal,
		SourceKindAgentPlan,
		SourceKindAgentRunFailed,
		SourceKindAgentException,
		SourceKindServiceFailure,
		SourceKindCarrierIntelEvent,
		SourceKindHOSViolation,
		SourceKindEDIInboundQuarantined,
		SourceKindBillingException,
		SourceKindDetentionOccurrence,
		SourceKindInboundMessage,
		SourceKindWorkerCredential,
		SourceKindMoveCoverage,
		SourceKindTelematicsStopVisit,
		SourceKindAgentQualityRegression,
		SourceKindAccountingSync,
		SourceKindAccountingReconciliation,
		SourceKindNoCheckCall,
		SourceKindEtaAtRisk,
		SourceKindMissedPickup,
		SourceKindMissedDelivery,
		SourceKindPODMissing,
		SourceKindTenderExhausted,
		SourceKindFactoring,
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

func (k SourceKind) OverseesAgents() bool {
	switch k {
	case SourceKindAgentProposal,
		SourceKindAgentPlan,
		SourceKindAgentRunFailed,
		SourceKindAgentException,
		SourceKindAgentQualityRegression:
		return true
	case SourceKindInsight,
		SourceKindServiceFailure,
		SourceKindCarrierIntelEvent,
		SourceKindHOSViolation,
		SourceKindEDIInboundQuarantined,
		SourceKindBillingException,
		SourceKindDetentionOccurrence,
		SourceKindInboundMessage,
		SourceKindWorkerCredential,
		SourceKindMoveCoverage,
		SourceKindTelematicsStopVisit,
		SourceKindAccountingSync,
		SourceKindAccountingReconciliation,
		SourceKindNoCheckCall,
		SourceKindEtaAtRisk,
		SourceKindMissedPickup,
		SourceKindMissedDelivery,
		SourceKindPODMissing,
		SourceKindTenderExhausted,
		SourceKindFactoring:
		return false
	default:
		return false
	}
}
