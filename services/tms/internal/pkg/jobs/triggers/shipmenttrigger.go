/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package triggers

import (
	"time"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// ShipmentTriggerParams defines dependencies for shipment-triggered jobs
type ShipmentTriggerParams struct {
	fx.In

	Logger     *logger.Logger
	JobService services.JobService
}

// ShipmentTrigger handles triggering background jobs based on shipment events
type ShipmentTrigger struct {
	logger     *zerolog.Logger
	jobService services.JobService
}

// ShipmentTriggerInterface defines methods for triggering shipment-related jobs
type ShipmentTriggerInterface interface {
	OnShipmentCreated(shipment *shipment.Shipment) error
	OnShipmentCompleted(shipment *shipment.Shipment) error
	OnShipmentStatusChanged(
		shipment *shipment.Shipment,
		oldStatus, newStatus shipment.Status,
	) error
	TriggerPatternAnalysisForCustomer(
		customerID, orgID, buID, userID pulid.ID,
		reason string,
	) error
}

// NewShipmentTrigger creates a new shipment trigger service
func NewShipmentTrigger(p ShipmentTriggerParams) ShipmentTriggerInterface {
	log := p.Logger.With().
		Str("service", "shipment_trigger").
		Logger()

	return &ShipmentTrigger{
		logger:     &log,
		jobService: p.JobService,
	}
}

// OnShipmentCreated triggers jobs when a new shipment is created
func (st *ShipmentTrigger) OnShipmentCreated(shp *shipment.Shipment) error {
	log := st.logger.With().
		Str("operation", "OnShipmentCreated").
		Str("shipment_id", shp.ID.String()).
		Str("customer_id", shp.CustomerID.String()).
		Str("organization_id", shp.OrganizationID.String()).
		Str("business_unit_id", shp.BusinessUnitID.String()).
		Logger()

	log.Info().
		Str("shipment_status", string(shp.Status)).
		Str("service_type_id", shp.ServiceTypeID.String()).
		Str("shipment_type_id", shp.ShipmentTypeID.String()).
		Msg("processing shipment created event for pattern analysis")

	// Trigger pattern analysis for this customer with a slight delay
	// This gives time for the shipment to be fully committed to the database
	taskInfo, err := st.scheduleDelayedPatternAnalysis(shp, "shipment_created", 30*time.Second)
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to schedule pattern analysis - shipment creation will continue")
		// Don't fail the shipment creation if job scheduling fails
		return nil
	}

	log.Info().
		Str("scheduled_job_id", taskInfo.ID).
		Time("scheduled_for", taskInfo.NextProcessAt).
		Msg("pattern analysis job scheduled successfully")

	return nil
}

// OnShipmentCompleted triggers jobs when a shipment is completed
func (st *ShipmentTrigger) OnShipmentCompleted(shp *shipment.Shipment) error {
	log := st.logger.With().
		Str("operation", "OnShipmentCompleted").
		Str("shipment_id", shp.ID.String()).
		Str("customer_id", shp.CustomerID.String()).
		Str("organization_id", shp.OrganizationID.String()).
		Str("business_unit_id", shp.BusinessUnitID.String()).
		Logger()

	log.Info().
		Str("shipment_status", string(shp.Status)).
		Msg("processing shipment completed event for comprehensive pattern analysis")

	// Trigger pattern analysis for completed shipments since they represent
	// the full pattern that could be automated
	taskInfo, err := st.scheduleDelayedPatternAnalysis(shp, "shipment_completed", 60*time.Second)
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to schedule pattern analysis for completed shipment")
		return nil
	}

	log.Info().
		Str("scheduled_job_id", taskInfo.ID).
		Time("scheduled_for", taskInfo.NextProcessAt).
		Msg("comprehensive pattern analysis scheduled for completed shipment")

	return nil
}

