package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ShipmentCDCHandlerParams struct {
	fx.In

	Logger           *logger.Logger
	StreamingService services.StreamingService
}

type ShipmentCDCHandler struct {
	l                *zerolog.Logger
	streamingService services.StreamingService
}

func NewShipmentCDCHandler(p ShipmentCDCHandlerParams) services.CDCEventHandler {
	log := p.Logger.With().
		Str("service", "shipment-cdc-handler").
		Logger()

	return &ShipmentCDCHandler{
		l:                &log,
		streamingService: p.StreamingService,
	}
}

func (h *ShipmentCDCHandler) GetTableName() string {
	return "shipments"
}

func (h *ShipmentCDCHandler) HandleEvent(event services.CDCEvent) error {
	h.l.Debug().
		Str("operation", event.Operation).
		Str("table", event.Table).
		Msg("Processing shipment CDC event")

	switch event.Operation {
	case "create":
		return h.handleCreate(event)
	case "update":
		return h.handleUpdate(event)
	case "delete":
		return h.handleDelete(event)
	case "read":
		// Ignore read events to avoid flooding during initial snapshot
		h.l.Debug().Msg("Ignoring read event from snapshot")
		return nil
	default:
		h.l.Warn().
			Str("operation", event.Operation).
			Msg("Unknown operation type in shipment CDC event")
		return nil
	}
}

func (h *ShipmentCDCHandler) handleCreate(event services.CDCEvent) error {
	if event.After == nil {
		return fmt.Errorf("create event missing 'after' data")
	}

	// Extract organization and business unit IDs for tenant isolation
	orgID, ok := event.After["organization_id"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid organization_id in shipment data")
	}

	buID, ok := event.After["business_unit_id"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid business_unit_id in shipment data")
	}

	// Convert to shipment domain object
	shipmentObj, err := h.convertToShipment(event.After)
	if err != nil {
		return fmt.Errorf("failed to convert to shipment object: %w", err)
	}

	// Broadcast to streaming service
	if err := h.streamingService.BroadcastToStream("shipments", orgID, buID, shipmentObj); err != nil {
		h.l.Error().
			Err(err).
			Str("org_id", orgID).
			Str("bu_id", buID).
			Msg("Failed to broadcast shipment create event")
		return err
	}

	h.l.Info().
		Str("shipment_id", shipmentObj.ID.String()).
		Str("pro_number", shipmentObj.ProNumber).
		Str("org_id", orgID).
		Str("bu_id", buID).
		Msg("Broadcasted shipment create event")

	return nil
}

func (h *ShipmentCDCHandler) handleUpdate(event services.CDCEvent) error {
	if event.After == nil {
		return fmt.Errorf("update event missing 'after' data")
	}

	// Extract organization and business unit IDs for tenant isolation
	orgID, ok := event.After["organization_id"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid organization_id in shipment data")
	}

	buID, ok := event.After["business_unit_id"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid business_unit_id in shipment data")
	}

	// Convert to shipment domain object
	shipmentObj, err := h.convertToShipment(event.After)
	if err != nil {
		return fmt.Errorf("failed to convert to shipment object: %w", err)
	}

	// Broadcast to streaming service
	if err := h.streamingService.BroadcastToStream("shipments", orgID, buID, shipmentObj); err != nil {
		h.l.Error().
			Err(err).
			Str("org_id", orgID).
			Str("bu_id", buID).
			Msg("Failed to broadcast shipment update event")
		return err
	}

	h.l.Info().
		Str("shipment_id", shipmentObj.ID.String()).
		Str("pro_number", shipmentObj.ProNumber).
		Str("org_id", orgID).
		Str("bu_id", buID).
		Msg("Broadcasted shipment update event")

	return nil
}

func (h *ShipmentCDCHandler) handleDelete(event services.CDCEvent) error {
	if event.Before == nil {
		return fmt.Errorf("delete event missing 'before' data")
	}

	// Extract organization and business unit IDs for tenant isolation
	orgID, ok := event.Before["organization_id"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid organization_id in shipment data")
	}

	buID, ok := event.Before["business_unit_id"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid business_unit_id in shipment data")
	}

	// For delete events, we send a special delete notification
	deleteEvent := map[string]any{
		"operation":        "delete",
		"shipment_id":      event.Before["id"],
		"pro_number":       event.Before["pro_number"],
		"organization_id":  orgID,
		"business_unit_id": buID,
		"deleted_at":       time.Now().Unix(),
	}

	// Broadcast to streaming service
	if err := h.streamingService.BroadcastToStream("shipments", orgID, buID, deleteEvent); err != nil {
		h.l.Error().
			Err(err).
			Str("org_id", orgID).
			Str("bu_id", buID).
			Msg("Failed to broadcast shipment delete event")
		return err
	}

	h.l.Info().
		Any("shipment_id", event.Before["id"]).
		Any("pro_number", event.Before["pro_number"]).
		Str("org_id", orgID).
		Str("bu_id", buID).
		Msg("Broadcasted shipment delete event")

	return nil
}

func (h *ShipmentCDCHandler) convertToShipment(data map[string]any) (*shipment.Shipment, error) {
	// This is a simplified conversion - you might want to use a more robust
	// mapping library or implement proper field mapping
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal shipment data: %w", err)
	}

	var shp shipment.Shipment
	if err := json.Unmarshal(jsonData, &shp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to shipment struct: %w", err)
	}

	return &shp, nil
}