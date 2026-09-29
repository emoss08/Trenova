// Package teamscope answers the question a manager's grant raises before they
// touch one worker's records: may they, and when their grant reaches only
// their own team, is this worker on it.
package teamscope

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/orgstructureservice"
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
		req *services.PermissionCheckRequest,
	) (*services.PermissionCheckResult, error)
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
	Actor      *services.RequestActor
	TenantInfo pagination.TenantInfo
	Resource   permission.Resource
	Operation  permission.Operation
	WorkerID   pulid.ID
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
		Scope:      worker.ApprovalScopeAll,
	})
	if err != nil {
		return fmt.Errorf("check the person's team: %w", err)
	}
	if !scope.Allowed {
		return ErrOutsideTeam
	}

	return nil
}
