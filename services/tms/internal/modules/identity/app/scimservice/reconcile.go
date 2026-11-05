package scimservice

import (
	"context"
	"errors"
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/app/roleservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/domain/scim"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	reconcileBatchSize   = 100
	pendingDirectoryPage = 500
)

func (s *Service) RequestPendingReconciles(ctx context.Context) (int, error) {
	directories, err := s.repo.ListPendingDirectories(ctx, pendingDirectoryPage)
	if err != nil {
		return 0, err
	}

	for _, directory := range directories {
		s.requester.RequestReconcile(ctx, pagination.TenantInfo{
			OrgID: directory.OrganizationID,
			BuID:  directory.BusinessUnitID,
		}, directory.DirectoryID)
	}

	return len(directories), nil
}

func (s *Service) ReconcilePending(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	directoryID pulid.ID,
) (*scimports.ReconcileResult, error) {
	dir := &scimports.Directory{
		ID:             directoryID,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}
	ctx = scopedTo(ctx, dir)

	links, err := s.repo.ListPending(ctx, directoryID, reconcileBatchSize+1)
	if err != nil {
		return nil, err
	}

	result := new(scimports.ReconcileResult)
	more := len(links) > reconcileBatchSize
	if more {
		links = links[:reconcileBatchSize]
	}

	grants := newAuthority(s, dir)
	var errs []error
	for _, link := range links {
		if err = s.reconcileUser(ctx, dir, link, grants); err != nil {
			result.Failed++
			errs = append(errs, err)
			s.l.Error("could not reconcile the roles a directory grants",
				zap.String("directoryId", directoryID.String()),
				zap.String("userId", link.UserID.String()),
				zap.Error(err))
			continue
		}
		result.Reconciled++
	}

	result.More = more && result.Reconciled > 0
	if result.Reconciled == 0 && len(errs) > 0 {
		return result, errors.Join(errs...)
	}

	return result, nil
}

func (s *Service) reconcileUser(
	ctx context.Context,
	dir *scimports.Directory,
	link *scim.User,
	grants *authority,
) error {
	authorized, err := s.authorizedRoles(ctx, dir, link, grants)
	if err != nil {
		return err
	}

	held, err := s.repo.ListGrants(ctx, dir.ID, link.UserID)
	if err != nil {
		return err
	}

	granted := make(map[pulid.ID]struct{}, len(held))
	for _, grant := range held {
		granted[grant.RoleID] = struct{}{}
		if _, keep := authorized[grant.RoleID]; keep {
			continue
		}
		if err = s.revokeGrant(ctx, dir, grant); err != nil {
			return err
		}
	}

	roleIDs := make([]pulid.ID, 0, len(authorized))
	for roleID := range authorized {
		if _, done := granted[roleID]; !done {
			roleIDs = append(roleIDs, roleID)
		}
	}
	slices.Sort(roleIDs)
	for _, roleID := range roleIDs {
		if err = s.grantRole(ctx, dir, link, roleID, authorized[roleID]); err != nil {
			return err
		}
	}

	return s.repo.CompleteReconcile(ctx, link)
}

