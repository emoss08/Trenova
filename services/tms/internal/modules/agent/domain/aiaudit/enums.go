package aiaudit

// Kind is what an AI audit event records.
type Kind string

const (
	KindRunStarted              = Kind("RunStarted")
	KindRunEnded                = Kind("RunEnded")
	KindModelCall               = Kind("ModelCall")
	KindToolCall                = Kind("ToolCall")
	KindToolRefused             = Kind("ToolRefused")
	KindProposalFiled           = Kind("ProposalFiled")
	KindProposalDecided         = Kind("ProposalDecided")
	KindProposalExecuted        = Kind("ProposalExecuted")
	KindProposalExecutionFailed = Kind("ProposalExecutionFailed")
	KindProposalSimulated       = Kind("ProposalSimulated")
	KindProposalExpired         = Kind("ProposalExpired")
	KindDelegationStarted       = Kind("DelegationStarted")
	KindDelegationEnded         = Kind("DelegationEnded")
)

func AllKinds() []Kind {
	return []Kind{
		KindRunStarted,
		KindRunEnded,
		KindModelCall,
		KindToolCall,
		KindToolRefused,
		KindProposalFiled,
		KindProposalDecided,
		KindProposalExecuted,
		KindProposalExecutionFailed,
		KindProposalSimulated,
		KindProposalExpired,
		KindDelegationStarted,
		KindDelegationEnded,
	}
}

func (k Kind) IsValid() bool {
	for _, known := range AllKinds() {
		if k == known {
			return true
		}
	}

	return false
}

func (k Kind) String() string { return string(k) }

// Outcome is how the thing an event records turned out.
type Outcome string

const (
	OutcomeStarted   = Outcome("Started")
	OutcomeCompleted = Outcome("Completed")
	OutcomeFailed    = Outcome("Failed")
	OutcomeRefused   = Outcome("Refused")
	OutcomeStopped   = Outcome("Stopped")
	OutcomeSucceeded = Outcome("Succeeded")
	OutcomeRan       = Outcome("Ran")
	OutcomeProposed  = Outcome("Proposed")
	OutcomeSimulated = Outcome("Simulated")
	OutcomeDenied    = Outcome("Denied")
	OutcomeUnknown   = Outcome("Unknown")
	OutcomeFiled     = Outcome("Filed")
	OutcomeAccepted  = Outcome("Accepted")
	OutcomeModified  = Outcome("Modified")
	OutcomeRejected  = Outcome("Rejected")
	OutcomeExpired   = Outcome("Expired")
	OutcomeExhausted = Outcome("Exhausted")
	OutcomeDeclined  = Outcome("Declined")
)

func AllOutcomes() []Outcome {
	return []Outcome{
		OutcomeStarted,
		OutcomeCompleted,
		OutcomeFailed,
		OutcomeRefused,
		OutcomeStopped,
		OutcomeSucceeded,
		OutcomeRan,
		OutcomeProposed,
		OutcomeSimulated,
		OutcomeDenied,
		OutcomeUnknown,
		OutcomeFiled,
		OutcomeAccepted,
		OutcomeModified,
		OutcomeRejected,
		OutcomeExpired,
		OutcomeExhausted,
		OutcomeDeclined,
	}
}

func (o Outcome) IsValid() bool {
	for _, known := range AllOutcomes() {
		if o == known {
			return true
		}
	}

	return false
}

func (o Outcome) String() string { return string(o) }

// PrincipalType is who an event's action was taken as: a person, an agent
// working unattended, or the system itself.
type PrincipalType string

const (
	PrincipalUser   = PrincipalType("User")
	PrincipalAgent  = PrincipalType("Agent")
	PrincipalSystem = PrincipalType("System")
)

func AllPrincipalTypes() []PrincipalType {
	return []PrincipalType{PrincipalUser, PrincipalAgent, PrincipalSystem}
}

