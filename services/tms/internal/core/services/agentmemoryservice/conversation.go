package agentmemoryservice

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	MaxConversationSuggestions   = 3
	conversationReasonChars      = 400
	conversationSuggestionWindow = 500
	conversationDismissalDays    = 30
)

func (s *Service) SuggestFromConversation(
	ctx context.Context,
	req *services.SuggestFromConversationRequest,
) (int, error) {
	if req == nil || len(req.Candidates) == 0 {
		return 0, nil
	}
	if req.AgentDefinitionID.IsNil() || req.ThreadID.IsNil() || req.SummaryID.IsNil() {
		return 0, errortypes.NewValidationError(
			"summaryId",
			errortypes.ErrRequired,
			"A suggestion from a conversation needs its agent, conversation and summary",
		)
	}

	known, err := s.knownSuggestions(ctx, req)
	if err != nil {
		return 0, err
	}

	created := 0
	for idx := range req.Candidates {
		if created == MaxConversationSuggestions {
			break
		}

		memory := s.conversationSuggestion(ctx, req, &req.Candidates[idx])
		if memory == nil {
			continue
		}

		key := comparableContent(memory.Content)
		if _, dup := known[key]; dup {
			continue
		}
		if existing, findErr := s.repo.FindActive(
			ctx,
			sameMemoryRequest(req.TenantInfo, memory),
		); findErr != nil {
			return created, findErr
		} else if existing != nil {
			continue
		}

		saved, createErr := s.repo.Create(ctx, memory)
		if createErr != nil {
			return created, createErr
		}
		known[key] = struct{}{}
		created++

		s.logChange(
			saved,
			nil,
			nil,
			permission.OpCreate,
			"Agent memory suggested from a conversation summary",
		)
		s.queueForRetrieval(ctx, saved)
	}

	return created, nil
}

func (s *Service) conversationSuggestion(
	ctx context.Context,
	req *services.SuggestFromConversationRequest,
	candidate *services.ConversationMemoryCandidate,
) *agent.Memory {
	kind := candidate.Kind
	if kind != agent.MemoryKindInstruction && kind != agent.MemoryKindFact {
		return nil
	}

	tainted := req.Taint.Tainted()
	if tainted && kind == agent.MemoryKindInstruction {
		return nil
	}

	content := strings.TrimSpace(candidate.Content)
	if content == "" || utf8.RuneCountInString(content) > agent.MaxMemoryContentChars {
		return nil
	}

	agentID := req.AgentDefinitionID
	memory := &agent.Memory{
		OrganizationID:    req.TenantInfo.OrgID,
		BusinessUnitID:    req.TenantInfo.BuID,
		Kind:              kind,
		Source:            agent.MemorySourceConversation,
		Status:            agent.MemoryStatusSuggested,
		Scope:             agent.MemoryScopeAgent,
		AgentDefinitionID: &agentID,
		Content:           content,
		Evidence: &agent.MemoryEvidence{
			Reason:    stringutils.Ellipsize(strings.TrimSpace(candidate.Reason), conversationReasonChars),
			ThreadID:  req.ThreadID,
			SummaryID: req.SummaryID,
		},
	}
	memory.Tainted = tainted

	if candidate.SubjectType != "" || candidate.SubjectID.IsNotNil() {
		if !candidate.SubjectType.IsValid() || candidate.SubjectID.IsNil() {
			return nil
		}
		subjectID := candidate.SubjectID
		memory.SubjectType = candidate.SubjectType
		memory.SubjectID = &subjectID
		if err := s.label(ctx, req.TenantInfo, memory); err != nil {
			s.l.Debug("conversation memory suggestion names a record that is not there",
				zap.String("subjectType", string(candidate.SubjectType)),
				zap.Error(err),
			)

			return nil
		}
	}

	me := errortypes.NewMultiError()
	memory.Validate(me)
	if me.HasErrors() {
		s.l.Debug("conversation memory suggestion is not a valid memory",
			zap.String("errors", jsonutils.MustToJSON(me)),
		)

		return nil
	}

	return memory
}

func (s *Service) knownSuggestions(
	ctx context.Context,
	req *services.SuggestFromConversationRequest,
) (map[string]struct{}, error) {
	now := timeutils.NowUnix()
	pending, err := s.repo.ListSuggestionContext(ctx, repositories.ListAgentMemorySuggestionContextRequest{
		TenantInfo:        req.TenantInfo,
		AgentDefinitionID: req.AgentDefinitionID,
		Now:               now,
		DismissedSince:    now - conversationDismissalDays*24*60*60,
		Limit:             conversationSuggestionWindow,
	})
	if err != nil {
		return nil, err
	}

	known := make(map[string]struct{}, len(pending))
	for _, memory := range pending {
		known[comparableContent(memory.Content)] = struct{}{}
	}

	return known, nil
}

func comparableContent(content string) string {
	return strings.ToLower(strings.Join(strings.Fields(content), " "))
}
