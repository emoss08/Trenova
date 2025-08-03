/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package handlers

import (
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/cdcutils"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/cdctypes"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

// ShipmentCDCHandlerParams defines dependencies required for initializing the ShipmentCDCHandler.
// This includes logger for structured logging and streaming service for real-time event broadcasting.
type ShipmentCDCHandlerParams struct {
	fx.In

	Logger           *logger.Logger
	StreamingService services.StreamingService
}

// ShipmentCDCHandler implements the CDCEventHandler interface
// and provides methods to handle Change Data Capture events for shipments.
// It processes create, update, and delete operations and broadcasts them to streaming clients.
type ShipmentCDCHandler struct {
	l                *zerolog.Logger
	streamingService services.StreamingService
}

// NewShipmentCDCHandler initializes a new instance of ShipmentCDCHandler with its dependencies.
//
// Parameters:
//   - p: ShipmentCDCHandlerParams containing dependencies.
//
// Returns:
//   - services.CDCEventHandler: A ready-to-use shipment CDC handler instance.
func NewShipmentCDCHandler(p ShipmentCDCHandlerParams) services.CDCEventHandler {
	log := p.Logger.With().
		Str("service", "shipment-cdc-handler").
		Logger()

	return &ShipmentCDCHandler{
		l:                &log,
		streamingService: p.StreamingService,
	}
}

// GetTableName returns the database table name that this handler monitors.
// This is used by the CDC system to route events to the appropriate handler.
//
// Returns:
//   - string: The table name "shipments".
func (h *ShipmentCDCHandler) GetTableName() string {
	return "shipments"
}

// HandleEvent processes Change Data Capture events for shipments.
// It routes events to appropriate handlers based on the operation type (create, update, delete).
// Read events are ignored to prevent flooding during initial snapshots.
//
// Parameters:
//   - event: CDCEvent containing the operation type and data payload.
//
// Returns:
//   - error: If event processing fails, nil otherwise.
func (h *ShipmentCDCHandler) HandleEvent(event *cdctypes.CDCEvent) error {
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
		// * Ignore read events to avoid flooding during initial snapshot
		h.l.Debug().Msg("Ignoring read event from snapshot")
		return nil
	default:
		h.l.Warn().
			Str("operation", event.Operation).
			Msg("Unknown operation type in shipment CDC event")
		return nil
	}
}

// handleCreate processes shipment creation events from the CDC stream.
// It extracts tenant information, converts the raw data to a shipment domain object,
// and broadcasts the event to connected streaming clients.
//
// Parameters:
//   - event: CDCEvent containing the newly created shipment data in the 'after' field.
//
// Returns:
//   - error: If validation, conversion, or broadcasting fails.
func (h *ShipmentCDCHandler) handleCreate(event *cdctypes.CDCEvent) error {
	if event.After == nil {
		return oops.
			In("shipment_cdc_handler").
			With("operation", event.Operation).
			With("table", event.Table).
			Time(time.Now()).
			New("create event missing 'after' data")
	}

	// * Extract organization and business unit IDs for tenant isolation
	orgID, buID, err := cdcutils.ExtractTenantInformation(event)
	if err != nil {
		return oops.
			In("shipment_cdc_handler").
			With("operation", event.Operation).
			Time(time.Now()).
			Wrapf(err, "failed to extract tenant information")
	}

	// * Convert to shipment domain object
	shipmentObj, err := h.convertToShipment(event.After)
	if err != nil {
		return oops.
			In("shipment_cdc_handler").
			With("org_id", orgID).
			With("bu_id", buID).
			Time(time.Now()).
			Wrapf(err, "failed to convert to shipment object")
	}

	h.l.Debug().
		Str("shipmentID", shipmentObj.ID.String()).
		Str("proNumber", shipmentObj.ProNumber).
		Str("orgID", orgID).
		Str("buID", buID).
		Str("status", string(shipmentObj.Status)).
		Bool("snapshot", event.Metadata.Source.Snapshot).
		Msg("Broadcasting shipment create event")

	// * Create event envelope with metadata
	eventEnvelope := map[string]any{
		"operation": "create",
		"timestamp": event.Metadata.Timestamp,
		"shipment":  shipmentObj,
		"metadata": map[string]any{
			"source":         event.Metadata.Source.Connector,
			"lsn":            event.Metadata.LSN,
			"transaction_id": event.Metadata.TransactionID,
			"snapshot":       event.Metadata.Source.Snapshot,
		},
	}

	// * Broadcast to streaming service
	if err = h.streamingService.BroadcastToStream("shipments", orgID, buID, eventEnvelope); err != nil {
		h.l.Error().
			Err(err).
			Str("shipment_id", shipmentObj.ID.String()).
			Str("org_id", orgID).
			Str("bu_id", buID).
			Msg("Failed to broadcast shipment create event")
		return oops.
			In("shipment_cdc_handler").
			With("shipment_id", shipmentObj.ID.String()).
			With("org_id", orgID).
			With("bu_id", buID).
			Time(time.Now()).
			Wrapf(err, "broadcast failed")
	}

	h.l.Info().
		Str("shipmentID", shipmentObj.ID.String()).
		Str("proNumber", shipmentObj.ProNumber).
		Str("orgID", orgID).
		Str("buID", buID).
		Str("status", string(shipmentObj.Status)).
		Msg("Successfully broadcasted shipment create event")

	return nil
}

