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
	Turns         *assistantturnservice.Service
	Workflows     serviceports.WorkflowStarter
	Proposals     repositories.AgentProposalRepository
	Plans         repositories.AgentPlanRepository
	Decisions     repositories.AgentDecisionRepository
}

type Service struct {
	l             *zap.Logger
	runs          repositories.AgentRunRepository
	conversations repositories.ConversationRepository
	turns         turnStarter
	workflows     serviceports.WorkflowStarter
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

func New(p Params) serviceports.DecisionFollowUps {
	return newService(p)
}

// NewResumer is the same service, as what a turn that ended asks to start the
// follow-ups it kept out.
func NewResumer(p Params) serviceports.DecisionFollowUpResumer {
	return newService(p)
}

func newService(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.assistantfollowup"),
		runs:          p.Runs,
		conversations: p.Conversations,
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
