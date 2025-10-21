package handlers

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/cdctypes"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/utils/cdcutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ShipmentCDCHandlerParams struct {
	fx.In

	Logger           *zap.Logger
	StreamingService services.StreamingService
}

type ShipmentCDCHandler struct {
	l                *zap.Logger
	streamingService services.StreamingService
}

func NewShipmentCDCHandler(p ShipmentCDCHandlerParams) services.CDCEventHandler {
	return &ShipmentCDCHandler{
		l:                p.Logger.With(zap.String("handler", "shipment-cdc-handler")),
		streamingService: p.StreamingService,
	}
}

func (h *ShipmentCDCHandler) GetTableName() string {
	return "shipments"
}

func (h *ShipmentCDCHandler) HandleEvent(ctx context.Context, event *cdctypes.CDCEvent) error {
	h.l.Debug(
		"Processing shipment CDC event",
		zap.String("operation", event.Operation),
		zap.String("table", event.Table),
	)

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	switch event.Operation {
	case "create":
		return h.handleCreate(ctx, event)
	case "update":
		return h.handleUpdate(ctx, event)
	case "delete":
		return h.handleDelete(ctx, event)
	case "read":
		h.l.Debug(
			"Ignoring read event from snapshot",
			zap.String("operation", event.Operation),
			zap.String("table", event.Table),
		)
		return nil
	default:
		h.l.Warn(
			"Unknown operation type in shipment CDC event",
			zap.String("operation", event.Operation),
		)
		return nil
	}
}

func (h *ShipmentCDCHandler) handleCreate(ctx context.Context, event *cdctypes.CDCEvent) error {
	_ = ctx // Context can be used for future cancellation/timeout logic
	if event.After == nil {
		// ! Create does not have an 'after' data
		return ErrCreateEventMissingAfterData
	}

	orgID, buID, err := cdcutils.ExtractTenantInformation(event)
	if err != nil {
		return fmt.Errorf("failed to extract tenant information: %w", err)
	}

	shipmentObj, err := h.convertToShipment(event.After)
	if err != nil {
		return fmt.Errorf("failed to convert to shipment object: %w", err)
	}

	h.l.Debug(
		"Broadcasting shipment create event",
		zap.String("shipmentID", shipmentObj.ID.String()),
		zap.String("proNumber", shipmentObj.ProNumber),
		zap.String("orgID", orgID),
		zap.String("buID", buID),
		zap.String("status", string(shipmentObj.Status)),
		zap.Bool("snapshot", event.Metadata.Source.Snapshot),
	)

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

	if err = h.streamingService.BroadcastToStream("shipments", orgID, buID, eventEnvelope); err != nil {
		h.l.Error(
			"Failed to broadcast shipment create event",
			zap.Error(err),
			zap.String("shipmentID", shipmentObj.ID.String()),
			zap.String("orgID", orgID),
			zap.String("buID", buID),
		)
		return fmt.Errorf("failed to broadcast shipment create event: %w", err)
	}

	h.l.Info(
		"Successfully broadcasted shipment create event",
		zap.String("shipmentID", shipmentObj.ID.String()),
		zap.String("proNumber", shipmentObj.ProNumber),
		zap.String("orgID", orgID),
		zap.String("buID", buID),
	)

	return nil
}