func (s *Service) authorizedRoles(
	ctx context.Context,
	dir *scimports.Directory,
	link *scim.User,
	grants *authority,
) (map[pulid.ID]pulid.ID, error) {
	authorized := make(map[pulid.ID]pulid.ID)
	if !link.Active {
		return authorized, nil
	}

	member, err := s.users.GetTenantMember(ctx, tenantports.GetTenantMemberRequest{
		UserID:     link.UserID,
		TenantInfo: dir.TenantInfo(),
		Now:        timeutils.NowUnix(),
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return authorized, nil
		}
		return nil, err
	}
	if member.ValidateStatus() != nil {
		return authorized, nil
	}

	sources, err := s.repo.RoleSources(ctx, dir.ID, link.UserID)
	if err != nil {
		return nil, err
	}

	refused := make(map[pulid.ID]*scim.RoleSource)
	for _, source := range sources {
		if _, done := authorized[source.RoleID]; done {
			continue
		}
		allowed, allowErr := grants.allows(ctx, source.SavedByID, source.RoleID)
		if allowErr != nil {
			return nil, allowErr
		}
		if allowed {
			authorized[source.RoleID] = source.SavedByID
			delete(refused, source.RoleID)
			continue
		}
		if _, noted := refused[source.RoleID]; !noted {
			refused[source.RoleID] = source
		}
	}

	for roleID, source := range refused {
		if _, ok := authorized[roleID]; ok {
			continue
		}
		s.record(ctx, dir, &event{
			action:       iam.ProvisioningActionUpdate,
			resourceType: resourceTypeRole,
			resourceID:   link.UserID,
			externalID:   source.DisplayName,
			status:       statusRefused,
			message: "The group mapping for role " + roleID.String() +
				" was saved by someone who can no longer grant it",
		})
	}

	return authorized, nil
}

func (s *Service) grantRole(
	ctx context.Context,
	dir *scimports.Directory,
	link *scim.User,
	roleID, grantorID pulid.ID,
) error {
	assignment := &rbac.UserRoleAssignment{UserID: link.UserID, RoleID: roleID}
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		if assignErr := s.roles.AssignRole(
			authctx.WithoutSessionRoleActivation(txCtx),
			roleservice.AssignRoleRequest{
				ActorID:        grantorID,
				OrganizationID: dir.OrganizationID,
				Assignment:     assignment,
			},
		); assignErr != nil {
			return assignErr
		}
		return s.repo.CreateGrant(txCtx, &scim.RoleGrant{
			OrganizationID: dir.OrganizationID,
			BusinessUnitID: dir.BusinessUnitID,
			DirectoryID:    dir.ID,
			UserID:         link.UserID,
			RoleID:         roleID,
			AssignmentID:   assignment.ID,
			GrantedByID:    grantorID,
		})
	})

	e := &event{
		action:       iam.ProvisioningActionUpdate,
		resourceType: resourceTypeRole,
		resourceID:   link.UserID,
		externalID:   roleID.String(),
	}
	switch {
	case err == nil:
		e.status = statusSucceeded
		e.comment = "Role granted from a SCIM group"
		e.metadata = map[string]any{"roleId": roleID.String(), "grantedById": grantorID.String()}
		s.record(ctx, dir, e)
		return nil
	case errors.Is(err, roleservice.ErrRoleAlreadyAssigned):
		return nil
	case errortypes.IsRefusal(err) || errortypes.IsAuthorizationError(err):
		s.recordFailure(ctx, dir, e, err)
		return nil
	default:
		return err
	}
}

func (s *Service) revokeGrant(
	ctx context.Context,
	dir *scimports.Directory,
	grant *scim.RoleGrant,
) error {
	err := s.roles.UnassignRole(ctx, roleservice.UnassignRoleRequest{
		OrganizationID: dir.OrganizationID,
		BusinessUnitID: dir.BusinessUnitID,
		AssignmentID:   grant.AssignmentID,
	})
	if err != nil && !errortypes.IsNotFoundError(err) {
		return err
	}

	s.record(ctx, dir, &event{
		action:       iam.ProvisioningActionUpdate,
		resourceType: resourceTypeRole,
		resourceID:   grant.UserID,
		externalID:   grant.RoleID.String(),
		status:       statusSucceeded,
		comment:      "Role a SCIM group granted was taken away",
		metadata:     map[string]any{"roleId": grant.RoleID.String()},
	})

	return nil
}

