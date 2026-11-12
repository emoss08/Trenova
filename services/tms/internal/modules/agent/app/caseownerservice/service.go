package caseownerservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/caseowner"
	"github.com/emoss08/trenova/internal/modules/agent/ports/caseownerports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordshareports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/approvaldelegation"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/orgstructureports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxCovered = 50

// Delegations reads who has handed their work to whom.
type Delegations interface {
	ListDelegations(
		ctx context.Context,
		req *orgstructureports.ListApprovalDelegationsRequest,
	) ([]*approvaldelegation.ApprovalDelegation, error)
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	Repo          caseownerports.Repository
	Permissions   permissionports.PermissionEngine
	Delegations   orgstructureports.OrgStructureRepository
	Auditor       permissionports.SecurityAuditor
	Notifications *notificationservice.Service
	Realtime      realtimeports.RealtimeService
	Shares        recordshareports.Repository
}

// Shares reads which records were shared with a person.
type Shares interface {
	SharedWith(ctx context.Context, req *recordshareports.SharedWithRequest) ([]pulid.ID, error)
}

type Service struct {
	l             *zap.Logger
	repo          caseownerports.Repository
	permissions   permissionports.PermissionEngine
	delegations   Delegations
	auditor       permissionports.SecurityAuditor
	notifications *notificationservice.Service
	realtime      realtimeports.RealtimeService
	shares        Shares
	now           func() int64
}

var _ caseownerports.Service = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.case-owner"),
		repo:          p.Repo,
		permissions:   p.Permissions,
		delegations:   p.Delegations,
		auditor:       p.Auditor,
		notifications: p.Notifications,
		realtime:      p.Realtime,
		shares:        p.Shares,
		now:           timeutils.NowUnix,
	}
}

func AsService(s *Service) caseownerports.Service { return s }

func (s *Service) allowed(
	ctx context.Context,
	actor *permissionports.RequestActor,
	operation permission.Operation,
) (bool, error) {
	result, err := s.permissions.Check(ctx, &permissionports.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       permission.ResourceWatchtower.String(),
		Operation:      operation,
	})
	if err != nil {
		return false, err
	}

	return result != nil && result.Allowed, nil
}

/*
Sight is whose cases the actor works: their own, and those of everyone who has
handed them their cases for now, by a delegation that covers cases or by
marking themselves away with the actor named to cover. Anything that is not a person acting for
themselves (an agent run, a system job) works every case, since it is not
what ownership keeps apart.
*/
func (s *Service) Sight(
	ctx context.Context,
	actor *permissionports.RequestActor,
) (caseowner.Sight, error) {
	if !actor.IsUser() {
		return caseowner.Sight{All: true}, nil
	}

	all, err := s.allowed(ctx, actor, permission.OpManage)
	if err != nil {
		return caseowner.Sight{}, err
	}

	covered, err := s.delegations.ListDelegations(
		ctx,
		&orgstructureports.ListApprovalDelegationsRequest{
			TenantInfo: actor.TenantInfo(),
			DelegateID: actor.UserID,
			ActiveAt:   s.now(),
			Limit:      maxCovered,
		},
	)
	if err != nil {
		return caseowner.Sight{}, err
	}

	away, err := s.repo.AwayFor(ctx, caseownerports.AwayRequest{
		TenantInfo: actor.TenantInfo(),
		Now:        s.now(),
		CoverID:    actor.UserID,
	})
	if err != nil {
		return caseowner.Sight{}, err
	}

	as := make([]pulid.ID, 0, len(covered)+len(away)+1)
	as = append(as, actor.UserID)
	for _, delegation := range covered {
		if delegation.Scope.Covers(approvaldelegation.ApprovalScopeCases) &&
			!slices.Contains(as, delegation.DelegatorID) {
			as = append(as, delegation.DelegatorID)
		}
	}
	for _, owner := range away {
		if !slices.Contains(as, owner) {
			as = append(as, owner)
		}
	}

	return caseowner.Sight{All: all, As: as, Viewer: actor.UserID}, nil
}

func (s *Service) Standing(
	ctx context.Context,
	req caseownerports.RecordRequest,
	actor *permissionports.RequestActor,
) (*caseownerports.Standing, error) {
	owner, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, err
	}
	reader, err := s.readerOf(ctx, actor)
	if err != nil {
		return nil, err
	}
	shared, err := s.sharedWith(ctx, req.TenantInfo, reader, []pulid.ID{req.RecordID})
	if err != nil {
		return nil, err
	}

	return reader.standing(owner, shared[req.RecordID]), nil
}

