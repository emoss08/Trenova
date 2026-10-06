package agentmemoryservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const fieldReplaces = "replacesMemoryId"

func (s *Service) planReplacement(
	ctx context.Context,
	req *services.RememberRequest,
	entity *agent.Memory,
) (*agent.Memory, bool, error) {
	notVisible := errortypes.NewValidationError(
		fieldReplaces,
		errortypes.ErrInvalid,
		"No memory with that id is visible here",
	)

	replaced, err := s.repo.GetByID(ctx, repositories.GetAgentMemoryByIDRequest{
		ID:         req.Replaces,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, false, notVisible
		}

		return nil, false, err
	}
	if !s.replacementReads(ctx, req, entity, replaced) {
		return nil, false, notVisible
	}

	inheritReaders(entity, replaced)

	if !Replaceable(replaced.Status) {
		already, err := s.alreadyReplaced(ctx, req.TenantInfo, entity, replaced)
		if err != nil {
			return nil, false, err
		}
		if !already {
			return nil, false, errortypes.NewValidationError(
				fieldReplaces,
				errortypes.ErrInvalid,
				"That memory is no longer kept, so there is nothing to replace; save this as a new memory",
			)
		}
	}

	return replaced, !ReplacedFreely(replaced, req.PersonUserID, entity.AgentDefinitionID), nil
}

func (s *Service) alreadyReplaced(
	ctx context.Context,
	tenant pagination.TenantInfo,
	entity *agent.Memory,
	replaced *agent.Memory,
) (bool, error) {
	same := sameMemoryRequest(tenant, entity)
	same.IncludeSuggested = true
	existing, err := s.repo.FindActive(ctx, same)
	if err != nil {
		return false, err
	}

	return existing != nil && existing.SupersedesID != nil &&
		*existing.SupersedesID == replaced.ID, nil
}

func Replaceable(status agent.MemoryStatus) bool {
	return status == agent.MemoryStatusActive || status == agent.MemoryStatusPaused
}

func ReplacedFreely(replaced *agent.Memory, personID pulid.ID, agentID *pulid.ID) bool {
	switch replaced.Scope {
	case agent.MemoryScopeUser:
		return personID.IsNotNil() && replaced.OwnerUserID != nil &&
			*replaced.OwnerUserID == personID
	case agent.MemoryScopeAgent:
		return agentID != nil && replaced.AgentDefinitionID != nil &&
			*replaced.AgentDefinitionID == *agentID
	default:
		return false
	}
}

func (s *Service) replacementReads(
	ctx context.Context,
	req *services.RememberRequest,
	entity *agent.Memory,
	replaced *agent.Memory,
) bool {
	if replaced.Status.IsSuggestion() {
		return false
	}
	if replaced.Scope == agent.MemoryScopeAgent {
		return entity.AgentDefinitionID != nil && replaced.AgentDefinitionID != nil &&
			*entity.AgentDefinitionID == *replaced.AgentDefinitionID
	}
	if replaced.Scope.Personal() && req.PersonUserID.IsNil() {
		return false
	}

	return s.promptReader(ctx, req.TenantInfo, req.PersonUserID).Reads(replaced)
}

func inheritReaders(entity, replaced *agent.Memory) {
	entity.SetAudience(
		replaced.Scope,
		pulid.ConvertFromPtr(replaced.OwnerUserID),
		pulid.ConvertFromPtr(replaced.RoleID),
	)
	if replaced.Scope == agent.MemoryScopeAgent {
		entity.AgentDefinitionID = pulid.ClonePointer(replaced.AgentDefinitionID)
	}
	if entity.SubjectType == "" && replaced.SubjectType != "" {
		entity.SubjectType = replaced.SubjectType
		entity.SubjectID = pulid.ClonePointer(replaced.SubjectID)
		entity.SubjectLabel = replaced.SubjectLabel
	}
	if entity.ToolName == "" {
		entity.ToolName = replaced.ToolName
	}
}

func (s *Service) retireReplaced(
	ctx context.Context,
	replaced *agent.Memory,
	replacement *agent.Memory,
	actor *services.RequestActor,
) {
	if !Replaceable(replaced.Status) || replaced.ID == replacement.ID {
		return
	}

	tenant := pagination.TenantInfo{OrgID: replaced.OrganizationID, BuID: replaced.BusinessUnitID}
	retired, err := s.repo.SetStatus(ctx, repositories.SetAgentMemoryStatusRequest{
		ID:         replaced.ID,
		TenantInfo: tenant,
		Status:     agent.MemoryStatusRetired,
		ByUserID:   StatusActor(actor),
		At:         timeutils.NowUnix(),
	})
	if err != nil {
		s.l.Warn("agent memory: the memory a restated one replaces could not be retired",
			zap.String("memory", replaced.ID.String()),
			zap.Error(err),
		)

		return
	}

	s.logChange(retired, jsonutils.MustToJSON(replaced), actor, permission.OpUpdate,
		"Agent memory replaced by "+replacement.ID.String())
	s.queueForRetrieval(ctx, retired)
}

func (s *Service) recordReplaced(
	ctx context.Context,
	replacement *agent.Memory,
	actor *services.RequestActor,
) {
	if !replacement.Replaces() {
		return
	}

	retired, err := s.repo.GetByID(ctx, repositories.GetAgentMemoryByIDRequest{
		ID: *replacement.SupersedesID,
		TenantInfo: pagination.TenantInfo{
			OrgID: replacement.OrganizationID,
			BuID:  replacement.BusinessUnitID,
		},
	})
	if err != nil {
		s.l.Warn("agent memory: the memory a new one replaced could not be read back",
			zap.String("memory", replacement.SupersedesID.String()),
			zap.Error(err),
		)

		return
	}
	if retired.Status != agent.MemoryStatusRetired {
		return
	}

	s.log(retired, actor, permission.OpUpdate, "Agent memory replaced by "+replacement.ID.String())
	s.queueForRetrieval(ctx, retired)
}
