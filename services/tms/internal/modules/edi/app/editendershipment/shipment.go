package editendershipment

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) Lock(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*shipment.Shipment, error) {
	return s.shipmentTenderRepo.LockShipment(ctx, editenderports.LockEDITenderShipmentRequest{
		ShipmentID: shipmentID,
		TenantInfo: tenantInfo,
	})
}

func (s *Service) SetTenderStatus(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
	status shipment.TenderStatus,
) error {
	return s.shipmentTenderRepo.SetShipmentTenderStatus(
		ctx,
		editenderports.SetEDIShipmentTenderStatusRequest{
			ShipmentID: shipmentID,
			TenantInfo: tenantInfo,
			Status:     status,
		},
	)
}

func (s *Service) CreateSystemComment(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
	comment string,
	metadata map[string]any,
) error {
	if s.shipmentCommentRepo == nil || s.userRepo == nil {
		return nil
	}

	systemUser, err := s.userRepo.GetSystemUser(ctx, "id")
	if err != nil {
		return err
	}

	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata[shipment.CommentMetadataOrigin] = shipment.CommentOriginEDI

	_, err = s.shipmentCommentRepo.Create(ctx, &shipment.ShipmentComment{
		ShipmentID:       shipmentID,
		OrganizationID:   tenantInfo.OrgID,
		BusinessUnitID:   tenantInfo.BuID,
		UserID:           systemUser.ID,
		Comment:          comment,
		Type:             shipment.CommentTypeStatusUpdate,
		Visibility:       shipment.CommentVisibilityOperations,
		Priority:         shipment.CommentPriorityNormal,
		Source:           shipment.CommentSourceSystem,
		Metadata:         metadata,
		MentionedUserIDs: []pulid.ID{},
	})
	return err
}
