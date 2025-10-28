package shipmentjobs

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/core/temporaljobs/searchjobs"
	"github.com/emoss08/trenova/pkg/meilisearchtype"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	ShipmentRepository repositories.ShipmentRepository
	AuditService       services.AuditService
	TemporalClient     client.Client
}

type Activities struct {
	sr             repositories.ShipmentRepository
	as             services.AuditService
	temporalClient client.Client
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		sr:             p.ShipmentRepository,
		as:             p.AuditService,
		temporalClient: p.TemporalClient,
	}
}

func (a *Activities) BulkDuplicateShipmentActivity(
	ctx context.Context,
	payload *DuplicateShipmentPayload,
) (*DuplicateShipmentResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting bulk duplicate shipment activity",
		"shipmentID", payload.ShipmentID.String(),
		"count", payload.Count,
		"overrideDates", payload.OverrideDates,
		"includeCommodities", payload.IncludeCommodities,
		"includeAdditionalCharges", payload.IncludeAdditionalCharges,
	)

	activity.RecordHeartbeat(ctx, "validating payload")
	if payload.Count <= 0 {
		return nil, temporaltype.NewInvalidInputError(
			"Count must be greater than 0",
			map[string]any{
				"count":      payload.Count,
				"shipmentID": payload.ShipmentID.String(),
			},
		).ToTemporalError()
	}

	if payload.Count > 100 {
		return nil, temporaltype.NewInvalidInputError(
			"Count must be less than 100",
			map[string]any{
				"count":      payload.Count,
				"shipmentID": payload.ShipmentID.String(),
			},
		).ToTemporalError()
	}

	activity.RecordHeartbeat(ctx, "preparing shipment duplication request")
	req := &repositories.DuplicateShipmentRequest{
		ShipmentID:               payload.ShipmentID,
		OrgID:                    payload.OrganizationID,
		BuID:                     payload.BusinessUnitID,
		UserID:                   payload.UserID,
		Count:                    payload.Count,
		OverrideDates:            payload.OverrideDates,
		IncludeCommodities:       payload.IncludeCommodities,
		IncludeAdditionalCharges: payload.IncludeAdditionalCharges,
	}

	activity.RecordHeartbeat(ctx, "duplicating shipments")
	shipments, err := a.sr.BulkDuplicate(ctx, req)
	if err != nil {
		logger.Error("Failed to duplicate shipment", "error", err)

		appErr := temporaltype.ClassifyError(err)
		if appErr.Type == temporaltype.ErrorTypeResourceNotFound {
			return nil, temporaltype.NewNonRetryableError(
				"Shipment not found",
				err,
			).ToTemporalError()
		}

		return nil, appErr.ToTemporalError()
	}

	proNumbers := make([]string, 0, len(shipments))
	for _, shp := range shipments {
		proNumbers = append(proNumbers, shp.ProNumber)
	}

	shipmentIDs := make([]pulid.ID, 0, len(shipments))
	for _, shp := range shipments {
		shipmentIDs = append(shipmentIDs, shp.ID)
	}

	activity.RecordHeartbeat(ctx, "logging shipment duplication")
	err = a.as.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceShipment,
		ResourceID:     payload.ShipmentID.String(),
		Operation:      permission.OpDuplicate,
		UserID:         payload.UserID,
		OrganizationID: payload.OrganizationID,
		BusinessUnitID: payload.BusinessUnitID,
	},
		audit.WithComment("Shipment duplicated"),
		audit.WithCategory("operations"),
		audit.WithMetadata(map[string]any{
			"proNumbers": proNumbers,
			"shipmentID": payload.ShipmentID.String(),
		}),
	)
	if err != nil {
		logger.Error("failed to log shipment duplication", zap.Error(err))
	}

	bp := &searchjobs.BulkIndexEntityPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: payload.OrganizationID,
			BusinessUnitID: payload.BusinessUnitID,
		},
		EntityType: meilisearchtype.EntityTypeShipment,
		EntityIDs:  shipmentIDs,
	}

	workflowID := fmt.Sprintf(
		"bulk-index-shipments-%s-%d",
		payload.ShipmentID.String(),
		time.Now().Unix(),
	)

	_, err = a.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: searchjobs.SearchTaskQueue,
		},
		searchjobs.BulkIndexEntityWorkflow,
		bp,
	)
	if err != nil {
		logger.Error("failed to execute workflow", zap.Error(err))
		return nil, temporaltype.NewRetryableError("failed to bulk index shipments", err).
			ToTemporalError()
	}

	return &DuplicateShipmentResult{
		Count:       len(shipments),
		ShipmentIDs: shipmentIDs,
		ProNumbers:  proNumbers,
		Result:      fmt.Sprintf("Successfully duplicated %d shipments", len(shipments)),
		Data: map[string]any{
			"shipmentCount":    len(shipments),
			"originalShipment": payload.ShipmentID.String(),
		},
	}, nil
}
