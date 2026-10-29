package editools

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/edi/app/edidelivery"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/pulid"
)

type ediRecordView struct {
	DeliveryStatus string `json:"deliveryStatus,omitempty"`
	Status         string `json:"status,omitempty"`
	Outcome        string `json:"outcome"`
}

var ediRecordLabels = map[string]string{
	"deliveryStatus":          "Delivery",
	changepreview.FieldStatus: "File status",
	"outcome":                 "What happens",
}

type plannedEDIRecord struct {
	change  *agentpreview.RecordChange
	refused bool
}

func refusedFirst(planned []plannedEDIRecord) (changes []*agentpreview.RecordChange, going int) {
	slices.SortStableFunc(planned, func(a, b plannedEDIRecord) int {
		switch {
		case a.refused == b.refused:
			return 0
		case a.refused:
			return -1
		default:
			return 1
		}
	})

	changes = make([]*agentpreview.RecordChange, 0, len(planned))
	for _, record := range planned {
		changes = append(changes, record.change)
		if !record.refused {
			going++
		}
	}

	return changes, going
}

func previewedIDs(ids []pulid.ID) ([]pulid.ID, bool) {
	if len(ids) <= agentpreview.MaxPreviewRecords {
		return ids, false
	}

	return ids[:agentpreview.MaxPreviewRecords], true
}

func messageLabel(message *edimessage.EDIMessage) string {
	label := fmt.Sprintf("%s %s", message.TransactionSet, message.InterchangeControlNumber)
	if message.Partner != nil && message.Partner.Name != "" {
		label += " to " + message.Partner.Name
	}

	return strings.TrimSpace(label)
}

func messagePartner(message *edimessage.EDIMessage) string {
	if message.Partner != nil && message.Partner.Name != "" {
		return message.Partner.Name
	}

	return tradingPartnerFallback
}

func deliveryChange(plan *edidelivery.DeliveryPlan) (*agentpreview.RecordChange, error) {
	message := plan.Message
	route := "over the partner's communication profile"
	if plan.Profile != nil && plan.Profile.Name != "" {
		route = fmt.Sprintf("over %s (%s)", plan.Profile.Name, plan.Profile.Method)
	}

	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceEDI,
		ID:       message.ID,
		Label:    messageLabel(message),
		Version:  changepreview.PinnedVersion(message.Version),
	}, &ediRecordView{DeliveryStatus: string(message.DeliveryStatus)}, &ediRecordView{
		DeliveryStatus: string(edimessage.MessageDeliveryStatusQueued),
		Outcome:        "Sent again " + route,
	}, toolpreview.Labels(ediRecordLabels))
	if err != nil {
		return nil, err
	}

	body := fmt.Sprintf(
		"The %s document with interchange control number %s, group %s and transaction "+
			"%s, exactly as it was generated.",
		message.TransactionSet,
		message.InterchangeControlNumber,
		message.GroupControlNumber,
		message.TransactionControlNumber,
	)
	if message.DeliveryLastError != "" {
		body += " The last attempt failed: " + message.DeliveryLastError
	}
	change.Message = &agentpreview.MessagePreview{
		Channel: agentpreview.MessageChannelEDI,
		To:      []string{messagePartner(message)},
		Subject: fmt.Sprintf("EDI %s %s", message.TransactionSet, message.InterchangeControlNumber),
		Body:    body,
	}

	return change, nil
}

func refusedRecord(
	id pulid.ID,
	label string,
	before *ediRecordView,
	refusal error,
) (*agentpreview.RecordChange, error) {
	after := *before
	after.Outcome = "Refused: " + strings.TrimSpace(refusal.Error())

	return toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceEDI,
		ID:       id,
		Label:    label,
	}, before, &after, toolpreview.Labels(ediRecordLabels))
}

func (t *RetryEDIMessageDeliveryTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	ids, err := ediRecordSet(t, &params, paramEDIMessageIDs)
	if err != nil {
		return nil, err
	}

	checked, more := previewedIDs(ids)
	planned := make([]plannedEDIRecord, 0, len(checked))
	for _, id := range checked {
		plan, planErr := t.deliveries.PlanRetryMessageDelivery(
			ctx,
			&edidelivery.RetryMessageDeliveryRequest{
				MessageID:  id,
				TenantInfo: toolparams.TenantFrom(params),
			},
		)
		if planErr != nil {
			if !changepreview.IsRefusal(planErr) && !errortypes.IsNotFoundError(planErr) {
				return nil, planErr
			}
			change, cErr := refusedRecord(id, "EDI message "+id.String(), &ediRecordView{}, planErr)
			if cErr != nil {
				return nil, cErr
			}
			planned = append(planned, plannedEDIRecord{change: change, refused: true})
			continue
		}
		change, cErr := deliveryChange(plan)
		if cErr != nil {
			return nil, cErr
		}
		planned = append(planned, plannedEDIRecord{change: change})
	}

	changes, going := refusedFirst(planned)
	preview := toolpreview.Build(fmt.Sprintf(
		"Would send %d of %d EDI %s checked to %s again.",
		going, len(checked), inflect.Pluralize("message", "messages", len(checked)),
		inflect.Pluralize("its trading partner", "their trading partners", going),
	), changes...)

	return finishBulkPreview(preview, going, more, len(ids)), nil
}

func (t *ReplayEDIMessageTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	req, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	summary := "Would send this EDI document to its trading partner again."
	plan, err := t.deliveries.PlanReplayMessageDelivery(ctx, req)
	if err != nil {
		if changepreview.IsRefusal(err) || errortypes.IsNotFoundError(err) {
			return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	change, err := deliveryChange(plan)
	if err != nil {
		return nil, err
	}
	summary = fmt.Sprintf(
		"Would send %s the %s it already received once more, exactly as it was generated; "+
			"it receives the document twice.",
		messagePartner(plan.Message), messageLabel(plan.Message),
	)

	return toolpreview.Build(summary, change), nil
}

func (t *ReprocessEDIInboundFilesTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	ids, err := ediRecordSet(t, &params, paramEDIInboundFileIDs)
	if err != nil {
		return nil, err
	}

	checked, more := previewedIDs(ids)
	planned := make([]plannedEDIRecord, 0, len(checked))
	for _, id := range checked {
		record, pErr := t.planFile(ctx, &params, id)
		if pErr != nil {
			return nil, pErr
		}
		planned = append(planned, record)
	}

	changes, going := refusedFirst(planned)
	preview := toolpreview.Build(fmt.Sprintf(
		"Would process %d of %d inbound EDI %s checked again. What each transaction "+
			"becomes, and the acknowledgments its partner is sent, are decided when it runs.",
		going, len(checked), inflect.Pluralize("file", "files", len(checked)),
	), changes...)
	preview.Partial = true

	return finishBulkPreview(preview, going, more, len(ids)), nil
}

func (t *ReprocessEDIInboundFilesTool) planFile(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
	id pulid.ID,
) (plannedEDIRecord, error) {
	file, err := t.files.GetInboundFile(ctx, edimessageports.GetEDIInboundFileByIDRequest{
		ID:         id,
		TenantInfo: toolparams.TenantFrom(*params),
	})
	if err != nil {
		if !errortypes.IsNotFoundError(err) {
			return plannedEDIRecord{}, err
		}
		change, cErr := refusedRecord(id, "Inbound file "+id.String(), &ediRecordView{}, err)

		return plannedEDIRecord{change: change, refused: true}, cErr
	}

	before := &ediRecordView{Status: string(file.Status)}
	label := file.FileName
	if label == "" {
		label = "Inbound file " + file.ID.String()
	}
	if refusal := ediinboundservice.CheckReprocessable(file); refusal != nil {
		change, cErr := refusedRecord(file.ID, label, before, refusal)

		return plannedEDIRecord{change: change, refused: true}, cErr
	}

	outcome := "Parsed and routed again"
	if file.FailureReason != "" {
		outcome += "; it was held back because " + file.FailureReason
	}
	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceEDI,
		ID:       file.ID,
		Label:    label,
		Version:  changepreview.PinnedVersion(file.Version),
	}, before, &ediRecordView{Status: string(file.Status), Outcome: outcome},
		toolpreview.Labels(ediRecordLabels))

	return plannedEDIRecord{change: change}, err
}

func finishBulkPreview(
	preview *agentpreview.ToolPreview,
	going int,
	more bool,
	total int,
) *agentpreview.ToolPreview {
	if more {
		preview.Partial = true
		preview.Summary += fmt.Sprintf(
			" %d were asked for; the rest are checked when it runs.", total,
		)
	}
	if going == 0 {
		toolpreview.Warn(preview, agentpreview.PreviewWarningWouldFail,
			"None of the checked records would go as they stand.")
	}

	return preview
}
