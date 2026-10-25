package driverportalrepository

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission/permissiondb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxUsernameLength       = 20
	defaultWorkerLoadsLimit = 25
	maxWorkerLoadsLimit     = 100
	workerPTOLimit          = 50

	resolveInvitationScopeReason = "resolve a driver portal invitation link before its tenant is known"
	emailAvailableScopeReason    = "check that a portal sign-in email is not already used by any account"
)

var (
	invitationErrors = dberror.Mapping{Entity: "PortalInvitation"}
	workerErrors     = dberror.Mapping{Entity: "Worker"}
	assignmentErrors = dberror.Mapping{Entity: "Assignment"}
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
	Quota  platformplanports.QuotaGuard `optional:"true"`
}

type portalAccessRepository struct {
	db    *postgres.Connection
	l     *zap.Logger
	quota platformplanports.QuotaGuard
}

func New(p Params) workerports.PortalAccessRepository {
	return &portalAccessRepository{
		db:    p.DB,
		l:     p.Logger.Named("postgres.driver-portal-repository"),
		quota: quotaservice.OrUnlimited(p.Quota),
	}
}

func (r *portalAccessRepository) CreateInvitation(
	ctx context.Context,
	entity *worker.PortalInvitation,
) (*worker.PortalInvitation, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.PortalInvitation, error) {
		if err := workerdb.PortalInvitations.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, fmt.Errorf(
				"create portal invitation: %w",
				invitationErrors.Map(err, entity.ID.String()),
			)
		}
		return entity, nil
	})
}

func (r *portalAccessRepository) UpdateInvitation(
	ctx context.Context,
	entity *worker.PortalInvitation,
) (*worker.PortalInvitation, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.PortalInvitation, error) {
		if err := workerdb.PortalInvitations.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, invitationErrors.Map(err, entity.ID.String())
		}
		return entity, nil
	})
}

func (r *portalAccessRepository) GetInvitationByID(
	ctx context.Context,
	req workerports.GetPortalInvitationByIDRequest,
) (*worker.PortalInvitation, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.PortalInvitation, error) {
		pi := workerdb.PortalInvitations
		entity, err := pi.Where(pi.ID.Eq(req.ID)).Load(pi.Worker()).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, invitationErrors.Map(err, req.ID.String())
		}
		return entity, nil
	})
}

func (r *portalAccessRepository) GetInvitationByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*worker.PortalInvitation, error) {
	ctx = dbscope.WithSystem(ctx, resolveInvitationScopeReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.PortalInvitation, error) {
		pi := workerdb.PortalInvitations
		entity, err := pi.AllTenants().
			Where(pi.TokenHash.Eq(tokenHash)).
			Load(pi.Worker().Organization()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, invitationErrors.Map(err, "")
		}
		return entity, nil
	})
}

func (r *portalAccessRepository) GetPendingInvitationForWorker(
	ctx context.Context,
	req workerports.GetPendingPortalInvitationRequest,
) (*worker.PortalInvitation, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.PortalInvitation, error) {
		pi := workerdb.PortalInvitations
		entity, err := pi.Where(
			pi.WorkerID.Eq(req.WorkerID),
			pi.Status.Eq(worker.PortalInvitationStatusPending),
		).
			Order(pi.CreatedAt.Desc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, invitationErrors.Map(err, req.WorkerID.String())
		}
		return entity, nil
	})
}

func (r *portalAccessRepository) ListInvitations(
	ctx context.Context,
	req *workerports.ListPortalInvitationsRequest,
) ([]*worker.PortalInvitation, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.PortalInvitation, error) {
		pi := workerdb.PortalInvitations
		items, err := pi.Where(keel.When(!req.WorkerID.IsNil(), pi.WorkerID.Eq(req.WorkerID))).
			Load(pi.InvitedBy()).
			Order(pi.CreatedAt.Desc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list portal invitations: %w", err)
		}
		return items, nil
	})
}

func (r *portalAccessRepository) GetWorkerByUserID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*worker.Worker, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.Worker, error) {
		w := workerdb.Workers
		entity, err := w.Where(w.UserID.Eq(tenantInfo.UserID)).
			Load(w.FleetCode(), w.Organization(), w.State(), w.Profile().LicenseState()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, workerErrors.Map(err, tenantInfo.UserID.String())
		}
		return entity, nil
	})
}

