// Package proposalexecutor runs the tool behind an approved agent proposal.
//
// Until this existed, approving a proposal recorded a decision and changed a
// status and nothing else happened: AgentTool.Execute was never called anywhere,
// and the Temporal workflow discarded the decision signal it received. An
// approval that does not act is worse than no approval at all, because the
// audit trail says a person authorized something that never occurred.
package proposalexecutor

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/toolsimulation"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// permissionChecker and actionLogger are the slices of PermissionEngine and
// AuditService this service actually uses. Depending on the narrow shape keeps
// the executor honest about its reach: it answers one question about the actor
// and writes one audit line, and nothing here can grow into a manifest read or
// an audit query without the dependency changing first.
type permissionChecker interface {
	Check(
		ctx context.Context,
		req *services.PermissionCheckRequest,
	) (*services.PermissionCheckResult, error)
}

type actionLogger interface {
	LogAction(params *services.LogActionParams, opts ...services.LogOption) error
}

// proposalOutcomeRecorder is the write half of the proposal repository. The
// executor never reads proposals; it is handed one and records what happened.
type proposalOutcomeRecorder interface {
	RecordExecution(
		ctx context.Context,
		req repositories.RecordAgentProposalExecutionRequest,
	) (*agent.AgentProposal, error)
	RecordSimulation(
		ctx context.Context,
		req repositories.RecordAgentProposalSimulationRequest,
	) (*agent.AgentProposal, error)
}

// definitionResolver finds the agent behind a proposal, through its run,
// so the executor can read the agent's simulation switch and its caps.
type definitionResolver interface {
	ForRun(
		ctx context.Context,
		tenant pagination.TenantInfo,
		runID pulid.ID,
	) (*agentdefinition.Definition, error)
}

type Params struct {
	fx.In

	Logger       *zap.Logger
	Tools        services.AgentToolRegistry
	ProposalRepo repositories.AgentProposalRepository
	Permissions  services.PermissionEngine
	AuditService services.AuditService
	// Versions is optional; without it a pinned proposal executes unchecked.
	Versions services.RecordVersionReader `optional:"true"`
	// Budgets is optional; without it a tool's daily cap is not enforced on
	// approval.
	Budgets     services.AgentBudgetService `optional:"true"`
	Runs        repositories.AgentRunRepository
	Definitions repositories.AgentDefinitionRepository
	DB          ports.DBConnection
}

type Service struct {
	l            *zap.Logger
	tools        services.AgentToolRegistry
	proposalRepo proposalOutcomeRecorder
	permissions  permissionChecker
	versions     services.RecordVersionReader
	audit        actionLogger
	budgets      services.AgentBudgetService
	definitions  definitionResolver
	db           ports.DBConnection
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.proposal-executor"),
		tools:        p.Tools,
		proposalRepo: p.ProposalRepo,
		permissions:  p.Permissions,
		versions:     p.Versions,
		audit:        p.AuditService,
		budgets:      p.Budgets,
		definitions:  runDefinitions{runs: p.Runs, definitions: p.Definitions},
		db:           p.DB,
	}
}

// runDefinitions reads a run and then its agent. A run without an agent, or
// an agent since deleted, resolves to nil: the proposal still executes, with
// no switch and no caps to read.
type runDefinitions struct {
	runs        repositories.AgentRunRepository
	definitions repositories.AgentDefinitionRepository
}

func (r runDefinitions) ForRun(
	ctx context.Context,
	tenant pagination.TenantInfo,
	runID pulid.ID,
) (*agentdefinition.Definition, error) {
	if r.runs == nil || r.definitions == nil || runID.IsNil() {
		return nil, nil
	}

	run, err := r.runs.GetByID(
		ctx,
		repositories.GetAgentRunByIDRequest{ID: runID, TenantInfo: &tenant},
	)
	if err != nil {
		return nil, err
	}
	if run.AgentDefinitionID.IsNil() {
		return nil, nil
	}

	definition, err := r.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         run.AgentDefinitionID,
		TenantInfo: tenant,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil
		}

		return nil, err
	}

	return definition, nil
}

