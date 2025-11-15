package consumers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	workflowservice "github.com/emoss08/trenova/internal/core/services/workflowservice"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// WorkflowTriggerConsumer listens to CDC events and triggers workflows
type WorkflowTriggerConsumer struct {
	logger         *zap.Logger
	reader         *kafka.Reader
	triggerService *workflowservice.TriggerService
}

// WorkflowTriggerConsumerParams holds dependencies for workflow trigger consumer
type WorkflowTriggerConsumerParams struct {
	Logger         *zap.Logger
	Reader         *kafka.Reader
	TriggerService *workflowservice.TriggerService
}

// NewWorkflowTriggerConsumer creates a new workflow trigger consumer
func NewWorkflowTriggerConsumer(p WorkflowTriggerConsumerParams) *WorkflowTriggerConsumer {
	return &WorkflowTriggerConsumer{
		logger:         p.Logger.Named("workflow-trigger-consumer"),
		reader:         p.Reader,
		triggerService: p.TriggerService,
	}
}

// Start begins consuming CDC events
func (c *WorkflowTriggerConsumer) Start(ctx context.Context) error {
	c.logger.Info("Starting workflow trigger consumer")

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Workflow trigger consumer stopped")
			return ctx.Err()
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				c.logger.Error("Error reading message", zap.Error(err))
				continue
			}

			if err := c.processMessage(ctx, msg); err != nil {
				c.logger.Error("Error processing message",
					zap.Error(err),
					zap.String("topic", msg.Topic),
					zap.Int64("offset", msg.Offset),
				)
			}
		}
	}
}

// CDCEvent represents a CDC event from Debezium
type CDCEvent struct {
	Payload struct {
		Before map[string]any `json:"before"`
		After  map[string]any `json:"after"`
		Source struct {
			Table string `json:"table"`
		} `json:"source"`
		Op string `json:"op"` // c=create, u=update, d=delete
	} `json:"payload"`
}

func (c *WorkflowTriggerConsumer) processMessage(ctx context.Context, msg kafka.Message) error {
	var event CDCEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal CDC event: %w", err)
	}

	// Handle different table events
	switch event.Payload.Source.Table {
	case "shipments":
		return c.handleShipmentEvent(ctx, &event)
	case "customers":
		return c.handleCustomerEvent(ctx, &event)
	default:
		// Ignore other tables
		return nil
	}
}

func (c *WorkflowTriggerConsumer) handleShipmentEvent(ctx context.Context, event *CDCEvent) error {
	// Only process updates
	if event.Payload.Op != "u" {
		return nil
	}

	before := event.Payload.Before
	after := event.Payload.After

	// Check if status changed
	if before["status"] == after["status"] {
		return nil // Status didn't change
	}

	// Parse shipment ID
	shipmentIDStr, ok := after["id"].(string)
	if !ok {
		return fmt.Errorf("shipment ID not found in CDC event")
	}

	shipmentID, err := pulid.MustParse(shipmentIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse shipment ID: %w", err)
	}

	// Parse organization and business unit IDs
	orgIDStr, ok := after["organization_id"].(string)
	if !ok {
		return fmt.Errorf("organization ID not found in CDC event")
	}

	orgID, err := pulid.MustParse(orgIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse organization ID: %w", err)
	}

	buIDStr, ok := after["business_unit_id"].(string)
	if !ok {
		return fmt.Errorf("business unit ID not found in CDC event")
	}

	buID, err := pulid.MustParse(buIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse business unit ID: %w", err)
	}

	// Get user ID (use system user or the user who made the change)
	var userID pulid.ID
	if userIDStr, ok := after["updated_by"].(string); ok {
		userID, _ = pulid.MustParse(userIDStr)
	}

	oldStatus, _ := before["status"].(string)
	newStatus, _ := after["status"].(string)

	c.logger.Info("Shipment status changed",
		zap.String("shipmentId", shipmentID.String()),
		zap.String("oldStatus", oldStatus),
		zap.String("newStatus", newStatus),
	)

	// Trigger workflows
	return c.triggerService.TriggerShipmentStatusChange(
		ctx,
		shipmentID,
		oldStatus,
		newStatus,
		orgID,
		buID,
		userID,
	)
}

func (c *WorkflowTriggerConsumer) handleCustomerEvent(ctx context.Context, event *CDCEvent) error {
	// Handle customer creation/updates
	after := event.Payload.After

	// Parse customer ID
	customerIDStr, ok := after["id"].(string)
	if !ok {
		return fmt.Errorf("customer ID not found in CDC event")
	}

	customerID, err := pulid.MustParse(customerIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse customer ID: %w", err)
	}

	// Parse organization and business unit IDs
	orgIDStr, ok := after["organization_id"].(string)
	if !ok {
		return fmt.Errorf("organization ID not found in CDC event")
	}

	orgID, err := pulid.MustParse(orgIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse organization ID: %w", err)
	}

	buIDStr, ok := after["business_unit_id"].(string)
	if !ok {
		return fmt.Errorf("business unit ID not found in CDC event")
	}

	buID, err := pulid.MustParse(buIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse business unit ID: %w", err)
	}

	var userID pulid.ID
	if userIDStr, ok := after["created_by"].(string); ok {
		userID, _ = pulid.MustParse(userIDStr)
	}

	// Determine event type
	var triggerType string
	switch event.Payload.Op {
	case "c":
		triggerType = "entity_created"
	case "u":
		triggerType = "entity_updated"
	default:
		return nil
	}

	c.logger.Info("Customer event",
		zap.String("customerId", customerID.String()),
		zap.String("eventType", triggerType),
	)

	// TODO: Map to workflow.TriggerType and trigger workflows
	// This requires converting the string to the proper enum type

	return nil
}

// Stop stops the consumer
func (c *WorkflowTriggerConsumer) Stop() error {
	c.logger.Info("Stopping workflow trigger consumer")
	return c.reader.Close()
}
