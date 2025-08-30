/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package shipmentjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	ShipmentRepository        repositories.ShipmentRepository
	ShipmentControlRepository repositories.ShipmentControlRepository
	AuditService              services.AuditService
}

type Activities struct {
	sr  repositories.ShipmentRepository
	scr repositories.ShipmentControlRepository
	as  services.AuditService
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		sr:  p.ShipmentRepository,
		scr: p.ShipmentControlRepository,
		as:  p.AuditService,
	}
}

func (sja *Activities) DuplicateShipmentActivity(
	ctx context.Context,
	payload *DuplicateShipmentPayload,
) (*DuplicateShipmentResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info(
		"Starting duplicate shipment activity... Duplicating %d shipments",
		payload.Count,
	)

	if payload.Count <= 0 {
		return nil, temporaltype.NewInvalidInputError(
			"Invalid duplication count",
			map[string]any{
				"count":      payload.Count,
				"shipmentID": payload.ShipmentID.String(),
			},
		).ToTemporalError()
	}

	if payload.Count > 100 {
		return nil, temporaltype.NewInvalidInputError(
			"Duplication count exceeds maximum limit of 100",
			map[string]any{
				"requested": payload.Count,
				"maximum":   100,
			},
		).ToTemporalError()
	}

	activity.RecordHeartbeat(ctx, "preparing duplication request")
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

	activity.RecordHeartbeat(ctx, "executing bulk duplication")
	shipments, err := sja.sr.BulkDuplicate(ctx, duplicateReq)
	if err != nil {
		logger.Error("Failed to duplicate shipments: %v", err)
		appErr := temporaltype.ClassifyError(err)
		if appErr.Type == temporaltype.ErrorTypeResourceNotFound {
			return nil, temporaltype.NewResourceNotFoundError(
				"Shipment",
				payload.ShipmentID.String(),
			).ToTemporalError()
		}
		return nil, appErr.ToTemporalError()
	}

	proNumbers := make([]string, 0, len(shipments))
	for _, shipment := range shipments {
		proNumbers = append(proNumbers, shipment.ProNumber)

		activity.RecordHeartbeat(
			ctx,
			fmt.Sprintf("logging audit action for shipment %s", shipment.ProNumber),
		)

		err = sja.as.LogAction(
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
			logger.Error("Failed to log shipment duplication: %v", err)
			// ! we will not return an error here because we want to continue the job
			// ! even if the log action fails
		}
	}

	return &DuplicateShipmentResult{
		Count:      len(shipments),
		ProNumbers: proNumbers,
		Result:     fmt.Sprintf("Successfully duplicated %d shipments", len(shipments)),
		Data: map[string]any{
			"shipmentCount":    len(shipments),
			"originalShipment": payload.ShipmentID.String(),
		},
	}, nil
}

func (sja *Activities) CancelShipmentsByCreatedAtActivity(
	ctx context.Context,
) (*CancelShipmentsByCreatedAtResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting cancel shipments by created at activity")

	activity.RecordHeartbeat(ctx, "fetching shipment controls")

	scs, err := sja.scr.List(ctx)
	if err != nil {
		logger.Error("Failed to get shipment controls: %v", err)
		return nil, temporaltype.NewRetryableError(
			"Failed to fetch shipment controls",
			err,
		).ToTemporalError()
	}

	skippedOrgs := make([]pulid.ID, 0)
	processedOrgs := make([]OrgCancellationResult, 0)
	totalCancelled := 0

	logger.Info("Processing %d organizations for shipment cancellation", len(scs))

	for _, sc := range scs {
		if !sc.AutoCancelShipments {
			logger.Info(
				"Skipping organization %s - auto cancel shipments is disabled",
				sc.OrganizationID,
			)
			skippedOrgs = append(skippedOrgs, sc.OrganizationID)
			continue // * skip the organization if auto cancel shipments is disabled
		}

		if sc.AutoCancelShipmentsThreshold == nil {
			logger.Info(
				"Skipping organization %s - auto cancel shipments threshold is not configured",
				sc.OrganizationID,
			)
			skippedOrgs = append(skippedOrgs, sc.OrganizationID)
			continue
		}

		activity.RecordHeartbeat(
			ctx,
			fmt.Sprintf("processing organization %s", sc.OrganizationID),
		)

		now := timeutils.NowUnix()
		daysInSeconds := int64(*sc.AutoCancelShipmentsThreshold * 24 * 60 * 60)
		thresholdDate := now - daysInSeconds

		logger.Info(
			"Processing organization %s for auto cancel with threshold %d days (created before %d)",
			sc.OrganizationID,
			*sc.AutoCancelShipmentsThreshold,
			thresholdDate,
		)

		shipments, err := sja.sr.BulkCancelShipmentsByCreatedAt(
			ctx,
			&repositories.BulkCancelShipmentsByCreatedAtRequest{
				OrgID:     sc.OrganizationID,
				BuID:      sc.BusinessUnitID,
				CreatedAt: thresholdDate,
			},
		)
		if err != nil {
			logger.Error(
				"Failed to cancel shipments for organization %s: %v",
				sc.OrganizationID,
				err,
			)
			continue
		}

		if len(shipments) > 0 {
			proNumbers := make([]string, 0, len(shipments))
			for _, shipment := range shipments {
				proNumbers = append(proNumbers, shipment.ProNumber)
			}

			orgResult := OrgCancellationResult{
				OrganizationID:   sc.OrganizationID,
				BusinessUnitID:   sc.BusinessUnitID,
				CancelledCount:   len(shipments),
				CancelledProNums: proNumbers,
			}
			processedOrgs = append(processedOrgs, orgResult)
			totalCancelled += len(shipments)

			logger.Info(
				"Cancelled %d shipments for organization %s",
				len(shipments),
				sc.OrganizationID,
			)
		}

		activity.RecordHeartbeat(
			ctx,
			fmt.Sprintf(
				"completed processing organization %s - cancelled %d shipments",
				sc.OrganizationID,
				len(shipments),
			),
		)
	}

	resultMessage := fmt.Sprintf(
		"Successfully processed %d organizations: %d shipments cancelled, %d organizations skipped",
		len(processedOrgs),
		totalCancelled,
		len(skippedOrgs),
	)

	logger.Info(resultMessage)

	return &CancelShipmentsByCreatedAtResult{
		TotalCancelled: totalCancelled,
		SkippedOrgs:    skippedOrgs,
		ProcessedOrgs:  processedOrgs,
		Result:         resultMessage,
		Data: map[string]any{
			"totalOrganizations": len(scs),
			"processedCount":     len(processedOrgs),
			"skippedCount":       len(skippedOrgs),
		},
	}, nil
}
