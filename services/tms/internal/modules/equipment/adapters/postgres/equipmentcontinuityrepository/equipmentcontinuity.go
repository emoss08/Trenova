package equipmentcontinuityrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentcontinuity/equipmentcontinuitydb"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentcontinuityports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const completedMoveExistsSQL = "SELECT EXISTS (SELECT 1 FROM shipment_moves WHERE id = $1 " +
	"AND organization_id = $2 AND business_unit_id = $3 AND status = 'Completed')"

var (
	errContinuityCycle = errors.New("equipment continuity cycle detected")

	continuityErrors = dberror.Mapping{Entity: "Equipment continuity"}
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) equipmentcontinuityports.EquipmentContinuityRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.equipment-continuity-repository"),
	}
}

func (r *repository) GetCurrent(
	ctx context.Context,
	req equipmentcontinuityports.GetCurrentEquipmentContinuityRequest,
) (*equipmentcontinuity.EquipmentContinuity, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*equipmentcontinuity.EquipmentContinuity, error) {
			ec := equipmentcontinuitydb.EquipmentContinuities
			entity, err := ec.Where(
				ec.EquipmentType.Eq(req.EquipmentType),
				ec.EquipmentID.Eq(req.EquipmentID),
				ec.IsCurrent.IsTrue(),
			).One(ctx, r.db.Keel(ctx))
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}
			if err != nil {
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *repository) GetEffectiveCurrent(
	ctx context.Context,
	req equipmentcontinuityports.GetCurrentEquipmentContinuityRequest,
) (*equipmentcontinuity.EquipmentContinuity, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*equipmentcontinuity.EquipmentContinuity, error) {
			entity, err := r.GetCurrent(ctx, req)
			if err != nil || entity == nil {
				return entity, err
			}

			visited := make(map[pulid.ID]struct{}, 4)
			for {
				if _, seen := visited[entity.ID]; seen {
					return nil, fmt.Errorf("%w for %s", errContinuityCycle, entity.ID)
				}
				visited[entity.ID] = struct{}{}

				valid, validErr := r.isEffectiveRow(ctx, entity)
				if validErr != nil {
					return nil, validErr
				}
				if valid {
					return entity, nil
				}
				if entity.PreviousContinuityID.IsNil() {
					return nil, nil
				}

				entity, err = r.GetByID(ctx, entity.PreviousContinuityID)
				if errortypes.IsNotFoundError(err) {
					return nil, nil
				}
				if err != nil {
					return nil, err
				}
			}
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	id pulid.ID,
) (*equipmentcontinuity.EquipmentContinuity, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*equipmentcontinuity.EquipmentContinuity, error) {
			entity, err := equipmentcontinuitydb.EquipmentContinuities.Get(ctx, r.db.Keel(ctx), id)
			if err != nil {
				return nil, continuityErrors.Map(err, id.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Advance(
	ctx context.Context,
	req equipmentcontinuityports.CreateEquipmentContinuityRequest,
) (*equipmentcontinuity.EquipmentContinuity, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*equipmentcontinuity.EquipmentContinuity, error) {
			current, err := r.GetCurrent(
				ctx,
				equipmentcontinuityports.GetCurrentEquipmentContinuityRequest{
					TenantInfo:    req.TenantInfo,
					EquipmentType: req.EquipmentType,
					EquipmentID:   req.EquipmentID,
				},
			)
			if err != nil {
				return nil, err
			}

			if current != nil {
				if err = r.supersede(ctx, current); err != nil {
					return nil, err
				}
			}

			entity := &equipmentcontinuity.EquipmentContinuity{
				OrganizationID:       req.TenantInfo.OrgID,
				BusinessUnitID:       req.TenantInfo.BuID,
				EquipmentType:        req.EquipmentType,
				EquipmentID:          req.EquipmentID,
				CurrentLocationID:    req.CurrentLocationID,
				PreviousContinuityID: previousID(current),
				SourceType:           req.SourceType,
				SourceShipmentID:     req.SourceShipmentID,
				SourceShipmentMoveID: req.SourceShipmentMoveID,
				SourceAssignmentID:   req.SourceAssignmentID,
				IsCurrent:            true,
			}

			if err = equipmentcontinuitydb.EquipmentContinuities.Create(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				return nil, fmt.Errorf("create equipment continuity: %w", err)
			}

			return entity, nil
		},
	)
}

func (r *repository) RollbackCurrentByMove(
	ctx context.Context,
	req equipmentcontinuityports.RollbackEquipmentContinuityByMoveRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		current, err := r.GetCurrent(
			ctx,
			equipmentcontinuityports.GetCurrentEquipmentContinuityRequest{
				TenantInfo:    req.TenantInfo,
				EquipmentType: req.EquipmentType,
				EquipmentID:   req.EquipmentID,
			},
		)
		if err != nil || current == nil {
			return err
		}

		if current.SourceShipmentMoveID != req.MoveID {
			return nil
		}

		return r.rollbackRow(ctx, current)
	})
}

