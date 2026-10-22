// Package teamscope answers the question a manager's grant raises before they
// touch one worker's records: may they, and when their grant reaches only
// their own team, is this worker on it.
package teamscope

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/app/orgstructureservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var (
	ErrOutsideTeam = errors.New(
		"that worker is not on the team of the person you act for, and their access " +
			"reaches only their own team",
	)
	ErrUnchecked = errors.New("the person's access could not be checked")
)

type Permissions interface {
	Check(
		ctx context.Context,
		req *permissionports.PermissionCheckRequest,
	) (*permissionports.PermissionCheckResult, error)
}

type Teams interface {
	CanActFor(
		ctx context.Context,
		req *orgstructureservice.ScopeRequest,
	) (*orgstructureservice.ScopeResult, error)
}

// Guard is the check the scheduling resolvers make. A team-scoped grant with
// no org structure behind it would widen silently, so it is refused.
type Guard struct {
	Permissions Permissions
	Teams       Teams
}

type Request struct {
	Actor      *permissionports.RequestActor
	TenantInfo pagination.TenantInfo
	Resource   permission.Resource
	Operation  permission.Operation
	WorkerID   pulid.ID
	// ApprovalScope is which delegations count toward the team; empty is all.
	ApprovalScope worker.ApprovalScope
}

func (r *Request) approvalScope() worker.ApprovalScope {
	if r.ApprovalScope == "" {
		return worker.ApprovalScopeAll
	}
	return r.ApprovalScope
}

func (g Guard) Require(ctx context.Context, req *Request) error {
	if g.Permissions == nil || req.Actor == nil {
		return ErrUnchecked
	}
	result, err := g.Permissions.Check(ctx, req.Actor.PermissionCheck(req.Resource, req.Operation))
	if err != nil {
		return fmt.Errorf("authorize %s on %s: %w", req.Operation, req.Resource, err)
	}
	if !result.Allowed {
		return fmt.Errorf("the person you act for may not %s %s records",
			req.Operation, req.Resource)
	}
	if result.DataScope != permission.DataScopeTeam {
		return nil
	}
	if g.Teams == nil {
		return ErrOutsideTeam
	}

	scope, err := g.Teams.CanActFor(ctx, &orgstructureservice.ScopeRequest{
		TenantInfo: req.TenantInfo,
		UserID:     req.Actor.UserID,
		WorkerID:   req.WorkerID,
		Scope:      req.approvalScope(),
	})
	if err != nil {
		return fmt.Errorf("check the person's team: %w", err)
	}
	if !scope.Allowed {
		return ErrOutsideTeam
	}

	return nil
}

type Managers interface {
	ManagerIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		userID pulid.ID,
		scope worker.ApprovalScope,
		asOf int64,
	) ([]pulid.ID, map[pulid.ID]pulid.ID, error)
}

// ListGuard is Guard for a list that spans workers: rather than answer for one
// worker, it names the managers a team-scoped grant narrows the list to.
type ListGuard struct {
	Permissions Permissions
	Managers    Managers
}

// Narrow returns the managers whose team the list is held to, or none when the
// grant reaches the whole organization. WorkerID is ignored.
func (g ListGuard) Narrow(ctx context.Context, req *Request) ([]pulid.ID, error) {
	if g.Permissions == nil || req.Actor == nil {
		return nil, ErrUnchecked
	}
	result, err := g.Permissions.Check(ctx, req.Actor.PermissionCheck(req.Resource, req.Operation))
	if err != nil {
		return nil, fmt.Errorf("authorize %s on %s: %w", req.Operation, req.Resource, err)
	}
	if !result.Allowed {
		return nil, fmt.Errorf("the person you act for may not %s %s records",
			req.Operation, req.Resource)
	}
	if result.DataScope != permission.DataScopeTeam {
		return nil, nil
	}
	if g.Managers == nil {
		return nil, ErrOutsideTeam
	}

	ids, _, err := g.Managers.ManagerIDs(
		ctx, req.TenantInfo, req.Actor.UserID, req.approvalScope(), 0,
	)
	if err != nil {
		return nil, fmt.Errorf("find the person's team: %w", err)
	}
	if len(ids) == 0 {
		return nil, ErrOutsideTeam
	}

	return ids, nil
}
