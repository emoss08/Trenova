package assistantservice

import (
	"context"
	"strings"
	"unicode/utf8"

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
	req repositories.GetThreadRequest,
) ([]conversation.Message, error) {
	// Reading the thread first is the authorization check: it is scoped by user,
	// so a thread belonging to someone else is not found rather than returned.
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}

	return s.conversations.ListMessages(ctx, repositories.ListMessagesRequest{
		ThreadID:   req.ID,
		TenantInfo: req.TenantInfo,
	})
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
	if content == "" {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("content", errortypes.ErrRequired, "Message cannot be empty")

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

	history, err := s.conversations.ListMessages(ctx, repositories.ListMessagesRequest{
		ThreadID:   thread.ID,
		TenantInfo: req.TenantInfo,
		Limit:      historyLimit,
	})
	if err != nil {
		return nil, err
	}

	turn, err := s.RunObserved(ctx, &TurnRequest{
		Definition: definition,
		Actor:      actor,
		History:    history,
		Input:      content,
	}, emit)
	if err != nil {
		return nil, err
	}

	saved, err := s.conversations.AppendTurn(ctx, repositories.AppendTurnRequest{
		ThreadID:   thread.ID,
		TenantInfo: req.TenantInfo,
		Messages:   turn.Messages,
	})
	if err != nil {
		return nil, err
	}

	s.titleIfUnnamed(ctx, thread, content)

	proposals, err := s.persistProposals(ctx, persistProposalsParams{
		Definition: definition,
		Thread:     thread,
		Actor:      actor,
		Saved:      saved,
		Actions:    turn.Proposals,
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
