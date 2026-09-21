package agent

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*ToolTrust)(nil)
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
	bun.BaseModel `bun:"table:agent_tool_trust,alias:att" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	AgentDefinitionID pulid.ID `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),notnull"`
	ToolName          string   `json:"toolName"          bun:"tool_name,type:VARCHAR(100),notnull"`

	Streak            int `json:"streak"            bun:"streak,type:INTEGER,notnull"`
	Approvals         int `json:"approvals"         bun:"approvals,type:INTEGER,notnull"`
	Modifications     int `json:"modifications"     bun:"modifications,type:INTEGER,notnull"`
	Rejections        int `json:"rejections"        bun:"rejections,type:INTEGER,notnull"`
	ExecutionFailures int `json:"executionFailures" bun:"execution_failures,type:INTEGER,notnull"`

	// EarnedTier is the tier the ledger granted, when the tool's current tier
	// on the agent was earned rather than chosen by a person. It is what lets
	// a demotion take back only what trust gave: a tier a person set by hand
	// never matches it and is never touched.
	EarnedTier     AutonomyTier `json:"earnedTier"     bun:"earned_tier,type:agent_autonomy_tier_enum,nullzero"`
	LastDecisionAt *int64       `json:"lastDecisionAt" bun:"last_decision_at,type:BIGINT,nullzero"`
	PromotedAt     *int64       `json:"promotedAt"     bun:"promoted_at,type:BIGINT,nullzero"`
	DemotedAt      *int64       `json:"demotedAt"      bun:"demoted_at,type:BIGINT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
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
	case TrustOutcomeApproved, TrustOutcomeModified, TrustOutcomeRejected, TrustOutcomeExecutionFailed:
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
func OutcomeOfDecision(decision DecisionType, modifications map[string]any) TrustOutcome {
	switch decision {
	case DecisionAccepted:
		if len(modifications) > 0 {
			return TrustOutcomeModified
		}

		return TrustOutcomeApproved
	case DecisionModified:
		return TrustOutcomeModified
	case DecisionRejected:
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
func (t *ToolTrust) HoldsEarnedTier(current AutonomyTier) bool {
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
				domainvalidation.ValidEnum[AutonomyTier]("Earned tier is invalid"),
			),
		),
	))
}

func (t *ToolTrust) GetID() pulid.ID { return t.ID }

func (t *ToolTrust) GetCreatedAt() int64 { return t.CreatedAt }

func (t *ToolTrust) GetOrganizationID() pulid.ID { return t.OrganizationID }

func (t *ToolTrust) GetBusinessUnitID() pulid.ID { return t.BusinessUnitID }

func (t *ToolTrust) GetTableName() string { return "agent_tool_trust" }

func (t *ToolTrust) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("att_")
		}
		t.CreatedAt = now
		t.UpdatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}

	return nil
}
