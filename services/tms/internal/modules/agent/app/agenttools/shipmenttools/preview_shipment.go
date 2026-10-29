package shipmenttools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/outboundpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
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

func (t *AddShipmentCommentTool) Preview(
	_ context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	entity, err := t.comment(&params)
	if err != nil {
		return nil, err
	}

	created, err := toolpreview.Create(
		toolpreview.Record{Resource: permission.ResourceShipmentComment, Label: "Shipment comment"},
		entity,
		toolpreview.Only(
			changepreview.PreviewFieldShipmentID,
			"comment",
			"type",
			"visibility",
			"priority",
		),
		toolpreview.WithRefs(map[string]permission.Resource{
			changepreview.PreviewFieldShipmentID: permission.ResourceShipment,
		}),
	)
	if err != nil {
		return nil, err
	}

	shared := toolpreview.Send(
		toolpreview.Record{
			Resource: permission.ResourceShipment,
			ID:       entity.ShipmentID,
			Label:    commentAudience(entity.Visibility),
		},
		&agentpreview.MessagePreview{
			Channel:    agentpreview.MessageChannelComment,
			To:         []string{commentAudience(entity.Visibility)},
			Body:       entity.Comment,
			Visibility: string(entity.Visibility),
		},
	)

	preview := toolpreview.Build(
		fmt.Sprintf("Would leave a %s note on the shipment, read by %s.",
			strings.ToLower(string(entity.Priority)),
			strings.ToLower(commentAudience(entity.Visibility))),
		created,
		shared,
	)
	if entity.Visibility == shipment.CommentVisibilityCustomer ||
		entity.Visibility == shipment.CommentVisibilityDriver {
		outboundpreview.WarnSensitiveContent(preview, entity.Comment)
	}

	return preview, nil
}

func commentAudience(visibility shipment.CommentVisibility) string {
	switch visibility {
	case shipment.CommentVisibilityCustomer:
		return "The customer"
	case shipment.CommentVisibilityDriver:
		return "The driver"
	case shipment.CommentVisibilityAccounting:
		return "Accounting"
	case shipment.CommentVisibilityOperations:
		return "Operations"
	case shipment.CommentVisibilityInternal:
		return "Internal staff"
	default:
		return "Internal staff"
	}
}

func (t *CancelShipmentTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	request, err := t.request(&params)
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
		next, moves := tenderservice.WithdrawnOfferStatus(offer.Status)
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