// handleUpdate processes shipment modification events from the CDC stream.
// It extracts tenant information, converts the updated data to a shipment domain object,
// identifies changed fields, and broadcasts the changes to connected streaming clients.
//
// Parameters:
//   - event: CDCEvent containing the updated shipment data in the 'after' field.
//
// Returns:
//   - error: If validation, conversion, or broadcasting fails.
func (h *ShipmentCDCHandler) handleUpdate(event *cdctypes.CDCEvent) error {
	if event.After == nil {
		return oops.
			In("shipment_cdc_handler").
			With("operation", event.Operation).
			With("table", event.Table).
			Time(time.Now()).
			New("update event missing 'after' data")
	}

	// * Extract organization and business unit IDs for tenant isolation
	orgID, buID, err := cdcutils.ExtractTenantInformation(event)
	if err != nil {
		return oops.
			In("shipment_cdc_handler").
			With("operation", event.Operation).
			Time(time.Now()).
			Wrapf(err, "failed to extract tenant information")
	}

	// * Convert to shipment domain object
	shipmentObj, err := h.convertToShipment(event.After)
	if err != nil {
		return oops.
			In("shipment_cdc_handler").
			With("org_id", orgID).
			With("bu_id", buID).
			Time(time.Now()).
			Wrapf(err, "failed to convert to shipment object")
	}

	// * Identify changed fields for better client handling
	var changedFields []string
	if event.Before != nil {
		for key, afterVal := range event.After {
			beforeVal, exists := event.Before[key]
			if !exists || beforeVal != afterVal {
				changedFields = append(changedFields, key)
			}
		}
	}

	h.l.Debug().
		Str("shipmentID", shipmentObj.ID.String()).
		Str("proNumber", shipmentObj.ProNumber).
		Str("orgID", orgID).
		Str("buID", buID).
		Strs("changed_fields", changedFields).
		Int("changed_count", len(changedFields)).
		Msg("Broadcasting shipment update event")

	// * Create event envelope with metadata
	eventEnvelope := map[string]any{
		"operation":      "update",
		"timestamp":      event.Metadata.Timestamp,
		"shipment":       shipmentObj,
		"changed_fields": changedFields,
		"metadata": map[string]any{
			"source":         event.Metadata.Source.Connector,
			"lsn":            event.Metadata.LSN,
			"transaction_id": event.Metadata.TransactionID,
		},
	}

	// * Broadcast to streaming service
	if err = h.streamingService.BroadcastToStream("shipments", orgID, buID, eventEnvelope); err != nil {
		h.l.Error().
			Err(err).
			Str("shipment_id", shipmentObj.ID.String()).
			Str("org_id", orgID).
			Str("bu_id", buID).
			Strs("changed_fields", changedFields).
			Msg("Failed to broadcast shipment update event")
		return oops.
			In("shipment_cdc_handler").
			With("shipment_id", shipmentObj.ID.String()).
			With("org_id", orgID).
			With("bu_id", buID).
			Time(time.Now()).
			Wrapf(err, "broadcast failed")
	}

	h.l.Info().
		Str("shipmentID", shipmentObj.ID.String()).
		Str("proNumber", shipmentObj.ProNumber).
		Str("orgID", orgID).
		Str("buID", buID).
		Int("changed_fields", len(changedFields)).
		Msg("Successfully broadcasted shipment update event")

	return nil
}

