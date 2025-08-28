package shipmentjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	ShipmentRepository repositories.ShipmentRepository
	AuditService       services.AuditService
}

type Activities struct {
	sr repositories.ShipmentRepository
	as services.AuditService
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		sr: p.ShipmentRepository,
		as: p.AuditService,
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
		return nil, err
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
