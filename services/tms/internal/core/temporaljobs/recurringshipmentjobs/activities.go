package recurringshipmentjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxDueSeriesPerTick = 200

	dataKeySeriesID   = "recurringShipmentId"
	dataKeySeriesName = "recurringShipmentName"

	eventTypeGenerationFailed  = "recurring_shipment_generation_failed"
	eventTypeOccurrenceSkipped = "recurring_shipment_occurrence_skipped"
	eventTypeSeriesPaused      = "recurring_shipment_paused"
	eventTypeSeriesExpired     = "recurring_shipment_expired"
)

type ActivitiesParams struct {
	fx.In

	Repo                repositories.RecurringShipmentRepository
	AuditService        services.AuditService
	NotificationService *notificationservice.Service
	Realtime            services.RealtimeService `optional:"true"`
	Logger              *zap.Logger
}

type Activities struct {
	repo         repositories.RecurringShipmentRepository
	auditService services.AuditService
	notification *notificationservice.Service
	realtime     services.RealtimeService
	logger       *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		repo:         p.Repo,
		auditService: p.AuditService,
		notification: p.NotificationService,
		realtime:     p.Realtime,
		logger:       p.Logger.Named("recurring-shipment-activities"),
	}
}

func (a *Activities) DispatchDueRecurringShipmentsActivity(
	ctx context.Context,
) (*DispatchDueRecurringShipmentsResult, error) {
	result := &DispatchDueRecurringShipmentsResult{}
	now := timeutils.NowUnix()

	due, err := a.repo.ListDue(ctx, &repositories.ListDueRecurringShipmentsRequest{
		Now:   now,
		Limit: maxDueSeriesPerTick,
	})
	if err != nil {
		a.logger.Error("failed to list due recurring shipments", zap.Error(err))
		return nil, err
	}

	for _, series := range due {
		recordActivityHeartbeat(ctx, series.ID.String())
		a.dispatchSeries(ctx, series, result)
	}

	result.CompletedAt = timeutils.NowUnix()

	return result, nil
}

func (a *Activities) dispatchSeries(
	ctx context.Context,
	series *recurringshipment.RecurringShipment,
	result *DispatchDueRecurringShipmentsResult,
) {
	log := a.logger.With(
		zap.String("recurringShipmentId", series.ID.String()),
		zap.String("orgId", series.OrganizationID.String()),
	)

	tenantInfo := pagination.TenantInfo{
		OrgID:  series.OrganizationID,
		BuID:   series.BusinessUnitID,
		UserID: series.EnteredByID,
	}

	generation, err := a.repo.Generate(ctx, &repositories.GenerateRecurringShipmentRequest{
		TenantInfo:          tenantInfo,
		RecurringShipmentID: series.ID,
		Trigger:             recurringshipment.RunTriggerAuto,
		RequestedBy:         series.EnteredByID,
	})
	if err != nil {
		log.Error("recurring shipment generation failed", zap.Error(err))
		result.Failed++
		a.recordFailure(ctx, series, tenantInfo, err)
		return
	}

	if generation.Shipment == nil {
		result.Skipped++
		if generation.Run != nil &&
			generation.Run.Status == recurringshipment.RunStatusSkipped &&
			generation.Series != nil {
			a.notifySeriesOwner(ctx, generation.Series,
				eventTypeOccurrenceSkipped,
				"Recurring shipment occurrence skipped",
				fmt.Sprintf(
					"An occurrence of %q was skipped: %s",
					generation.Series.Name, generation.Run.Detail,
				))
		}
		a.notifyIfExpired(ctx, generation.Series)
		return
	}

	result.Dispatched++

	if auditErr := a.auditService.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipment,
			ResourceID:     generation.Shipment.GetID().String(),
			Operation:      permission.OpCreate,
			UserID:         series.EnteredByID,
			CurrentState:   jsonutils.MustToJSON(generation.Shipment),
			OrganizationID: generation.Shipment.OrganizationID,
			BusinessUnitID: generation.Shipment.BusinessUnitID,
		},
		auditservice.WithComment(
			fmt.Sprintf("Shipment generated from recurring series %q", series.Name),
		),
	); auditErr != nil {
		log.Error("failed to log generated shipment audit action", zap.Error(auditErr))
	}

	if publishErr := realtimeinvalidation.Publish(
		ctx,
		a.realtime,
		&realtimeinvalidation.PublishParams{
			OrganizationID: generation.Shipment.OrganizationID,
			BusinessUnitID: generation.Shipment.BusinessUnitID,
			Resource:       "shipments",
			Action:         "created",
			RecordID:       generation.Shipment.ID,
			Entity:         generation.Shipment,
		},
	); publishErr != nil {
		log.Warn("failed to publish generated shipment invalidation", zap.Error(publishErr))
	}

	a.notifyIfExpired(ctx, generation.Series)
}

