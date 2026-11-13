package agentmemoryservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	fieldTeamName   = "teamName"
	upkeepListLimit = 25
)

func (s *Service) Upkeep(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]*agentmemoryports.MemoryUpkeep, error) {
	window := agentmemory.UpkeepWindow(timeutils.NowUnix())
	memories, err := s.repo.ListUpkeep(ctx, &agentmemoryports.ListAgentMemoryUpkeepRequest{
		TenantInfo: tenant,
		Window:     window,
		Limit:      upkeepListLimit,
	})
	if err != nil {
		return nil, err
	}

	out := make([]*agentmemoryports.MemoryUpkeep, 0, len(memories))
	for _, memory := range memories {
		if reason := memory.UpkeepReason(window); reason != "" {
			out = append(out, &agentmemoryports.MemoryUpkeep{Memory: memory, Reason: reason})
		}
	}

	return out, nil
}

func (s *Service) Confirm(
	ctx context.Context,
	req agentmemoryports.ReviewAgentMemoryRequest,
	actor *permissionports.RequestActor,
) (*agentmemory.Memory, error) {
	if !actor.IsUser() {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrForbidden,
			"Only a person can say a memory still holds",
		)
	}

	current, err := s.repo.GetByID(ctx, agentmemoryports.GetAgentMemoryByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if current.Status != agentmemory.MemoryStatusActive {
		return nil, errortypes.NewBusinessError(
			"Only a memory agents are reading can be confirmed; this one is {0}",
			strings.ToLower(string(current.Status)),
		)
	}
	if current.Scope == agentmemory.MemoryScopeUser &&
		(current.OwnerUserID == nil || *current.OwnerUserID != actor.UserID) {
		return nil, errortypes.NewAuthorizationError(
			"A memory kept for one person is theirs to confirm",
		)
	}

	confirmed, err := s.repo.Confirm(ctx, &agentmemoryports.ConfirmAgentMemoryQuery{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
		ByUserID:   actor.UserID,
		At:         timeutils.NowUnix(),
		Version:    req.Version,
	})
	if err != nil {
		return nil, err
	}

	s.security.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceAgentMemory,
		ResourceID:     confirmed.ID.String(),
		Operation:      permission.OpUpdate,
		Actor:          actor.AuditActor(),
		OrganizationID: confirmed.OrganizationID,
		BusinessUnitID: confirmed.BusinessUnitID,
		Before:         jsonutils.MustToJSON(current),
		After:          jsonutils.MustToJSON(confirmed),
		Comment:        "Agent memory confirmed: a person said it still holds",
	})

	return confirmed, nil
}

func (s *Service) TeamRole(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID pulid.ID,
	name string,
) (pulid.ID, error) {
	held, err := s.heldRoles(ctx, tenant, userID)
	if err != nil {
		return pulid.Nil, err
	}
	if len(held) == 0 {
		return pulid.Nil, errortypes.NewValidationError(
			fieldTeamName,
			errortypes.ErrInvalid,
			"The person holds no role to keep this for; keep it for them or the organization",
		)
	}

	names := make([]string, 0, len(held))
	for _, role := range held {
		names = append(names, role.Name)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		if len(held) == 1 {
			return held[0].ID, nil
		}

		return pulid.Nil, errortypes.NewValidationError(
			fieldTeamName,
			errortypes.ErrRequired,
			"The person holds several roles: {0}. Ask which team this is for, then pass "+
				"that role's name as teamName",
			strings.Join(names, ", "),
		)
	}
	for _, role := range held {
		if strings.EqualFold(role.Name, name) {
			return role.ID, nil
		}
	}

	return pulid.Nil, errortypes.NewValidationError(
		fieldTeamName,
		errortypes.ErrInvalid,
		"The person holds no role named \"{0}\". Their roles: {1}",
		name, strings.Join(names, ", "),
	)
}

func (s *Service) Outcomes(
	ctx context.Context,
	tenant pagination.TenantInfo,
	since int64,
) ([]agentmemoryports.MemoryOutcome, error) {
	return s.repo.ListOutcomes(ctx, &agentmemoryports.ListAgentMemoryOutcomesRequest{
		TenantInfo: tenant,
		Since:      since,
	})
}

func (s *Service) LapseOffers(
	ctx context.Context,
	tenant pagination.TenantInfo,
	now int64,
) (int64, error) {
	return s.repo.LapseOffers(ctx, &agentmemoryports.LapseAgentMemoryOffersRequest{
		TenantInfo: tenant,
		Before:     agentmemory.OffersLapseBefore(now),
		At:         now,
	})
}

func (s *Service) ResolveSubject(
	ctx context.Context,
	tenant pagination.TenantInfo,
	subjectType agentmemory.MemorySubjectType,
	name string,
) (pulid.ID, error) {
	return s.subjectNames.Resolve(ctx, tenant, subjectType, name)
}
