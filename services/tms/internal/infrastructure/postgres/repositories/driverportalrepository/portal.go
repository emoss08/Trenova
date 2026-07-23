package driverportalrepository

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxUsernameLength = 20

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type portalAccessRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.PortalAccessRepository {
	return &portalAccessRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.driver-portal-repository"),
	}
}

func (r *portalAccessRepository) CreateInvitation(
	ctx context.Context,
	entity *worker.PortalInvitation,
) (*worker.PortalInvitation, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create portal invitation: %w", err)
	}
	return entity, nil
}

func (r *portalAccessRepository) UpdateInvitation(
	ctx context.Context,
	entity *worker.PortalInvitation,
) (*worker.PortalInvitation, error) {
	cols := buncolgen.PortalInvitationColumns
	ov := entity.Version
	entity.Version++
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		entity.Version = ov
		return nil, fmt.Errorf("update portal invitation: %w", err)
	}
	err = dberror.CheckRowsAffected(results, "PortalInvitation", entity.ID.String())
	if err != nil {
		entity.Version = ov
		return nil, err
	}
	return entity, nil
}

func (r *portalAccessRepository) GetInvitationByID(
	ctx context.Context,
	req repositories.GetPortalInvitationByIDRequest,
) (*worker.PortalInvitation, error) {
	cols := buncolgen.PortalInvitationColumns
	entity := new(worker.PortalInvitation)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.PortalInvitationScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Relation(buncolgen.PortalInvitationRelations.Worker).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "PortalInvitation")
	}
	return entity, nil
}

func (r *portalAccessRepository) GetInvitationByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*worker.PortalInvitation, error) {
	cols := buncolgen.PortalInvitationColumns
	rel := buncolgen.PortalInvitationRelations
	entity := new(worker.PortalInvitation)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.TokenHash.Eq(), tokenHash).
		Relation(rel.Worker).
		Relation(buncolgen.Rel(rel.Worker, buncolgen.WorkerRelations.Organization)).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "PortalInvitation")
	}
	return entity, nil
}

func (r *portalAccessRepository) GetPendingInvitationForWorker(
	ctx context.Context,
	req repositories.GetPendingPortalInvitationRequest,
) (*worker.PortalInvitation, error) {
	cols := buncolgen.PortalInvitationColumns
	entity := new(worker.PortalInvitation)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.PortalInvitationScopeTenant(sq, req.TenantInfo).
				Where(cols.WorkerID.Eq(), req.WorkerID).
				Where(cols.Status.Eq(), worker.PortalInvitationStatusPending)
		}).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "PortalInvitation")
	}
	return entity, nil
}

func (r *portalAccessRepository) ListInvitations(
	ctx context.Context,
	req *repositories.ListPortalInvitationsRequest,
) ([]*worker.PortalInvitation, error) {
	cols := buncolgen.PortalInvitationColumns
	items := make([]*worker.PortalInvitation, 0, 8)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.PortalInvitationScopeTenant(sq, req.TenantInfo)
			if !req.WorkerID.IsNil() {
				sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
			}
			return sq
		}).
		Relation(buncolgen.PortalInvitationRelations.InvitedBy).
		Order(cols.CreatedAt.OrderDesc())
	if err := query.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list portal invitations: %w", err)
	}
	return items, nil
}

func (r *portalAccessRepository) GetWorkerByUserID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*worker.Worker, error) {
	cols := buncolgen.WorkerColumns
	rel := buncolgen.WorkerRelations
	entity := new(worker.Worker)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerScopeTenant(sq, tenantInfo).
				Where(cols.UserID.Eq(), tenantInfo.UserID)
		}).
		Relation(rel.FleetCode).
		Relation(rel.Organization).
		Relation(rel.State).
		Relation(rel.Profile).
		Relation(
			buncolgen.Rel(rel.Profile, buncolgen.WorkerProfileRelations.LicenseState),
		).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Worker")
	}
	return entity, nil
}

func (r *portalAccessRepository) GetWorkerForPortalManagement(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.Worker, error) {
	cols := buncolgen.WorkerColumns
	rel := buncolgen.WorkerRelations
	entity := new(worker.Worker)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerScopeTenant(sq, tenantInfo).
				Where(cols.ID.Eq(), workerID)
		}).
		Relation(rel.Organization).
		Relation(rel.PortalUser).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Worker")
	}
	return entity, nil
}

