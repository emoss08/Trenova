package capturejobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/captureservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Capture *captureservice.Service
	Items   repositories.CaptureItemRepository
	Logger  *zap.Logger
}

type Activities struct {
	capture *captureservice.Service
	items   repositories.CaptureItemRepository
	l       *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		capture: p.Capture,
		items:   p.Items,
		l:       p.Logger.Named("capture-activities"),
	}
}

// ProcessCaptureBatchActivity reads a batch, heartbeating per page so a
// worker lost in the middle of a long stack is noticed in minutes.
//
// The names carry their domain because Temporal's activity registry is keyed
// by method name across a task queue.
func (a *Activities) ProcessCaptureBatchActivity(
	ctx context.Context,
	payload *ProcessBatchPayload,
) (*ProcessResult, error) {
	return a.capture.ProcessBatch(
		ctx,
		pagination.TenantInfo{OrgID: payload.OrganizationID, BuID: payload.BusinessUnitID},
		payload.BatchID,
		func(pagesRead int) {
			activity.RecordHeartbeat(ctx, pagesRead)
		},
	)
}

// AutoFileCaptureItemActivity files an item as the person who captured it,
// to wherever processing proposed. The filing runs every check a person's
// own filing would, so an item this person could not file stays in intake.
func (a *Activities) AutoFileCaptureItemActivity(
	ctx context.Context,
	payload *AutoFilePayload,
) error {
	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}

	item, err := a.items.GetByID(
		ctx,
		repositories.GetCaptureItemByIDRequest{ID: payload.ItemID, TenantInfo: tenantInfo},
	)
	if err != nil {
		return err
	}
	suggestion := item.Suggestion()
	if item.Status != capture.ItemProposed || !suggestion.HasRecord() {
		return nil
	}

	_, err = a.capture.FileItem(ctx, &captureservice.FileItemInput{
		TenantInfo:     tenantInfo,
		ItemID:         item.ID,
		TargetType:     suggestion.ResourceType,
		TargetID:       *suggestion.ResourceID,
		DocumentTypeID: suggestion.DocumentTypeID,
		Automatic:      true,
	})
	if err != nil {
		a.l.Info("an automatic filing was refused; the item stays in intake",
			zap.String("itemId", item.ID.String()), zap.Error(err))
	}

	return nil
}

func (a *Activities) RecordCaptureFiledActivity(
	ctx context.Context,
	payload *RecordFiledPayload,
) error {
	_, err := a.capture.RecordFiled(
		ctx,
		tenantOf(&payload.FileItemPayload),
		payload.ItemID,
		payload.DocumentID,
	)

	return err
}

func (a *Activities) RecordCaptureFilingFailedActivity(
	ctx context.Context,
	payload *RecordFailedPayload,
) error {
	_, err := a.capture.RecordFilingFailed(
		ctx,
		tenantOf(&payload.FileItemPayload),
		payload.ItemID,
		payload.Message,
	)

	return err
}

func (a *Activities) CaptureMaintenanceActivity(ctx context.Context) (*MaintenanceResult, error) {
	return a.capture.Maintain(ctx)
}

func tenantOf(payload *FileItemPayload) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}
}
