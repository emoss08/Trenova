package ediservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// PartnerNotice is one EDI message a shipment write would send a trading
// partner: a 204 change to a partner the load was tendered to, or a 214
// cancellation to a linked shipment's owner.
type PartnerNotice struct {
	RecordID       pulid.ID
	PartnerID      pulid.ID
	OrganizationID pulid.ID
	PartnerName    string
	TransactionSet edi.TransactionSet
	Purpose        string
	StatusCode     string
	Changed        []string
	HeldForReview  bool
}

// tenderChangePlan is the 204 change a shipment update would send, and the
// recipients whose baseline it differs from.
type tenderChangePlan struct {
	payload    *edi.LoadTenderPayload
	hash       string
	recipients []*edi.TenderRecipient
}

// planTenderChanges decides which recipients a shipment update reaches: the
// tender payload has to change, and each recipient's last baseline has to
// differ from the new one.
func (s *Service) planTenderChanges(
	ctx context.Context,
	original, updated *shipment.Shipment,
	actor *services.RequestActor,
) (*tenderChangePlan, error) {
	if original == nil || updated == nil {
		return nil, nil
	}
	if original.ID != updated.ID || original.OrganizationID != updated.OrganizationID {
		return nil, nil
	}

	oldPayload := buildTenderPayload(original)
	newPayload := buildTenderPayload(updated)
	newPayload.PurposeCode = edi.LoadTenderPurposeChange
	newHash := tenderPayloadHash(&newPayload)
	if tenderPayloadHash(&oldPayload) == newHash {
		return nil, nil
	}

	recipients, err := s.tenderRecipientRepo.ListActiveTenderRecipientsForSourceShipment(
		ctx,
		repositories.ListEDITenderRecipientsForSourceShipmentRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  updated.OrganizationID,
				BuID:   updated.BusinessUnitID,
				UserID: actorUserID(actor),
			},
			SourceShipmentID: updated.ID,
		},
	)
	if err != nil {
		return nil, err
	}

	plan := &tenderChangePlan{
		payload:    &newPayload,
		hash:       newHash,
		recipients: make([]*edi.TenderRecipient, 0, len(recipients)),
	}
	for _, recipient := range recipients {
		if recipient == nil || recipient.LatestBaselineHash == newHash {
			continue
		}
		plan.recipients = append(plan.recipients, recipient)
	}

	return plan, nil
}

// PreviewTenderChanges names the partners a shipment update would send a 204
// change to, and what changed for each. An internal recipient reviews the
// change before it applies; an external one is sent it.
func (s *Service) PreviewTenderChanges(
	ctx context.Context,
	original, updated *shipment.Shipment,
	actor *services.RequestActor,
) ([]*PartnerNotice, error) {
	plan, err := s.planTenderChanges(ctx, original, updated, actor)
	if err != nil || plan == nil {
		return nil, err
	}

	notices := make([]*PartnerNotice, 0, len(plan.recipients))
	for _, recipient := range plan.recipients {
		diff := diffTenderPayloads(&recipient.LatestBaselinePayload, plan.payload)
		if len(diff) == 0 {
			continue
		}
		changed := make([]string, 0, len(diff))
		for key := range diff {
			changed = append(changed, key)
		}
		slices.Sort(changed)

		notices = append(notices, &PartnerNotice{
			RecordID:       recipient.ID,
			PartnerID:      recipient.EDIPartnerID,
			OrganizationID: recipient.RecipientOrganizationID,
			PartnerName:    s.partnerName(ctx, recipient),
			TransactionSet: edi.TransactionSet204,
			Purpose:        "Change",
			Changed:        changed,
			HeldForReview:  recipient.RecipientKind != edi.TenderRecipientKindExternal,
		})
	}

	return notices, nil
}

// PreviewCancelNotices names the linked shipments a cancellation reaches: the
// owner of each active link is sent the cancellation, with its reason, as an
// internal 214.
func (s *Service) PreviewCancelNotices(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*PartnerNotice, error) {
	if s.shipmentLinkRepo == nil {
		return nil, nil
	}

	links, err := s.shipmentLinkRepo.GetShipmentLinksByShipmentID(
		ctx,
		repositories.GetEDIShipmentLinksByShipmentIDRequest{
			ShipmentID: shipmentID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	notices := make([]*PartnerNotice, 0, len(links))
	for _, link := range links {
		if link == nil || link.Status != edi.ShipmentLinkStatusActive {
			continue
		}
		opposite := link.TargetOrganizationID
		if link.TargetOrganizationID == tenantInfo.OrgID && link.TargetShipmentID == shipmentID {
			opposite = link.SourceOrganizationID
		}
		notices = append(notices, &PartnerNotice{
			RecordID:       link.ID,
			OrganizationID: opposite,
			PartnerName:    "Linked trading partner",
			TransactionSet: edi.TransactionSet214,
			Purpose:        "Cancel",
			StatusCode:     "A7",
		})
	}

	return notices, nil
}

func (s *Service) partnerName(ctx context.Context, recipient *edi.TenderRecipient) string {
	if recipient.EDIPartnerID.IsNil() || s.partnerRepo == nil {
		return "Linked trading partner"
	}

	partner, err := s.partnerRepo.GetByID(ctx, repositories.GetEDIPartnerByIDRequest{
		ID: recipient.EDIPartnerID,
		TenantInfo: pagination.TenantInfo{
			OrgID: recipient.SourceOrganizationID,
			BuID:  recipient.SourceBusinessUnitID,
		},
	})
	if err != nil || partner == nil || strings.TrimSpace(partner.Name) == "" {
		return "EDI partner"
	}

	return partner.Name
}
