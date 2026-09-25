package shipmentholdservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *service) PreviewCreate(
	ctx context.Context,
	req *repositories.CreateShipmentHoldRequest,
	actor *services.RequestActor,
) (*shipment.ShipmentHold, error) {
	return s.planCreate(ctx, req, actor)
}

func (s *service) PreviewRelease(
	ctx context.Context,
	req *repositories.ReleaseShipmentHoldRequest,
	actor *services.RequestActor,
) (*shipment.ShipmentHold, error) {
	_, toRelease, err := s.planRelease(ctx, req, actor)

	return toRelease, err
}

func (s *service) planCreate(
	ctx context.Context,
	req *repositories.CreateShipmentHoldRequest,
	actor *services.RequestActor,
) (*shipment.ShipmentHold, error) {
	if req == nil {
		return nil, errShipmentHoldRequestRequired()
	}
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	userID, err := requireHoldUser(actor)
	if err != nil {
		return nil, err
	}

	if err = s.ensureShipmentExists(ctx, req.ShipmentID, req.TenantInfo); err != nil {
		return nil, err
	}

	reason, err := s.getActiveHoldReason(ctx, req.HoldReasonID, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	startedAt := timeutils.NowUnix()
	if req.StartedAt != nil {
		startedAt = *req.StartedAt
	}

	entity := &shipment.ShipmentHold{
		ShipmentID:        req.ShipmentID,
		OrganizationID:    req.TenantInfo.OrgID,
		BusinessUnitID:    req.TenantInfo.BuID,
		HoldReasonID:      idPtr(reason.ID),
		Type:              reason.Type,
		Severity:          reason.DefaultSeverity,
		ReasonCode:        reason.Code,
		Notes:             strings.TrimSpace(req.Notes),
		Source:            shipment.HoldSourceUser,
		BlocksDispatch:    reason.DefaultBlocksDispatch,
		BlocksDelivery:    reason.DefaultBlocksDelivery,
		BlocksBilling:     reason.DefaultBlocksBilling,
		VisibleToCustomer: reason.DefaultVisibleToCustomer,
		StartedAt:         startedAt,
		CreatedByID:       idPtr(userID),
	}
	applyCreateOverrides(entity, req)

	if multiErr := validateHold(entity); multiErr != nil {
		return nil, multiErr
	}

	return entity, nil
}

func (s *service) planRelease(
	ctx context.Context,
	req *repositories.ReleaseShipmentHoldRequest,
	actor *services.RequestActor,
) (original, released *shipment.ShipmentHold, err error) {
	if req == nil {
		return nil, nil, errShipmentHoldRequestRequired()
	}
	if multiErr := req.Validate(); multiErr != nil {
		return nil, nil, multiErr
	}

	userID, err := requireHoldUser(actor)
	if err != nil {
		return nil, nil, err
	}

	original, err = s.repo.GetByID(ctx, &repositories.GetShipmentHoldByIDRequest{
		HoldID:     req.HoldID,
		ShipmentID: req.ShipmentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}
	if !original.IsActive() {
		return nil, nil, errortypes.NewBusinessError("Shipment hold is already released").
			WithParam("holdId", req.HoldID.String())
	}

	releasedAt := timeutils.NowUnix()
	toRelease := *original
	toRelease.ReleasedAt = &releasedAt
	toRelease.ReleasedByID = idPtr(userID)
	released = &toRelease

	return original, released, nil
}

func errShipmentHoldRequestRequired() error {
	return errortypes.NewValidationError(
		"request",
		errortypes.ErrRequired,
		"Shipment hold request is required",
	)
}
