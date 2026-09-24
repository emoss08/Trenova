package assistantservice

import (
	"context"
	"strings"
	"unicode/utf8"

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
	// turns are the ones that carry the thread of the question. Tool results
	// count toward it, and the runtime cuts the older ones down before the
	// replay, so the limit is set by how many turns should stay in view
	// rather than by how much data those turns fetched.
	historyLimit = 120
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
	if err = assertChatAgent(definition); err != nil {
		return nil, err
	}
	if err = s.assertMayUseAgent(ctx, actor, definition); err != nil {
		return nil, err
	}
	if req.SubjectID.IsNotNil() {
		if err = s.assertSubjectReadable(ctx, actor, req.SubjectType); err != nil {
			return nil, err
		}
	}

	origin := req.Origin
	if origin == "" {
		origin = conversation.ThreadOriginPanel
	}

	thread := &conversation.Thread{
		OrganizationID:    req.TenantInfo.OrgID,
		BusinessUnitID:    req.TenantInfo.BuID,
		UserID:            actor.UserID,
		AgentDefinitionID: definition.ID,
		Title:             strings.TrimSpace(req.Title),
		Status:            conversation.ThreadStatusActive,
		Origin:            origin,
		SubjectType:       req.SubjectType,
		SubjectID:         req.SubjectID,
	}

	multiErr := errortypes.NewMultiError()
	thread.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.conversations.CreateThread(ctx, thread)
	if err != nil {
		return nil, err
	}
	created.CanContinue = true

	return created, nil
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
	result, err := s.conversations.ListThreads(ctx, req)
	if err != nil {
		return nil, err
	}
	if err = s.markContinuable(
		ctx,
		threadReader(req.UserID, req.TenantInfo),
		result.Items...,
	); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) GetThread(
	ctx context.Context,
	req repositories.GetThreadRequest,
) (*conversation.Thread, error) {
	thread, err := s.conversations.GetThread(ctx, req)
	if err != nil {
		return nil, err
	}
	if err = s.markContinuable(ctx, threadReader(req.UserID, req.TenantInfo), thread); err != nil {
		return nil, err
	}

	return thread, nil
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

	s.runtime.MarkToolEffects(messages)
	s.nameDelegatedSteps(ctx, req.Thread.TenantInfo, messages)

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

// UpdateThread changes what a person may change about their own
// conversation. The thread is read under their user id, so someone else's
// is not found rather than renamed.
func (s *Service) UpdateThread(
	ctx context.Context,
	req *services.UpdateThreadRequest,
	actor *services.RequestActor,
) (*conversation.Thread, error) {
	thread, err := s.conversations.GetThread(ctx, repositories.GetThreadRequest{
		ID:         req.ThreadID,
		UserID:     actor.UserID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		thread.Title = strings.TrimSpace(*req.Title)
	}
	if req.Pinned != nil {
		thread.Pinned = *req.Pinned
	}
	if req.Keep && !thread.Origin.Listed() {
		thread.Origin = conversation.ThreadOriginDesk
	}

	multiErr := errortypes.NewMultiError()
	thread.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	updated, err := s.conversations.UpdateThread(ctx, thread)
	if err != nil {
		return nil, err
	}
	if err = s.markContinuable(ctx, actor, updated); err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) DeleteThread(
	ctx context.Context,
	req repositories.GetThreadRequest,
) error {
	return s.conversations.DeleteThread(ctx, req)
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
