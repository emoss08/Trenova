package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/internal/core/services/tenderservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var (
	_ serviceports.ToolPreviewer = (*addShipmentCommentTool)(nil)
	_ serviceports.ToolPreviewer = (*updateShipmentTool)(nil)
	_ serviceports.ToolPreviewer = (*cancelShipmentTool)(nil)
)

// shipmentPartnerNotices is what a shipment write tells its EDI partners: a
// 204 change to those it was tendered to, a 214 cancellation to linked
// shipments.
type shipmentPartnerNotices interface {
	PreviewTenderChanges(
		ctx context.Context,
		original, updated *shipment.Shipment,
		actor *serviceports.RequestActor,
	) ([]*ediservice.PartnerNotice, error)
	PreviewCancelNotices(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
	) ([]*ediservice.PartnerNotice, error)
}

// liveTenderReader is what cancelling a shipment withdraws from carriers.
type liveTenderReader interface {
	PreviewLiveTenders(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
	) ([]*tender.Tender, error)
}

var updateShipmentRefs = map[string]permission.Resource{
	"customerId":     permission.ResourceCustomer,
	"serviceTypeId":  permission.ResourceServiceType,
	"shipmentTypeId": permission.ResourceShipmentType,
	"tractorTypeId":  permission.ResourceEquipmentType,
	"trailerTypeId":  permission.ResourceEquipmentType,
}

func (t *addShipmentCommentTool) Preview(
	_ context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	entity, err := t.comment(params)
	if err != nil {
		return nil, err
	}

	created, err := toolpreview.Create(
		toolpreview.Record{Resource: permission.ResourceShipmentComment, Label: "Shipment comment"},
		entity,
		toolpreview.Only("shipmentId", "comment", "type", "visibility", "priority"),
		toolpreview.WithRefs(map[string]permission.Resource{
			"shipmentId": permission.ResourceShipment,
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
		&agent.MessagePreview{
			Channel:    agent.MessageChannelComment,
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
		warnSensitiveContent(preview, entity.Comment)
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

func (t *updateShipmentTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	before, after, err := t.plan(ctx, params)
	if err != nil {
		return nil, err
	}

	change, err := toolpreview.Changed(
		shipmentRecord(before),
		before,
		after,
		toolpreview.Only(
			"customerId", "serviceTypeId", "shipmentTypeId", "tractorTypeId", "trailerTypeId",
			"bol", "pieces", "weight", "temperatureMin", "temperatureMax",
		),
		toolpreview.WithRefs(updateShipmentRefs),
	)
	if err != nil {
		return nil, err
	}

	changes := []*agent.RecordChange{change}
	if t.partners != nil {
		notices, nErr := t.partners.PreviewTenderChanges(ctx, before, after, params.Actor)
		if nErr != nil {
			return nil, nErr
		}
		for _, notice := range notices {
			changes = append(changes, partnerNoticeSend(notice, ""))
		}
	}

	summary := fmt.Sprintf("Would change %s on shipment %s.",
		countOf(len(change.Fields), "field"), before.ProNumber)
	if partners := len(changes) - 1; partners > 0 {
		summary += fmt.Sprintf(" %s tendered the load would be sent an EDI 204 change.",
			countOf(partners, "partner"))
	}

	return toolpreview.Build(summary, changes...), nil
}

func (t *cancelShipmentTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	request, err := t.request(params)
	if err != nil {
		return nil, err
	}

	cancelled, err := t.shipments.PreviewCancel(ctx, request, params.Actor)
	if err != nil {
		return nil, err
	}

	change, err := toolpreview.Changed(
		shipmentRecord(cancelled.Before),
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
	changes := []*agent.RecordChange{change}

	if t.partners != nil {
		notices, nErr := t.partners.PreviewCancelNotices(ctx, request.TenantInfo, request.ShipmentID)
		if nErr != nil {
			return nil, nErr
		}
		for _, notice := range notices {
			changes = append(changes, partnerNoticeSend(notice, request.CancelReason))
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
			countOf(withdrawn, "live tender"))
	}

	return toolpreview.Build(summary, changes...), nil
}

// partnerNoticeSend is one EDI message to a trading partner.
func partnerNoticeSend(notice *ediservice.PartnerNotice, reason string) *agent.RecordChange {
	subject := fmt.Sprintf("EDI %s %s", notice.TransactionSet, strings.ToLower(notice.Purpose))
	body := ""
	switch {
	case notice.StatusCode != "":
		body = "Shipment status " + notice.StatusCode + " (canceled)."
		if reason != "" {
			body += " Reason: " + reason
		}
	case len(notice.Changed) > 0:
		body = "Changed: " + strings.Join(notice.Changed, ", ") + "."
	}
	if notice.HeldForReview {
		body += " The partner reviews the change before it applies."
	}

	return toolpreview.Send(
		toolpreview.Record{
			Resource: permission.ResourceEDI,
			ID:       notice.RecordID,
			Label:    notice.PartnerName,
		},
		&agent.MessagePreview{
			Channel: agent.MessageChannelEDI,
			To:      []string{notice.PartnerName},
			Subject: subject,
			Body:    strings.TrimSpace(body),
		},
	)
}

// tenderWithdrawalChanges is a live tender cancelled with the shipment: the
// tender itself, and a withdrawal for every carrier holding an offer.
func tenderWithdrawalChanges(entity *tender.Tender) ([]*agent.RecordChange, error) {
	changes := make([]*agent.RecordChange, 0, len(entity.Offers)+1)
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
	record.Operation = agent.PreviewOperationArchive
	changes = append(changes, record)

	for _, offer := range entity.Offers {
		if offer == nil {
			continue
		}
		next, moves := tenderservice.WithdrawnOfferStatus(offer.Status)
		if !moves || next != tender.OfferStatusWithdrawn {
			continue
		}
		channel := agent.MessageChannelEmail
		to := offer.RecipientEmail
		if offer.Channel == tender.ChannelEDI {
			channel = agent.MessageChannelEDI
			to = carrierNameOfOffer(offer) + " (EDI)"
		}
		changes = append(changes, toolpreview.Send(
			toolpreview.Record{
				Resource: permission.ResourceCarrier,
				ID:       offer.CarrierID,
				Label:    carrierNameOfOffer(offer),
			},
			&agent.MessagePreview{
				Channel: channel,
				To:      []string{to},
				Subject: "Tender offer withdrawn",
				Body:    "The shipment was canceled; the offer is no longer open.",
			},
		))
	}

	return changes, nil
}

func carrierNameOfOffer(offer *tender.TenderOffer) string {
	if offer.Carrier != nil && strings.TrimSpace(offer.Carrier.Name) != "" {
		return offer.Carrier.Name
	}

	return "Carrier"
}

func shipmentRecord(entity *shipment.Shipment) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceShipment,
		ID:       entity.ID,
		Label:    entity.ProNumber,
		Version:  pinnedVersion(entity.Version),
	}
}
