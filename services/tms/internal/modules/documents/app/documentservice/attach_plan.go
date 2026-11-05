package documentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AttachLineageRequest struct {
	DocumentID   pulid.ID
	ResourceType string
	ResourceID   string
	TenantInfo   pagination.TenantInfo
	Actor        *permissionports.RequestActor
}

type AttachLineagePlan struct {
	Current    *document.Document
	Attached   *document.Document
	ShipmentID pulid.ID
	Unchanged  bool
}

func ApplyLineageMove(entity *document.Document, resourceType, resourceID string) {
	entity.ResourceType = resourceType
	entity.ResourceID = resourceID
}

func (s *Service) PreviewAttachLineage(
	ctx context.Context,
	req *AttachLineageRequest,
) (*AttachLineagePlan, error) {
	return s.planAttach(ctx, req)
}

func (s *Service) planAttach(
	ctx context.Context,
	req *AttachLineageRequest,
) (*AttachLineagePlan, error) {
	current, err := s.visibleDocument(ctx, documentports.GetDocumentByIDRequest{
		ID:         req.DocumentID,
		TenantInfo: req.TenantInfo,
	}, req.Actor)
	if err != nil {
		return nil, err
	}

	plan := &AttachLineagePlan{Current: current}
	if current.ResourceType == req.ResourceType && current.ResourceID == req.ResourceID {
		plan.Unchanged = true
		plan.Attached = current

		return plan, nil
	}

	if req.ResourceType == "shipment" {
		plan.ShipmentID, err = pulid.MustParse(req.ResourceID)
		if err != nil {
			return nil, errortypes.NewValidationError(
				"shipmentId",
				errortypes.ErrInvalid,
				"Invalid shipment ID",
			)
		}
		if err = s.requireShipment(ctx, plan.ShipmentID, req.TenantInfo); err != nil {
			return nil, err
		}
	}

	attached := *current
	ApplyLineageMove(&attached, req.ResourceType, req.ResourceID)
	plan.Attached = &attached

	return plan, nil
}