func (p PrincipalType) IsValid() bool {
	switch p {
	case PrincipalUser, PrincipalAgent, PrincipalSystem:
		return true
	default:
		return false
	}
}

// Purpose separates live work from evaluation replays, which the trail keeps
// but hides by default.
type Purpose string

const (
	PurposeLive       = Purpose("Live")
	PurposeEvaluation = Purpose("Evaluation")
)

func AllPurposes() []Purpose {
	return []Purpose{PurposeLive, PurposeEvaluation}
}

func (p Purpose) IsValid() bool {
	return p == PurposeLive || p == PurposeEvaluation
}

// ExportFormat is the file an export is written as.
type ExportFormat string

const (
	ExportFormatCSV  = ExportFormat("CSV")
	ExportFormatJSON = ExportFormat("JSON")
)

func AllExportFormats() []ExportFormat {
	return []ExportFormat{ExportFormatCSV, ExportFormatJSON}
}

func (f ExportFormat) IsValid() bool {
	return f == ExportFormatCSV || f == ExportFormatJSON
}

func (f ExportFormat) ContentType() string {
	if f == ExportFormatJSON {
		return "application/json"
	}

	return "text/csv; charset=utf-8"
}

func (f ExportFormat) Extension() string {
	if f == ExportFormatJSON {
		return "json"
	}

	return "csv"
}

// ExportStatus is how far an export got.
type ExportStatus string

const (
	ExportStatusPending   = ExportStatus("Pending")
	ExportStatusRunning   = ExportStatus("Running")
	ExportStatusSucceeded = ExportStatus("Succeeded")
	ExportStatusFailed    = ExportStatus("Failed")
	ExportStatusExpired   = ExportStatus("Expired")
)

func AllExportStatuses() []ExportStatus {
	return []ExportStatus{
		ExportStatusPending,
		ExportStatusRunning,
		ExportStatusSucceeded,
		ExportStatusFailed,
		ExportStatusExpired,
	}
}

func (s ExportStatus) IsValid() bool {
	for _, known := range AllExportStatuses() {
		if s == known {
			return true
		}
	}

	return false
}

func (s ExportStatus) Terminal() bool {
	switch s {
	case ExportStatusSucceeded, ExportStatusFailed, ExportStatusExpired:
		return true
	case ExportStatusPending, ExportStatusRunning:
		return false
	default:
		return false
	}
}

// VerificationStatus is what the last check of a tenant's chain found.
type VerificationStatus string

const (
	VerificationVerified = VerificationStatus("Verified")
	VerificationMismatch = VerificationStatus("Mismatch")
	// VerificationKeyMissing is a chain holding rows signed with a key that is
	// no longer configured, so they cannot be checked.
	VerificationKeyMissing = VerificationStatus("KeyMissing")
)

func AllVerificationStatuses() []VerificationStatus {
	return []VerificationStatus{
		VerificationVerified,
		VerificationMismatch,
		VerificationKeyMissing,
	}
}

func (s VerificationStatus) IsValid() bool {
	switch s {
	case VerificationVerified, VerificationMismatch, VerificationKeyMissing:
		return true
	default:
		return false
	}
}

// Source names a table the projector reads, and the watermark kept for it.
type Source string

const (
	SourceAgentRuns      = Source("agent_runs")
	SourceAssistantTurns = Source("assistant_turns")
	SourceUsage          = Source("ai_usage_records")
	SourceRunSteps       = Source("agent_run_steps")
	SourceRunEvents      = Source("agent_run_events")
	SourceProposals      = Source("agent_proposals")
	SourceDecisions      = Source("agent_decisions")
)

func AllSources() []Source {
	return []Source{
		SourceAgentRuns,
		SourceAssistantTurns,
		SourceUsage,
		SourceRunSteps,
		SourceRunEvents,
		SourceProposals,
		SourceDecisions,
	}
}

func (s Source) String() string { return string(s) }
