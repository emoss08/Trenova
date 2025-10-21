package shipmentrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/pulid"
)

func (r *repository) prepareForCreation(
	ctx context.Context,
	entity *shipment.Shipment,
	userID pulid.ID,
) error {
	proNumber, err := r.generator.GenerateShipmentProNumber(
		ctx,
		entity.OrganizationID,
		entity.BusinessUnitID,
	)
	if err != nil {
		return fmt.Errorf("generate shipment pro number: %w", err)
	}

	entity.ProNumber = proNumber

	r.calculator.CalculateTotals(ctx, entity, userID)

	if err = r.calculator.CalculateStatus(entity); err != nil {
		return fmt.Errorf("calculate shipment status: %w", err)
	}

	if err = r.calculator.CalculateTimestamps(entity); err != nil {
		return fmt.Errorf("calculate shipment timestamps: %w", err)
	}

	return nil
}
