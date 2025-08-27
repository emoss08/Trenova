/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package shipment

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/infrastructure/external/temporaljobs/payloads"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"go.temporal.io/sdk/activity"
)

// DuplicateShipmentResult contains the result of a duplicate shipment operation
type DuplicateShipmentResult struct {
	ShipmentCount int      `json:"shipmentCount"`
	ProNumbers    []string `json:"proNumbers"`
}

// ActivityProvider provides shipment activities with dependencies
type ActivityProvider struct {
	logger              *zerolog.Logger
	shipmentRepo        repositories.ShipmentRepository
	auditService        services.AuditService
	notificationService services.NotificationService
}

// NewActivityProvider creates a new shipment activity provider
func NewActivityProvider(
	logger *logger.Logger,
	shipmentRepo repositories.ShipmentRepository,
	auditService services.AuditService,
	notificationService services.NotificationService,
) *ActivityProvider {
	log := logger.With().
		Str("component", "temporal-shipment-activities").
		Logger()

	return &ActivityProvider{
		logger:              &log,
		shipmentRepo:        shipmentRepo,
		auditService:        auditService,
		notificationService: notificationService,
	}
}

// DuplicateShipmentActivity duplicates a shipment using injected dependencies
func (p *ActivityProvider) DuplicateShipmentActivity(
	ctx context.Context,
	payload *payloads.DuplicateShipmentPayload,
) (*DuplicateShipmentResult, error) {
	p.logger.Info().
		Str("shipmentId", payload.ShipmentID.String()).
		Int("count", payload.Count).
		Msg("duplicating shipment with dependencies")

	// Record heartbeat for monitoring
	activity.RecordHeartbeat(ctx, "preparing duplication request")

	// Prepare the duplication request
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

	// Execute the duplication
	activity.RecordHeartbeat(ctx, "executing bulk duplication")
	shipments, err := p.shipmentRepo.BulkDuplicate(ctx, duplicateReq)
	if err != nil {
		p.logger.Error().Err(err).Msg("failed to duplicate shipments")
		return nil, fmt.Errorf("failed to duplicate shipments: %w", err)
	}

	// Extract pro numbers for the result
	proNumbers := make([]string, 0, len(shipments))
	for _, shipment := range shipments {
		proNumbers = append(proNumbers, shipment.ProNumber)

		// Log audit entry for each duplicated shipment
		activity.RecordHeartbeat(
			ctx,
			fmt.Sprintf("logging audit for shipment %s", shipment.ProNumber),
		)

		if auditErr := p.auditService.LogAction(
			&services.LogActionParams{
				Resource:       permission.ResourceShipment,
				ResourceID:     payload.ShipmentID.String(),
				Action:         permission.ActionDuplicate,
				UserID:         payload.UserID,
				OrganizationID: payload.OrganizationID,
				BusinessUnitID: payload.BusinessUnitID,
			},
			audit.WithComment("Shipment duplicated via Temporal workflow"),
			audit.WithCategory("operations"),
			audit.WithMetadata(map[string]any{
				"proNumber":      shipment.ProNumber,
				"customerID":     shipment.CustomerID.String(),
				"duplicateCount": payload.Count,
				"workflowEngine": "temporal",
			}),
			audit.WithTags(
				"shipment-duplication",
				"temporal-workflow",
				fmt.Sprintf("customer-%s", shipment.CustomerID.String()),
			),
		); auditErr != nil {
			p.logger.Error().Err(auditErr).
				Str("shipmentId", shipment.ID.String()).
				Msg("failed to log shipment duplication audit")
			// Continue processing even if audit logging fails
		}
	}

	p.logger.Info().
		Int("count", len(shipments)).
		Strs("proNumbers", proNumbers).
		Msg("shipments duplicated successfully")

	return &DuplicateShipmentResult{
		ShipmentCount: len(shipments),
		ProNumbers:    proNumbers,
	}, nil
}


// SendJobCompletionNotificationWithService sends a notification using the injected service
func (p *ActivityProvider) SendJobCompletionNotificationWithService(
	ctx context.Context,
	payload *payloads.JobCompletionNotificationPayload,
) error {
	p.logger.Info().
		Str("jobType", payload.JobType).
		Bool("success", payload.Success).
		Msg("sending job completion notification")

	activity.RecordHeartbeat(ctx, "preparing notification request")

	// Prepare the notification request
	notificationReq := &services.JobCompletionNotificationRequest{
		JobID:          payload.JobID,
		JobType:        payload.JobType,
		UserID:         payload.UserID,
		OrganizationID: payload.OrganizationID,
		BusinessUnitID: payload.BusinessUnitID,
		Success:        payload.Success,
		Result:         payload.Result,
		Data:           payload.Data,
	}

	// Send the notification
	if err := p.notificationService.SendJobCompletionNotification(ctx, notificationReq); err != nil {
		p.logger.Error().Err(err).Msg("failed to send job completion notification")
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return nil
}

// ActivityDefinition defines an activity with its configuration
type ActivityDefinition struct {
	Name        string
	Fn          any
	Description string
}

// RegisterActivities registers all shipment-related activities
func RegisterActivities() []ActivityDefinition {
	// These are placeholder activities that will be replaced by the provider methods
	// The actual activities are registered in RegisterActivityProviders
	return []ActivityDefinition{}
}
