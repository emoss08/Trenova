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
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
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
}

type Service struct {
	l            *zap.Logger
	tools        services.AgentToolRegistry
	proposalRepo proposalOutcomeRecorder
	permissions  permissionChecker
	versions     services.RecordVersionReader
	audit        actionLogger
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.proposal-executor"),
		tools:        p.Tools,
		proposalRepo: p.ProposalRepo,
		permissions:  p.Permissions,
		versions:     p.Versions,
		audit:        p.AuditService,
	}
}

// ErrToolMissing reports a proposal naming a tool the registry no longer has.
var ErrToolMissing = errors.New("proposal names a tool this system does not provide")

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
	tool, ok := s.tools.Get(proposal.ToolName)
	if !ok {
		err := fmt.Errorf("%w: %s", ErrToolMissing, proposal.ToolName)
		s.recordFailure(ctx, proposal, err)

		return err
	}

	if err := s.assertActorMayRun(ctx, tool, actor); err != nil {
		s.recordFailure(ctx, proposal, err)

		return err
	}

	if err := s.assertTargetUnchanged(ctx, proposal); err != nil {
		s.recordFailure(ctx, proposal, err)

		return err
	}

	params := mergeParams(proposal.ToolParams, modifications)

	// The proposal id is the idempotency key. It is stable across retries of the
	// same approval and distinct between proposals, which is exactly what a tool
	// guarding against double execution needs.
	err := tool.Execute(ctx, services.ToolExecuteParams{
		OrganizationID: proposal.OrganizationID,
		BusinessUnitID: proposal.BusinessUnitID,
		Actor:          actor,
		IdempotencyKey: proposal.ID.String(),
		RunID:          proposal.RunID,
		Params:         params,
	})
	if err != nil {
		s.l.Error("approved proposal failed to execute",
			zap.String("proposal", proposal.ID.String()),
			zap.String("tool", proposal.ToolName),
			zap.Error(err),
		)
		s.recordFailure(ctx, proposal, err)

		return err
	}

	s.recordSuccess(ctx, proposal, actor, params)

	return nil
}

// ErrTargetChanged reports a record that moved on since the change to it was
// proposed. It is a business outcome, not a transport failure: the approval
// was for the record as described, and that record no longer exists in that
// form.
var ErrTargetChanged = errors.New("the record changed since this was proposed")

// assertTargetUnchanged compares the pinned version with the record's current
// one. A proposal without a pin — an older one, or a tool with no single
// target — passes; only a pin that no longer matches refuses.
func (s *Service) assertTargetUnchanged(ctx context.Context, proposal *agent.AgentProposal) error {
	if proposal.TargetID.IsNil() || s.versions == nil {
		return nil
	}

	current, err := s.versions.Version(ctx, pagination.TenantInfo{
		OrgID: proposal.OrganizationID,
		BuID:  proposal.BusinessUnitID,
	}, services.ToolTarget{
		Resource: permission.Resource(proposal.TargetResource),
		ID:       proposal.TargetID,
	})
	if err != nil {
		return fmt.Errorf("%w: the %s could not be read (%w)", ErrTargetChanged, proposal.TargetResource, err)
	}

	if current != proposal.TargetVersion {
		return fmt.Errorf(
			"%w: the %s is at version %d and was at %d when this was proposed. "+
				"Review the current record and ask again",
			ErrTargetChanged, proposal.TargetResource, current, proposal.TargetVersion,
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
	result, err := s.permissions.Check(ctx, &services.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       tool.PermissionResource().String(),
		Operation:      tool.PermissionOperation(),
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
				tool.PermissionOperation(),
				tool.PermissionResource(),
			),
		)
	}

	return nil
}

// mergeParams overlays an approver's modifications onto the proposed parameters.
// A nil or empty modification map leaves the proposal untouched.
func mergeParams(proposed, modifications map[string]any) map[string]any {
	merged := make(map[string]any, len(proposed)+len(modifications))
	for key, value := range proposed {
		merged[key] = value
	}
	for key, value := range modifications {
		merged[key] = value
	}

	return merged
}

func (s *Service) recordSuccess(
	ctx context.Context,
	proposal *agent.AgentProposal,
	actor *services.RequestActor,
	params map[string]any,
) {
	now := timeutils.NowUnix()
	if _, err := s.proposalRepo.RecordExecution(
		ctx,
		repositories.RecordAgentProposalExecutionRequest{
			ID:         proposal.ID,
			Status:     agent.ProposalStatusExecuted,
			ExecutedAt: &now,
			TenantInfo: tenantOf(proposal),
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

	auditActor := actor.AuditActor()
	if err := s.audit.LogAction(&services.LogActionParams{
		Resource:   permission.ResourceAgentProposal,
		ResourceID: proposal.ID.String(),
		// Recorded against approval rather than a dedicated execute operation:
		// approving is the act a person took, and executing is its consequence.
		Operation:      permission.OpApprove,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(map[string]any{"tool": proposal.ToolName, "params": params}),
		OrganizationID: proposal.OrganizationID,
		BusinessUnitID: proposal.BusinessUnitID,
		Critical:       true,
	}, auditservice.WithComment("Approved agent proposal executed")); err != nil {
		s.l.Error("failed to log proposal execution audit", zap.Error(err))
	}
}

func (s *Service) recordFailure(
	ctx context.Context,
	proposal *agent.AgentProposal,
	cause error,
) {
	if _, err := s.proposalRepo.RecordExecution(
		ctx,
		repositories.RecordAgentProposalExecutionRequest{
			ID:             proposal.ID,
			Status:         agent.ProposalStatusExecutionFailed,
			ExecutionError: cause.Error(),
			TenantInfo:     tenantOf(proposal),
		},
	); err != nil {
		s.l.Error("failed to record proposal execution failure",
			zap.String("proposal", proposal.ID.String()),
			zap.Error(err),
		)
	}
}

func tenantOf(proposal *agent.AgentProposal) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: proposal.OrganizationID,
		BuID:  proposal.BusinessUnitID,
	}
}