// ErrToolMissing reports a proposal naming a tool the registry no longer has.
var ErrToolMissing = errors.New("proposal names a tool this system does not provide")

// ErrTenantMismatch is a proposal decided by someone outside its tenant.
var ErrTenantMismatch = errors.New("proposal does not belong to the approver's organization")

// ErrTaintedNeedsPerson is a write taint held, proposed by a run that had
// read content written outside the organization, being run by anything but a
// person's decision.
var ErrTaintedNeedsPerson = errors.New(
	"this change was held because the agent had read outside content before proposing " +
		"it, so only a person can approve it",
)

// Approval is an approved proposal to run. ExpectedTargetVersion is the
// version its target is expected at when an earlier step of the same plan
// changed that record first; without it the target must still be at the
// version pinned when the change was proposed.
type Approval struct {
	Proposal              *agent.AgentProposal
	Modifications         map[string]any
	Actor                 *services.RequestActor
	ExpectedTargetVersion *int64
}

// Outcome is what running an approval left behind: the version the write
// left its target at, when the target is pinned and could be read.
type Outcome struct {
	TargetVersion *int64
}

// Execute runs an approved proposal's tool.
//
// The approver is the actor, not the agent. Everything the tool does is
// therefore attributed to the person who approved it and checked against their
// permissions, which is the only honest reading of what happened: a human
// authorized this write.
//
// Modifications from the decision are applied over the proposed parameters, so
// approving with changes runs what the approver actually agreed to rather than
// what the model originally asked for.
func (s *Service) Execute(
	ctx context.Context,
	proposal *agent.AgentProposal,
	modifications map[string]any,
	actor *services.RequestActor,
) error {
	_, err := s.Run(ctx, &Approval{
		Proposal:      proposal,
		Modifications: modifications,
		Actor:         actor,
	})

	return err
}

// Run is Execute for a caller that needs what the write left behind, such
// as a plan whose next step changes the same record.
func (s *Service) Run(ctx context.Context, approval *Approval) (*Outcome, error) {
	ctx, span := startExecute(ctx, approval.Proposal, approval.Modifications, approval.Actor)
	defer span.End()

	outcome, err := s.execute(ctx, approval)
	if err != nil {
		aitrace.MarkFailed(span, executeFailure(err))
	}

	return outcome, err
}

func (s *Service) execute(ctx context.Context, approval *Approval) (*Outcome, error) {
	proposal := approval.Proposal
	actor := approval.Actor
	// The tenant the write runs in is the proposal's and the principal is
	// the approver's. The two are asserted to agree here, where they meet,
	// rather than trusted to have been scoped alike by every caller.
	if actor == nil || proposal.OrganizationID != actor.OrganizationID ||
		proposal.BusinessUnitID != actor.BusinessUnitID {
		return nil, ErrTenantMismatch
	}

	tool, params, err := s.admit(ctx, approval)
	if err != nil {
		s.recordFailureBy(ctx, proposal, err, actor)

		return nil, err
	}

	policy := tool.Policy()
	execParams := ExecutionParams(proposal, &policy, params, actor)

	egress := policy.Classified(execParams).Egress
	if err = assertTaintDecidedByPerson(proposal, egress, actor); err != nil {
		s.recordFailureAs(ctx, proposal, err, egress, actor)

		return nil, err
	}
	proposal.EgressClass = recordedEgress(proposal.EgressClass, egress)

	definition, err := s.definitionFor(ctx, proposal)
	if err != nil {
		s.recordFailureBy(ctx, proposal, err, actor)

		return nil, err
	}

	approved := &approvedRun{
		tool:     tool,
		proposal: proposal,
		actor:    actor,
		policy:   &policy,
		params:   &execParams,
	}

	// An agent in simulation gets a preview in place of the write, however
	// the proposal was decided: the approval is real and recorded, the
	// change is not.
	if definition != nil && definition.SimulationMode {
		return &Outcome{}, s.simulate(ctx, approved)
	}

	if err = s.assertWithinBudget(ctx, definition, tool.Name()); err != nil {
		s.recordFailureBy(ctx, proposal, err, actor)

		return nil, err
	}

	after, err := s.run(ctx, approved)
	if err != nil {
		return nil, err
	}

	return &Outcome{TargetVersion: after}, nil
}

