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
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func (s *Service) ListAdvances(
	ctx context.Context,
	req *repositories.ListPayAdvancesRequest,
) (*pagination.ListResult[*driverpay.PayAdvance], error) {
	return s.advanceRepo.List(ctx, req)
}

func (s *Service) ListAdvancesConnection(
	ctx context.Context,
	req *repositories.ListPayAdvanceConnectionRequest,
) (*pagination.CursorListResult[*driverpay.PayAdvance], error) {
	return s.advanceRepo.ListConnection(ctx, req)
}

func (s *Service) GetAdvance(
	ctx context.Context,
	req repositories.GetPayAdvanceByIDRequest,
) (*driverpay.PayAdvance, error) {
	return s.advanceRepo.GetByID(ctx, req)
}

func (s *Service) IssueAdvance(
	ctx context.Context,
	entity *driverpay.PayAdvance,
	actor *serviceports.RequestActor,
) (*driverpay.PayAdvance, error) {
	if err := requireActor(actor, "Pay advance issuance"); err != nil {
		return nil, err
	}
	entity.Status = driverpay.AdvanceStatusOutstanding
	entity.RecoveredMinor = 0
	entity.WrittenOffMinor = 0

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	entity.CreatedByID = actor.UserID
	created, err := s.advanceRepo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logAdvanceAudit(created, nil, actor.UserID, permission.OpCreate, "Pay advance issued")
	return created, nil
}

func (s *Service) UpdateAdvance(
	ctx context.Context,
	entity *driverpay.PayAdvance,
	actor *serviceports.RequestActor,
) (*driverpay.PayAdvance, error) {
	if err := requireActor(actor, "Pay advance update"); err != nil {
		return nil, err
	}
	previous, err := s.advanceRepo.GetByID(ctx, repositories.GetPayAdvanceByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}
	if previous.Status != driverpay.AdvanceStatusOutstanding {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only outstanding advances with no recovery activity can be edited",
		)
	}

	entity.RecoveredMinor = previous.RecoveredMinor
	entity.WrittenOffMinor = previous.WrittenOffMinor
	entity.SyncStatus()

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	updated, err := s.advanceRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logAdvanceAudit(updated, previous, actor.UserID, permission.OpUpdate, "Pay advance updated")
	return updated, nil
}

func (s *Service) WriteOffAdvance(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	advanceID pulid.ID,
	reason string,
	actor *serviceports.RequestActor,
) (*driverpay.PayAdvance, error) {
	if err := requireActor(actor, "Pay advance write-off"); err != nil {
		return nil, err
	}
	if reason == "" {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"A write-off reason is required",
		)
	}
	entity, err := s.advanceRepo.GetByID(ctx, repositories.GetPayAdvanceByIDRequest{
		ID:         advanceID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	outstanding := entity.OutstandingMinor()
	if outstanding <= 0 {
		return nil, errortypes.NewValidationError(
			"advanceId",
			errortypes.ErrInvalidOperation,
			"Advance has no outstanding balance to write off",
		)
	}

	previous := *entity
	now := timeutils.NowUnix()
	entity.WrittenOffMinor += outstanding
	entity.WriteOffReason = reason
	entity.WrittenOffByID = actor.UserID
	entity.WrittenOffAt = &now
	entity.SyncStatus()

	updated, err := s.advanceRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logAdvanceAudit(updated, &previous, actor.UserID, permission.OpCancel,
		"Pay advance written off: "+reason)
	return updated, nil
}

func (s *Service) logAdvanceAudit(
	current, previous *driverpay.PayAdvance,
	userID pulid.ID,
	operation permission.Operation,
	comment string,
) {
	params := &serviceports.LogActionParams{
		Resource:       permission.ResourcePayAdvance,
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
		s.l.Error("failed to log pay advance audit action", zap.Error(err))
	}
}
