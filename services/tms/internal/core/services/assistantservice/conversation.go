package assistantservice

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const (
	// historyLimit bounds how much of a long conversation is replayed. The whole
	// thread would eventually exceed any context window, and the most recent
	// turns are the ones that carry the thread of the question.
	historyLimit = 40
	// defaultPageLimit is how much of a thread the client reads at a time;
	// maxPageLimit is the most it may ask for in one page.
	defaultPageLimit = 50
	maxPageLimit     = 200
	// maxThreadMessages is where a conversation must be continued in a new
	// one. Every message past the model's history window is history the
	// client still has to render.
	maxThreadMessages = 400
	// maxTitleRunes bounds a title derived from the first message.
	maxTitleRunes = 60
)

func (s *Service) StartThread(
	ctx context.Context,
	req *services.StartThreadRequest,
	actor *services.RequestActor,
) (*conversation.Thread, error) {
	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         req.AgentDefinitionID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if !definition.Enabled {
		return nil, errortypes.NewBusinessError(
			"Agent {0} is disabled and cannot be used", definition.Name,
		)
	}

	thread := &conversation.Thread{
		OrganizationID:    req.TenantInfo.OrgID,
		BusinessUnitID:    req.TenantInfo.BuID,
		UserID:            actor.UserID,
		AgentDefinitionID: definition.ID,
		Title:             strings.TrimSpace(req.Title),
		Status:            conversation.ThreadStatusActive,
	}

	multiErr := errortypes.NewMultiError()
	thread.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.conversations.CreateThread(ctx, thread)
}

// assertWithinBudget refuses a turn once the agent's monthly budget is
// spent. The person asking is told which cap, because the alternative is a
// conversation that stops answering without saying why.
func (s *Service) assertWithinBudget(
	ctx context.Context,
	definition *agentdefinition.Definition,
) error {
	if s.budgets == nil {
		return nil
	}

	refusal, err := s.budgets.CheckRun(ctx, definition)
	if err != nil {
		return err
	}
	if refusal.Refused() {
		return errortypes.NewBusinessError(refusal.Message(definition.Name))
	}

	return nil
}

func (s *Service) ListThreads(
	ctx context.Context,
	req repositories.ListThreadsRequest,
) (*pagination.ListResult[*conversation.Thread], error) {
	return s.conversations.ListThreads(ctx, req)
}

func (s *Service) GetThread(
	ctx context.Context,
	req repositories.GetThreadRequest,
) (*conversation.Thread, error) {
	return s.conversations.GetThread(ctx, req)
}

