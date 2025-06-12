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
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

type DuplicateShipmentHandlerParams struct {
	fx.In

	Logger             *logger.Logger
	AuditService       services.AuditService
	ShipmentRepository repositories.ShipmentRepository
}

type DuplicateShipmentHandler struct {
	l            *zerolog.Logger
	shipmentRepo repositories.ShipmentRepository
	as           services.AuditService
}

func NewDuplicateShipmentHandler(p DuplicateShipmentHandlerParams) jobs.JobHandler {
	log := p.Logger.With().
		Str("handler", "duplicate_shipment").
		Logger()

	return &DuplicateShipmentHandler{
		l:            &log,
		shipmentRepo: p.ShipmentRepository,
		as:           p.AuditService,
	}
}

func (dsh *DuplicateShipmentHandler) JobType() jobs.JobType {
	return jobs.JobTypeDuplicateShipment
}

func (dsh *DuplicateShipmentHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	jobStartTime := time.Now()

	log := dsh.l.With().
		Str("job_id", task.ResultWriter().TaskID()).
		Str("job_type", task.Type()).
		Time("job_started_at", jobStartTime).
		Logger()

	log.Info().Msg("starting duplicate shipment job")

	var payload jobs.DuplicateShipmentPayload
	if err := jobs.UnmarshalPayload(task.Payload(), &payload); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal payload")
		return oops.
			In("duplicate_shipment_handler").
			With("payload", task.Payload()).
			Tags("unmarshal").
			Time(time.Now()).
			Wrap(err)
	}

	log.Info().Interface("payload", payload).Msg("payload")

	duplicateReq := &repositories.DuplicateShipmentRequest{
		ShipmentID:               payload.ShipmentID,
		OrgID:                    payload.OrganizationID,
		BuID:                     payload.BusinessUnitID,
		UserID:                   payload.UserID,
		Count:                    payload.Count,
		OverrideDates:            payload.OverrideDates,
		IncludeCommodities:       payload.IncludeCommodities,
		IncludeAdditionalCharges: payload.IncludeAdditionalCharges,
	}

	log.Info().Interface("duplicate_req", duplicateReq).Msg("duplicate shipment request")

	shipments, err := dsh.shipmentRepo.BulkDuplicate(ctx, duplicateReq)
	if err != nil {
		log.Error().Err(err).Msg("failed to duplicate shipment")
		return oops.
			In("duplicate_shipment_handler").
			With("req", duplicateReq).
			Tags("duplicate").
			Time(time.Now()).
			Wrap(err)
	}

	for _, shipment := range shipments {
		// Log the update if the insert was successful
		err = dsh.as.LogAction(
			&services.LogActionParams{
				Resource:       permission.ResourceShipment,
				ResourceID:     payload.ShipmentID.String(),
				Action:         permission.ActionDuplicate,
				UserID:         payload.UserID,
				OrganizationID: payload.OrganizationID,
				BusinessUnitID: payload.BusinessUnitID,
			},
			audit.WithComment("Shipment duplicated"),
			audit.WithCategory("operations"),
			audit.WithMetadata(map[string]any{
				"proNumber":  shipment.ProNumber,
				"customerID": shipment.CustomerID.String(),
			}),
			audit.WithTags(
				"shipment-duplication",
				fmt.Sprintf("customer-%s", shipment.CustomerID.String()),
			),
		)
		if err != nil {
			log.Error().Err(err).Msg("failed to log shipment duplication")
			// ! we will not return an error here because we want to continue the job
			// ! even if the log action fails
		}
	}

	if _, err := task.ResultWriter().Write([]byte(fmt.Sprintf("copied %d shipments", len(shipments)))); err != nil {
		log.Error().Err(err).Msg("failed to write result")
		return oops.
			In("duplicate_shipment_handler").
			Tags("write_result").
			Time(time.Now()).
			Wrap(err)
	}

	log.Info().Int("count", len(shipments)).Msg("shipments duplicated successfully")

	return nil
}
