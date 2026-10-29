package agentdecisionports

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentapprovalports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type DecideAgentProposalRequest struct {
	ProposalID    pulid.ID
	Decision      agentdecision.DecisionType
	Modifications map[string]any
	ReasonCode    string
	Note          string
	TenantInfo    pagination.TenantInfo
	// WithinPlan says the decision is one step of a plan being decided as a
	// whole. The run's workflow is signalled once by the plan, not once per
	// step, and a step's execution failure is reported to the caller so the
	// plan can stop rather than logged and swallowed.
	WithinPlan bool
	// PreviewDigest is the digest of the preview the decider was shown. An
	// approval naming a digest that no longer matches is refused, and nothing
	// is recorded.
	PreviewDigest string
	// StepPreview, StepPreviewReviewed and ExpectedTargetVersion are set only
	// by the plan service, never from a request: the step's preview projected
	// with the plan, whether the plan's digest was reviewed, and the version
	// the step's target is expected at once the steps before it have run.
	StepPreview           *agentpreview.ProposalPreview
	StepPreviewReviewed   bool
	ExpectedTargetVersion *int64
}

// DecisionOutcome is a recorded decision plus what happened when it ran.
// The decision is the durable fact and is recorded whatever the tool did;
// ExecutionError is set when an approval's write did not go through.
type DecisionOutcome struct {
	Decision       *agentdecision.AgentDecision
	ExecutionError error
	// ExecutedTargetVersion is the version the approved write left its
	// target at, when the target is pinned and could be read.
	ExecutedTargetVersion *int64
}

// Result is the outcome as its decider is told it: the decision, whether
// the approved write ran, and why it did not. An approval waiting out its
// undo window has not run yet; one refused because its record moved on ran
// nothing.
func (o *DecisionOutcome) Result(proposalID pulid.ID) AgentProposalDecisionResult {
	result := AgentProposalDecisionResult{ProposalID: proposalID, Decision: o.Decision}
	if o.ExecutionError != nil {
		result.Error = o.ExecutionError.Error()
		result.Refused = errors.Is(o.ExecutionError, agentplan.ErrTargetChanged)

		return result
	}

	result.Executed = o.Decision != nil && o.Decision.CommitsAt == nil &&
		(o.Decision.Decision == agentdecision.DecisionAccepted ||
			o.Decision.Decision == agentdecision.DecisionModified)

	return result
}

type AgentDecisionService interface {
	// DecideWithOutcome records a decision, runs what an approval approved
	// and reports whether it went through. A write that failed, or an
	// approval refused because its record moved on, is the outcome, not an
	// error: the decision is recorded either way.
	DecideWithOutcome(
		ctx context.Context,
		req *DecideAgentProposalRequest,
		actor *permissionports.RequestActor,
	) (*DecisionOutcome, error)
	// DecideOwn is DecideWithOutcome for a proposal raised in one of the
	// actor's own conversations; any other is not found. Its approval waits
	// out the undo window, unless it is refused at once.
	DecideOwn(
		ctx context.Context,
		req *DecideAgentProposalRequest,
		actor *permissionports.RequestActor,
	) (*DecisionOutcome, error)
	// AssertOwnProposal refuses a proposal not raised in one of the actor's
	// own conversations, as not found.
	AssertOwnProposal(
		ctx context.Context,
		proposalID pulid.ID,
		tenant pagination.TenantInfo,
		actor *permissionports.RequestActor,
	) error
	// Defers says whether an approval of this kind waits out the undo window.
	Defers(decision agentdecision.DecisionType) bool
	// ApproveDeferred records approvals that commit together when one undo
	// window closes. Each is checked on its own and reports its own outcome.
	ApproveDeferred(
		ctx context.Context,
		reqs []*DecideAgentProposalRequest,
		actor *permissionports.RequestActor,
	) ([]DeferredApproval, error)
	agentapprovalports.ApprovalCommitter
}

// DeferredApproval is one proposal of an approval made in the undo window:
// the decision recorded for it, or why it was refused.
type DeferredApproval struct {
	ProposalID     pulid.ID
	Decision       *agentdecision.AgentDecision
	Err            error
	ExecutionError error
}

// AgentProposalDecisionResult is what became of one decided proposal. Error
// is written for the person; Executed says the approved write went through;
// Refused says the approval was recorded but refused because the record it
// changes moved on since it was proposed, so nothing was changed.
type AgentProposalDecisionResult struct {
	ProposalID pulid.ID                     `json:"proposalId"`
	Decision   *agentdecision.AgentDecision `json:"decision"`
	Error      string                       `json:"error,omitempty"`
	Executed   bool                         `json:"executed"`
	Refused    bool                         `json:"refused"`
}