func (h *ShipmentCDCHandler) handleUpdate(ctx context.Context, event *cdctypes.CDCEvent) error {
	_ = ctx // Context can be used for future cancellation/timeout logic
	if event.After == nil {
		// ! Update should always have an 'after' data if configured correctly
		return ErrUpdateEventMissingAfterData
	}

	orgID, buID, err := cdcutils.ExtractTenantInformation(event)
	if err != nil {
		return fmt.Errorf("failed to extract tenant information: %w", err)
	}

	shipmentObj, err := h.convertToShipment(event.After)
	if err != nil {
		return fmt.Errorf("failed to convert to shipment object: %w", err)
	}

	var changedFields []string
	if event.Before != nil {
		for key, afterVal := range event.After {
			beforeVal, exists := event.Before[key]
			if !exists || beforeVal != afterVal {
				changedFields = append(changedFields, key)
			}
		}
	}

	h.l.Debug(
		"Broadcasting shipment update event",
		zap.String("shipmentID", shipmentObj.ID.String()),
		zap.String("proNumber", shipmentObj.ProNumber),
		zap.String("orgID", orgID),
		zap.String("buID", buID),
		zap.Strings("changedFields", changedFields),
	)

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

	if err = h.streamingService.BroadcastToStream("shipments", orgID, buID, eventEnvelope); err != nil {
		h.l.Error(
			"Failed to broadcast shipment update event",
			zap.Error(err),
			zap.String("shipmentID", shipmentObj.ID.String()),
			zap.String("orgID", orgID),
			zap.String("buID", buID),
		)
		return fmt.Errorf("failed to broadcast shipment update event: %w", err)
	}

	h.l.Info(
		"Successfully broadcasted shipment update event",
		zap.String("shipmentID", shipmentObj.ID.String()),
		zap.String("proNumber", shipmentObj.ProNumber),
		zap.String("orgID", orgID),
		zap.String("buID", buID),
	)

	return nil
}

func (h *ShipmentCDCHandler) handleDelete(ctx context.Context, event *cdctypes.CDCEvent) error {
	_ = ctx // Context can be used for future cancellation/timeout logic
	if event.Before == nil {
		return ErrDeleteEventMissingBeforeData
	}

	orgID, buID, err := cdcutils.ExtractTenantInformation(event)
	if err != nil {
		return fmt.Errorf("failed to extract tenant information: %w", err)
	}

	shipmentID := cdcutils.ExtractStringField(event.Before, "id")
	proNumber := cdcutils.ExtractStringField(event.Before, "pro_number")

	if shipmentID == "" {
		return ErrDeleteEventMissingShipmentID
	}

	h.l.Debug(
		"Broadcasting shipment delete event",
		zap.String("shipmentID", shipmentID),
		zap.String("proNumber", proNumber),
		zap.String("orgID", orgID),
		zap.String("buID", buID),
	)

	deleteEvent := map[string]any{
		"operation":        "delete",
		"timestamp":        event.Metadata.Timestamp,
		"shipment_id":      shipmentID,
		"pro_number":       proNumber,
		"organization_id":  orgID,
		"business_unit_id": buID,
		"deleted_at":       utils.NowUnix(),
		"metadata": map[string]any{
			"source":         event.Metadata.Source.Connector,
			"lsn":            event.Metadata.LSN,
			"transaction_id": event.Metadata.TransactionID,
		},
	}

	if err = h.streamingService.BroadcastToStream("shipments", orgID, buID, deleteEvent); err != nil {
		h.l.Error(
			"Failed to broadcast shipment delete event",
			zap.Error(err),
			zap.String("shipmentID", shipmentID),
			zap.String("orgID", orgID),
			zap.String("buID", buID),
		)
		return fmt.Errorf("failed to broadcast shipment delete event: %w", err)
	}

	h.l.Info(
		"Successfully broadcasted shipment delete event",
		zap.String("shipmentID", shipmentID),
		zap.String("proNumber", proNumber),
		zap.String("orgID", orgID),
		zap.String("buID", buID),
	)

	return nil
}

func (h *ShipmentCDCHandler) convertToShipment(data map[string]any) (*shipment.Shipment, error) {
	if data == nil {
		return nil, ErrNilDataProvided
	}

	normalizedData := make(map[string]any)
	for k, v := range data {
		normalizedData[k] = cdctypes.ConvertAvroOptionalField(v)
	}

	jsonData, err := sonic.Marshal(normalizedData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal shipment data: %w", err)
	}

	var shp shipment.Shipment
	if err = sonic.Unmarshal(jsonData, &shp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to shipment struct: %w", err)
	}

	if shp.ID.String() == "" {
		return nil, ErrShipmentIDMissing
	}

	return &shp, nil
}
