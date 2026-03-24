package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetCurrentEquipmentContinuityRequest struct {
	TenantInfo    pagination.TenantInfo             `json:"tenantInfo"`
	EquipmentType equipmentcontinuity.EquipmentType `json:"equipmentType"`
	EquipmentID   pulid.ID                          `json:"equipmentId"`
}

type CreateEquipmentContinuityRequest struct {
	TenantInfo           pagination.TenantInfo             `json:"tenantInfo"`
	EquipmentType        equipmentcontinuity.EquipmentType `json:"equipmentType"`
	EquipmentID          pulid.ID                          `json:"equipmentId"`
	CurrentLocationID    pulid.ID                          `json:"currentLocationId"`
	SourceType           equipmentcontinuity.SourceType    `json:"sourceType"`
	SourceShipmentID     pulid.ID                          `json:"sourceShipmentId"`
	SourceShipmentMoveID pulid.ID                          `json:"sourceShipmentMoveId"`
	SourceAssignmentID   pulid.ID                          `json:"sourceAssignmentId"`
}

type RollbackEquipmentContinuityByMoveRequest struct {
	TenantInfo    pagination.TenantInfo             `json:"tenantInfo"`
	EquipmentType equipmentcontinuity.EquipmentType `json:"equipmentType"`
	EquipmentID   pulid.ID                          `json:"equipmentId"`
	MoveID        pulid.ID                          `json:"moveId"`
}

type RollbackEquipmentContinuityByShipmentRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	ShipmentID pulid.ID              `json:"shipmentId"`
}

type EquipmentContinuityRepository interface {
	GetCurrent(
		ctx context.Context,
		req GetCurrentEquipmentContinuityRequest,
	) (*equipmentcontinuity.EquipmentContinuity, error)
	GetEffectiveCurrent(
		ctx context.Context,
		req GetCurrentEquipmentContinuityRequest,
	) (*equipmentcontinuity.EquipmentContinuity, error)
	GetByID(
		ctx context.Context,
		id pulid.ID,
	) (*equipmentcontinuity.EquipmentContinuity, error)
	Advance(
		ctx context.Context,
		req CreateEquipmentContinuityRequest,
	) (*equipmentcontinuity.EquipmentContinuity, error)
	RollbackCurrentByMove(
		ctx context.Context,
		req RollbackEquipmentContinuityByMoveRequest,
	) error
	RollbackCurrentByShipment(
		ctx context.Context,
		req RollbackEquipmentContinuityByShipmentRequest,
	) error
}
