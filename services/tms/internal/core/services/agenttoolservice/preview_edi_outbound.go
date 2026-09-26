package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const shipmentFieldTenderStatus = "tenderStatus"

func (t *sendEDILoadTenderTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	req, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	summary := "Would tender this shipment to another organization over EDI."
	plan, err := t.tenders.PlanLoadTender(ctx, req)
	if err != nil {
		if isRefusal(err) || errortypes.IsNotFoundError(err) {
			return warnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	receiver := plan.SourcePartner.Name
	sent := toolpreview.Send(toolpreview.Record{
		Resource: permission.ResourceEDI,
		ID:       plan.SourcePartner.ID,
		Label:    receiver,
	}, &agent.MessagePreview{
		Channel: agent.MessageChannelEDI,
		To:      []string{receiver},
		Subject: "Load tender for shipment " + plan.SourceShipment.ProNumber,
		Body:    tenderBody(&plan.Payload),
	})

	tendered := *plan.SourceShipment
	status := shipment.TenderStatusTendered
	tendered.TenderStatus = &status
	marked, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipment,
		ID:       plan.SourceShipment.ID,
		Label:    plan.SourceShipment.ProNumber,
		Version:  pinnedVersion(plan.SourceShipment.Version),
	}, plan.SourceShipment, &tendered, toolpreview.Only(shipmentFieldTenderStatus))
	if err != nil {
		return nil, err
	}

	summary = fmt.Sprintf(
		"Would tender shipment %s to %s over EDI and mark it Tendered until they answer.",
		plan.SourceShipment.ProNumber, receiver,
	)
	if plan.Status == edi.TransferStatusMappingRequired {
		summary += fmt.Sprintf(
			" %s has to map %d of its customers, locations or other records to its own "+
				"before it can accept.",
			receiver, len(plan.Mapping.Unresolved),
		)
	}

	return toolpreview.Build(summary, sent, marked), nil
}

func tenderBody(payload *edi.LoadTenderPayload) string {
	lines := make([]string, 0, 4)
	if payload.BOL != "" {
		lines = append(lines, "BOL: "+payload.BOL)
	}
	if payload.CustomerLabel != "" {
		lines = append(lines, "Customer: "+payload.CustomerLabel)
	}
	for moveIdx := range payload.Moves {
		stops := payload.Moves[moveIdx].Stops
		for stopIdx := range stops {
			stop := &stops[stopIdx]
			lines = append(lines, strings.TrimSpace(fmt.Sprintf("%s: %s %s",
				stop.Type, stop.LocationLabel, stop.LocationCity)))
		}
	}
	if payload.TotalChargeAmount.Valid {
		lines = append(lines, "Total charge: "+payload.TotalChargeAmount.Decimal.StringFixed(2))
	}

	return strings.Join(lines, "\n")
}

func (t *sendEDIStatusUpdateTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	req, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	summary := "Would send the trading partner an EDI 214 shipment status."
	document, err := t.documents.PlanGenerateDocument(ctx, req)
	if err != nil {
		if isRefusal(err) || errortypes.IsNotFoundError(err) {
			return warnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	partner := tradingPartnerFallback
	if document.Profile != nil {
		switch {
		case document.Profile.Partner != nil && document.Profile.Partner.Name != "":
			partner = document.Profile.Partner.Name
		case document.Profile.Name != "":
			partner = document.Profile.Name
		}
	}
	sent := toolpreview.Send(toolpreview.Record{
		Resource: permission.ResourceEDI,
		ID:       req.EDIPartnerID,
		Label:    partner,
	}, &agent.MessagePreview{
		Channel: agent.MessageChannelEDI,
		To:      []string{partner},
		Subject: "EDI 214 shipment status",
		Body:    document.RawX12,
	})

	summary = fmt.Sprintf(
		"Would send %s an EDI 214 shipment status of %d segments, rendered as below; the "+
			"control numbers are assigned when it is sent.",
		partner, document.SegmentCount,
	)
	if notes := diagnosticNotes(document); notes != "" {
		summary += " The partner's rules noted: " + notes + "."
	}

	return toolpreview.Build(summary, sent), nil
}

func diagnosticNotes(document *ediservice.EDIDocumentPreview) string {
	notes := make([]string, 0, len(document.Diagnostics))
	for _, diagnostic := range document.Diagnostics {
		if message := strings.TrimSpace(diagnostic.Message); message != "" {
			notes = append(notes, strings.TrimRight(message, "."))
		}
	}

	return strings.Join(notes, "; ")
}