// OnShipmentStatusChanged triggers jobs when shipment status changes
func (st *ShipmentTrigger) OnShipmentStatusChanged(
	shp *shipment.Shipment,
	oldStatus, newStatus shipment.Status,
) error {
	log := st.logger.With().
		Str("operation", "OnShipmentStatusChanged").
		Str("shipment_id", shp.ID.String()).
		Str("customer_id", shp.CustomerID.String()).
		Str("organization_id", shp.OrganizationID.String()).
		Str("business_unit_id", shp.BusinessUnitID.String()).
		Str("old_status", string(oldStatus)).
		Str("new_status", string(newStatus)).
		Logger()

	// Schedule status update notification job
	payload := &services.ShipmentStatusUpdatePayload{
		JobBasePayload: services.JobBasePayload{
			OrganizationID: shp.OrganizationID,
			BusinessUnitID: shp.BusinessUnitID,
			Timestamp:      timeutils.NowUnix(),
		},
		ShipmentID: shp.ID,
		OldStatus:  string(oldStatus),
		NewStatus:  string(newStatus),
	}

	opts := services.CriticalJobOptions()
	taskInfo, err := st.jobService.ScheduleShipmentStatusUpdate(payload, opts)
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to schedule shipment status update job")
		return nil
	}

	log.Info().
		Str("scheduled_job_id", taskInfo.ID).
		Str("queue", opts.Queue).
		Msg("shipment status update job scheduled successfully")

	return nil
}

// TriggerPatternAnalysisForCustomer manually triggers pattern analysis for a specific customer
func (st *ShipmentTrigger) TriggerPatternAnalysisForCustomer(
	customerID, orgID, buID, userID pulid.ID, reason string,
) error {
	log := st.logger.With().
		Str("operation", "TriggerPatternAnalysisForCustomer").
		Str("customer_id", customerID.String()).
		Str("organization_id", orgID.String()).
		Str("business_unit_id", buID.String()).
		Str("user_id", userID.String()).
		Str("reason", reason).
		Logger()

	// Analyze last 90 days of shipments
	endDate := timeutils.NowUnix()
	startDate := endDate - (90 * 86400)

	log.Info().
		Int64("analysis_start_date", startDate).
		Int64("analysis_end_date", endDate).
		Int64("analysis_days", 90).
		Msg("triggering manual pattern analysis for customer") // 90 days ago

	payload := &services.PatternAnalysisPayload{
		JobBasePayload: services.JobBasePayload{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			UserID:         userID,
			Timestamp:      timeutils.NowUnix(),
		},
		MinFrequency:  3, // Look for patterns with at least 3 occurrences
		TriggerReason: reason,
	}

	opts := services.PatternAnalysisOptions()
	opts.UniqueKey = "pattern_analysis_customer_" + customerID.String()

	taskInfo, err := st.jobService.SchedulePatternAnalysis(payload, opts)
	if err != nil {
		log.Error().
			Err(err).
			Str("unique_key", opts.UniqueKey).
			Msg("failed to schedule pattern analysis")
		return err
	}

	log.Info().
		Str("scheduled_job_id", taskInfo.ID).
		Str("queue", opts.Queue).
		Str("unique_key", opts.UniqueKey).
		Time("scheduled_for", taskInfo.NextProcessAt).
		Msg("manual pattern analysis scheduled successfully for customer")

	return nil
}

// scheduleDelayedPatternAnalysis schedules pattern analysis with a delay
func (st *ShipmentTrigger) scheduleDelayedPatternAnalysis(
	shp *shipment.Shipment,
	reason string,
	delay time.Duration,
) (*asynq.TaskInfo, error) {
	payload := &services.PatternAnalysisPayload{
		JobBasePayload: services.JobBasePayload{
			OrganizationID: shp.OrganizationID,
			BusinessUnitID: shp.BusinessUnitID,
			Timestamp:      timeutils.NowUnix(),
		},
		MinFrequency:  2, // More aggressive detection for real-time triggers
		TriggerReason: reason,
	}

	opts := services.PatternAnalysisOptions()
	// Unique key prevents duplicate analysis for same customer within the delay period
	opts.UniqueKey = "pattern_analysis_shipment_" + shp.CustomerID.String()

	return st.jobService.EnqueueIn(services.JobTypeAnalyzePatterns, payload, delay, opts)
}
