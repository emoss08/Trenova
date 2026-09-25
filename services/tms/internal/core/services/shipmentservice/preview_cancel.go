package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *service) PreviewCancel(
	ctx context.Context,
	req *repositories.CancelShipmentRequest,
	actor *services.RequestActor,
) (*services.ShipmentCancelPreview, error) {
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

	return &services.ShipmentCancelPreview{Before: original, After: &after}, nil
}

// planCancel reads the shipment a cancellation is for, refuses one already
// canceled, and stamps the request with who cancels it and when.
func (s *service) planCancel(
	ctx context.Context,
	req *repositories.CancelShipmentRequest,
	actor *services.RequestActor,
) (*shipment.Shipment, error) {
	if req == nil {
		return nil, cancelRequestRequired()
	}

	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID: req.ShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.TenantInfo.OrgID,
			BuID:  req.TenantInfo.BuID,
		},
	})
	if err != nil {
		return nil, err
	}

	if original.IsCanceled() {
		return nil, errortypes.NewBusinessError("shipment is already canceled")
	}

	req.CanceledByID = actor.AuditActor().UserID
	req.CanceledAt = timeutils.NowUnix()

	return original, nil
}

func cancelRequestRequired() error {
	multiErr := errortypes.NewMultiError()
	multiErr.Add("request", errortypes.ErrRequired, "Cancel request is required")

	return multiErr
}