/*
sharedWith is which of the records were shared with the reader. A share
brings a case someone else owns into the reader's sight: they read it and
ask about it, and its owner still decides it. A reader who sees every case,
or is not a person, needs no share and is not looked up.
*/
func (s *Service) sharedWith(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	reader caseReader,
	recordIDs []pulid.ID,
) (map[pulid.ID]bool, error) {
	shared := make(map[pulid.ID]bool, len(recordIDs))
	if !reader.person || reader.sight.All || len(recordIDs) == 0 {
		return shared, nil
	}

	ids, err := s.shares.SharedWith(ctx, &recordshareports.SharedWithRequest{
		TenantInfo: tenantInfo,
		UserID:     reader.userID,
		RecordIDs:  recordIDs,
	})
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		shared[id] = true
	}

	return shared, nil
}

// Standings reads the reader once and every owner in one query, so a page of
// cases costs what one case does.
func (s *Service) Standings(
	ctx context.Context,
	req caseownerports.RecordsRequest,
	actor *permissionports.RequestActor,
) (map[pulid.ID]*caseownerports.Standing, error) {
	out := make(map[pulid.ID]*caseownerports.Standing, len(req.RecordIDs))
	if len(req.RecordIDs) == 0 {
		return out, nil
	}

	owners, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, err
	}
	reader, err := s.readerOf(ctx, actor)
	if err != nil {
		return nil, err
	}

	shared, err := s.sharedWith(ctx, req.TenantInfo, reader, req.RecordIDs)
	if err != nil {
		return nil, err
	}

	owned := make(map[pulid.ID]*caseowner.Owner, len(owners))
	for _, owner := range owners {
		owned[owner.RecordID] = owner
	}
	for _, id := range req.RecordIDs {
		out[id] = reader.standing(owned[id], shared[id])
	}

	return out, nil
}

// caseReader is how one actor stands to any case: whose cases they work and
// what they may do with one. It is read once and applied to each owner.
type caseReader struct {
	person    bool
	userID    pulid.ID
	sight     caseowner.Sight
	canAssign bool
	mayWork   bool
}

func (s *Service) readerOf(
	ctx context.Context,
	actor *permissionports.RequestActor,
) (caseReader, error) {
	if !actor.IsUser() {
		return caseReader{}, nil
	}

	sight, err := s.Sight(ctx, actor)
	if err != nil {
		return caseReader{}, err
	}
	canAssign, err := s.allowed(ctx, actor, permission.OpAssign)
	if err != nil {
		return caseReader{}, err
	}
	mayWork, err := s.allowed(ctx, actor, permission.OpUpdate)
	if err != nil {
		return caseReader{}, err
	}

	return caseReader{
		person:    true,
		userID:    actor.UserID,
		sight:     sight,
		canAssign: canAssign,
		mayWork:   mayWork,
	}, nil
}

// standing is how the reader stands to one case. shared says the record was
// shared with them, which lets them see a case they do not work.
func (r caseReader) standing(owner *caseowner.Owner, shared bool) *caseownerports.Standing {
	if !r.person {
		return &caseownerports.Standing{Owner: owner, CanWork: true, CanSee: true}
	}

	ownerID := pulid.Nil
	if owner != nil {
		ownerID = owner.OwnerID
	}

	return &caseownerports.Standing{
		Owner:     owner,
		Covering:  owner != nil && ownerID != r.userID && r.sight.Works(ownerID),
		CanWork:   r.sight.Works(ownerID),
		CanSee:    r.sight.Sees(ownerID) || shared,
		CanClaim:  owner == nil && r.mayWork,
		CanAssign: r.canAssign,
	}
}

func (s *Service) AssertMayWork(
	ctx context.Context,
	req caseownerports.RecordRequest,
	actor *permissionports.RequestActor,
) error {
	if req.RecordID.IsNil() || !actor.IsUser() {
		return nil
	}

	standing, err := s.Standing(ctx, req, actor)
	if err != nil {
		return err
	}
	if !standing.CanWork {
		return errortypes.NewAuthorizationError(
			"This case is owned by someone else. Its owner decides it, or someone who assigns " +
				"cases can hand it to you.",
		)
	}

	return nil
}

func tenantOf(actor *permissionports.RequestActor, userID pulid.ID) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  actor.OrganizationID,
		BuID:   actor.BusinessUnitID,
		UserID: userID,
	}
}