// handleDelete processes shipment deletion events from the CDC stream.
// It extracts tenant information from the 'before' data and creates a special delete notification
// that includes the shipment ID, pro number, and deletion timestamp for client synchronization.
//
// Parameters:
//   - event: CDCEvent containing the deleted shipment data in the 'before' field.
//
// Returns:
//   - error: If validation or broadcasting fails.
func (h *ShipmentCDCHandler) handleDelete(event *cdctypes.CDCEvent) error {
	if event.Before == nil {
		return oops.
			In("shipment_cdc_handler").
			With("operation", event.Operation).
			With("table", event.Table).
			Time(time.Now()).
			New("delete event missing 'before' data")
	}

	// * Extract organization and business unit IDs for tenant isolation
	orgID, buID, err := cdcutils.ExtractTenantInformation(event)
	if err != nil {
		return oops.
			In("shipment_cdc_handler").
			With("operation", event.Operation).
			Time(time.Now()).
			Wrapf(err, "failed to extract tenant information")
	}

	// * Extract key fields safely
	shipmentID := cdcutils.ExtractStringField(event.Before, "id")
	proNumber := cdcutils.ExtractStringField(event.Before, "pro_number")

	if shipmentID == "" {
		return oops.
			In("shipment_cdc_handler").
			With("org_id", orgID).
			With("bu_id", buID).
			Time(time.Now()).
			New("delete event missing shipment ID")
	}

	h.l.Debug().
		Str("shipmentID", shipmentID).
		Str("proNumber", proNumber).
		Str("orgID", orgID).
		Str("buID", buID).
		Msg("Broadcasting shipment delete event")

	// * Create comprehensive delete notification
	deleteEvent := map[string]any{
		"operation":        "delete",
		"timestamp":        event.Metadata.Timestamp,
		"shipment_id":      shipmentID,
		"pro_number":       proNumber,
		"organization_id":  orgID,
		"business_unit_id": buID,
		"deleted_at":       timeutils.NowUnix(),
		"metadata": map[string]any{
			"source":         event.Metadata.Source.Connector,
			"lsn":            event.Metadata.LSN,
			"transaction_id": event.Metadata.TransactionID,
		},
	}

	// * Broadcast to streaming service
	if err = h.streamingService.BroadcastToStream("shipments", orgID, buID, deleteEvent); err != nil {
		h.l.Error().
			Err(err).
			Str("shipment_id", shipmentID).
			Str("org_id", orgID).
			Str("bu_id", buID).
			Msg("Failed to broadcast shipment delete event")
		return oops.
			In("shipment_cdc_handler").
			With("shipment_id", shipmentID).
			With("org_id", orgID).
			With("bu_id", buID).
			Time(time.Now()).
			Wrapf(err, "broadcast failed")
	}

	h.l.Info().
		Str("shipmentID", shipmentID).
		Str("proNumber", proNumber).
		Str("orgID", orgID).
		Str("buID", buID).
		Msg("Successfully broadcasted shipment delete event")

	return nil
}

// convertToShipment transforms raw CDC event data into a strongly-typed shipment domain object.
// It handles Avro optional field formats and ensures proper type conversion for all fields.
//
// Parameters:
//   - data: Raw map containing shipment field data from the CDC event.
//
// Returns:
//   - *shipment.Shipment: Converted shipment domain object.
//   - error: If conversion fails.
func (h *ShipmentCDCHandler) convertToShipment(data map[string]any) (*shipment.Shipment, error) {
	if data == nil {
		return nil, oops.
			In("shipment_cdc_handler").
			Time(time.Now()).
			New("nil data provided for shipment conversion")
	}

	// * First, normalize the data by converting Avro optional fields
	normalizedData := make(map[string]any)
	for k, v := range data {
		normalizedData[k] = cdctypes.ConvertAvroOptionalField(v)
	}

	// * Marshal the normalized data to JSON for intermediate conversion
	jsonData, err := sonic.Marshal(normalizedData)
	if err != nil {
		return nil, oops.
			In("shipment_cdc_handler").
			Time(time.Now()).
			With("data_keys", len(data)).
			Wrapf(err, "failed to marshal shipment data")
	}

	// * Unmarshal JSON into the shipment struct using sonic for performance
	var shp shipment.Shipment
	if err = sonic.Unmarshal(jsonData, &shp); err != nil {
		return nil, oops.
			In("shipment_cdc_handler").
			Time(time.Now()).
			With("json_size", len(jsonData)).
			Wrapf(err, "failed to unmarshal to shipment struct")
	}

	// * Validate critical fields
	if shp.ID.String() == "" {
		return nil, oops.
			In("shipment_cdc_handler").
			Time(time.Now()).
			New("shipment ID is missing or invalid")
	}

	return &shp, nil
}