func (r *portalAccessRepository) ActivatePortalAccess(
	ctx context.Context,
	req *repositories.ActivatePortalAccessRequest,
) (*tenant.User, error) {
	var created *tenant.User
	err := r.db.DB().RunInTx(ctx, nil, func(txCtx context.Context, tx bun.Tx) error {
		invitation, err := r.lockInvitation(txCtx, tx, req.Invitation)
		if err != nil {
			return err
		}

		wrk, err := r.lockWorker(txCtx, tx, invitation)
		if err != nil {
			return err
		}

		if err = r.ensureEmailAvailable(txCtx, tx, req.User.EmailAddress); err != nil {
			return err
		}

		role, err := r.ensureDriverRole(txCtx, tx, invitation, req)
		if err != nil {
			return err
		}

		req.User.Username, err = r.uniqueUsername(txCtx, tx, invitation, req.User.Username)
		if err != nil {
			return err
		}

		if _, err = tx.NewInsert().Model(req.User).Exec(txCtx); err != nil {
			return fmt.Errorf("create portal user: %w", err)
		}

		membership := &tenant.OrganizationMembership{
			IsDefault:      true,
			BusinessUnitID: invitation.BusinessUnitID,
			UserID:         req.User.ID,
			OrganizationID: invitation.OrganizationID,
			GrantedByID:    invitation.InvitedByID,
		}
		if _, err = tx.NewInsert().Model(membership).Exec(txCtx); err != nil {
			return fmt.Errorf("create portal user membership: %w", err)
		}

		assignment := &permission.UserRoleAssignment{
			ID:             pulid.MustNew("ura_"),
			UserID:         req.User.ID,
			OrganizationID: invitation.OrganizationID,
			RoleID:         role.ID,
			AssignedBy:     invitation.InvitedByID,
			AssignedAt:     timeutils.NowUnix(),
		}
		if _, err = tx.NewInsert().Model(assignment).Exec(txCtx); err != nil {
			return fmt.Errorf("assign driver role: %w", err)
		}

		if err = r.linkWorker(txCtx, tx, wrk, req.User.ID); err != nil {
			return err
		}

		now := timeutils.NowUnix()
		invitation.Status = worker.PortalInvitationStatusAccepted
		invitation.AcceptedAt = &now
		invitation.AcceptedUserID = &req.User.ID
		if _, err = tx.NewUpdate().
			Model(invitation).
			WherePK().
			Column(
				buncolgen.PortalInvitationColumns.Status.Bare(),
				buncolgen.PortalInvitationColumns.AcceptedAt.Bare(),
				buncolgen.PortalInvitationColumns.AcceptedUserID.Bare(),
				buncolgen.PortalInvitationColumns.UpdatedAt.Bare(),
			).
			Exec(txCtx); err != nil {
			return fmt.Errorf("mark invitation accepted: %w", err)
		}

		created = req.User
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (r *portalAccessRepository) lockInvitation(
	ctx context.Context,
	tx bun.Tx,
	ref *worker.PortalInvitation,
) (*worker.PortalInvitation, error) {
	cols := buncolgen.PortalInvitationColumns
	invitation := new(worker.PortalInvitation)
	err := tx.NewSelect().
		Model(invitation).
		Where(cols.ID.Eq(), ref.ID).
		Where(cols.OrganizationID.Eq(), ref.OrganizationID).
		Where(cols.BusinessUnitID.Eq(), ref.BusinessUnitID).
		For("UPDATE").
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "PortalInvitation")
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
	tx bun.Tx,
	invitation *worker.PortalInvitation,
) (*worker.Worker, error) {
	cols := buncolgen.WorkerColumns
	wrk := new(worker.Worker)
	err := tx.NewSelect().
		Model(wrk).
		Where(cols.ID.Eq(), invitation.WorkerID).
		Where(cols.OrganizationID.Eq(), invitation.OrganizationID).
		Where(cols.BusinessUnitID.Eq(), invitation.BusinessUnitID).
		For("UPDATE").
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Worker")
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

func (r *portalAccessRepository) ensureEmailAvailable(
	ctx context.Context,
	tx bun.Tx,
	email string,
) error {
	cols := buncolgen.UserColumns
	exists, err := tx.NewSelect().
		Model((*tenant.User)(nil)).
		Where(cols.EmailAddress.Expr("LOWER({}) = LOWER(?)"), email).
		Exists(ctx)
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
	tx bun.Tx,
	invitation *worker.PortalInvitation,
	req *repositories.ActivatePortalAccessRequest,
) (*permission.Role, error) {
	cols := buncolgen.RoleColumns
	role := new(permission.Role)
	err := tx.NewSelect().
		Model(role).
		Where(cols.OrganizationID.Eq(), invitation.OrganizationID).
		Where(cols.BusinessUnitID.Eq(), invitation.BusinessUnitID).
		Where(cols.Name.Eq(), req.RoleName).
		Where(cols.IsSystem.IsTrue()).
		Limit(1).
		Scan(ctx)
	if err == nil {
		return role, nil
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
	if _, err = tx.NewInsert().Model(role).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create driver role: %w", err)
	}
	return role, nil
}

func (r *portalAccessRepository) uniqueUsername(
	ctx context.Context,
	tx bun.Tx,
	invitation *worker.PortalInvitation,
	base string,
) (string, error) {
	cols := buncolgen.UserColumns
	candidate := base
	for suffix := 2; ; suffix++ {
		exists, err := tx.NewSelect().
			Model((*tenant.User)(nil)).
			Where(cols.BusinessUnitID.Eq(), invitation.BusinessUnitID).
			Where(cols.CurrentOrganizationID.Eq(), invitation.OrganizationID).
			Where(cols.Username.Eq(), candidate).
			Exists(ctx)
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
	tx bun.Tx,
	wrk *worker.Worker,
	userID pulid.ID,
) error {
	cols := buncolgen.WorkerColumns
	results, err := tx.NewUpdate().
		Model((*worker.Worker)(nil)).
		Where(cols.ID.Eq(), wrk.ID).
		Where(cols.OrganizationID.Eq(), wrk.OrganizationID).
		Where(cols.BusinessUnitID.Eq(), wrk.BusinessUnitID).
		Where(cols.UserID.IsNull()).
		Set(cols.UserID.Set(), userID).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("link worker to portal user: %w", err)
	}
	return dberror.CheckRowsAffected(results, "Worker", wrk.ID.String())
}

func (r *portalAccessRepository) RevokePortalAccess(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) error {
	return r.db.DB().RunInTx(ctx, nil, func(txCtx context.Context, tx bun.Tx) error {
		wcols := buncolgen.WorkerColumns
		wrk := new(worker.Worker)
		err := tx.NewSelect().
			Model(wrk).
			Where(wcols.ID.Eq(), workerID).
			Where(wcols.OrganizationID.Eq(), tenantInfo.OrgID).
			Where(wcols.BusinessUnitID.Eq(), tenantInfo.BuID).
			For("UPDATE").
			Scan(txCtx)
		if err != nil {
			return dberror.HandleNotFoundError(err, "Worker")
		}

		now := timeutils.NowUnix()
		icols := buncolgen.PortalInvitationColumns
		if _, err = tx.NewUpdate().
			Model((*worker.PortalInvitation)(nil)).
			Where(icols.WorkerID.Eq(), workerID).
			Where(icols.OrganizationID.Eq(), tenantInfo.OrgID).
			Where(icols.BusinessUnitID.Eq(), tenantInfo.BuID).
			Where(icols.Status.Eq(), worker.PortalInvitationStatusPending).
			Set(icols.Status.Set(), worker.PortalInvitationStatusRevoked).
			Set(icols.UpdatedAt.Set(), now).
			Exec(txCtx); err != nil {
			return fmt.Errorf("revoke pending invitations: %w", err)
		}

		if wrk.UserID.IsNil() {
			return nil
		}

		ucols := buncolgen.UserColumns
		if _, err = tx.NewUpdate().
			Model((*tenant.User)(nil)).
			Where(ucols.ID.Eq(), wrk.UserID).
			Set(ucols.Status.Set(), domaintypes.StatusInactive).
			Set(ucols.UpdatedAt.Set(), now).
			Exec(txCtx); err != nil {
			return fmt.Errorf("deactivate portal user: %w", err)
		}

		if _, err = tx.NewUpdate().
			Model((*worker.Worker)(nil)).
			Where(wcols.ID.Eq(), workerID).
			Where(wcols.OrganizationID.Eq(), tenantInfo.OrgID).
			Where(wcols.BusinessUnitID.Eq(), tenantInfo.BuID).
			Set(wcols.UserID.SetNull()).
			Set(wcols.UpdatedAt.Set(), now).
			Exec(txCtx); err != nil {
			return fmt.Errorf("unlink worker portal user: %w", err)
		}
		return nil
	})
}

func (r *portalAccessRepository) WorkerAssignedToShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	shipmentID pulid.ID,
) (bool, error) {
	acols := buncolgen.AssignmentColumns
	mcols := buncolgen.ShipmentMoveColumns
	exists, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*shipment.Assignment)(nil)).
		Join("JOIN shipment_moves AS sm ON "+mcols.ID.Qualified()+" = "+acols.ShipmentMoveID.Qualified()).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AssignmentScopeTenant(sq, tenantInfo).
				Where(mcols.ShipmentID.Eq(), shipmentID).
				WhereGroup(" AND ", func(inner *bun.SelectQuery) *bun.SelectQuery {
					return inner.
						Where(acols.PrimaryWorkerID.Eq(), workerID).
						WhereOr(acols.SecondaryWorkerID.Eq(), workerID)
				})
		}).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("check worker shipment assignment: %w", err)
	}
	return exists, nil
}