func (s *Service) ListMessages(
	ctx context.Context,
	req services.ListThreadMessagesRequest,
) (*services.ThreadMessagesPage, error) {
	// Reading the thread first is the authorization check: it is scoped by user,
	// so a thread belonging to someone else is not found rather than returned.
	if _, err := s.conversations.GetThread(ctx, req.Thread); err != nil {
		return nil, err
	}

	limit := pageLimit(req.Limit)

	// One more than the page says whether there is a page above it without
	// a second query for the count of what is left.
	messages, err := s.conversations.ListMessages(ctx, repositories.ListMessagesRequest{
		ThreadID:       req.Thread.ID,
		TenantInfo:     req.Thread.TenantInfo,
		Limit:          limit + 1,
		BeforeSequence: req.BeforeSequence,
	})
	if err != nil {
		return nil, err
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[len(messages)-limit:]
	}

	total, err := s.conversations.CountMessages(ctx, repositories.CountMessagesRequest{
		ThreadID:   req.Thread.ID,
		TenantInfo: req.Thread.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	return &services.ThreadMessagesPage{
		Results: messages,
		HasMore: hasMore,
		Total:   total,
		Limit:   maxThreadMessages,
	}, nil
}

func pageLimit(requested int) int {
	switch {
	case requested <= 0:
		return defaultPageLimit
	case requested > maxPageLimit:
		return maxPageLimit
	default:
		return requested
	}
}

// assertRoom refuses a turn on a thread that has reached its length. The
// thread is what the client renders and what every turn replays a slice of;
// one that grows without end slows the page it lives on and buries the
// question under its own history. The limit is generous for a conversation
// and the refusal says what to do instead.
func (s *Service) assertRoom(
	ctx context.Context,
	thread *conversation.Thread,
	tenantInfo pagination.TenantInfo,
) error {
	count, err := s.conversations.CountMessages(ctx, repositories.CountMessagesRequest{
		ThreadID:   thread.ID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if count < maxThreadMessages {
		return nil
	}

	return errortypes.NewBusinessError(
		"This conversation has reached its limit of {0} messages. Start a new conversation to continue",
		maxThreadMessages,
	)
}

func (s *Service) DeleteThread(
	ctx context.Context,
	req repositories.GetThreadRequest,
) error {
	return s.conversations.DeleteThread(ctx, req)
}

// SendMessage runs a turn and saves it.
//
// The turn is persisted whatever the outcome, refusals included. A refused turn
// is the record that someone tried to use the assistant for something it does not
// do, which is exactly the thing worth keeping.
func (s *Service) SendMessage(
	ctx context.Context,
	req *services.SendMessageRequest,
	actor *services.RequestActor,
) (*services.SendMessageResult, error) {
	return s.SendMessageStream(ctx, req, actor, nil)
}

// SendMessageStream is SendMessage with the turn reported to emit as it
// happens. The reply is still saved whole at the end: what the reader watched
// arrive and what the thread shows afterwards are the same messages.
func (s *Service) SendMessageStream(
	ctx context.Context,
	req *services.SendMessageRequest,
	actor *services.RequestActor,
	emit services.AssistantStreamEmitter,
) (*services.SendMessageResult, error) {
	content := strings.TrimSpace(req.Content)
	multiErr := errortypes.NewMultiError()
	if content == "" {
		multiErr.Add("content", errortypes.ErrRequired, "Message cannot be empty")
	}
	page := req.Page.Normalized()
	if page != nil {
		page.Validate("context", multiErr)
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	thread, err := s.conversations.GetThread(ctx, repositories.GetThreadRequest{
		ID:         req.ThreadID,
		UserID:     actor.UserID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         thread.AgentDefinitionID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if !definition.Enabled {
		return nil, errortypes.NewBusinessError(
			"Agent {0} is disabled and cannot be used", definition.Name,
		)
	}

	if err = s.assertWithinBudget(ctx, definition); err != nil {
		return nil, err
	}

	if err = s.assertRoom(ctx, thread, req.TenantInfo); err != nil {
		return nil, err
	}

	history, err := s.conversations.ListMessages(ctx, repositories.ListMessagesRequest{
		ThreadID:   thread.ID,
		TenantInfo: req.TenantInfo,
		Limit:      historyLimit,
	})
	if err != nil {
		return nil, err
	}

	// The reader's choice is resolved against what they may actually pick, then
	// kept on the thread so the picker still shows it after a reload. A choice
	// that no longer resolves is cleared rather than carried, so the stored
	// preference and the model that answers cannot drift apart.
	// A send that says nothing about the model keeps the thread's saved
	// choice; only a choice, including the choice of "automatic", replaces it.
	if req.ProviderChosen {
		if chosen := s.resolvePreference(ctx, req.PreferredProviderID, *actor); chosen != thread.PreferredProviderID {
			thread.PreferredProviderID = chosen
			if _, uErr := s.conversations.UpdateThread(ctx, thread); uErr != nil {
				s.logger.Warn("could not save the chosen model on the thread",
					zap.String("thread", thread.ID.String()),
					zap.Error(uErr),
				)
			}
		}
	}

	turn, runErr := s.RunObserved(ctx, &TurnRequest{
		Definition:          definition,
		Actor:               actor,
		History:             history,
		Input:               content,
		Page:                page,
		PreferredProviderID: thread.PreferredProviderID,
		ThreadID:            thread.ID,
		Proposals:           s.proposalOutcomes(ctx, thread, req.TenantInfo),
	}, emit)

	// The save does not ride the request context. The two ways a turn is most
	// often lost — the person pressing Stop, or closing the tab — both cancel
	// that context, and a save that honoured the cancellation would be the one
	// act the cancellation defeated. Nothing here waits on the reader.
	keep := context.WithoutCancel(ctx)

	if runErr != nil {
		if turn != nil {
			attachPageContext(turn.Messages, page)
			if _, saveErr := s.conversations.AppendTurn(keep, repositories.AppendTurnRequest{
				ThreadID:   thread.ID,
				TenantInfo: req.TenantInfo,
				Messages:   turn.Messages,
			}); saveErr != nil {
				s.logger.Error("could not keep an interrupted turn",
					zap.String("thread", thread.ID.String()),
					zap.Error(saveErr),
				)
			}
		}

		return nil, runErr
	}

	attachPageContext(turn.Messages, page)

	saved, err := s.conversations.AppendTurn(keep, repositories.AppendTurnRequest{
		ThreadID:   thread.ID,
		TenantInfo: req.TenantInfo,
		Messages:   turn.Messages,
	})
	if err != nil {
		return nil, err
	}

	s.titleIfUnnamed(keep, thread, content)

	proposals, err := s.persistProposals(keep, persistProposalsParams{
		Definition: definition,
		Thread:     thread,
		Actor:      actor,
		Saved:      saved,
		Actions:    turn.Actions,
		Model:      turn.Model,
		Input:      content,
	})
	if err != nil {
		s.logProposalPersistFailure(thread, err)
	}

	return &services.SendMessageResult{
		Thread:              thread,
		Messages:            saved,
		Reply:               turn.Reply,
		Refused:             !turn.Decision.Allowed,
		Proposals:           proposals,
		ProposalsUnrecorded: err != nil,
	}, nil
}

// titleIfUnnamed names a thread from its first message so the list is readable
// without asking the model for a summary.
func (s *Service) titleIfUnnamed(
	ctx context.Context,
	thread *conversation.Thread,
	firstMessage string,
) {
	if strings.TrimSpace(thread.Title) != "" {
		return
	}

	thread.Title = truncateRunes(firstMessage, maxTitleRunes)
	if _, err := s.conversations.UpdateThread(ctx, thread); err != nil {
		// A missing title is cosmetic, so it must not fail a turn that otherwise
		// succeeded and is already saved.
		s.logger.Warn("failed to title thread", zap.Error(err))
	}
}

func truncateRunes(s string, limit int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= limit {
		return s
	}

	runes := []rune(s)

	return strings.TrimSpace(string(runes[:limit])) + "…"
}

// attachPageContext records the page on the person's turn only: the assistant
// and tool messages that follow are about the same page, but the fact worth
// keeping is what the person was looking at when they asked.
func attachPageContext(messages []conversation.Message, page *agent.PageContext) {
	if page == nil {
		return
	}
	for i := range messages {
		if messages[i].Role == conversation.RoleUser {
			messages[i].PageContext = page
			return
		}
	}
}
