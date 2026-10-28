package workforcetools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/app/teamscope"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/approvaldelegation"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type LeaveWorkers interface {
	CaseWorker(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		caseID pulid.ID,
	) (pulid.ID, error)
	EntryWorker(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		entryID pulid.ID,
	) (pulid.ID, error)
}

// LeaveScope is the check the leave resolvers make: a driver manager's
// team-scoped leave grant reaches only the cases of the workers on their team.
// The worker is always the one the case or day belongs to, never one taken
// from alongside it.
type LeaveScope struct {
	guard   teamscope.Guard
	workers LeaveWorkers
}

func NewLeaveScope(
	permissions teamscope.Permissions,
	teams teamscope.Teams,
	workers LeaveWorkers,
) LeaveScope {
	return LeaveScope{
		guard:   teamscope.Guard{Permissions: permissions, Teams: teams},
		workers: workers,
	}
}

func (s LeaveScope) worker(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
	operation permission.Operation,
	workerID pulid.ID,
) error {
	return s.guard.Require(ctx, &teamscope.Request{
		Actor:         params.Actor,
		TenantInfo:    toolparams.TenantFrom(*params),
		Resource:      permission.ResourceWorkerLeave,
		Operation:     operation,
		WorkerID:      workerID,
		ApprovalScope: approvaldelegation.ApprovalScopeTimeOff,
	})
}

func (s LeaveScope) leaveCase(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
	operation permission.Operation,
	caseID pulid.ID,
) error {
	if s.workers == nil {
		return teamscope.ErrUnchecked
	}
	workerID, err := s.workers.CaseWorker(ctx, toolparams.TenantFrom(*params), caseID)
	if err != nil {
		return err
	}
	return s.worker(ctx, params, operation, workerID)
}

func (s LeaveScope) entry(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
	operation permission.Operation,
	entryID pulid.ID,
) error {
	if s.workers == nil {
		return teamscope.ErrUnchecked
	}
	workerID, err := s.workers.EntryWorker(ctx, toolparams.TenantFrom(*params), entryID)
	if err != nil {
		return err
	}
	return s.worker(ctx, params, operation, workerID)
}
