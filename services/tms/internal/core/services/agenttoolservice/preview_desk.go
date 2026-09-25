package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/carrierintelservice"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/internal/core/services/workerservice"
	"github.com/emoss08/trenova/shared/timeutils"
)

var (
	_ serviceports.ToolPreviewer = (*escalateDetentionTool)(nil)
	_ serviceports.ToolPreviewer = (*placeWorkerDispatchHoldTool)(nil)
	_ serviceports.ToolPreviewer = (*acknowledgeCarrierIntelEventTool)(nil)
	_ serviceports.ToolPreviewer = (*resolveCarrierIntelEventTool)(nil)
)

var detentionEscalationFields = []string{"requiresApproval", "notificationStatus"}

var detentionEvidenceFields = []string{fieldKind, "source", "summary"}

func detentionOccurrenceRecord(occurrence *detention.DetentionOccurrence) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceDetentionPolicy,
		ID:       occurrence.ID,
		Label:    fmt.Sprintf("Detention, %d billable minutes", occurrence.BillableMinutes),
		Version:  previewVersion(occurrence.Version),
	}
}

func (t *escalateDetentionTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	occurrenceID, reason, err := t.arguments(params)
	if err != nil {
		return nil, err
	}

	occurrence, err := t.occurrence(ctx, occurrenceID, params)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	plan, err := planUpdate(
		detentionOccurrenceRecord(occurrence),
		occurrence,
		func(escalated *detention.DetentionOccurrence) error {
			return escalated.Escalate(now)
		},
		toolpreview.Only(detentionEscalationFields...),
	)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would put this detention occurrence (%d billable minutes) in front of a person: %s",
		occurrence.BillableMinutes,
		reason,
	)
	evidence, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceDetentionPolicy,
		Label:    "Detention evidence",
	}, detentionservice.EscalationEvidence(occurrence, reason, now),
		toolpreview.Only(detentionEvidenceFields...),
	)
	if err != nil {
		return nil, err
	}

	return plan.preview(summary, evidence), nil
}

func (t *placeWorkerDispatchHoldTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.arguments(params)
	if err != nil {
		return nil, err
	}

	driver, err := t.workers.Get(ctx, repositories.GetWorkerByIDRequest{
		ID:         request.WorkerID,
		TenantInfo: request.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	plan, err := planUpdate(toolpreview.Record{
		Resource: permission.ResourceWorker,
		ID:       driver.ID,
		Label:    driver.FullName(),
		Version:  previewVersion(driver.Version),
	}, driver, func(held *worker.Worker) error {
		return workerservice.ApplyDispatchHold(held, &request)
	}, toolpreview.Only("canBeAssigned"))
	if err != nil {
		return nil, err
	}

	return plan.preview(fmt.Sprintf(
		"Would stop %s being given new freight: %s",
		driver.FullName(),
		request.Reason,
	)), nil
}

var carrierIntelVolatileFields = []string{"acknowledgedAt", "resolvedAt"}

func carrierIntelRecord(event *carrierintel.CarrierIntelEvent) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceCarrierIntelligence,
		ID:       event.ID,
		Label:    event.Summary,
		Version:  previewVersion(event.Version),
	}
}

func (t *carrierIntelEventTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.arguments(params)
	if err != nil {
		return nil, err
	}

	event, err := t.intel.GetEvent(ctx, request.TenantInfo, request.EventID)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	opts := []toolpreview.Option{toolpreview.Volatile(carrierIntelVolatileFields...)}
	if !t.resolves {
		return t.previewAcknowledge(event, request, now, opts)
	}

	plan, err := planUpdate(
		carrierIntelRecord(event),
		event,
		func(closed *carrierintel.CarrierIntelEvent) error {
			return carrierintelservice.PlanResolve(closed, request, now)
		},
		opts...,
	)
	if err != nil {
		return nil, err
	}

	return plan.preview(fmt.Sprintf(
		"Would close the %s carrier finding %q as %s: %s",
		event.Severity,
		event.Summary,
		request.Resolution,
		request.Note,
	)), nil
}

func (t *carrierIntelEventTool) previewAcknowledge(
	event *carrierintel.CarrierIntelEvent,
	request *carrierintelservice.ResolveEventRequest,
	now int64,
	opts []toolpreview.Option,
) (*agent.ToolPreview, error) {
	summary := fmt.Sprintf(
		"Would mark the %s carrier finding %q as seen: %s",
		event.Severity,
		event.Summary,
		request.Note,
	)
	if event.Status.IsClosed() {
		return warnWouldFail(
			toolpreview.Build(summary),
			fmt.Errorf("carrier finding %s is already %s", event.ID, event.Status),
		), nil
	}

	acknowledged := true
	plan, err := planUpdate(
		carrierIntelRecord(event),
		event,
		func(seen *carrierintel.CarrierIntelEvent) error {
			acknowledged = seen.Acknowledge(request.TenantInfo.UserID, now)

			return nil
		},
		opts...,
	)
	if err != nil {
		return nil, err
	}
	if !acknowledged {
		return toolpreview.Build(fmt.Sprintf(
			"The carrier finding %q is already %s, so marking it as seen would change nothing.",
			event.Summary,
			event.Status,
		)), nil
	}

	return plan.preview(summary), nil
}
