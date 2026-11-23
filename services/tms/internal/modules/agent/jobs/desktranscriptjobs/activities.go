package desktranscriptjobs

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/app/desktranscriptexport"
	"github.com/emoss08/trenova/internal/modules/agent/domain/desktranscript"
	"github.com/emoss08/trenova/internal/modules/agent/ports/desktranscriptports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
)

const (
	errTypeExportNotFound = "DeskTranscriptExportNotFound"
	errTypeInvalidInput   = "InvalidInput"
)

type ExportResult struct {
	ExportID          string                      `json:"exportId"`
	Status            desktranscript.ExportStatus `json:"status"`
	ConversationCount int                         `json:"conversationCount"`
}

type CleanupResult = desktranscriptexport.CleanupResult

func heartbeat(ctx context.Context) desktranscriptexport.Heartbeat {
	return func(details ...any) {
		activity.RecordHeartbeat(ctx, details...)
	}
}

type ActivitiesParams struct {
	fx.In

	Exports *desktranscriptexport.Exports
	Slots   kernelports.HeavyJobSlots
}

type Activities struct {
	exports *desktranscriptexport.Exports
	slots   kernelports.HeavyJobSlots
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{exports: p.Exports, slots: p.Slots}
}

func (a *Activities) RunDeskTranscriptExportActivity(
	ctx context.Context,
	payload *desktranscriptports.ExportPayload,
) (*ExportResult, error) {
	if payload == nil || payload.ExportID.IsNil() || payload.OrganizationID.IsNil() ||
		payload.BusinessUnitID.IsNil() {
		return nil, temporal.NewNonRetryableApplicationError(
			"an export needs its id and tenant", errTypeInvalidInput, nil,
		)
	}
	ctx = dbscope.WithValidTenant(ctx, payload.TenantInfo().DBTenant())
	release, err := a.slots.Hold(ctx, kernelports.HeavyJobExport)
	if err != nil {
		return nil, err
	}
	defer release()

	export, err := a.exports.Run(ctx, payload.TenantInfo(), payload.ExportID, heartbeat(ctx))
	if err != nil {
		var notFound *errortypes.NotFoundError
		if errors.As(err, &notFound) {
			return nil, temporal.NewNonRetryableApplicationError(
				err.Error(), errTypeExportNotFound, err,
			)
		}

		return nil, err
	}

	return &ExportResult{
		ExportID:          export.ID.String(),
		Status:            export.Status,
		ConversationCount: export.ConversationCount,
	}, nil
}

func (a *Activities) CleanupDeskTranscriptExportsActivity(
	ctx context.Context,
) (*CleanupResult, error) {
	return a.exports.Cleanup(ctx, heartbeat(ctx))
}
