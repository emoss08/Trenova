package ediservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type CancelInboundTenderRequest struct {
	TenantInfo pagination.TenantInfo
	Partner    *edi.EDIPartner
	Key        string
	ShipmentID pulid.ID
	Control    string
}

// CancelInboundTender carries out a partner's cancellation that its flow allowed: a tender still
// waiting for a planner is cancelled, and a load already made from it is cancelled with the
// partner's reason. It returns what a person should know about it.
func (s *Service) CancelInboundTender(
	ctx context.Context,
	req *CancelInboundTenderRequest,
) ([]string, error) {
	reason := fmt.Sprintf("Cancelled by %s in tender %s", req.Partner.Name, req.Control)
	now := timeutils.NowUnix()
	if req.Key != "" {
		prior, err := s.transferRepo.GetActionableInboundTransferByExternalReference(
			ctx,
			ediports.GetActionableInboundEDITransferByExternalReferenceRequest{
				TenantInfo:        req.TenantInfo,
				PartnerID:         req.Partner.ID,
				ExternalReference: req.Key,
			},
		)
		switch {
		case err == nil:
			MarkTransferCanceled(prior, permissionports.SystemPrincipalID, now)
			prior.FailureReason = reason
			prior.ProcessedAt = &now
			_, err = s.transferRepo.UpdateTransfer(ctx, prior)
			return nil, err
		case !errortypes.IsNotFoundError(err):
			return nil, err
		}
	}
	if req.ShipmentID.IsNil() {
		return []string{fmt.Sprintf(
			"%s cancelled tender %s, which matches no pending tender or load",
			req.Partner.Name,
			req.Key,
		)}, nil
	}
	_, err := s.shipmentSvc.Cancel(ctx, &shipmentports.CancelShipmentRequest{
		TenantInfo:   req.TenantInfo,
		ShipmentID:   req.ShipmentID,
		CanceledByID: permissionports.SystemPrincipalID,
		CanceledAt:   now,
		CancelReason: reason,
	}, &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeSystem,
		PrincipalID:    permissionports.SystemPrincipalID,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	})
	if errortypes.IsBusinessError(err) || errortypes.IsMultiError(err) || errortypes.IsError(err) {
		return []string{fmt.Sprintf(
			"%s cancelled tender %s, and its load could not be cancelled: %s",
			req.Partner.Name,
			req.Key,
			err.Error(),
		)}, nil
	}
	return nil, err
}
