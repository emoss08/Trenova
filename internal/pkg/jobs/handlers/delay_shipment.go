package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/jobs"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

type DelayShipmentHandlerParams struct {
	fx.In

	Logger              *logger.Logger
	AuditService        services.AuditService
	ShipmentRepository  repositories.ShipmentRepository
	NotificationService services.NotificationService
}

type DelayShipmentHandler struct {
	l                   *zerolog.Logger
	shipmentRepo        repositories.ShipmentRepository
	as                  services.AuditService
	notificationService services.NotificationService
}

func NewDelayShipmentHandler(p DelayShipmentHandlerParams) jobs.JobHandler {
	log := p.Logger.With().
		Str("handler", "delay_shipment").
		Logger()

	return &DelayShipmentHandler{
		l:                   &log,
		shipmentRepo:        p.ShipmentRepository,
		as:                  p.AuditService,
		notificationService: p.NotificationService,
	}
}

func (dsh *DelayShipmentHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	jobStartTime := time.Now()

	log := dsh.l.With().
		Str("jobID", task.ResultWriter().TaskID()).
		Str("jobType", task.Type()).
		Time("jobStartedAt", jobStartTime).
		Logger()

	log.Info().Msg("starting delay shipment job")

	var payload jobs.DelayShipmentPayload
	if err := jobs.UnmarshalPayload(task.Payload(), &payload); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal payload")
		return oops.
			In("delay_shipment_handler").
			With("payload", task.Payload()).
			Tags("unmarshal").
			Time(time.Now()).
			Wrap(err)
	}

	log.Info().Interface("payload", payload).Msg("payload")

	shipments, err := dsh.shipmentRepo.DelayShipments(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to process delayed shipments")
		if _, err := task.ResultWriter().Write(fmt.Appendf(nil, "failed to process delayed shipments: %s", err.Error())); err != nil {
			log.Error().Err(err).Msg("failed to write result")
		}

		return oops.
			In("delay_shipment_handler").
			With("payload", payload).
			Tags("process_delayed_shipments").
			Time(time.Now()).
			Wrap(err)
	}

	if len(shipments) == 0 {
		log.Info().Msg("no shipments found that need to be delayed")
		if _, err := task.ResultWriter().Write(fmt.Appendf(nil, "no shipments found that need to be delayed")); err != nil {
			log.Error().Err(err).Msg("failed to write result")
		}
		return nil
	}

	processedCount := 0
	for _, shp := range shipments {
		if shp.OwnerID != nil {
			notificationReq := &services.JobCompletionNotificationRequest{
				JobID:          task.ResultWriter().TaskID(),
				JobType:        "delay_shipment",
				UserID:         pulid.ConvertFromPtr(shp.OwnerID),
				OrganizationID: shp.OrganizationID,
				BusinessUnitID: shp.BusinessUnitID,
				Success:        true,
				Result: fmt.Sprintf(
					"Shipment %s has been marked as delayed due to missed appointment window. Please update the shipment schedule.",
					shp.ProNumber,
				),
				Data: map[string]any{
					"shipmentId": shp.ID.String(),
					"proNumber":  shp.ProNumber,
					"status":     "Delayed",
				},
			}

			if err := dsh.notificationService.SendJobCompletionNotification(ctx, notificationReq); err != nil {
				log.Error().
					Err(err).
					Str("shipmentID", shp.ID.String()).
					Msg("failed to send delay notification")
			}
		}

		err := dsh.as.LogAction(
			&services.LogActionParams{
				Resource:       permission.ResourceShipment,
				ResourceID:     shp.ID.String(),
				Action:         permission.ActionUpdate,
				UserID:         payload.UserID,
				OrganizationID: shp.OrganizationID,
				BusinessUnitID: shp.BusinessUnitID,
			},
			audit.WithComment(
				"Shipment automatically marked as delayed due to missed appointment",
			),
			audit.WithCategory("operations"),
			audit.WithMetadata(map[string]any{
				"proNumber":      shp.ProNumber,
				"customerID":     shp.CustomerID.String(),
				"previousStatus": "InTransit", // _ Assumed previous status
				"newStatus":      "Delayed",
			}),
			audit.WithTags(
				"shipment-delay",
				"automated-delay",
				fmt.Sprintf("customer-%s", shp.CustomerID.String()),
			),
		)
		if err != nil {
			log.Error().
				Err(err).
				Str("shipmentID", shp.ID.String()).
				Msg("failed to log delay action")
			// ! Continue processing other shipments even if audit logging fails
		}

		processedCount++
	}

	if _, err := task.ResultWriter().Write(fmt.Appendf(nil, "Successfully delayed %d shipments", processedCount)); err != nil {
		log.Error().Err(err).Msg("failed to write result")
		return oops.
			In("delay_shipment_handler").
			Tags("write_result").
			Time(time.Now()).
			Wrap(err)
	}

	log.Info().
		Int("total", len(shipments)).
		Int("processed", processedCount).
		Msg("delay shipment job completed successfully")

	return nil
}

func (dsh *DelayShipmentHandler) JobType() jobs.JobType {
	return jobs.JobTypeDelayShipment
}
