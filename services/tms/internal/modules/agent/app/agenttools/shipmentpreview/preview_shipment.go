package shipmentpreview

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// shipmentPartnerNotices is what a shipment write tells its EDI partners: a
// 204 change to those it was tendered to, a 214 cancellation to linked
// shipments.
type ShipmentPartnerNotices interface {
	PreviewTenderChanges(
		ctx context.Context,
		original, updated *shipment.Shipment,
		actor *permissionports.RequestActor,
	) ([]*ediservice.PartnerNotice, error)
	PreviewCancelNotices(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
	) ([]*ediservice.PartnerNotice, error)
}

var UpdateShipmentRefs = map[string]permission.Resource{
	changepreview.PreviewFieldCustomerID:     permission.ResourceCustomer,
	changepreview.PreviewFieldServiceTypeID:  permission.ResourceServiceType,
	changepreview.PreviewFieldShipmentTypeID: permission.ResourceShipmentType,
	changepreview.PreviewFieldTractorTypeID:  permission.ResourceEquipmentType,
	changepreview.PreviewFieldTrailerTypeID:  permission.ResourceEquipmentType,
}

// partnerNoticeSend is one EDI message to a trading partner.
func PartnerNoticeSend(notice *ediservice.PartnerNotice, reason string) *agentpreview.RecordChange {
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
		&agentpreview.MessagePreview{
			Channel: agentpreview.MessageChannelEDI,
			To:      []string{notice.PartnerName},
			Subject: subject,
			Body:    strings.TrimSpace(body),
		},
	)
}

func ShipmentRecord(entity *shipment.Shipment) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceShipment,
		ID:       entity.ID,
		Label:    entity.ProNumber,
		Version:  changepreview.PinnedVersion(entity.Version),
	}
}