func (a *Activities) recordFailure(
	ctx context.Context,
	series *recurringshipment.RecurringShipment,
	tenantInfo pagination.TenantInfo,
	genErr error,
) {
	occurrenceAt := timeutils.NowUnix()
	if series.NextOccurrenceAt != nil {
		occurrenceAt = *series.NextOccurrenceAt
	}

	updated, err := a.repo.RecordGenerationFailure(
		ctx,
		&repositories.RecordRecurringGenerationFailureRequest{
			TenantInfo:          tenantInfo,
			RecurringShipmentID: series.ID,
			OccurrenceAt:        occurrenceAt,
			Detail:              genErr.Error(),
		},
	)
	if err != nil {
		a.logger.Error("failed to record recurring generation failure", zap.Error(err))
		return
	}

	if updated.Status == recurringshipment.StatusPaused {
		a.notifySeriesOwner(ctx, updated,
			eventTypeSeriesPaused,
			"Recurring shipment paused",
			fmt.Sprintf(
				"%q failed %d consecutive times and has been paused. Review the series and resume it once the issue is resolved.",
				updated.Name, updated.ConsecutiveFailures,
			))
		return
	}

	a.notifySeriesOwner(ctx, updated,
		eventTypeGenerationFailed,
		"Recurring shipment generation failed",
		fmt.Sprintf(
			"A shipment could not be generated from %q. The next occurrence will still be attempted on schedule.",
			updated.Name,
		))
}

func (a *Activities) notifyIfExpired(
	ctx context.Context,
	series *recurringshipment.RecurringShipment,
) {
	if series == nil || series.Status != recurringshipment.StatusExpired {
		return
	}

	a.notifySeriesOwner(ctx, series,
		eventTypeSeriesExpired,
		"Recurring shipment completed",
		fmt.Sprintf(
			"%q has reached the end of its schedule and will no longer generate shipments.",
			series.Name,
		))
}

func (a *Activities) notifySeriesOwner(
	ctx context.Context,
	series *recurringshipment.RecurringShipment,
	eventType, title, message string,
) {
	if series.EnteredByID.IsNil() {
		return
	}

	if _, err := a.notification.Create(ctx, &notification.Notification{
		OrganizationID: series.OrganizationID,
		BusinessUnitID: &series.BusinessUnitID,
		TargetUserID:   &series.EnteredByID,
		Channel:        notification.ChannelUser,
		EventType:      eventType,
		Priority:       notification.PriorityHigh,
		Title:          title,
		Message:        message,
		Data: map[string]any{
			dataKeySeriesID:   series.ID.String(),
			dataKeySeriesName: series.Name,
		},
		Source: "recurringshipmentjobs.DispatchDueRecurringShipments",
	}); err != nil {
		a.logger.Warn("failed to notify recurring shipment owner",
			zap.String("recurringShipmentId", series.ID.String()), zap.Error(err))
	}
}

func recordActivityHeartbeat(ctx context.Context, details ...any) {
	defer func() {
		_ = recover()
	}()

	activity.RecordHeartbeat(ctx, details...)
}