func (s *Service) admit(
	ctx context.Context,
	approval *Approval,
) (services.AgentTool, map[string]any, error) {
	proposal := approval.Proposal
	modifications := approval.Modifications
	tool, ok := s.tools.Get(proposal.ToolName)
	if !ok {
		return nil, nil, fmt.Errorf("%w: %s", ErrToolMissing, proposal.ToolName)
	}

	if err := s.assertActorMayRun(ctx, tool, approval.Actor); err != nil {
		return nil, nil, err
	}

	if err := s.assertTargetUnchanged(ctx, proposal, approval.ExpectedTargetVersion); err != nil {
		return nil, nil, err
	}

	if err := refuseOwnerChange(modifications); err != nil {
		return nil, nil, err
	}

	params := MergeParams(proposal.ToolParams, modifications)
	if len(modifications) > 0 {
		// What the approver changed is checked once more here, where it
		// runs: the decision that carried it was checked when it was made,
		// and the tool may have changed. It may not point the write at
		// another record, and it must fit the tool's own schema.
		if err := refuseRetarget(tool, proposal.ToolParams, params); err != nil {
			return nil, nil, err
		}
		if err := toolschema.CheckSubsets(
			tool.ParamSchema(),
			proposal.ToolParams,
			modifications,
		); err != nil {
			return nil, nil, err
		}
		if err := validateParams(tool, params); err != nil {
			return nil, nil, err
		}
	}

	return tool, params, nil
}

// ExecutionParams is what the tool behind a proposal is handed when it runs
// for an approver, and when it previews for one.
func ExecutionParams(
	proposal *agent.AgentProposal,
	policy *services.ToolPolicy,
	params map[string]any,
	actor *services.RequestActor,
) services.ToolExecuteParams {
	// The proposal id is the idempotency key. It is stable across retries of the
	// same approval and distinct between proposals, which is exactly what a tool
	// guarding against double execution needs.
	execParams := services.ToolExecuteParams{
		OrganizationID: proposal.OrganizationID,
		BusinessUnitID: proposal.BusinessUnitID,
		Actor:          actor,
		IdempotencyKey: proposal.ID.String(),
		RunID:          proposal.RunID,
		Params:         params,
		ProposalID:     proposal.ID,
	}
	if policy.CarriesTaint {
		execParams.Taint = proposalTaint(proposal)
	}

	return execParams
}

type approvedRun struct {
	tool     services.AgentTool
	proposal *agent.AgentProposal
	actor    *services.RequestActor
	policy   *services.ToolPolicy
	params   *services.ToolExecuteParams
}

func (s *Service) run(ctx context.Context, r *approvedRun) (*int64, error) {
	proposal := r.proposal
	toolCtx, toolSpan := startTool(ctx, proposal, r.policy)
	defer toolSpan.End()
	writeCtx, write := startWrite(toolCtx, proposal, false)
	result, err := services.ExecuteTool(writeCtx, r.tool, *r.params)
	if err != nil {
		aitrace.MarkFailed(write, aitrace.OutcomeFailed)
		write.End()
		finishTool(toolSpan, aitrace.OutcomeFailed)
		s.l.Error("approved proposal failed to execute",
			zap.String("proposal", proposal.ID.String()),
			zap.String("tool", proposal.ToolName),
			zap.Error(err),
		)
		s.recordFailureBy(ctx, proposal, err, r.actor)

		return nil, err
	}
	after := s.versionAfter(writeCtx, proposal)
	if after != nil {
		write.SetAttributes(aitrace.AIVersionAfter.Int64(*after))
	}
	write.End()
	finishTool(toolSpan, aitrace.OutcomeRan)

	s.recordSuccess(ctx, executionSuccess{
		proposal:      proposal,
		actor:         r.actor,
		params:        r.params.Params,
		result:        result,
		targetVersion: after,
	})

	return after, nil
}

