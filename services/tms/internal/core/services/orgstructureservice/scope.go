package orgstructureservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// ScopeRequest asks whether a user may act on a worker, and under what scope.
type ScopeRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	WorkerID   pulid.ID
	Scope      worker.ApprovalScope
	AsOf       int64
}

// ScopeResult says whether the user answers for the worker, and through whom.
// The delegator is carried back so an approval can record that it was made in
// somebody's place rather than in the delegate's own right.
type ScopeResult struct {
	Allowed bool
	// OnBehalfOf is set only when the answer came through a delegation.
	OnBehalfOf pulid.ID
}

// ManagerIDs is the set of people a user answers for: themselves, plus anybody
// who has delegated to them inside the requested scope. One list rather than a
// query each, because the roster question is the expensive half.
func (s *Service) ManagerIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
	scope worker.ApprovalScope,
	asOf int64,
) ([]pulid.ID, map[pulid.ID]pulid.ID, error) {
	if userID.IsNil() {
		return nil, nil, nil
	}
	if asOf <= 0 {
		asOf = timeutils.NowUnix()
	}
	if scope == "" {
		scope = worker.ApprovalScopeAll
	}

	ids := []pulid.ID{userID}
	// via maps a delegator onto themselves, so a caller that finds the worker
	// through one can say whose authority it used.
	via := make(map[pulid.ID]pulid.ID, 2)

	delegations, err := s.repo.ListDelegations(ctx, &repositories.ListApprovalDelegationsRequest{
		TenantInfo: tenantInfo,
		DelegateID: userID,
		ActiveAt:   asOf,
	})
	if err != nil {
		return nil, nil, err
	}
	for _, delegation := range delegations {
		if delegation == nil || !delegation.Scope.Covers(scope) {
			continue
		}
		if !delegation.IsActive(asOf) {
			continue
		}
		ids = append(ids, delegation.DelegatorID)
		via[delegation.DelegatorID] = delegation.DelegatorID
	}

	return ids, via, nil
}

// CanActFor answers the authorization question every HR area asks: may this
// user approve for this worker. It is deliberately not "is this user an admin"
// — the org-wide grant is checked by the resolver before this is reached, and
// this narrows it to the people the user actually manages.
func (s *Service) CanActFor(ctx context.Context, req *ScopeRequest) (*ScopeResult, error) {
	managerIDs, via, err := s.ManagerIDs(
		ctx,
		req.TenantInfo,
		req.UserID,
		req.Scope,
		req.AsOf,
	)
	if err != nil {
		return nil, err
	}
	if len(managerIDs) == 0 {
		return &ScopeResult{}, nil
	}

	// The direct answer first: if the user manages the worker in their own
	// right, no delegation was used and none should be recorded.
	own, err := s.repo.ManagesWorker(
		ctx,
		req.TenantInfo,
		[]pulid.ID{req.UserID},
		req.WorkerID,
	)
	if err != nil {
		return nil, err
	}
	if own {
		return &ScopeResult{Allowed: true}, nil
	}

	for delegatorID := range via {
		reached, mErr := s.repo.ManagesWorker(
			ctx,
			req.TenantInfo,
			[]pulid.ID{delegatorID},
			req.WorkerID,
		)
		if mErr != nil {
			return nil, mErr
		}
		if reached {
			return &ScopeResult{Allowed: true, OnBehalfOf: delegatorID}, nil
		}
	}

	return &ScopeResult{}, nil
}

// MyTeam is everyone a user answers for, directly or through a terminal they
// run, plus anyone delegated to them.
func (s *Service) MyTeam(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
	includeInactive bool,
) ([]repositories.TeamMemberRow, error) {
	managerIDs, _, err := s.ManagerIDs(
		ctx,
		tenantInfo,
		userID,
		worker.ApprovalScopeAll,
		0,
	)
	if err != nil {
		return nil, err
	}

	return s.repo.TeamMembers(ctx, &repositories.TeamScopeRequest{
		TenantInfo:      tenantInfo,
		ManagerIDs:      managerIDs,
		IncludeInactive: includeInactive,
	})
}

