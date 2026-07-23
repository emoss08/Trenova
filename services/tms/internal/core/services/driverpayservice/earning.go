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

func (s *Service) ListEarnings(
	ctx context.Context,
	req *repositories.ListRecurringEarningsRequest,
) (*pagination.ListResult[*driverpay.RecurringEarning], error) {
	return s.earningRepo.List(ctx, req)
}

func (s *Service) ListEarningsConnection(
	ctx context.Context,
	req *repositories.ListRecurringEarningConnectionRequest,
) (*pagination.CursorListResult[*driverpay.RecurringEarning], error) {
	return s.earningRepo.ListConnection(ctx, req)
}

func (s *Service) GetEarning(
	ctx context.Context,
	req repositories.GetRecurringEarningByIDRequest,
) (*driverpay.RecurringEarning, error) {
	return s.earningRepo.GetByID(ctx, req)
}

func (s *Service) CreateEarning(
	ctx context.Context,
	entity *driverpay.RecurringEarning,
	actor *serviceports.RequestActor,
) (*driverpay.RecurringEarning, error) {
	if err := requireActor(actor, "Recurring earning creation"); err != nil {
		return nil, err
	}
	if err := s.resolvePayCode(
		ctx,
		pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		entity.PayCodeID,
		driverpay.PayCodeDirectionEarning,
	); err != nil {
		return nil, err
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	entity.CreatedByID = actor.UserID
	created, err := s.earningRepo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logEarningAudit(created, nil, actor.UserID, permission.OpCreate,
		"Recurring earning created")
	return created, nil
}

func (s *Service) UpdateEarning(
	ctx context.Context,
	entity *driverpay.RecurringEarning,
	actor *serviceports.RequestActor,
) (*driverpay.RecurringEarning, error) {
	if err := requireActor(actor, "Recurring earning update"); err != nil {
		return nil, err
	}
	if err := s.resolvePayCode(
		ctx,
		pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		entity.PayCodeID,
		driverpay.PayCodeDirectionEarning,
	); err != nil {
		return nil, err
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	previous, err := s.earningRepo.GetByID(ctx, repositories.GetRecurringEarningByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}
	entity.PaidToDateMinor = previous.PaidToDateMinor

	updated, err := s.earningRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logEarningAudit(updated, previous, actor.UserID, permission.OpUpdate,
		"Recurring earning updated")
	return updated, nil
}

func (s *Service) logEarningAudit(
	current, previous *driverpay.RecurringEarning,
	userID pulid.ID,
	operation permission.Operation,
	comment string,
) {
	params := &serviceports.LogActionParams{
		Resource:       permission.ResourceRecurringEarning,
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
		s.l.Error("failed to log recurring earning audit action", zap.Error(err))
	}
}
