package shipmenttools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendercancel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// liveTenderReader is what cancelling a shipment withdraws from carriers.
type liveTenderReader interface {
	PreviewLiveTenders(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
	) ([]*tender.Tender, error)
}

func (t *CancelShipmentTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	request, err := t.request(params)
	if err != nil {
		return nil, err
	}

	cancelled, err := t.shipments.PreviewCancel(ctx, request, params.Actor)
	if err != nil {
		return nil, err
	}

	change, err := toolpreview.Changed(
		shipmentpreview.ShipmentRecord(cancelled.Before),
		cancelled.Before,
		cancelled.After,
		toolpreview.Only("status", "cancelReason", "canceledById", "canceledAt"),
		toolpreview.WithRefs(map[string]permission.Resource{
			"canceledById": permission.ResourceUser,
		}),
		toolpreview.Volatile("canceledAt"),
	)
	if err != nil {
		return nil, err
	}
	changes := []*agentpreview.RecordChange{change}

	if t.partners != nil {
		notices, nErr := t.partners.PreviewCancelNotices(
			ctx,
			request.TenantInfo,
			request.ShipmentID,
		)
		if nErr != nil {
			return nil, nErr
		}
		for _, notice := range notices {
			changes = append(
				changes,
				shipmentpreview.PartnerNoticeSend(notice, request.CancelReason),
			)
		}
	}

	withdrawn := 0
	if t.tenders != nil {
		live, lErr := t.tenders.PreviewLiveTenders(ctx, request.TenantInfo, request.ShipmentID)
		if lErr != nil {
			return nil, lErr
		}
		for _, entity := range live {
			withdrawals, wErr := tenderWithdrawalChanges(entity)
			if wErr != nil {
				return nil, wErr
			}
			changes = append(changes, withdrawals...)
		}
		withdrawn = len(live)
	}

	summary := fmt.Sprintf(
		"Would cancel shipment %s, releasing its assignments and stopping it being billed. "+
			"Reason: %s",
		cancelled.Before.ProNumber, request.CancelReason,
	)
	if withdrawn > 0 {
		summary += fmt.Sprintf(" %s would be withdrawn from carriers.",
			changepreview.CountOf(withdrawn, "live tender"))
	}

	return toolpreview.Build(summary, changes...), nil
}

// tenderWithdrawalChanges is a live tender cancelled with the shipment: the
// tender itself, and a withdrawal for every carrier holding an offer.
func tenderWithdrawalChanges(entity *tender.Tender) ([]*agentpreview.RecordChange, error) {
	changes := make([]*agentpreview.RecordChange, 0, len(entity.Offers)+1)
	canceled := *entity
	canceled.Status = tender.StatusCanceled
	record, err := toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceTender,
			ID:       entity.ID,
			Label:    string(entity.Mode) + " tender",
		},
		entity,
		&canceled,
		toolpreview.Only("status"),
	)
	if err != nil {
		return nil, err
	}
	record.Operation = agentpreview.PreviewOperationArchive
	changes = append(changes, record)

	for _, offer := range entity.Offers {
		if offer == nil {
			continue
		}
		next, moves := tendercancel.WithdrawnOfferStatus(offer.Status)
		if !moves || next != tender.OfferStatusWithdrawn {
			continue
		}
		channel := agentpreview.MessageChannelEmail
		to := offer.RecipientEmail
		if offer.Channel == tender.ChannelEDI {
			channel = agentpreview.MessageChannelEDI
			to = CarrierNameOfOffer(offer) + " (EDI)"
		}
		changes = append(changes, toolpreview.Send(
			toolpreview.Record{
				Resource: permission.ResourceCarrier,
				ID:       offer.CarrierID,
				Label:    CarrierNameOfOffer(offer),
			},
			&agentpreview.MessagePreview{
				Channel: channel,
				To:      []string{to},
				Subject: "Tender offer withdrawn",
				Body:    "The shipment was canceled; the offer is no longer open.",
			},
		))
	}

	return changes, nil
}

func CarrierNameOfOffer(offer *tender.TenderOffer) string {
	if offer.Carrier != nil && strings.TrimSpace(offer.Carrier.Name) != "" {
		return offer.Carrier.Name
	}

	return changepreview.LabelCarrier
}