func (r *portalAccessRepository) ExistsWorkerForUser(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		w := workerdb.Workers
		exists, err := w.Where(w.UserID.Eq(tenantInfo.UserID)).Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			return false, fmt.Errorf("check worker linkage for user: %w", err)
		}
		return exists, nil
	})
}

func (r *portalAccessRepository) GetWorkerForPortalManagement(
	ctx context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.Worker, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.Worker, error) {
		w := workerdb.Workers
		entity, err := w.Where(w.ID.Eq(workerID)).
			Load(w.Organization(), w.PortalUser()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, workerErrors.Map(err, workerID.String())
		}
		return entity, nil
	})
}

func (r *portalAccessRepository) ActivatePortalAccess(
	ctx context.Context,
	req *workerports.ActivatePortalAccessRequest,
) (*tenant.User, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tenant.User, error) {
		invitation, err := r.lockInvitation(ctx, req.Invitation)
		if err != nil {
			return nil, err
		}

		if err = quotaservice.EnforceAll(ctx, r.quota, platformplanports.QuotaRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: invitation.OrganizationID,
				BuID:  invitation.BusinessUnitID,
			},
			Meter:    platformcatalog.MeterUserSeats,
			Quantity: 1,
		}); err != nil {
			return nil, err
		}

		wrk, err := r.lockWorker(ctx, invitation)
		if err != nil {
			return nil, err
		}

		if err = r.ensureEmailAvailable(ctx, req.User.EmailAddress); err != nil {
			return nil, err
		}

		role, err := r.ensureDriverRole(ctx, invitation, req)
		if err != nil {
			return nil, err
		}

		req.User.Username, err = r.uniqueUsername(ctx, invitation, req.User.Username)
		if err != nil {
			return nil, err
		}

		db := r.db.Keel(ctx)
		if err = tenantdb.Users.Create(ctx, db, req.User); err != nil {
			return nil, fmt.Errorf("create portal user: %w", err)
		}

		if err = tenantdb.OrganizationMemberships.Create(ctx, db, &tenant.OrganizationMembership{
			IsDefault:      true,
			BusinessUnitID: invitation.BusinessUnitID,
			UserID:         req.User.ID,
			OrganizationID: invitation.OrganizationID,
			GrantedByID:    invitation.InvitedByID,
		}); err != nil {
			return nil, fmt.Errorf("create portal user membership: %w", err)
		}

		if err = permissiondb.UserRoleAssignments.Create(ctx, db, &permission.UserRoleAssignment{
			ID:             pulid.MustNew("ura_"),
			UserID:         req.User.ID,
			OrganizationID: invitation.OrganizationID,
			RoleID:         role.ID,
			AssignedBy:     invitation.InvitedByID,
			AssignedAt:     timeutils.NowUnix(),
		}); err != nil {
			return nil, fmt.Errorf("assign driver role: %w", err)
		}

		if err = r.linkWorker(ctx, wrk, req.User.ID); err != nil {
			return nil, err
		}

		pi := workerdb.PortalInvitations
		now := timeutils.NowUnix()
		invitation.Status = worker.PortalInvitationStatusAccepted
		invitation.AcceptedAt = &now
		invitation.AcceptedUserID = &req.User.ID
		if err = pi.Save(
			ctx,
			db,
			invitation,
			pi.Status,
			pi.AcceptedAt,
			pi.AcceptedUserID,
		); err != nil {
			return nil, fmt.Errorf(
				"mark invitation accepted: %w",
				invitationErrors.Map(err, invitation.ID.String()),
			)
		}

		return req.User, nil
	})
}

func (r *portalAccessRepository) lockInvitation(
	ctx context.Context,
	ref *worker.PortalInvitation,
) (*worker.PortalInvitation, error) {
	pi := workerdb.PortalInvitations
	invitation, err := pi.Where(pi.ID.Eq(ref.ID)).
		Lock(keel.ForUpdate).
		One(ctx, r.db.Keel(ctx))
	if err != nil {
		return nil, invitationErrors.Map(err, ref.ID.String())
	}
	if !invitation.IsAcceptable(timeutils.NowUnix()) {
		return nil, errortypes.NewValidationError(
			"token",
			errortypes.ErrInvalid,
			"This invitation is no longer valid. Ask your carrier to send a new one.",
		)
	}
	return invitation, nil
}

