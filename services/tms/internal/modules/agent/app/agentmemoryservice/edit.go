package agentmemoryservice

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	fieldEditOld = "old"
	fieldEditNew = "new"
)

func (s *Service) PlanEdit(
	ctx context.Context,
	req *agentmemoryports.EditAgentMemoryRequest,
) (*agentmemoryports.RememberRequest, error) {
	current, err := s.repo.GetByID(ctx, agentmemoryports.GetAgentMemoryByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errMemoryNotVisible()
		}

		return nil, err
	}
	if current.Status.IsSuggestion() || !s.editorReads(ctx, req, current) {
		return nil, errMemoryNotVisible()
	}
	if !Replaceable(current.Status) {
		return nil, errortypes.NewValidationError(
			"memoryId",
			errortypes.ErrInvalid,
			"That memory is no longer kept, so there is nothing to edit; save a new memory instead",
		)
	}

	content, err := EditedContent(current, req.Old, req.New)
	if err != nil {
		return nil, err
	}

	return &agentmemoryports.RememberRequest{
		TenantInfo:   req.TenantInfo,
		Kind:         current.Kind,
		Content:      content,
		SubjectType:  current.SubjectType,
		SubjectID:    pulid.ConvertFromPtr(current.SubjectID),
		ToolName:     current.ToolName,
		ExpiresAt:    current.ExpiresAt,
		RunID:        req.RunID,
		ProposalID:   req.ProposalID,
		Taint:        req.Taint,
		Scope:        current.Scope,
		OwnerUserID:  pulid.ConvertFromPtr(current.OwnerUserID),
		RoleID:       pulid.ConvertFromPtr(current.RoleID),
		PersonUserID: req.PersonUserID,
		Replaces:     current.ID,
	}, nil
}

func (s *Service) editorReads(
	ctx context.Context,
	req *agentmemoryports.EditAgentMemoryRequest,
	memory *agentmemory.Memory,
) bool {
	switch {
	case memory.Scope == agentmemory.MemoryScopeAgent:
		return req.AgentDefinitionID.IsNotNil() && memory.AgentDefinitionID != nil &&
			*memory.AgentDefinitionID == req.AgentDefinitionID
	case memory.Scope.Personal():
		return req.PersonUserID.IsNotNil() &&
			s.promptReader(ctx, req.TenantInfo, req.PersonUserID).Reads(memory)
	default:
		return true
	}
}

func EditedContent(memory *agentmemory.Memory, old, replacement string) (string, error) {
	if old == "" {
		return "", errortypes.NewValidationError(
			fieldEditOld,
			errortypes.ErrRequired,
			"Say which words of the memory to change",
		)
	}

	switch count := strings.Count(memory.Content, old); count {
	case 0:
		return "", errortypes.NewValidationError(
			fieldEditOld,
			errortypes.ErrInvalid,
			"Those words are not in the memory as it stands; read it with agent_list_memories "+
				"and its id, then copy the words exactly",
		)
	case 1:
	default:
		return "", errortypes.NewValidationError(
			fieldEditOld,
			errortypes.ErrInvalid,
			"Those words appear {0} times in the memory; include enough of the words around "+
				"them to name one place",
			count,
		)
	}

	content := strings.TrimSpace(strings.Replace(memory.Content, old, replacement, 1))
	if content == "" {
		return "", errortypes.NewValidationError(
			fieldEditNew,
			errortypes.ErrInvalid,
			"The edit would leave the memory empty; retire it instead",
		)
	}
	if limit := memory.Kind.MaxContentChars(); utf8.RuneCountInString(content) > limit {
		return "", errortypes.NewValidationError(
			fieldEditNew,
			errortypes.ErrInvalid,
			"The edit would make the memory longer than {0} characters",
			limit,
		)
	}
	if content == strings.TrimSpace(memory.Content) {
		return "", errortypes.NewValidationError(
			fieldEditNew,
			errortypes.ErrInvalid,
			"The edit changes nothing",
		)
	}

	return content, nil
}

func errMemoryNotVisible() error {
	return errortypes.NewValidationError(
		"memoryId",
		errortypes.ErrInvalid,
		"No memory with that id is visible here",
	)
}
