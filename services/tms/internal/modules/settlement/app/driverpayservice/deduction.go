package driverpayservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/recurringpay"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *Service) ListDeductions(
	ctx context.Context,
	req *driverpayports.ListRecurringDeductionsRequest,
) (*pagination.ListResult[*recurringpay.RecurringDeduction], error) {
	return s.deductionRepo.List(ctx, req)
}

func (s *Service) ListDeductionsConnection(
	ctx context.Context,
	req *driverpayports.ListRecurringDeductionConnectionRequest,
) (*pagination.CursorListResult[*recurringpay.RecurringDeduction], error) {
	return s.deductionRepo.ListConnection(ctx, req)
}

func (s *Service) GetDeduction(
	ctx context.Context,
	req driverpayports.GetRecurringDeductionByIDRequest,
) (*recurringpay.RecurringDeduction, error) {
	return s.deductionRepo.GetByID(ctx, req)
}

func (s *Service) CreateDeduction(
	ctx context.Context,
	entity *recurringpay.RecurringDeduction,
	autoLinkEscrow bool,
	actor *permissionports.RequestActor,
) (*recurringpay.RecurringDeduction, error) {
	if err := settlementshared.RequireActor(actor, "Recurring deduction creation"); err != nil {
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
	entity *recurringpay.RecurringDeduction,
	actor *permissionports.RequestActor,
) (*recurringpay.RecurringDeduction, error) {
	if err := settlementshared.RequireActor(actor, "Recurring deduction update"); err != nil {
		return nil, err
	}
	if err := s.CheckDeduction(ctx, entity, false); err != nil {
		return nil, err
	}

	previous, err := s.deductionRepo.GetByID(ctx, driverpayports.GetRecurringDeductionByIDRequest{
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
	current, previous *recurringpay.RecurringDeduction,
	userID pulid.ID,
	operation permission.Operation,
	comment string,
) {
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceRecurringDeduction,
		ResourceID:     current.ID.String(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	options := []auditports.LogOption{auditservice.WithComment(comment)}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
		options = append(options, auditservice.WithDiff(previous, current))
	}
	if err := s.auditService.LogAction(params, options...); err != nil {
		s.l.Error("failed to log recurring deduction audit action", zap.Error(err))
	}
}
