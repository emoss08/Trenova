package driverpayservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *Service) ListPayCodes(
	ctx context.Context,
	req *repositories.ListPayCodesRequest,
) (*pagination.ListResult[*driverpay.PayCode], error) {
	if err := s.payCodeRepo.EnsureSystemDefaults(ctx, req.Filter.TenantInfo); err != nil {
		return nil, err
	}
	return s.payCodeRepo.List(ctx, req)
}

func (s *Service) ListPayCodesConnection(
	ctx context.Context,
	req *repositories.ListPayCodeConnectionRequest,
) (*pagination.CursorListResult[*driverpay.PayCode], error) {
	if err := s.payCodeRepo.EnsureSystemDefaults(ctx, req.Filter.TenantInfo); err != nil {
		return nil, err
	}
	return s.payCodeRepo.ListConnection(ctx, req)
}

func (s *Service) ListActivePayCodes(
	ctx context.Context,
	req repositories.ListActivePayCodesRequest,
) ([]*driverpay.PayCode, error) {
	if err := s.payCodeRepo.EnsureSystemDefaults(ctx, req.TenantInfo); err != nil {
		return nil, err
	}
	return s.payCodeRepo.ListActive(ctx, req)
}

func (s *Service) GetPayCode(
	ctx context.Context,
	req repositories.GetPayCodeByIDRequest,
) (*driverpay.PayCode, error) {
	return s.payCodeRepo.GetByID(ctx, req)
}

func (s *Service) CreatePayCode(
	ctx context.Context,
	entity *driverpay.PayCode,
	actor *serviceports.RequestActor,
) (*driverpay.PayCode, error) {
	if err := requireActor(actor, "Pay code creation"); err != nil {
		return nil, err
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.payCodeRepo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logPayCodeAudit(created, nil, actor.UserID, permission.OpCreate, "Pay code created")
	return created, nil
}

func (s *Service) UpdatePayCode(
	ctx context.Context,
	entity *driverpay.PayCode,
	actor *serviceports.RequestActor,
) (*driverpay.PayCode, error) {
	if err := requireActor(actor, "Pay code update"); err != nil {
		return nil, err
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	previous, err := s.payCodeRepo.GetByID(ctx, repositories.GetPayCodeByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}
	if previous.IsSystem && previous.Code != entity.Code {
		return nil, errortypes.NewValidationError(
			"code",
			errortypes.ErrInvalidOperation,
			"System pay codes keep their code; rename the display name or create a custom code instead",
		)
	}
	entity.Direction = previous.Direction
	entity.IsSystem = previous.IsSystem

	updated, err := s.payCodeRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logPayCodeAudit(updated, previous, actor.UserID, permission.OpUpdate, "Pay code updated")
	return updated, nil
}

func (s *Service) resolvePayCode(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	payCodeID pulid.ID,
	direction driverpay.PayCodeDirection,
) error {
	code, err := s.payCodeRepo.GetByID(ctx, repositories.GetPayCodeByIDRequest{
		ID:         payCodeID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if code.Direction != direction {
		return errortypes.NewValidationError(
			"payCodeId",
			errortypes.ErrInvalid,
			"Pay code "+code.Code+" is a "+code.Direction.String()+" code and cannot be used here",
		)
	}
	return nil
}

func (s *Service) logPayCodeAudit(
	current, previous *driverpay.PayCode,
	userID pulid.ID,
	operation permission.Operation,
	comment string,
) {
	params := &serviceports.LogActionParams{
		Resource:       permission.ResourcePayCode,
		ResourceID:     current.ID.String(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	options := []serviceports.LogOption{auditservice.WithComment(comment)}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
		options = append(options, auditservice.WithDiff(previous, current))
	}
	if err := s.auditService.LogAction(params, options...); err != nil {
		s.l.Error("failed to log pay code audit action", zap.Error(err))
	}
}
