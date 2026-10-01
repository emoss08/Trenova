package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
)

const idempotencyKeyConstraint = "uq_shipments_idempotency_key"

func (s *service) findIdempotentCreate(
	ctx context.Context,
	entity *shipment.Shipment,
) (*shipment.Shipment, bool, error) {
	if entity.IdempotencyKey == "" {
		return nil, false, nil
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
	id, err := s.repo.FindIDByIdempotencyKey(ctx, &repositories.IdempotencyKeyLookupRequest{
		TenantInfo:     tenantInfo,
		IdempotencyKey: entity.IdempotencyKey,
	})
	if err != nil || id.IsNil() {
		return nil, false, err
	}

	existing, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:              id,
		TenantInfo:      tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{ExpandShipmentDetails: true},
	})
	if err != nil {
		return nil, false, err
	}
	return existing, true, nil
}

func isIdempotencyKeyConflict(err error) bool {
	return dberror.IsUniqueConstraintViolation(err) &&
		dberror.ExtractConstraintName(err) == idempotencyKeyConstraint
}
