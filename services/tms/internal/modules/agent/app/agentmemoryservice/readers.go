package agentmemoryservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type roleReader interface {
	GetUserRoleAssignments(
		ctx context.Context,
		userID, orgID pulid.ID,
	) ([]*rbac.UserRoleAssignment, error)
	GetRolesWithInheritance(ctx context.Context, roleIDs []pulid.ID) ([]*rbac.Role, error)
}

func rolesOrNil(roles permissionports.RoleRepository) roleReader {
	if roles == nil {
		return nil
	}

	return roles
}

// heldRoles are the roles a person holds directly, unexpired, with their names.
func (s *Service) heldRoles(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) ([]*rbac.Role, error) {
	if s.roles == nil || userID.IsNil() {
		return nil, nil
	}

	assignments, err := s.roles.GetUserRoleAssignments(ctx, userID, tenant.OrgID)
	if err != nil {
		return nil, err
	}

	roles := make([]*rbac.Role, 0, len(assignments))
	for _, assignment := range assignments {
		if assignment == nil || assignment.IsExpired() || assignment.Role == nil {
			continue
		}
		roles = append(roles, assignment.Role)
	}
	slices.SortFunc(roles, func(a, b *rbac.Role) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	return roles, nil
}

func (s *Service) Reader(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) (agentmemory.MemoryReader, error) {
	reader := agentmemory.MemoryReader{UserID: userID}
	if s.roles == nil || userID.IsNil() {
		return reader, nil
	}

	held, err := s.heldRoles(ctx, tenant, userID)
	if err != nil {
		return reader, err
	}
	if len(held) == 0 {
		return reader, nil
	}

	ids := make([]pulid.ID, 0, len(held))
	for _, role := range held {
		ids = append(ids, role.ID)
	}
	// A role inherits what its parents grant, and a memory kept for the
	// parent role is part of what its holders work under.
	closure, err := s.roles.GetRolesWithInheritance(ctx, ids)
	if err != nil {
		return reader, err
	}
	for _, role := range closure {
		if role != nil && !slices.Contains(ids, role.ID) {
			ids = append(ids, role.ID)
		}
	}
	reader.RoleIDs = ids

	return reader, nil
}

// promptReader is Reader for a prompt or a recall, which go on without the
// person's roles rather than fail when they cannot be read.
func (s *Service) promptReader(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) agentmemory.MemoryReader {
	reader, err := s.Reader(ctx, tenant, userID)
	if err != nil {
		s.l.Warn(
			"agent memory: the person's roles could not be read; reading their own memories only",
			zap.String("user", userID.String()),
			zap.Error(err),
		)
	}

	return reader
}

func (s *Service) SavingMode(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) (agentmemory.MemorySavingMode, error) {
	if userID.IsNil() {
		return agentmemory.MemorySavingAutomatic, nil
	}

	preference, err := s.repo.GetPreference(ctx, agentmemoryports.GetAgentMemoryPreferenceRequest{
		TenantInfo: tenant,
		UserID:     userID,
	})
	if err != nil {
		return "", err
	}
	if preference.AsksFirst() {
		return agentmemory.MemorySavingAskFirst, nil
	}

	return agentmemory.MemorySavingAutomatic, nil
}
