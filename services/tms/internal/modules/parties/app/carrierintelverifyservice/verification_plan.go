package carrierintelverifyservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/ports/carrierassignmentports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelverification"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// PlanVerifyEquipment is the verification record VerifyEquipment would make
// before the provider is asked: everything but the result.
func (s *Service) PlanVerifyEquipment(
	ctx context.Context,
	req *VerifyEquipmentRequest,
) (*carrierintelverification.CarrierEquipmentVerification, *carrier.Carrier, error) {
	assignment, err := s.assignmentRepo.GetByID(
		ctx,
		&carrierassignmentports.GetCarrierAssignmentByIDRequest{
			TenantInfo:          req.TenantInfo,
			CarrierAssignmentID: req.CarrierAssignmentID,
		},
	)
	if err != nil {
		return nil, nil, err
	}

	carrierEntity, err := s.carrierRepo.GetByID(ctx, carrierports.GetCarrierByIDRequest{
		ID:         assignment.CarrierID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}

	entity := &carrierintelverification.CarrierEquipmentVerification{
		OrganizationID:      req.TenantInfo.OrgID,
		BusinessUnitID:      req.TenantInfo.BuID,
		CarrierAssignmentID: assignment.ID,
		ShipmentMoveID:      assignment.ShipmentMoveID,
		CarrierID:           carrierEntity.ID,
		ExpectedDOTNumber:   carrierEntity.DOTNumber,
		UnitType:            req.UnitType,
		VIN:                 req.VIN,
		PlateNumber:         req.PlateNumber,
		PlateState:          req.PlateState,
		UnitNumber:          req.UnitNumber,
		VerifiedByID:        req.TenantInfo.UserID,
		VerifiedAt:          s.now(),
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, nil, multiErr
	}

	return entity, carrierEntity, nil
}