func (r *portalAccessRepository) ListWorkerPTO(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) ([]*worker.WorkerPTO, error) {
	cols := buncolgen.WorkerPTOColumns
	items := make([]*worker.WorkerPTO, 0, 16)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerPTOScopeTenant(sq, tenantInfo).
				Where(cols.WorkerID.Eq(), workerID)
		}).
		Order(cols.StartDate.OrderDesc()).
		Limit(50).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list worker pto: %w", err)
	}
	return items, nil
}

func (r *portalAccessRepository) UpdateAssignmentAck(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	assignmentID pulid.ID,
	ack shipment.AssignmentAck,
	reason string,
) (*shipment.Assignment, error) {
	acols := buncolgen.AssignmentColumns
	entity := new(shipment.Assignment)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AssignmentScopeTenant(sq, tenantInfo).
				Where(acols.ID.Eq(), assignmentID).
				Where(acols.ArchivedAt.IsNull()).
				WhereGroup(" AND ", func(inner *bun.SelectQuery) *bun.SelectQuery {
					return inner.
						Where(acols.PrimaryWorkerID.Eq(), workerID).
						WhereOr(acols.SecondaryWorkerID.Eq(), workerID)
				})
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Assignment")
	}

	now := timeutils.NowUnix()
	entity.AckStatus = ack
	entity.AckAt = &now
	entity.AckReason = reason

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Column(
			acols.AckStatus.Bare(),
			acols.AckAt.Bare(),
			acols.AckReason.Bare(),
			acols.UpdatedAt.Bare(),
		).
		WherePK().
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update assignment acknowledgment: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "Assignment", entity.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *portalAccessRepository) WorkerAssignedToMove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	moveID pulid.ID,
) (bool, error) {
	acols := buncolgen.AssignmentColumns
	exists, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*shipment.Assignment)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AssignmentScopeTenant(sq, tenantInfo).
				Where(acols.ShipmentMoveID.Eq(), moveID).
				Where(acols.ArchivedAt.IsNull()).
				WhereGroup(" AND ", func(inner *bun.SelectQuery) *bun.SelectQuery {
					return inner.
						Where(acols.PrimaryWorkerID.Eq(), workerID).
						WhereOr(acols.SecondaryWorkerID.Eq(), workerID)
				})
		}).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("check worker move assignment: %w", err)
	}
	return exists, nil
}