func (s *Service) ListDelegations(
	ctx context.Context,
	req *repositories.ListApprovalDelegationsRequest,
) ([]*worker.ApprovalDelegation, error) {
	return s.repo.ListDelegations(ctx, req)
}

// DelegateRequest hands approval to somebody else for a while.
type DelegateRequest struct {
	TenantInfo  pagination.TenantInfo
	DelegatorID pulid.ID
	DelegateID  pulid.ID
	Scope       worker.ApprovalScope
	StartsAt    int64
	EndsAt      *int64
	Reason      string
	UserID      pulid.ID
}

// Delegate records a delegation. A delegation widens what the delegate may act
// on; it never widens what the delegator could approve in the first place, so
// there is nothing to check about the delegator's own grants here.
func (s *Service) Delegate(
	ctx context.Context,
	req *DelegateRequest,
) (*worker.ApprovalDelegation, error) {
	entity := &worker.ApprovalDelegation{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		DelegatorID:    req.DelegatorID,
		DelegateID:     req.DelegateID,
		Scope:          req.Scope,
		StartsAt:       req.StartsAt,
		EndsAt:         req.EndsAt,
		Reason:         req.Reason,
		CreatedByID:    req.UserID,
	}
	if entity.Scope == "" {
		entity.Scope = worker.ApprovalScopeAll
	}
	if entity.StartsAt <= 0 {
		entity.StartsAt = timeutils.NowUnix()
	}
	entity.Normalise()

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if err := s.checkNoOverlap(ctx, entity); err != nil {
		return nil, err
	}

	created, err := s.repo.CreateDelegation(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource: permission.ResourceApprovalDelegation, resourceID: created.GetResourceID(),
		operation: permission.OpCreate, userID: req.UserID, tenant: req.TenantInfo,
		current: created,
		comment: "Approval delegated for " + created.Scope.String(),
	})
	s.publish(
		ctx,
		req.TenantInfo,
		realtimeDelegation,
		permission.OpCreate,
		created.ID,
		req.UserID,
	)

	return created, nil
}

// checkNoOverlap refuses a second live delegation of the same scope between the
// same two people. Two overlapping delegations are not more authority than
// one, and the pair is impossible to revoke confidently.
func (s *Service) checkNoOverlap(
	ctx context.Context,
	entity *worker.ApprovalDelegation,
) error {
	existing, err := s.repo.ListDelegations(ctx, &repositories.ListApprovalDelegationsRequest{
		TenantInfo:  pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		DelegatorID: entity.DelegatorID,
		DelegateID:  entity.DelegateID,
		ActiveAt:    entity.StartsAt,
	})
	if err != nil {
		return err
	}
	for _, row := range existing {
		if row == nil || row.ID == entity.ID {
			continue
		}
		if row.Scope.Covers(entity.Scope) || entity.Scope.Covers(row.Scope) {
			return errortypes.NewValidationError(
				"delegateId",
				errortypes.ErrInvalidOperation,
				"That approval is already delegated to this person for the same period",
			)
		}
	}
	return nil
}

// RevokeDelegation calls a delegation back. The row is kept rather than
// deleted so an approval made under it can still be explained.
func (s *Service) RevokeDelegation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	userID pulid.ID,
) (*worker.ApprovalDelegation, error) {
	original, err := s.repo.GetDelegationByID(ctx, &repositories.GetApprovalDelegationByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if original.RevokedAt != nil && *original.RevokedAt > 0 {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalidOperation,
			"That delegation has already been revoked",
		)
	}

	previous := *original
	now := timeutils.NowUnix()
	original.RevokedAt = &now
	original.RevokedByID = userID

	updated, err := s.repo.UpdateDelegation(ctx, original)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource: permission.ResourceApprovalDelegation, resourceID: updated.GetResourceID(),
		operation: permission.OpUpdate, userID: userID, tenant: tenantInfo,
		current: updated, previous: &previous, comment: "Delegation revoked",
	})
	s.publish(ctx, tenantInfo, realtimeDelegation, permission.OpUpdate, updated.ID, userID)

	return updated, nil
}
