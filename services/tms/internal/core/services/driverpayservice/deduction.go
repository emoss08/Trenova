package driverpayservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *Service) ListDeductions(
	ctx context.Context,
	req *repositories.ListRecurringDeductionsRequest,
) (*pagination.ListResult[*driverpay.RecurringDeduction], error) {
	return s.deductionRepo.List(ctx, req)
}

func (s *Service) ListDeductionsConnection(
	ctx context.Context,
	req *repositories.ListRecurringDeductionConnectionRequest,
) (*pagination.CursorListResult[*driverpay.RecurringDeduction], error) {
	return s.deductionRepo.ListConnection(ctx, req)
}

func (s *Service) GetDeduction(
	ctx context.Context,
	req repositories.GetRecurringDeductionByIDRequest,
) (*driverpay.RecurringDeduction, error) {
	return s.deductionRepo.GetByID(ctx, req)
}

func (s *Service) CreateDeduction(
	ctx context.Context,
	entity *driverpay.RecurringDeduction,
	autoLinkEscrow bool,
	actor *serviceports.RequestActor,
) (*driverpay.RecurringDeduction, error) {
	if err := requireActor(actor, "Recurring deduction creation"); err != nil {
		return nil, err
	}
	if err := s.CheckDeduction(ctx, entity, autoLinkEscrow); err != nil {
		return nil, err
	}

	entity.CreatedByID = actor.UserID
	created, err := s.deductionRepo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logDeductionAudit(created, nil, actor.UserID, permission.OpCreate,
		"Recurring deduction created")
	return created, nil
}

func (s *Service) UpdateDeduction(
	ctx context.Context,
	entity *driverpay.RecurringDeduction,
	actor *serviceports.RequestActor,
) (*driverpay.RecurringDeduction, error) {
	if err := requireActor(actor, "Recurring deduction update"); err != nil {
		return nil, err
	}
	if err := s.CheckDeduction(ctx, entity, false); err != nil {
		return nil, err
	}

	previous, err := s.deductionRepo.GetByID(ctx, repositories.GetRecurringDeductionByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}
	entity.DeductedToDateMinor = previous.DeductedToDateMinor

	updated, err := s.deductionRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logDeductionAudit(updated, previous, actor.UserID, permission.OpUpdate,
		"Recurring deduction updated")
	return updated, nil
}

func (s *Service) logDeductionAudit(
	current, previous *driverpay.RecurringDeduction,
	userID pulid.ID,
	operation permission.Operation,
	comment string,
) {
	params := &serviceports.LogActionParams{
		Resource:       permission.ResourceRecurringDeduction,
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
		s.l.Error("failed to log recurring deduction audit action", zap.Error(err))
	}
}