func (s *Service) versionAfter(ctx context.Context, proposal *agent.AgentProposal) *int64 {
	if proposal.TargetID.IsNil() || s.versions == nil {
		return nil
	}

	version, err := s.versions.Version(ctx, tenantOf(proposal), services.ToolTarget{
		Resource: permission.Resource(proposal.TargetResource),
		ID:       proposal.TargetID,
	})
	if err != nil {
		s.l.Warn("could not read the version an approved write left its record at",
			zap.String("proposal", proposal.ID.String()),
			zap.Error(err),
		)

		return nil
	}

	return &version
}

// CheckModifications validates what an approver changed before the decision
// that carries it is recorded: the merged parameters must fit the tool's
// schema, and a tool that checks its own arguments gets to check them. It
// returns the parameters as they would run.
//
// Checking here rather than at execution alone is the difference between a
// form that says which value is wrong and a decision that is recorded,
// audited, and then fails. A change that would not run is refused as a
// validation error, before anything is written.
func (s *Service) CheckModifications(
	ctx context.Context,
	proposal *agent.AgentProposal,
	modifications map[string]any,
	actor *services.RequestActor,
) (map[string]any, error) {
	if actor == nil || proposal.OrganizationID != actor.OrganizationID ||
		proposal.BusinessUnitID != actor.BusinessUnitID {
		return nil, ErrTenantMismatch
	}

	tool, ok := s.tools.Get(proposal.ToolName)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrToolMissing, proposal.ToolName)
	}

	if err := refuseOwnerChange(modifications); err != nil {
		return nil, err
	}

	params := MergeParams(proposal.ToolParams, modifications)
	if err := refuseRetarget(tool, proposal.ToolParams, params); err != nil {
		return nil, err
	}
	if err := toolschema.CheckSubsets(
		tool.ParamSchema(),
		proposal.ToolParams,
		modifications,
	); err != nil {
		return nil, err
	}
	if err := validateParams(tool, params); err != nil {
		return nil, err
	}

	if validator, checks := tool.(services.ToolValidator); checks {
		if err := validator.Validate(ctx, services.ToolExecuteParams{
			OrganizationID: proposal.OrganizationID,
			BusinessUnitID: proposal.BusinessUnitID,
			Actor:          actor,
			IdempotencyKey: proposal.ID.String(),
			RunID:          proposal.RunID,
			Params:         params,
		}); err != nil {
			return nil, err
		}
	}

	return params, nil
}

func (s *Service) definitionFor(
	ctx context.Context,
	proposal *agent.AgentProposal,
) (*agentdefinition.Definition, error) {
	if s.definitions == nil {
		return nil, nil
	}

	return s.definitions.ForRun(ctx, pagination.TenantInfo{
		OrgID: proposal.OrganizationID,
		BuID:  proposal.BusinessUnitID,
	}, proposal.RunID)
}

// ErrBudgetSpent reports a tool past its agent's daily cap. It is a business
// outcome the approver has to hear: the change they cleared did not happen.
var ErrBudgetSpent = errors.New("the agent's budget for this tool is spent")

func (s *Service) assertWithinBudget(
	ctx context.Context,
	definition *agentdefinition.Definition,
	toolName string,
) error {
	if s.budgets == nil || definition == nil {
		return nil
	}

	refusal, err := s.budgets.CheckTool(ctx, services.CheckToolBudgetRequest{
		Definition: definition,
		ToolName:   toolName,
	})
	if err != nil {
		return err
	}
	if refusal.Refused() {
		return fmt.Errorf("%w: %s", ErrBudgetSpent, refusal.Message(definition.Name))
	}

	return nil
}

