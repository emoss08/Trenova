package workflow

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/messaging/rabbitmq"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Publisher *rabbitmq.WorkflowPublisher
	Logger    *logger.Logger
}

type Service struct {
	publisher *rabbitmq.WorkflowPublisher
	l         *zerolog.Logger
}

func NewService(p Params) *Service {
	log := p.Logger.With().
		Str("service", "workflow").
		Logger()

	return &Service{
		publisher: p.Publisher,
		l:         &log,
	}
}

func (s *Service) TriggerShipmentWorkflow(
	ctx context.Context,
	workflowType Type,
	shipmentID pulid.ID,
	tenantID pulid.ID,
	payload *ShipmentWorkflowPayload,
) error {
	msg := NewMessage(workflowType, shipmentID, "shipment", tenantID, payload)

	s.l.Info().
		Str("workflowType", string(workflowType)).
		Str("shipmentID", shipmentID.String()).
		Str("tenantID", tenantID.String()).
		Msg("triggering shipment workflow")

	return s.publisher.Publish(ctx, string(workflowType), msg)
}
