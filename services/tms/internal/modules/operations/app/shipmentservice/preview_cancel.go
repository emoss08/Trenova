package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *service) PreviewCancel(
	ctx context.Context,
	req *shipmentports.CancelShipmentRequest,
	actor *permissionports.RequestActor,
) (*shipmentserviceports.ShipmentCancelPreview, error) {
	if req == nil {
		return nil, cancelRequestRequired()
	}

	request := *req
	original, err := s.planCancel(ctx, &request, actor)
	if err != nil {
		return nil, err
	}

	after := *original
	after.ApplyCancel(request.CanceledByID, request.CanceledAt, request.CancelReason)

	return &shipmentserviceports.ShipmentCancelPreview{Before: original, After: &after}, nil
}

func (s *service) planCancel(
	ctx context.Context,
	req *shipmentports.CancelShipmentRequest,
	actor *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	if req == nil {
		return nil, cancelRequestRequired()
	}

	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID: req.ShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.TenantInfo.OrgID,
			BuID:  req.TenantInfo.BuID,
		},
	})
	if err != nil {
		return nil, err
	}

	if err = shipmentstate.ValidateCancel(original); err != nil {
		return nil, err
	}

	req.CanceledByID = actor.AuditActor().UserID
	req.CanceledAt = timeutils.NowUnix()
	req.ExpectedVersion = original.Version

	return original, nil
}

func cancelRequestRequired() error {
	multiErr := errortypes.NewMultiError()
	multiErr.Add("request", errortypes.ErrRequired, "Cancel request is required")

	return multiErr
}