func (s *Service) simulate(ctx context.Context, r *approvedRun) error {
	proposal := r.proposal
	actor := r.actor
	toolCtx, toolSpan := startTool(ctx, proposal, r.policy)
	writeCtx, write := startWrite(toolCtx, proposal, true)
	preview := toolsimulation.InSnapshot(writeCtx, s.db, r.tool, r.params)
	write.End()
	finishTool(toolSpan, aitrace.OutcomeSimulated)
	toolSpan.End()
	now := timeutils.NowUnix()

	if _, err := s.proposalRepo.RecordSimulation(
		ctx,
		repositories.RecordAgentProposalSimulationRequest{
			ID:               proposal.ID,
			TenantInfo:       tenantOf(proposal),
			SimulatedAt:      now,
			Simulation:       preview,
			ExecutedByUserID: executorOf(actor),
		},
	); err != nil {
		s.l.Error("failed to record proposal simulation",
			zap.String("proposal", proposal.ID.String()), zap.Error(err))

		return err
	}

	auditActor := actor.AuditActor()
	if err := s.audit.LogAction(&services.LogActionParams{
		Resource:      permission.ResourceAgentProposal,
		ResourceID:    proposal.ID.String(),
		Operation:     permission.OpUpdate,
		UserID:        auditActor.UserID,
		PrincipalType: auditActor.PrincipalType,
		PrincipalID:   auditActor.PrincipalID,
		APIKeyID:      auditActor.APIKeyID,
		CurrentState: jsonutils.MustToJSON(map[string]any{
			"tool":       r.tool.Name(),
			"params":     r.params.Params,
			"simulation": preview,
		}),
		OrganizationID: proposal.OrganizationID,
		BusinessUnitID: proposal.BusinessUnitID,
	}, auditservice.WithComment("Agent proposal simulated: the agent is in simulation, nothing was changed")); err != nil {
		s.l.Error("failed to log proposal simulation audit", zap.Error(err))
	}

	return nil
}

// ErrTargetChanged reports a record that moved on since the change to it was
// proposed. It is a business outcome, not a transport failure: the approval
// was for the record as described, and that record no longer exists in that
// form.
var ErrTargetChanged = errors.New("the record changed since this was proposed")

// assertTargetUnchanged compares the pinned version with the record's current
// one. A proposal without a pin — an older one, or a tool with no single
// target — passes; only a pin that no longer matches refuses. An expected
// version stands in for the pin when an earlier step of the same plan changed
// the record: that step's write is the one change the approver allowed for.
func (s *Service) assertTargetUnchanged(
	ctx context.Context,
	proposal *agent.AgentProposal,
	expected *int64,
) error {
	if proposal.TargetID.IsNil() || s.versions == nil {
		return nil
	}

	pinned := proposal.TargetVersion
	if expected != nil {
		pinned = *expected
	}

	current, err := s.versions.Version(ctx, pagination.TenantInfo{
		OrgID: proposal.OrganizationID,
		BuID:  proposal.BusinessUnitID,
	}, services.ToolTarget{
		Resource: permission.Resource(proposal.TargetResource),
		ID:       proposal.TargetID,
	})
	if err != nil {
		return fmt.Errorf(
			"%w: the %s could not be read (%w)",
			ErrTargetChanged,
			proposal.TargetResource,
			err,
		)
	}

	if current != pinned {
		return fmt.Errorf(
			"%w: the %s is at version %d and was at %d when this was proposed. "+
				"Review the current record and ask again",
			ErrTargetChanged, proposal.TargetResource, current, pinned,
		)
	}

	return nil
}

// assertActorMayRun checks the tool's own permission rather than a fixed one.
//
// The decision service historically checked billing-queue approval for every
// proposal, which was right when the only agent was the billing agent and wrong
// as soon as a second one existed. A proposal to reassign a move should require
// permission over moves.
func (s *Service) assertActorMayRun(
	ctx context.Context,
	tool services.AgentTool,
	actor *services.RequestActor,
) error {
	// A self-scoped tool needs no grant: it runs only for the person it was
	// proposed for, which the tool checks against the owner the runtime
	// recorded, and only a person has records of that kind.
	if services.IsSelfScoped(tool) {
		if actor == nil || actor.PrincipalType != services.PrincipalTypeUser {
			return errortypes.NewValidationError("actor", errortypes.ErrForbidden,
				"Only the person this change is for can approve it")
		}
		return nil
	}

	policy := tool.Policy()
	result, err := s.permissions.Check(ctx, &services.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       policy.Resource.String(),
		Operation:      policy.Operation,
	})
	if err != nil {
		return err
	}

	if !result.Allowed {
		return errortypes.NewValidationError(
			"actor",
			errortypes.ErrForbidden,
			fmt.Sprintf(
				"You do not have permission to %s %s, which this proposal would do",
				policy.Operation,
				policy.Resource,
			),
		)
	}

	return nil
}

