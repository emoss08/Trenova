package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	tenderResponseAccept   = "A"
	transferFieldReason    = "rejectionReason"
	tradingPartnerFallback = "The trading partner"
)

var transferPreviewLabels = map[string]string{
	fieldStatus:         "Tender status",
	transferFieldReason: "Reason given to the partner",
}

func (t *ediTransferDecisionTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	call, outcome, err := t.outcome(ctx, &params)
	if err != nil {
		if isRefusal(err) || errortypes.IsNotFoundError(err) {
			return warnWouldFail(toolpreview.Build(t.decision.description), err), nil
		}

		return nil, err
	}

	summary := t.decision.summary(outcome, call)
	preview := toolpreview.Build(summary)
	if outcome.refusal == nil {
		changes, cErr := tenderDecisionChanges(outcome, call.tenant.OrgID)
		if cErr != nil {
			return nil, cErr
		}
		preview = toolpreview.Build(summary, changes...)
	}

	return warnWouldFail(preview, outcome.refusal), nil
}

func tenderDecisionChanges(outcome *tenderOutcome, orgID pulid.ID) ([]*agent.RecordChange, error) {
	record := toolpreview.Record{
		Resource: permission.ResourceEDI,
		ID:       outcome.before.ID,
		Label:    transferLabel(outcome.before, orgID),
		Version:  pinnedVersion(outcome.before.Version),
	}
	status, err := toolpreview.Changed(record, outcome.before, outcome.after,
		toolpreview.Only(fieldStatus, transferFieldReason),
		toolpreview.Labels(transferPreviewLabels),
	)
	if err != nil {
		return nil, err
	}

	changes := []*agent.RecordChange{status}
	if outcome.approval != nil && outcome.approval.Shipment != nil {
		created, cErr := enteredShipmentChanges(&serviceports.ShipmentCreatePlan{
			Shipment: outcome.approval.Shipment,
		})
		if cErr != nil {
			return nil, cErr
		}
		changes = append(changes, created...)
	}
	if response := tenderResponseChange(outcome.after); response != nil {
		changes = append(changes, response)
	}

	return changes, nil
}

func tenderResponseChange(after *edi.EDITransfer) *agent.RecordChange {
	if after == nil || after.InboundMessageID.IsNil() ||
		(after.Status != edi.TransferStatusRejected && after.Status != edi.TransferStatusProcessing) {
		return nil
	}

	response := ediservice.TenderResponseFor(after)
	if response == nil {
		return nil
	}

	partner := tradingPartnerFallback
	if after.TargetPartner != nil && after.TargetPartner.Name != "" {
		partner = after.TargetPartner.Name
	}

	lines := []string{
		"Response: " + tenderResponseWord(response.ResponseCode),
	}
	if response.BOL != "" {
		lines = append(lines, "BOL: "+response.BOL)
	}
	if response.ReasonCode != "" {
		lines = append(lines, "Reason code: "+response.ReasonCode)
	}
	if response.RejectionReason != "" {
		lines = append(lines, "Reason: "+response.RejectionReason)
	}

	return toolpreview.Send(toolpreview.Record{
		Resource: permission.ResourceEDI,
		ID:       after.TargetPartnerID,
		Label:    partner,
	}, &agent.MessagePreview{
		Channel: agent.MessageChannelEDI,
		To:      []string{partner},
		Subject: "990 response to load tender " + strings.TrimSpace(response.BOL),
		Body:    strings.Join(lines, "\n"),
	})
}

func tenderResponseWord(code string) string {
	if code == tenderResponseAccept {
		return "Accepted (A)"
	}

	return "Declined (D)"
}

func tenderStops(transfer *edi.EDITransfer) int {
	stops := 0
	for _, move := range transfer.TenderPayload.Moves {
		stops += len(move.Stops)
	}

	return stops
}

func tenderSubject(outcome *tenderOutcome, call *tenderCall) string {
	return "load tender" + transferIdentity(outcome.before, call.tenant.OrgID)
}

func tellsPartner(transfer *edi.EDITransfer) string {
	if transfer.InboundMessageID.IsNotNil() {
		return "The partner is sent a 990"
	}

	return "The other organization sees it"
}

func summarizeAcceptTender(outcome *tenderOutcome, call *tenderCall) string {
	summary := fmt.Sprintf(
		"Would accept the %s, %s with %d %s, and create the shipment from it.",
		tenderSubject(outcome, call),
		outcome.before.TenderPayload.CustomerLabel,
		tenderStops(outcome.before),
		stringutils.Pluralize("stop", "stops", tenderStops(outcome.before)),
	)
	if outcome.refusal == nil {
		summary += " " + tellsPartner(outcome.before) + " accepted."
	}

	return summary
}

func summarizeDeclineTender(outcome *tenderOutcome, call *tenderCall) string {
	return fmt.Sprintf(
		"Would decline the %s. %s declined, with the reason: %s",
		tenderSubject(outcome, call),
		tellsPartner(outcome.before),
		call.reason,
	)
}

func summarizeCancelTender(outcome *tenderOutcome, call *tenderCall) string {
	return fmt.Sprintf(
		"Would withdraw the %s. The receiving organization can no longer accept it and "+
			"the shipment's tender is marked canceled.",
		tenderSubject(outcome, call),
	)
}

func summarizeExpireTender(outcome *tenderOutcome, call *tenderCall) string {
	return fmt.Sprintf(
		"Would close the %s as expired; nobody can accept it afterwards.",
		tenderSubject(outcome, call),
	)
}
