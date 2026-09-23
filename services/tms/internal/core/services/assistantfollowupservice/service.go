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

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/assistantjobs"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Runs          repositories.AgentRunRepository
	Conversations repositories.ConversationRepository
	Definitions   repositories.AgentDefinitionRepository
	Permissions   serviceports.PermissionEngine
	Turns         *assistantturnservice.Service
	Workflows     serviceports.WorkflowStarter
}

type Service struct {
	l             *zap.Logger
	runs          repositories.AgentRunRepository
	conversations repositories.ConversationRepository
	definitions   repositories.AgentDefinitionRepository
	permissions   serviceports.PermissionEngine
	turns         turnStarter
	workflows     serviceports.WorkflowStarter
}

// turnStarter is the part of the turn service a follow-up needs.
type turnStarter interface {
	StartTurn(
		ctx context.Context,
		req assistantturnservice.StartRequest,
		start func(turn *conversation.AssistantTurn) (string, error),
	) (*conversation.AssistantTurn, error)
}

func New(p Params) serviceports.DecisionFollowUps {
	return &Service{
		l:             p.Logger.Named("service.assistantfollowup"),
		runs:          p.Runs,
		conversations: p.Conversations,
		definitions:   p.Definitions,
		permissions:   p.Permissions,
		turns:         p.Turns,
		workflows:     p.Workflows,
	}
}

// FollowUp starts the turn in which the agent reports a decision, when the
// decision was on something a conversation raised. Anything else, a
// background run's proposal most often, is not a conversation's to answer.
func (s *Service) FollowUp(ctx context.Context, req serviceports.DecisionFollowUpRequest) {
	if req.ProposalID.IsNil() == req.PlanID.IsNil() {
		s.l.Error("a decision follow-up must name one proposal or one plan",
			zap.String("proposal", req.ProposalID.String()),
			zap.String("plan", req.PlanID.String()),
		)
		return
	}

	// The follow-up outlives the request that decided: a person closing the
	// tab after clicking approve still gets the report when they come back.
	ctx = context.WithoutCancel(ctx)

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
	actor := serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
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
		run, startErr := assistantjobs.StartTurnWorkflow(ctx, s.workflows, turn,
			assistantjobs.TurnStart{
				Actor: actor,
				Request: assistantjobs.AssistantTurnRequest{
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
	req serviceports.DecisionFollowUpRequest,
) (*conversation.Thread, error) {
	tenant := req.TenantInfo
	run, err := s.runs.GetByID(ctx, repositories.GetAgentRunByIDRequest{
		ID:         req.RunID,
		TenantInfo: &tenant,
	})
	if err != nil {
		return nil, err
	}
	if run.SubjectType != agent.SubjectAssistantThread || run.SubjectID.IsNil() {
		return nil, nil
	}

	return s.conversations.GetThreadOwned(ctx, repositories.GetThreadOwnedRequest{
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
	owner *serviceports.RequestActor,
	tenant pagination.TenantInfo,
) bool {
	if s.definitions == nil || s.permissions == nil {
		s.l.Warn("decision follow-up skipped: agent access cannot be checked",
			zap.String("thread", thread.ID.String()),
		)
		return false
	}

	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
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
// being busy is ordinary — the person asked something while the change ran —
// and the outcome reaches the agent on that turn anyway, since every turn is
// told what became of the conversation's proposals.
func (s *Service) logStartFailure(start assistantturnservice.StartRequest, err error) {
	s.l.Info("decision follow-up not started",
		zap.String("thread", start.ThreadID.String()),
		zap.Error(err),
	)
}
