// Package assistantfollowupservice has the agent report what came of a
// decision, in the conversation that raised the proposal.
//
// The report used to be asked for by the card that was clicked, and only by
// that card. A proposal approved from the Desk's decisions, from AI Control or
// as part of a plan changed state without a word in the conversation, and
// even the card asked before the change had run. The decision service now
// tells this one after every decision, once the change has run or failed,
// and the conversation gets its turn whoever decided and wherever.
package assistantfollowupservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantturnservice"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/assistantjobs/assistantcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Runs          agentports.AgentRunRepository
	Conversations agentports.ConversationRepository
	Definitions   agentdefinitionports.AgentDefinitionRepository
	Permissions   permissionports.PermissionEngine
	Turns         *assistantturnservice.Service
	Workflows     kernelports.WorkflowStarter
	Proposals     agentports.AgentProposalRepository
	Plans         agentports.AgentPlanRepository
	Decisions     agentports.AgentDecisionRepository
}

type Service struct {
	l             *zap.Logger
	runs          agentports.AgentRunRepository
	conversations agentports.ConversationRepository
	definitions   agentdefinitionports.AgentDefinitionRepository
	permissions   permissionports.PermissionEngine
	turns         turnStarter
	workflows     kernelports.WorkflowStarter
	decided       decidedReader
}

// turnStarter is the part of the turn service a follow-up needs.
type turnStarter interface {
	StartTurn(
		ctx context.Context,
		req assistantturnservice.StartRequest,
		start func(turn *conversation.AssistantTurn) (string, error),
	) (*conversation.AssistantTurn, error)
}

func New(p Params) agentports.DecisionFollowUps {
	return newService(p)
}

// NewResumer is the same service, as what a turn that ended asks to start the
// follow-ups it kept out.
func NewResumer(p Params) agentports.DecisionFollowUpResumer {
	return newService(p)
}

func newService(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.assistantfollowup"),
		runs:          p.Runs,
		conversations: p.Conversations,
		definitions:   p.Definitions,
		permissions:   p.Permissions,
		turns:         p.Turns,
		workflows:     p.Workflows,
		decided: decidedReader{
			proposals: p.Proposals,
			plans:     p.Plans,
			decisions: p.Decisions,
		},
	}
}

// FollowUp starts the turn in which the agent reports a decision, when the
// decision was on something a conversation raised. Anything else, a
// background run's proposal most often, is not a conversation's to answer.
func (s *Service) FollowUp(ctx context.Context, req agentports.DecisionFollowUpRequest) {
	if req.ProposalID.IsNil() == req.PlanID.IsNil() {
		s.l.Error("a decision follow-up must name one proposal or one plan",
			zap.String("proposal", req.ProposalID.String()),
			zap.String("plan", req.PlanID.String()),
		)
		return
	}

	// The follow-up outlives the request that decided: a person closing the
	// tab after clicking approve still gets the report when they come back.
	ctx = dbscope.WithValidTenant(context.WithoutCancel(ctx), req.TenantInfo.DBTenant())

	thread, err := s.threadFor(ctx, req)
	if err != nil {
		s.l.Warn("decision follow-up skipped: its conversation could not be read",
			zap.String("run", req.RunID.String()),
			zap.Error(err),
		)
		return
	}
	if thread == nil {
		return
	}

	// The turn runs as the person whose conversation it is. The decision may
	// have been made by someone else; the report is addressed to the owner,
	// with the owner's access, as every turn in their conversation is.
	actor := permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    thread.UserID,
		UserID:         thread.UserID,
		OrganizationID: thread.OrganizationID,
		BusinessUnitID: thread.BusinessUnitID,
	}
	if !s.ownerMayUseAgent(ctx, thread, &actor, req.TenantInfo) {
		return
	}
	start := assistantturnservice.StartRequest{
		ThreadID:   thread.ID,
		UserID:     thread.UserID,
		TenantInfo: req.TenantInfo,
		Origin:     conversation.AssistantTurnOriginDecisionFollowUp,
	}

	// A worker answers it like any other turn, so the report survives the
	// request that decided and can be watched and stopped like any reply.
	_, err = s.turns.StartTurn(ctx, start, func(turn *conversation.AssistantTurn) (string, error) {
		run, startErr := assistantcontract.StartTurnWorkflow(ctx, s.workflows, turn,
			assistantcontract.TurnStart{
				Actor: actor,
				Request: assistantcontract.AssistantTurnRequest{
					FollowUpProposalID: req.ProposalID,
					FollowUpPlanID:     req.PlanID,
				},
			})
		if startErr != nil {
			return "", startErr
		}

		return run.GetID(), nil
	})
	if err != nil {
		s.logStartFailure(start, err)
	}
}

// threadFor is the conversation that raised the decided proposal, or nil when
// a conversation did not raise it.
func (s *Service) threadFor(
	ctx context.Context,
	req agentports.DecisionFollowUpRequest,
) (*conversation.Thread, error) {
	tenant := req.TenantInfo
	run, err := s.runs.GetByID(ctx, agentports.GetAgentRunByIDRequest{
		ID:         req.RunID,
		TenantInfo: &tenant,
	})
	if err != nil {
		return nil, err
	}
	if run.SubjectType != agent.SubjectAssistantThread || run.SubjectID.IsNil() {
		return nil, nil
	}

	return s.conversations.GetThreadOwned(ctx, agentports.GetThreadOwnedRequest{
		ID:         run.SubjectID,
		TenantInfo: tenant,
	})
}

// ownerMayUseAgent reports whether the conversation's owner may still use its
// agent. One who lost access keeps the conversation to read, and the decision
// stands, but the agent is not asked to report it: the turn would be refused.
func (s *Service) ownerMayUseAgent(
	ctx context.Context,
	thread *conversation.Thread,
	owner *permissionports.RequestActor,
	tenant pagination.TenantInfo,
) bool {
	if s.definitions == nil || s.permissions == nil {
		s.l.Warn("decision follow-up skipped: agent access cannot be checked",
			zap.String("thread", thread.ID.String()),
		)
		return false
	}

	definition, err := s.definitions.GetByID(ctx, agentdefinitionports.GetAgentDefinitionByIDRequest{
		ID:         thread.AgentDefinitionID,
		TenantInfo: tenant,
	})
	if err != nil {
		s.l.Info("decision follow-up skipped: the conversation's agent could not be read",
			zap.String("thread", thread.ID.String()),
			zap.Error(err),
		)
		return false
	}

	allowed, err := s.permissions.MayUseAgent(ctx, owner, definition)
	if err != nil {
		s.l.Warn("decision follow-up skipped: agent access could not be checked",
			zap.String("thread", thread.ID.String()),
			zap.Error(err),
		)
		return false
	}
	if !allowed {
		s.l.Info("decision follow-up skipped: the owner may no longer use the agent",
			zap.String("thread", thread.ID.String()),
			zap.String("agent", definition.ID.String()),
		)
	}

	return allowed
}

// logStartFailure records a follow-up that never began. The conversation
// being busy is ordinary — the person asked something while the change ran,
// or approved a second card while the first was being reported. The turn in
// the way read the proposal before it was decided, so it cannot report it;
// when it ends it resumes this follow-up (ResumeFollowUps).
func (s *Service) logStartFailure(start assistantturnservice.StartRequest, err error) {
	s.l.Info("decision follow-up not started",
		zap.String("thread", start.ThreadID.String()),
		zap.Error(err),
	)
}