func (r *portalAccessRepository) lockWorker(
	ctx context.Context,
	invitation *worker.PortalInvitation,
) (*worker.Worker, error) {
	w := workerdb.Workers
	wrk, err := w.Where(w.ID.Eq(invitation.WorkerID)).
		Lock(keel.ForUpdate).
		One(ctx, r.db.Keel(ctx))
	if err != nil {
		return nil, workerErrors.Map(err, invitation.WorkerID.String())
	}
	if !wrk.UserID.IsNil() {
		return nil, errortypes.NewValidationError(
			"token",
			errortypes.ErrInvalid,
			"This driver already has portal access.",
		)
	}
	return wrk, nil
}

func (r *portalAccessRepository) ensureEmailAvailable(ctx context.Context, email string) error {
	ctx = dbscope.WithSystem(ctx, emailAvailableScopeReason)
	exists, err := dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		u := tenantdb.Users
		return u.AllTenants().
			Where(keel.DynFunc("lower", u.EmailAddress).Eq(keel.DynFunc("lower", email))).
			Exists(ctx, r.db.Keel(ctx))
	})
	if err != nil {
		return fmt.Errorf("check portal user email: %w", err)
	}
	if exists {
		return errortypes.NewValidationError(
			"email",
			errortypes.ErrInvalid,
			"An account with this email address already exists. Contact your carrier to resolve this.",
		)
	}
	return nil
}

func (r *portalAccessRepository) ensureDriverRole(
	ctx context.Context,
	invitation *worker.PortalInvitation,
	req *workerports.ActivatePortalAccessRequest,
) (*permission.Role, error) {
	ro := permissiondb.Roles
	db := r.db.Keel(ctx)
	role, err := ro.Where(ro.Name.Eq(req.RoleName), ro.IsSystem.IsTrue()).First(ctx, db)
	if err == nil {
		return role, nil
	}
	if !dberror.IsNotFoundError(err) {
		return nil, fmt.Errorf("find driver role: %w", err)
	}

	role = &permission.Role{
		ID:             pulid.MustNew("rol_"),
		BusinessUnitID: invitation.BusinessUnitID,
		OrganizationID: invitation.OrganizationID,
		Name:           req.RoleName,
		Description:    req.RoleDescription,
		MaxSensitivity: permission.SensitivityInternal,
		IsSystem:       true,
		CreatedBy:      invitation.InvitedByID,
	}
	if err = ro.Create(ctx, db, role); err != nil {
		return nil, fmt.Errorf("create driver role: %w", err)
	}
	return role, nil
}

func (r *portalAccessRepository) uniqueUsername(
	ctx context.Context,
	invitation *worker.PortalInvitation,
	base string,
) (string, error) {
	u := tenantdb.Users
	db := r.db.Keel(ctx)
	candidate := base
	for suffix := 2; ; suffix++ {
		exists, err := u.Where(
			u.CurrentOrganizationID.Eq(invitation.OrganizationID),
			u.Username.Eq(candidate),
		).Exists(ctx, db)
		if err != nil {
			return "", fmt.Errorf("check portal username: %w", err)
		}
		if !exists {
			return candidate, nil
		}
		tag := strconv.Itoa(suffix)
		trimmed := base
		if len(trimmed)+len(tag) > maxUsernameLength {
			trimmed = strings.TrimRight(trimmed[:maxUsernameLength-len(tag)], "-_.")
		}
		candidate = trimmed + tag
	}
}

func (r *portalAccessRepository) linkWorker(
	ctx context.Context,
	wrk *worker.Worker,
	userID pulid.ID,
) error {
	w := workerdb.Workers
	affected, err := w.Where(w.ID.Eq(wrk.ID), w.UserID.IsNull()).
		Update(ctx, r.db.Keel(ctx), w.UserID.To(userID))
	if err != nil {
		return fmt.Errorf("link worker to portal user: %w", err)
	}
	return dberror.CheckAffected(affected, "Worker", wrk.ID.String())
}

func (r *portalAccessRepository) RevokePortalAccess(
	ctx context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		w := workerdb.Workers
		wrk, err := w.Where(w.ID.Eq(workerID)).Lock(keel.ForUpdate).One(ctx, db)
		if err != nil {
			return workerErrors.Map(err, workerID.String())
		}

		pi := workerdb.PortalInvitations
		if _, err = pi.Where(
			pi.WorkerID.Eq(workerID),
			pi.Status.Eq(worker.PortalInvitationStatusPending),
		).Update(ctx, db, pi.Status.To(worker.PortalInvitationStatusRevoked)); err != nil {
			return fmt.Errorf("revoke pending invitations: %w", err)
		}

		if wrk.UserID.IsNil() {
			return nil
		}

		u := tenantdb.Users
		if _, err = u.Where(u.ID.Eq(wrk.UserID)).
			Update(ctx, db, u.Status.To(domaintypes.StatusInactive)); err != nil {
			return fmt.Errorf("deactivate portal user: %w", err)
		}

		if _, err = w.Where(w.ID.Eq(workerID)).Update(ctx, db, w.UserID.ToNull()); err != nil {
			return fmt.Errorf("unlink worker portal user: %w", err)
		}
		return nil
	})
}

