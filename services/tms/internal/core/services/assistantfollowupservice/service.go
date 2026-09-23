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
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/assistantjobs"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// inProcessTimeout bounds a follow-up answered in this process. It is a
// sentence or two about something that already happened; a turn still going
// after this long is stuck, and holding the conversation's one live slot for
// longer would refuse the person's next question.
const inProcessTimeout = 5 * time.Minute

type Params struct {
	fx.In

	Logger        *zap.Logger
	Config        *config.Config
	Runs          repositories.AgentRunRepository
	Conversations repositories.ConversationRepository
	Assistant     serviceports.AssistantService
	Turns         *assistantturnservice.Service
	Workflows     serviceports.WorkflowStarter `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	ai            *config.AIConfig
	runs          repositories.AgentRunRepository
	conversations repositories.ConversationRepository
	assistant     serviceports.AssistantService
	turns         turnStarter
	workflows     serviceports.WorkflowStarter
	// inProcess runs a follow-up that no worker will pick up. It is a field
	// so a test can run it where it can wait for it.
	inProcess func(run func())
}

// turnStarter is the part of the turn service a follow-up needs.
type turnStarter interface {
	Start(ctx context.Context, req assistantturnservice.StartRequest) (*conversation.AssistantTurn, error)
	StartDurable(
		ctx context.Context,
		req assistantturnservice.StartRequest,
		start func(turn *conversation.AssistantTurn) (string, error),
	) (*conversation.AssistantTurn, error)
	Observe(
		ctx context.Context,
		turn *conversation.AssistantTurn,
		emit serviceports.AssistantStreamEmitter,
	) (serviceports.AssistantStreamEmitter, func(serviceports.StreamEvent))
	Complete(
		ctx context.Context,
		turn *conversation.AssistantTurn,
		status conversation.AssistantTurnStatus,
		cause error,
	)
}

func New(p Params) serviceports.DecisionFollowUps {
	var ai *config.AIConfig
	if p.Config != nil {
		ai = p.Config.GetAIConfig()
	}

	return &Service{
		l:             p.Logger.Named("service.assistantfollowup"),
		ai:            ai,
		runs:          p.Runs,
		conversations: p.Conversations,
		assistant:     p.Assistant,
		turns:         p.Turns,
		workflows:     p.Workflows,
		inProcess:     func(run func()) { go run() },
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
	message := &serviceports.SendMessageRequest{
		ThreadID:           thread.ID,
		TenantInfo:         req.TenantInfo,
		FollowUpProposalID: req.ProposalID,
		FollowUpPlanID:     req.PlanID,
	}
	start := assistantturnservice.StartRequest{
		ThreadID:   thread.ID,
		UserID:     thread.UserID,
		TenantInfo: req.TenantInfo,
		Origin:     conversation.AssistantTurnOriginDecisionFollowUp,
	}

	if s.durable() {
		s.startDurable(ctx, start, actor, message)
		return
	}
	s.startInProcess(ctx, start, &actor, message)
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

func (s *Service) durable() bool {
	return s.ai.DurableTurnsEnabled() && s.workflows != nil && s.workflows.Enabled()
}

func (s *Service) startDurable(
	ctx context.Context,
	start assistantturnservice.StartRequest,
	actor serviceports.RequestActor,
	message *serviceports.SendMessageRequest,
) {
	_, err := s.turns.StartDurable(ctx, start, func(turn *conversation.AssistantTurn) (string, error) {
		return assistantjobs.StartTurnWorkflow(ctx, s.workflows, turn, assistantjobs.TurnStart{
			Actor: actor,
			Request: assistantjobs.AssistantTurnRequest{
				FollowUpProposalID: message.FollowUpProposalID,
				FollowUpPlanID:     message.FollowUpPlanID,
			},
		})
	})
	if err != nil {
		s.logStartFailure(start, err)
	}
}

// startInProcess records the turn now, so a reader asking for the
// conversation's live reply finds it the moment the decision returns, and
// answers it off the request.
func (s *Service) startInProcess(
	ctx context.Context,
	start assistantturnservice.StartRequest,
	actor *serviceports.RequestActor,
	message *serviceports.SendMessageRequest,
) {
	turn, err := s.turns.Start(ctx, start)
	if err != nil {
		s.logStartFailure(start, err)
		return
	}

	s.inProcess(func() {
		runCtx, cancel := context.WithTimeout(ctx, inProcessTimeout)
		defer cancel()

		observed, closeStream := s.turns.Observe(runCtx, turn, nil)
		result, runErr := s.assistant.SendMessageStream(runCtx, message, actor, observed)
		closeStream(assistantturnservice.Ending(result, runErr))

		refused := runErr == nil && result != nil && result.Refused
		// Completing rides the parent context: a turn that timed out still
		// has to be closed, or it holds the conversation's live slot.
		s.turns.Complete(ctx, turn, assistantturnservice.StatusFor(refused, runErr), runErr)
		if runErr != nil && !errors.Is(runErr, context.Canceled) {
			s.l.Warn("decision follow-up did not finish",
				zap.String("thread", turn.ThreadID.String()),
				zap.Error(runErr),
			)
		}
	})
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
