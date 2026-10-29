package findingtools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionreview"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelreviewservice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelevent"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/shared/timeutils"
)

var detentionEscalationFields = []string{"requiresApproval", "notificationStatus"}

var detentionEvidenceFields = []string{changepreview.FieldKind, "source", "summary"}

func detentionOccurrenceRecord(occurrence *detention.DetentionOccurrence) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceDetention,
		ID:       occurrence.ID,
		Label:    fmt.Sprintf("Detention, %d billable minutes", occurrence.BillableMinutes),
		Version:  changepreview.PreviewVersion(occurrence.Version),
	}
}

func (t *EscalateDetentionTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	occurrenceID, reason, err := t.arguments(&params)
	if err != nil {
		return nil, err
	}

	occurrence, err := t.occurrence(ctx, occurrenceID, &params)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	plan, err := changepreview.PlanUpdate(
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
		Resource: permission.ResourceDetention,
		Label:    "Detention evidence",
	}, detentionreview.EscalationEvidence(occurrence, reason, now),
		toolpreview.Only(detentionEvidenceFields...),
	)
	if err != nil {
		return nil, err
	}

	return plan.PlanPreview(summary, evidence), nil
}

func (t *PlaceWorkerDispatchHoldTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	request, err := t.arguments(&params)
	if err != nil {
		return nil, err
	}

	driver, err := t.workers.Get(ctx, workerports.GetWorkerByIDRequest{
		ID:         request.WorkerID,
		TenantInfo: request.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	plan, err := changepreview.PlanUpdate(toolpreview.Record{
		Resource: permission.ResourceWorker,
		ID:       driver.ID,
		Label:    driver.FullName(),
		Version:  changepreview.PreviewVersion(driver.Version),
	}, driver, func(held *worker.Worker) error {
		return workerservice.ApplyDispatchHold(held, &request)
	}, toolpreview.Only("canBeAssigned"))
	if err != nil {
		return nil, err
	}

	return plan.PlanPreview(fmt.Sprintf(
		"Would stop %s being given new freight: %s",
		driver.FullName(),
		request.Reason,
	)), nil
}

var carrierIntelVolatileFields = []string{"acknowledgedAt", "resolvedAt"}

func carrierIntelRecord(event *carrierintelevent.CarrierIntelEvent) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceCarrierIntelligence,
		ID:       event.ID,
		Label:    event.Summary,
		Version:  changepreview.PreviewVersion(event.Version),
	}
}

func (t *carrierIntelEventTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	request, err := t.arguments(&params)
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

	plan, err := changepreview.PlanUpdate(
		carrierIntelRecord(event),
		event,
		func(closed *carrierintelevent.CarrierIntelEvent) error {
			return carrierintelreviewservice.PlanResolve(closed, request, now)
		},
		opts...,
	)
	if err != nil {
		return nil, err
	}

	return plan.PlanPreview(fmt.Sprintf(
		"Would close the %s carrier finding %q as %s: %s",
		event.Severity,
		event.Summary,
		request.Resolution,
		request.Note,
	)), nil
}

func (t *carrierIntelEventTool) previewAcknowledge(
	event *carrierintelevent.CarrierIntelEvent,
	request *carrierintelreviewservice.ResolveEventRequest,
	now int64,
	opts []toolpreview.Option,
) (*agentpreview.ToolPreview, error) {
	summary := fmt.Sprintf(
		"Would mark the %s carrier finding %q as seen: %s",
		event.Severity,
		event.Summary,
		request.Note,
	)
	if event.Status.IsClosed() {
		return changepreview.WarnWouldFail(
			toolpreview.Build(summary),
			fmt.Errorf("carrier finding %s is already %s", event.ID, event.Status),
		), nil
	}

	acknowledged := true
	plan, err := changepreview.PlanUpdate(
		carrierIntelRecord(event),
		event,
		func(seen *carrierintelevent.CarrierIntelEvent) error {
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

	return plan.PlanPreview(summary), nil
}
