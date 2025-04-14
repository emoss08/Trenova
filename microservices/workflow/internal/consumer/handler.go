package consumer

import (
	"context"
	"fmt"
	"log"

	"github.com/emoss08/trenova/microservices/workflow/internal/model"
	"github.com/hatchet-dev/hatchet/pkg/client"
)

type HatchetHandler struct {
	client client.Client
}

func NewHatchetHandler(hatchetClient client.Client) *HatchetHandler {
	return &HatchetHandler{
		client: hatchetClient,
	}
}

func (h *HatchetHandler) HandleShipmentMessage(ctx context.Context, msg *model.Message) error {
	log.Printf("Processing shipment message: %s, ID: %s", msg.Type, msg.ID)

	// Push the event to Hatchet
	err := h.client.Event().Push(
		ctx,
		string(msg.Type),
		msg,
		client.WithEventMetadata(map[string]string{
			"tenantId":   msg.TenantID,
			"entityType": msg.EntityType,
			"messageId":  msg.ID,
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to push event to Hatchet: %w", err)
	}

	log.Printf("Successfully pushed shipment event to Hatchet: %s for entity %s",
		msg.Type, msg.EntityID)
	return nil
}
