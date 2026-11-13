package agentmemoryservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// These are the entry points the Desk's memory page keeps and changes
// memories through (agentmemorydesk.Core), so what a person does there goes
// through the same dedupe, consolidation, taint and audit as every writer.

func (s *Service) Keep(
	ctx context.Context,
	req *agentmemoryports.RememberRequest,
	actor *permissionports.RequestActor,
	manages bool,
) (*agentmemory.Memory, error) {
	return s.remember(ctx, req, actor, manages)
}

func (s *Service) SetStatusAs(
	ctx context.Context,
	req *agentmemoryports.SetAgentMemoryStatusRequest,
	actor *permissionports.RequestActor,
	manages bool,
) (*agentmemory.Memory, error) {
	return s.setStatus(ctx, req, actor, manages)
}

func (s *Service) QueueForRetrieval(ctx context.Context, memory *agentmemory.Memory) {
	s.queueForRetrieval(ctx, memory)
}

func (s *Service) LogChange(
	ctx context.Context,
	entity *agentmemory.Memory,
	previous map[string]any,
	actor *permissionports.RequestActor,
	op permission.Operation,
	comment string,
) {
	s.logChange(ctx, entity, previous, actor, op, comment)
}

func (s *Service) RecordReview(
	ctx context.Context,
	before, after *agentmemory.Memory,
	actor *permissionports.RequestActor,
) {
	s.recordReview(ctx, before, after, actor)
}

func (s *Service) RecordReplaced(
	ctx context.Context,
	replacement *agentmemory.Memory,
	actor *permissionports.RequestActor,
) {
	s.recordReplaced(ctx, replacement, actor)
}

func (s *Service) HeldRoles(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) ([]*rbac.Role, error) {
	return s.heldRoles(ctx, tenant, userID)
}