func (r *portalAccessRepository) ListDriverShipmentComments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*shipment.ShipmentComment, error) {
	cols := buncolgen.ShipmentCommentColumns
	items := make([]*shipment.ShipmentComment, 0, 8)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ShipmentCommentScopeTenant(sq, tenantInfo).
				Where(cols.ShipmentID.Eq(), shipmentID).
				Where(cols.Visibility.Eq(), shipment.CommentVisibilityDriver)
		}).
		Relation(buncolgen.ShipmentCommentRelations.User).
		Order(cols.CreatedAt.OrderDesc()).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list driver shipment comments: %w", err)
	}
	return items, nil
}

func (r *portalAccessRepository) ListWorkerLoads(
	ctx context.Context,
	req *repositories.ListWorkerLoadsRequest,
) ([]*shipment.Assignment, error) {
	acols := buncolgen.AssignmentColumns
	arel := buncolgen.AssignmentRelations
	moveAlias := "shipment_move"
	moveStatus := buncolgen.ShipmentMoveColumns.Status.WithAlias(moveAlias)

	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	items := make([]*shipment.Assignment, 0, limit)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AssignmentScopeTenant(sq, req.TenantInfo).
				WhereGroup(" AND ", func(inner *bun.SelectQuery) *bun.SelectQuery {
					return inner.
						Where(acols.PrimaryWorkerID.Eq(), req.WorkerID).
						WhereOr(acols.SecondaryWorkerID.Eq(), req.WorkerID)
				}).
				Where(acols.ArchivedAt.IsNull())
		}).
		Relation(arel.ShipmentMove).
		Relation(buncolgen.Rel(arel.ShipmentMove, buncolgen.ShipmentMoveRelations.Shipment)).
		Relation(
			buncolgen.Rel(arel.ShipmentMove, buncolgen.ShipmentMoveRelations.Stops),
			func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Order(buncolgen.StopColumns.Sequence.OrderAsc())
			},
		).
		Relation(
			buncolgen.Rel(
				arel.ShipmentMove,
				buncolgen.ShipmentMoveRelations.Stops,
				buncolgen.StopRelations.Location,
			),
		).
		Relation(arel.Tractor).
		Relation(arel.Trailer).
		Order(acols.CreatedAt.OrderDesc()).
		Limit(limit)

	if len(req.Statuses) > 0 {
		query = query.Where(moveStatus.In(), bun.List(req.Statuses))
	}

	if err := query.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list worker loads: %w", err)
	}
	return items, nil
}