// refuseOwnerChange keeps whose records a self-scoped call is about out of
// an approver's reach. The runtime recorded the owner from the turn's actor;
// a change that named someone else would let an approval land on another
// person's records, so it is refused as a field error rather than merged.
func refuseOwnerChange(modifications map[string]any) error {
	if _, retargets := modifications[services.SelfScopeOwnerParam]; !retargets {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	multiErr.Add(
		services.SelfScopeOwnerParam,
		errortypes.ErrForbidden,
		"Whose records this change is for is set when it is proposed and cannot be changed",
	)

	return multiErr
}

// refuseRetarget keeps an approver's changes on the record the change was
// proposed for. A change that pointed the write at another record would be a
// different proposal, approved without anyone having proposed it, and would
// escape the staleness check, which compares the record that was pinned.
func refuseRetarget(tool services.AgentTool, proposed, merged map[string]any) error {
	targeted, ok := tool.(services.TargetedTool)
	if !ok {
		return nil
	}

	before, hadTarget := targeted.Target(proposed)
	after, hasTarget := targeted.Target(merged)
	if hadTarget == hasTarget && (!hadTarget || before == after) {
		return nil
	}

	field := "modifications"
	if names := services.TargetParameters(tool, proposed); len(names) > 0 {
		field = names[0]
	}

	multiErr := errortypes.NewMultiError()
	multiErr.Add(
		field,
		errortypes.ErrForbidden,
		"The record this change is for is set when it is proposed and cannot be changed. "+
			"Reject it and ask the agent again for the record you mean",
	)

	return multiErr
}

// validateParams checks the parameters as they would run against the tool's
// schema. The owner the runtime records on a self-scoped call is not one of
// the tool's declared parameters, so it is set aside for the check and the
// schema judges only what the model and the approver supplied.
func validateParams(tool services.AgentTool, params map[string]any) error {
	if _, owned := params[services.SelfScopeOwnerParam]; !owned || !services.IsSelfScoped(tool) {
		return toolschema.Validate(tool.ParamSchema(), params)
	}

	declared := make(map[string]any, len(params)-1)
	for key, value := range params {
		if key != services.SelfScopeOwnerParam {
			declared[key] = value
		}
	}

	return toolschema.Validate(tool.ParamSchema(), declared)
}

// MergeParams overlays an approver's modifications onto the proposed parameters.
// A nil or empty modification map leaves the proposal untouched. The owner of a
// self-scoped call is always the one stored on the proposal: it is taken from
// the proposed parameters after the overlay, never from the modifications.
func MergeParams(proposed, modifications map[string]any) map[string]any {
	merged := make(map[string]any, len(proposed)+len(modifications))
	for key, value := range proposed {
		merged[key] = value
	}
	for key, value := range modifications {
		merged[key] = value
	}

	if owner, owned := proposed[services.SelfScopeOwnerParam]; owned {
		merged[services.SelfScopeOwnerParam] = owner
	} else {
		delete(merged, services.SelfScopeOwnerParam)
	}

	return merged
}

// executionSuccess is an approved proposal that ran: who ran it, with what,
// and what the tool reports it made.
type executionSuccess struct {
	proposal      *agent.AgentProposal
	actor         *services.RequestActor
	params        map[string]any
	result        *agent.ToolExecutionResult
	targetVersion *int64
}

func (s *Service) recordSuccess(ctx context.Context, success executionSuccess) {
	proposal := success.proposal
	now := timeutils.NowUnix()
	if _, err := s.proposalRepo.RecordExecution(
		ctx,
		repositories.RecordAgentProposalExecutionRequest{
			ID:                    proposal.ID,
			Status:                agent.ProposalStatusExecuted,
			ExecutedAt:            &now,
			ExecutionResult:       success.result,
			EgressClass:           proposal.EgressClass,
			TenantInfo:            tenantOf(proposal),
			ExecutedByUserID:      executorOf(success.actor),
			ExecutedTargetVersion: success.targetVersion,
		},
	); err != nil {
		// The tool already ran, so failing to record that is a reporting problem
		// rather than a correctness one for the write itself — but it leaves the
		// proposal looking unexecuted, so it is logged loudly.
		s.l.Error("executed a proposal but could not record the outcome",
			zap.String("proposal", proposal.ID.String()),
			zap.Error(err),
		)
	}

	auditActor := success.actor.AuditActor()
	if err := s.audit.LogAction(&services.LogActionParams{
		Resource:   permission.ResourceAgentProposal,
		ResourceID: proposal.ID.String(),
		// Recorded against approval rather than a dedicated execute operation:
		// approving is the act a person took, and executing is its consequence.
		Operation:     permission.OpApprove,
		UserID:        auditActor.UserID,
		PrincipalType: auditActor.PrincipalType,
		PrincipalID:   auditActor.PrincipalID,
		APIKeyID:      auditActor.APIKeyID,
		CurrentState: jsonutils.MustToJSON(map[string]any{
			"tool":   proposal.ToolName,
			"params": success.params,
			"result": success.result,
		}),
		OrganizationID: proposal.OrganizationID,
		BusinessUnitID: proposal.BusinessUnitID,
		Critical:       true,
	}, auditservice.WithComment("Approved agent proposal executed")); err != nil {
		s.l.Error("failed to log proposal execution audit", zap.Error(err))
	}
}

func (s *Service) recordFailureBy(
	ctx context.Context,
	proposal *agent.AgentProposal,
	cause error,
	actor *services.RequestActor,
) {
	s.recordFailureAs(ctx, proposal, cause, proposal.EgressClass, actor)
}

func (s *Service) recordFailureAs(
	ctx context.Context,
	proposal *agent.AgentProposal,
	cause error,
	egress agent.EgressClass,
	actor *services.RequestActor,
) {
	if _, err := s.proposalRepo.RecordExecution(
		ctx,
		repositories.RecordAgentProposalExecutionRequest{
			ID:               proposal.ID,
			Status:           agent.ProposalStatusExecutionFailed,
			ExecutionError:   cause.Error(),
			EgressClass:      egress,
			TenantInfo:       tenantOf(proposal),
			ExecutedByUserID: executorOf(actor),
		},
	); err != nil {
		s.l.Error("failed to record proposal execution failure",
			zap.String("proposal", proposal.ID.String()),
			zap.Error(err),
		)
	}
}

// assertTaintDecidedByPerson refuses to run a write that leaves the
// organization, from a run that had read outside content, on anything but a
// person's decision: an API key, an agent or the system clearing it would be
// the automatic execution the taint rule exists to stop. The class is checked
// as the write would run and as it was proposed, so an approver's change
// cannot talk the check out of a class the proposal already had.
func assertTaintDecidedByPerson(
	proposal *agent.AgentProposal,
	egress agent.EgressClass,
	actor *services.RequestActor,
) error {
	if !proposal.RequiresPerson(egress) && !proposal.RequiresPerson(proposal.EgressClass) {
		return nil
	}
	if actor != nil && actor.PrincipalType == services.PrincipalTypeUser &&
		actor.UserID.IsNotNil() {
		return nil
	}

	return ErrTaintedNeedsPerson
}

// recordedEgress is the class a proposal keeps once it runs: where it
// reached as it ran, or where it was proposed to reach when the tool no
// longer says.
func recordedEgress(proposed, ran agent.EgressClass) agent.EgressClass {
	if ran.IsValid() {
		return ran
	}

	return proposed
}

// proposalTaint is the taint a write that keeps it carries into what it
// makes. A proposal recorded as tainted whose marks were not kept still
// carries one, naming its run.
func proposalTaint(proposal *agent.AgentProposal) *agent.RunTaint {
	if !proposal.Tainted {
		return nil
	}
	if proposal.Taint.Tainted() {
		return proposal.Taint.Clone()
	}

	return agent.RunRecordTaint(proposal.RunID, timeutils.NowUnix())
}

func tenantOf(proposal *agent.AgentProposal) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: proposal.OrganizationID,
		BuID:  proposal.BusinessUnitID,
	}
}