func assignedTo(workerID pulid.ID) keel.Pred {
	a := shipmentdb.Assignments
	return keel.Or(a.PrimaryWorkerID.Eq(workerID), a.SecondaryWorkerID.Eq(workerID))
}

func (r *portalAccessRepository) WorkerAssignedToShipment(
	ctx context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
	shipmentID pulid.ID,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		a := shipmentdb.Assignments
		exists, err := a.Where(
			a.ShipmentMove().ShipmentID.Eq(shipmentID),
			assignedTo(workerID),
		).Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			return false, fmt.Errorf("check worker shipment assignment: %w", err)
		}
		return exists, nil
	})
}

func (r *portalAccessRepository) ListWorkerPTO(
	ctx context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
) ([]*worker.WorkerPTO, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.WorkerPTO, error) {
		pto := workerdb.WorkerPTOs
		items, err := pto.Where(pto.WorkerID.Eq(workerID)).
			Order(pto.StartDate.Desc()).
			Limit(workerPTOLimit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list worker pto: %w", err)
		}
		return items, nil
	})
}

func (r *portalAccessRepository) UpdateAssignmentAck(
	ctx context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
	assignmentID pulid.ID,
	ack shipment.AssignmentAck,
	reason string,
) (*shipment.Assignment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.Assignment, error) {
		a := shipmentdb.Assignments
		db := r.db.Keel(ctx)
		entity, err := a.Where(
			a.ID.Eq(assignmentID),
			a.ArchivedAt.IsNull(),
			assignedTo(workerID),
		).One(ctx, db)
		if err != nil {
			return nil, assignmentErrors.Map(err, assignmentID.String())
		}

		now := timeutils.NowUnix()
		entity.AckStatus = ack
		entity.AckAt = &now
		entity.AckReason = reason

		if err = a.Save(ctx, db, entity, a.AckStatus, a.AckAt, a.AckReason); err != nil {
			return nil, assignmentErrors.Map(err, entity.ID.String())
		}
		return entity, nil
	})
}

func (r *portalAccessRepository) WorkerAssignedToMove(
	ctx context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
	moveID pulid.ID,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		a := shipmentdb.Assignments
		exists, err := a.Where(
			a.ShipmentMoveID.Eq(moveID),
			a.ArchivedAt.IsNull(),
			assignedTo(workerID),
		).Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			return false, fmt.Errorf("check worker move assignment: %w", err)
		}
		return exists, nil
	})
}

func (r *portalAccessRepository) ListDriverShipmentComments(
	ctx context.Context,
	_ pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*shipment.ShipmentComment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*shipment.ShipmentComment, error) {
		sc := shipmentdb.ShipmentComments
		items, err := sc.Where(
			sc.ShipmentID.Eq(shipmentID),
			sc.Visibility.Eq(shipment.CommentVisibilityDriver),
			sc.DeletedAt.IsNull(),
		).
			Load(sc.User()).
			Order(sc.CreatedAt.Desc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list driver shipment comments: %w", err)
		}
		return items, nil
	})
}

func (r *portalAccessRepository) ListWorkerLoads(
	ctx context.Context,
	req *workerports.ListWorkerLoadsRequest,
) ([]*shipment.Assignment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*shipment.Assignment, error) {
		limit := req.Limit
		if limit <= 0 || limit > maxWorkerLoadsLimit {
			limit = defaultWorkerLoadsLimit
		}

		a := shipmentdb.Assignments
		move := a.ShipmentMove()
		stops := move.Stops()
		items, err := a.Where(
			assignedTo(req.WorkerID),
			a.ArchivedAt.IsNull(),
			keel.When(len(req.Statuses) > 0, move.Status.In(req.Statuses...)),
		).
			Load(
				move.Shipment(),
				stops.Order(stops.Sequence.Asc()).Load(stops.Location()),
				a.Tractor(),
				a.Trailer(),
			).
			Order(a.CreatedAt.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list worker loads: %w", err)
		}
		return items, nil
	})
}
