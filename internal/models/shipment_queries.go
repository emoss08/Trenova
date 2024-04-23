package models

import (
	"context"

	gen "github.com/emoss08/trenova/internal/ent"
)

func HandleVoidedShipment(
	ctx context.Context, m *gen.ShipmentMutation, client *gen.Client,
) error {
	// If the shipment is voided then reset the order and mark the status as voided.
	shipmentStatus, _ := m.Status()

	if shipmentStatus == "Voided" {
		// Reset the order.
		if err := resetShipment(ctx, m, client); err != nil {
			return err
		}

		// Mark the status as voided.
		if err := markOrderVoided(ctx, m, client); err != nil {
			return err
		}
	}

	return nil
}

func resetShipment(
	ctx context.Context, m *gen.ShipmentMutation, client *gen.Client,
) error {
	shipmentID, _ := m.ID()

	// Update the shipment based on ID.
	_, err := client.Shipment.UpdateOneID(shipmentID).
		SetStatus("New").
		SetReadyToBill(false).
		SetBilled(false).
		SetTransferredToBilling(false).
		SetTransferredToBillingDate(nil).
		Save(ctx)
	if err != nil {
		return err
	}

	return nil
}

func markOrderVoided(
	ctx context.Context, m *gen.ShipmentMutation, client *gen.Client,
) error {
	shipmentID, _ := m.ID()

	// Update the shipment based on ID.
	_, err := client.Shipment.UpdateOneID(shipmentID).
		SetStatus("Voided").
		Save(ctx)
	if err != nil {
		return err
	}

	return nil
}