func (s *Service) revokeUserGrants(
	ctx context.Context,
	dir *scimports.Directory,
	userID pulid.ID,
) error {
	grants, err := s.repo.ListGrants(ctx, dir.ID, userID)
	if err != nil {
		return err
	}
	for _, grant := range grants {
		if err = s.revokeGrant(ctx, dir, grant); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) MappingsChanged(ctx context.Context, tenantInfo pagination.TenantInfo) error {
	ctx = dbscope.EnsureTenant(ctx, tenantInfo.DBTenant())
	directories, err := s.repo.MarkOrganizationPending(ctx)
	if err != nil {
		return err
	}

	kernelports.AfterCommit(ctx, func(committed context.Context) {
		for _, directoryID := range directories {
			s.requester.RequestReconcile(committed, tenantInfo, directoryID)
		}
	})

	return nil
}

func (s *Service) RevokeDirectoryGrants(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	directoryID pulid.ID,
) error {
	ctx = dbscope.EnsureTenant(ctx, tenantInfo.DBTenant())
	assignments, err := s.repo.DeleteDirectoryAssignments(ctx, directoryID)
	if err != nil {
		return err
	}
	if len(assignments) == 0 {
		return nil
	}

	kernelports.AfterCommit(ctx, func(committed context.Context) {
		var actor kernelports.AuditActor
		if tenantInfo.UserID.IsNotNil() {
			actor = permissionports.UserActor(tenantInfo).AuditActor()
		}
		for _, assignment := range assignments {
			s.auditor.RecordChange(committed, &permissionports.SecurityChange{
				Resource:       permission.ResourceRole,
				ResourceID:     assignment.RoleID.String(),
				Operation:      permission.OpUnassign,
				Actor:          actor,
				OrganizationID: tenantInfo.OrgID,
				BusinessUnitID: tenantInfo.BuID,
				Before:         assignment,
				Comment:        "Role unassigned because its SCIM directory was deleted",
				Metadata: map[string]any{
					"assignmentId":    assignment.ID.String(),
					"userId":          assignment.UserID.String(),
					"scimDirectoryId": directoryID.String(),
				},
			})
		}
		s.notifier.OrganizationChanged(committed, tenantInfo)
	})

	return nil
}

type authority struct {
	s        *Service
	dir      *scimports.Directory
	standing map[pulid.ID]bool
	verdicts map[[2]pulid.ID]bool
}

func newAuthority(s *Service, dir *scimports.Directory) *authority {
	return &authority{
		s:        s,
		dir:      dir,
		standing: make(map[pulid.ID]bool),
		verdicts: make(map[[2]pulid.ID]bool),
	}
}

func (a *authority) allows(ctx context.Context, grantorID, roleID pulid.ID) (bool, error) {
	key := [2]pulid.ID{grantorID, roleID}
	if verdict, ok := a.verdicts[key]; ok {
		return verdict, nil
	}

	good, err := a.inGoodStanding(ctx, grantorID)
	if err != nil {
		return false, err
	}
	if !good {
		a.verdicts[key] = false
		return false, nil
	}

	tenantInfo := a.dir.TenantInfo()
	tenantInfo.UserID = grantorID
	checkCtx := authctx.WithoutSessionRoleActivation(
		dbscope.WithTenant(ctx, tenantInfo.DBTenant()),
	)
	err = a.s.roles.RequireCanGrantRole(checkCtx, tenantInfo, roleID)
	switch {
	case err == nil:
		a.verdicts[key] = true
		return true, nil
	case errortypes.IsRefusal(err) || errortypes.IsAuthorizationError(err):
		a.verdicts[key] = false
		return false, nil
	default:
		return false, err
	}
}

func (a *authority) inGoodStanding(ctx context.Context, userID pulid.ID) (bool, error) {
	if good, ok := a.standing[userID]; ok {
		return good, nil
	}

	member, err := a.s.users.GetTenantMember(ctx, tenantports.GetTenantMemberRequest{
		UserID:     userID,
		TenantInfo: a.dir.TenantInfo(),
		Now:        timeutils.NowUnix(),
	})
	switch {
	case err == nil:
		good := member.ValidateStatus() == nil
		a.standing[userID] = good
		return good, nil
	case errortypes.IsNotFoundError(err):
		a.standing[userID] = false
		return false, nil
	default:
		return false, err
	}
}