func (r *repository) RollbackCurrentByShipment(
	ctx context.Context,
	req equipmentcontinuityports.RollbackEquipmentContinuityByShipmentRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		ec := equipmentcontinuitydb.EquipmentContinuities
		rows, err := ec.Where(
			ec.SourceShipmentID.Eq(req.ShipmentID),
			ec.IsCurrent.IsTrue(),
		).All(ctx, r.db.Keel(ctx))
		if err != nil {
			return err
		}

		for _, row := range rows {
			if err = r.rollbackRow(ctx, row); err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *repository) rollbackRow(
	ctx context.Context,
	current *equipmentcontinuity.EquipmentContinuity,
) error {
	if err := r.supersede(ctx, current); err != nil {
		return err
	}

	if current.PreviousContinuityID.IsNil() {
		return nil
	}

	previous, err := r.GetByID(ctx, current.PreviousContinuityID)
	if err != nil {
		return err
	}

	ec := equipmentcontinuitydb.EquipmentContinuities
	previous.IsCurrent = true
	previous.SupersededAt = nil
	if err = ec.Save(ctx, r.db.Keel(ctx), previous, ec.IsCurrent, ec.SupersededAt); err != nil {
		return fmt.Errorf(
			"restore previous equipment continuity: %w",
			continuityErrors.Map(err, previous.ID.String()),
		)
	}

	return nil
}

func (r *repository) supersede(
	ctx context.Context,
	entity *equipmentcontinuity.EquipmentContinuity,
) error {
	if entity == nil || !entity.IsCurrent {
		return nil
	}

	ec := equipmentcontinuitydb.EquipmentContinuities
	entity.IsCurrent = false
	entity.SupersededAt = new(timeutils.NowUnix())
	if err := ec.Save(ctx, r.db.Keel(ctx), entity, ec.IsCurrent, ec.SupersededAt); err != nil {
		return fmt.Errorf(
			"supersede equipment continuity: %w",
			continuityErrors.Map(err, entity.ID.String()),
		)
	}

	return nil
}

func previousID(entity *equipmentcontinuity.EquipmentContinuity) pulid.ID {
	if entity == nil {
		return pulid.Nil
	}

	return entity.ID
}

func (r *repository) isEffectiveRow(
	ctx context.Context,
	entity *equipmentcontinuity.EquipmentContinuity,
) (bool, error) {
	switch entity.SourceType {
	case equipmentcontinuity.SourceTypeManualLocate:
		return true, nil
	case equipmentcontinuity.SourceTypeAssignment:
		if entity.SourceShipmentMoveID.IsNil() {
			return false, nil
		}

		return keel.QueryValue[bool](
			ctx,
			r.db.Keel(ctx),
			completedMoveExistsSQL,
			entity.SourceShipmentMoveID.String(),
			entity.OrganizationID.String(),
			entity.BusinessUnitID.String(),
		)
	default:
		return false, nil
	}
}
