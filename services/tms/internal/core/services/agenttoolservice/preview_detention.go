package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/pulid"
)

var (
	_ serviceports.ToolPreviewer = (*approveDetentionTool)(nil)
	_ serviceports.ToolPreviewer = (*waiveDetentionTool)(nil)
	_ serviceports.ToolPreviewer = (*sendDetentionNoticeTool)(nil)
)

func (t *approveDetentionTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, detail, err := t.request(ctx, params)
	if err != nil {
		return nil, err
	}

	change, err := t.detention.PreviewApprove(ctx, &request)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would approve the %s %s detention charge %s for billing, releasing it onto the "+
			"customer's invoice (collectability %d, %s; %d evidence records): %s",
		change.Before.BillableAmount.StringFixed(2),
		change.Before.Currency,
		detentionPlace(change.Before),
		detail.Collectability.Score,
		detail.Collectability.Band,
		len(detail.Evidence),
		request.Note,
	)

	return detentionDecisionPreview(summary, change, "approvedAt")
}

func (t *waiveDetentionTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	change, err := t.detention.PreviewWaive(ctx, &request)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would waive the %s %s detention charge %s as %s, giving up that revenue: %s",
		change.Before.BillableAmount.StringFixed(2),
		change.Before.Currency,
		detentionPlace(change.Before),
		request.Reason,
		request.Note,
	)

	return detentionDecisionPreview(summary, change, "waivedAt")
}

func (t *sendDetentionNoticeTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	notice, err := t.detention.PreviewOccurrenceNotice(ctx, &request)
	if err != nil {
		return nil, err
	}

	message := &agent.MessagePreview{
		To:      notice.Recipients,
		Subject: notice.Content.Subject,
		Body:    notice.Content.Text,
	}
	if notice.Attachment != "" {
		message.Attachments = []string{notice.Attachment}
	}
	if notice.Content.VersionID != nil {
		message.TemplateVersionID = *notice.Content.VersionID
	}

	send := toolpreview.Send(toolpreview.Record{
		Resource: permission.ResourceCustomer,
		ID:       notice.Before.CustomerID,
		Label:    notice.Before.CustomerName,
	}, emailMessagePreview(notice.Sender, message))

	status, err := toolpreview.Changed(
		detentionRecord(notice.Before),
		notice.Before,
		notice.After,
		toolpreview.Volatile("noticeSentAt"),
	)
	if err != nil {
		return nil, err
	}

	preview := toolpreview.Build(
		fmt.Sprintf(
			"Would send the customer the %s detention notice %s, to %s.",
			strings.ToLower(string(notice.Kind)),
			detentionPlace(notice.Before),
			strings.Join(notice.Recipients, ", "),
		),
		send,
		status,
	)
	warnSuppressedRecipients(preview, notice.Sender)

	return preview, nil
}

// detentionDecisionPreview is an approval or a waiver: the occurrence's
// values before and after, and what the shipment's detention charge comes to
// on either side.
func detentionDecisionPreview(
	summary string,
	change *detentionservice.OccurrenceChange,
	volatile ...string,
) (*agent.ToolPreview, error) {
	occurrence, err := toolpreview.Changed(
		detentionRecord(change.Before),
		change.Before,
		change.After,
		toolpreview.Volatile(volatile...),
		toolpreview.WithRefs(map[string]permission.Resource{
			"approvedById": permission.ResourceUser,
			"waivedById":   permission.ResourceUser,
		}),
		toolpreview.Ignore("evidence", "notices", "evidenceHead"),
	)
	if err != nil {
		return nil, err
	}

	toolpreview.AttachMoney(
		occurrence,
		shipmentDetentionMoney(change),
		toolpreview.SensitiveAs("billableAmount"),
	)

	return toolpreview.Build(summary, occurrence), nil
}

// shipmentDetentionMoney is the shipment's detention charge, occurrence by
// occurrence, with the decided one as it stands and as it would stand.
func shipmentDetentionMoney(change *detentionservice.OccurrenceChange) *agent.MoneyPreview {
	occurrences := change.Shipment
	if !containsOccurrence(occurrences, change.Before.ID) {
		occurrences = append([]*detention.DetentionOccurrence{change.Before}, occurrences...)
	}

	lines := make([]agent.MoneyLine, 0, len(occurrences))
	for _, occurrence := range occurrences {
		if occurrence == nil {
			continue
		}
		before, after := occurrence, occurrence
		if occurrence.ID == change.Before.ID {
			before, after = change.Before, change.After
		}
		lines = append(lines, agent.MoneyLine{
			Label:  detentionLineLabel(occurrence),
			Before: knownAmount(before.ChargedAmount()),
			After:  knownAmount(after.ChargedAmount()),
		})
	}

	return toolpreview.MoneyBlock(change.Before.Currency, lines...)
}

func containsOccurrence(occurrences []*detention.DetentionOccurrence, id pulid.ID) bool {
	for _, occurrence := range occurrences {
		if occurrence != nil && occurrence.ID == id {
			return true
		}
	}

	return false
}

func detentionRecord(occurrence *detention.DetentionOccurrence) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceDetentionPolicy,
		ID:       occurrence.ID,
		Label:    "Detention " + detentionPlace(occurrence),
		Version:  pinnedVersion(occurrence.Version),
	}
}

func detentionPlace(occurrence *detention.DetentionOccurrence) string {
	parts := make([]string, 0, 2)
	if name := strings.TrimSpace(occurrence.LocationName); name != "" {
		parts = append(parts, "at "+name)
	}
	if pro := strings.TrimSpace(occurrence.ShipmentProNumber); pro != "" {
		parts = append(parts, "on "+pro)
	}
	if len(parts) == 0 {
		return "on its shipment"
	}

	return strings.Join(parts, " ")
}

func detentionLineLabel(occurrence *detention.DetentionOccurrence) string {
	name := strings.TrimSpace(occurrence.LocationName)
	if name == "" {
		name = "Stop"
	}
	if occurrence.StopType == "" {
		return "Detention at " + name
	}

	return fmt.Sprintf("Detention at %s (%s)", name, strings.ToLower(string(occurrence.StopType)))
}
