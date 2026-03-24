package shipmentjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Repo         repositories.ShipmentRepository
	AuditService services.AuditService
	Realtime     services.RealtimeService
	Logger       *zap.Logger
}

type Activities struct {
	repo         repositories.ShipmentRepository
	auditService services.AuditService
	realtime     services.RealtimeService
	logger       *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		repo:         p.Repo,
		auditService: p.AuditService,
		realtime:     p.Realtime,
		logger:       p.Logger.Named("shipment-activities"),
	}
}

func (a *Activities) BulkDuplicateShipmentsActivity(
	ctx context.Context,
	payload *BulkDuplicateShipmentsPayload,
) (*BulkDuplicateShipmentsResult, error) {
	a.logger.Info(
		"Starting shipment bulk duplication activity",
		zap.String("shipmentId", payload.ShipmentID.String()),
		zap.Int("count", payload.Count),
	)

	recordActivityHeartbeat(ctx, "duplicating shipments")

	duplicated, err := a.repo.BulkDuplicate(ctx, &repositories.BulkDuplicateShipmentRequest{
		TenantInfo:    paginationTenantInfoFromPayload(payload),
		ShipmentID:    payload.ShipmentID,
		Count:         payload.Count,
		OverrideDates: payload.OverrideDates,
	})
	if err != nil {
		a.logger.Error("Shipment bulk duplication failed", zap.Error(err))
		return nil, err
	}

	shipmentIDs := make([]pulid.ID, 0, len(duplicated))
	for _, entity := range duplicated {
		shipmentIDs = append(shipmentIDs, entity.ID)

		if auditErr := a.auditService.LogAction(
			&services.LogActionParams{
				Resource:       permission.ResourceShipment,
				ResourceID:     entity.GetID().String(),
				Operation:      permission.OpCreate,
				UserID:         payload.RequestedBy,
				CurrentState:   jsonutils.MustToJSON(entity),
				OrganizationID: entity.OrganizationID,
				BusinessUnitID: entity.BusinessUnitID,
			},
			auditservice.WithComment("Shipment duplicated"),
		); auditErr != nil {
			a.logger.Error("failed to log duplicated shipment audit action", zap.Error(auditErr))
		}
	}

	if len(duplicated) > 0 {
		if publishErr := realtimeinvalidation.Publish(ctx, a.realtime, &realtimeinvalidation.PublishParams{
			OrganizationID: payload.OrganizationID,
			BusinessUnitID: payload.BusinessUnitID,
			ActorUserID:    payload.RequestedBy,
			ActorType:      services.PrincipalTypeUser,
			ActorID:        payload.RequestedBy,
			Resource:       "shipments",
			Action:         "bulk_created",
		}); publishErr != nil {
			a.logger.Warn(
				"failed to publish duplicated shipment invalidation",
				zap.Error(publishErr),
			)
		}
	}

	return &BulkDuplicateShipmentsResult{
		ShipmentIDs:      shipmentIDs,
		DuplicatedCount:  len(duplicated),
		CompletedAt:      timeutils.NowUnix(),
		SourceShipmentID: payload.ShipmentID,
	}, nil
}

func paginationTenantInfoFromPayload(payload *BulkDuplicateShipmentsPayload) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.RequestedBy,
	}
}

func (a *Activities) AutoDelayShipmentsActivity(
	ctx context.Context,
) (*AutoDelayShipmentsResult, error) {
	a.logger.Info("Starting shipment auto delay activity")
	recordActivityHeartbeat(ctx, "auto-delaying shipments")

	delayedShipments, err := a.repo.AutoDelayShipments(ctx)
	if err != nil {
		a.logger.Error("Shipment auto delay failed", zap.Error(err))
		return nil, err
	}

	shipmentIDs := make([]pulid.ID, 0, len(delayedShipments))
	for _, entity := range delayedShipments {
		shipmentIDs = append(shipmentIDs, entity.ID)

		if publishErr := realtimeinvalidation.Publish(ctx, a.realtime, &realtimeinvalidation.PublishParams{
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
			Resource:       "shipments",
			Action:         "delayed",
			RecordID:       entity.ID,
			Entity:         entity,
		}); publishErr != nil {
			a.logger.Warn("failed to publish shipment delay invalidation", zap.Error(publishErr))
		}
	}

	return &AutoDelayShipmentsResult{
		ShipmentIDs:  shipmentIDs,
		DelayedCount: len(delayedShipments),
		CompletedAt:  timeutils.NowUnix(),
	}, nil
}

func (a *Activities) AutoCancelShipmentsActivity(
	ctx context.Context,
) (*AutoCancelShipmentsResult, error) {
	a.logger.Info("Starting shipment auto cancel activity")
	recordActivityHeartbeat(ctx, "auto-canceling shipments")

	canceledShipments, err := a.repo.RunAutoCancelShipments(ctx)
	if err != nil {
		a.logger.Error("Shipment auto cancel failed", zap.Error(err))
		return nil, err
	}

	shipmentIDs := make([]pulid.ID, 0, len(canceledShipments))
	for _, entity := range canceledShipments {
		shipmentIDs = append(shipmentIDs, entity.ID)
	}

	if len(canceledShipments) > 0 {
		for _, tenantInfo := range uniqueShipmentTenants(canceledShipments) {
			if publishErr := realtimeinvalidation.Publish(ctx, a.realtime, &realtimeinvalidation.PublishParams{
				OrganizationID: tenantInfo.OrgID,
				BusinessUnitID: tenantInfo.BuID,
				Resource:       "shipments",
				Action:         "bulk_canceled",
			}); publishErr != nil {
				a.logger.Warn(
					"failed to publish shipment auto cancel invalidation",
					zap.Error(publishErr),
				)
			}
		}
	}

	return &AutoCancelShipmentsResult{
		ShipmentIDs:   shipmentIDs,
		CanceledCount: len(canceledShipments),
		CompletedAt:   timeutils.NowUnix(),
	}, nil
}

func uniqueShipmentTenants(entities []*shipment.Shipment) []pagination.TenantInfo {
	seen := make(map[string]struct{}, len(entities))
	tenants := make([]pagination.TenantInfo, 0, len(entities))

	for _, entity := range entities {
		key := entity.OrganizationID.String() + ":" + entity.BusinessUnitID.String()
		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}
		tenants = append(tenants, pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		})
	}

	return tenants
}

func recordActivityHeartbeat(ctx context.Context, details ...any) {
	defer func() {
		_ = recover()
	}()

	activity.RecordHeartbeat(ctx, details...)
}
