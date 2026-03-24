package equipmentcontinuityrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
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

func New(p Params) repositories.EquipmentContinuityRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.equipment-continuity-repository"),
	}
}

func (r *repository) GetCurrent(
	ctx context.Context,
	req repositories.GetCurrentEquipmentContinuityRequest,
) (*equipmentcontinuity.EquipmentContinuity, error) {
	entity := new(equipmentcontinuity.EquipmentContinuity)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("ec.organization_id = ?", req.TenantInfo.OrgID).
		Where("ec.business_unit_id = ?", req.TenantInfo.BuID).
		Where("ec.equipment_type = ?", req.EquipmentType).
		Where("ec.equipment_id = ?", req.EquipmentID).
		Where("ec.is_current = ?", true).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetEffectiveCurrent(
	ctx context.Context,
	req repositories.GetCurrentEquipmentContinuityRequest,
) (*equipmentcontinuity.EquipmentContinuity, error) {
	current, err := r.GetCurrent(ctx, req)
	if err != nil || current == nil {
		return current, err
	}

	visited := make(map[pulid.ID]struct{}, 4)
	entity := current
	for entity != nil {
		if _, ok := visited[entity.ID]; ok {
			return nil, fmt.Errorf("equipment continuity cycle detected for %s", entity.ID)
		}
		visited[entity.ID] = struct{}{}

		valid, err := r.isEffectiveRow(ctx, entity)
		if err != nil {
			return nil, err
		}
		if valid {
			return entity, nil
		}
		if entity.PreviousContinuityID.IsNil() {
			return nil, nil
		}

		entity, err = r.GetByID(ctx, entity.PreviousContinuityID)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}
			return nil, err
		}
	}

	return nil, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	id pulid.ID,
) (*equipmentcontinuity.EquipmentContinuity, error) {
	entity := new(equipmentcontinuity.EquipmentContinuity)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("ec.id = ?", id).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Equipment continuity")
	}

	return entity, nil
}

func (r *repository) Advance(
	ctx context.Context,
	req repositories.CreateEquipmentContinuityRequest,
) (*equipmentcontinuity.EquipmentContinuity, error) {
	current, err := r.GetCurrent(ctx, repositories.GetCurrentEquipmentContinuityRequest{
		TenantInfo:    req.TenantInfo,
		EquipmentType: req.EquipmentType,
		EquipmentID:   req.EquipmentID,
	})
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

	if _, err = r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create equipment continuity: %w", err)
	}

	return entity, nil
}

func (r *repository) RollbackCurrentByMove(
	ctx context.Context,
	req repositories.RollbackEquipmentContinuityByMoveRequest,
) error {
	current, err := r.GetCurrent(ctx, repositories.GetCurrentEquipmentContinuityRequest{
		TenantInfo:    req.TenantInfo,
		EquipmentType: req.EquipmentType,
		EquipmentID:   req.EquipmentID,
	})
	if err != nil || current == nil {
		return err
	}

	if current.SourceShipmentMoveID != req.MoveID {
		return nil
	}

	return r.rollbackRow(ctx, current)
}

func (r *repository) RollbackCurrentByShipment(
	ctx context.Context,
	req repositories.RollbackEquipmentContinuityByShipmentRequest,
) error {
	rows := make([]*equipmentcontinuity.EquipmentContinuity, 0)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&rows).
		Where("ec.organization_id = ?", req.TenantInfo.OrgID).
		Where("ec.business_unit_id = ?", req.TenantInfo.BuID).
		Where("ec.source_shipment_id = ?", req.ShipmentID).
		Where("ec.is_current = ?", true).
		Scan(ctx); err != nil {
		if dberror.IsNotFoundError(err) {
			return nil
		}
		return err
	}

	for _, row := range rows {
		if err := r.rollbackRow(ctx, row); err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) rollbackRow(
	ctx context.Context,
	current *equipmentcontinuity.EquipmentContinuity,
) error {
	if current == nil {
		return nil
	}

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

	now := timeutils.NowUnix()
	ov := previous.Version
	previous.Version++
	previous.IsCurrent = true
	previous.SupersededAt = nil
	previous.UpdatedAt = now

	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(previous).
		Column("is_current", "superseded_at", "version", "updated_at").
		Where("id = ?", previous.ID).
		Where("version = ?", ov).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("restore previous equipment continuity: %w", err)
	}

	return dberror.CheckRowsAffected(result, "Equipment continuity", previous.ID.String())
}

func (r *repository) supersede(
	ctx context.Context,
	entity *equipmentcontinuity.EquipmentContinuity,
) error {
	if entity == nil || !entity.IsCurrent {
		return nil
	}

	now := timeutils.NowUnix()
	ov := entity.Version
	entity.Version++
	entity.IsCurrent = false
	entity.SupersededAt = &now
	entity.UpdatedAt = now

	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Column("is_current", "superseded_at", "version", "updated_at").
		Where("id = ?", entity.ID).
		Where("version = ?", ov).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("supersede equipment continuity: %w", err)
	}

	return dberror.CheckRowsAffected(result, "Equipment continuity", entity.ID.String())
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
	if entity == nil {
		return false, nil
	}

	switch entity.SourceType {
	case equipmentcontinuity.SourceTypeManualLocate:
		return true, nil
	case equipmentcontinuity.SourceTypeAssignment:
		if entity.SourceShipmentMoveID.IsNil() {
			return false, nil
		}

		count, err := r.db.DBForContext(ctx).
			NewSelect().
			Table("shipment_moves").
			Where("id = ?", entity.SourceShipmentMoveID).
			Where("organization_id = ?", entity.OrganizationID).
			Where("business_unit_id = ?", entity.BusinessUnitID).
			Where("status = ?", "Completed").
			Count(ctx)
		if err != nil {
			return false, err
		}
		return count > 0, nil
	default:
		return false, nil
	}
}
