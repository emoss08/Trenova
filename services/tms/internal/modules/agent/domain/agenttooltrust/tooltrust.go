package agenttooltrust

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*ToolTrust)(nil)
)

// ToolTrust is the ledger behind earned autonomy: one row per agent and tool,
// counting how the people deciding on that agent's proposals for that tool
// have been deciding.
//
// The streak is what promotion reads. It counts approvals in a row that
// changed nothing; a modification says the agent was nearly right, a
// rejection says it was wrong, and a failed execution says the world did not
// agree with it, and each of those sends the streak back to zero. The totals
// are for people: they answer "how has this tool been going" on the agent's
// page without a query over decisions.
type ToolTrust struct {
	keel.Table `keel:"agent_tool_trust"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:att_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	AgentDefinitionID pulid.ID `json:"agentDefinitionId" keel:"agent_definition_id"`
	ToolName          string   `json:"toolName"          keel:"tool_name"`

	Streak            int `json:"streak"            keel:"streak"`
	Approvals         int `json:"approvals"         keel:"approvals"`
	Modifications     int `json:"modifications"     keel:"modifications"`
	Rejections        int `json:"rejections"        keel:"rejections"`
	ExecutionFailures int `json:"executionFailures" keel:"execution_failures"`

	// EarnedTier is the tier the ledger granted, when the tool's current tier
	// on the agent was earned rather than chosen by a person. It is what lets
	// a demotion take back only what trust gave: a tier a person set by hand
	// never matches it and is never touched.
	EarnedTier     agent.AutonomyTier `json:"earnedTier"     keel:"earned_tier,nullzero"`
	LastDecisionAt *int64             `json:"lastDecisionAt" keel:"last_decision_at"`
	PromotedAt     *int64             `json:"promotedAt"     keel:"promoted_at"`
	DemotedAt      *int64             `json:"demotedAt"      keel:"demoted_at"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

// TrustOutcome is what one decision or execution taught the ledger.
type TrustOutcome string

const (
	// TrustOutcomeApproved is an acceptance that changed nothing about the
	// proposal: the only outcome that lengthens the streak.
	TrustOutcomeApproved = TrustOutcome("Approved")
	// TrustOutcomeModified is an acceptance with the parameters changed.
	TrustOutcomeModified = TrustOutcome("Modified")
	// TrustOutcomeRejected is a proposal a person refused.
	TrustOutcomeRejected = TrustOutcome("Rejected")
	// TrustOutcomeExecutionFailed is an approved or automatic run of the tool
	// that did not go through.
	TrustOutcomeExecutionFailed = TrustOutcome("ExecutionFailed")
)

func (o TrustOutcome) IsValid() bool {
	switch o {
	case TrustOutcomeApproved,
		TrustOutcomeModified,
		TrustOutcomeRejected,
		TrustOutcomeExecutionFailed:
		return true
	default:
		return false
	}
}

// Clean reports whether the outcome extends a streak of clean approvals.
func (o TrustOutcome) Clean() bool { return o == TrustOutcomeApproved }

// Setback reports whether the outcome is one that should cost the tool a
// tier it earned. A modification is not: the person kept the action and
// changed a detail, which resets the streak but does not say the agent
// should be trusted less than before.
func (o TrustOutcome) Setback() bool {
	return o == TrustOutcomeRejected || o == TrustOutcomeExecutionFailed
}

// OutcomeOfDecision maps a recorded decision onto what it teaches the ledger.
func OutcomeOfDecision(
	decision agentdecision.DecisionType,
	modifications map[string]any,
) TrustOutcome {
	switch decision {
	case agentdecision.DecisionAccepted:
		if len(modifications) > 0 {
			return TrustOutcomeModified
		}

		return TrustOutcomeApproved
	case agentdecision.DecisionModified:
		return TrustOutcomeModified
	case agentdecision.DecisionRejected:
		return TrustOutcomeRejected
	default:
		return TrustOutcomeRejected
	}
}

// ReadyForPromotion reports whether the streak has reached the threshold.
func (t *ToolTrust) ReadyForPromotion(threshold int) bool {
	return threshold > 0 && t.Streak >= threshold
}

// HoldsEarnedTier reports whether the tool's current tier on its agent is the
// one the ledger granted, so a setback may take it back.
func (t *ToolTrust) HoldsEarnedTier(current agent.AutonomyTier) bool {
	return t.EarnedTier.IsValid() && t.EarnedTier == current
}

func (t *ToolTrust) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&t.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&t.AgentDefinitionID, validation.Required.Error("Agent is required")),
		validation.Field(&t.ToolName, validation.Required.Error("Tool name is required")),
		validation.Field(&t.EarnedTier,
			validation.When(t.EarnedTier != "",
				domainvalidation.ValidEnum[agent.AutonomyTier]("Earned tier is invalid"),
			),
		),
	))
}

func (t *ToolTrust) GetID() pulid.ID { return t.ID }

func (t *ToolTrust) GetCreatedAt() int64 { return t.CreatedAt }

func (t *ToolTrust) GetOrganizationID() pulid.ID { return t.OrganizationID }

func (t *ToolTrust) GetBusinessUnitID() pulid.ID { return t.BusinessUnitID }

func (t *ToolTrust) GetTableName() string { return "agent_tool_trust" }
