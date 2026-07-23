package driversettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type CreateDisputeRequest struct {
	TenantInfo        pagination.TenantInfo
	WorkerID          pulid.ID
	SubmittedByUserID pulid.ID
	SettlementID      pulid.ID
	SettlementLineID  *pulid.ID
	Category          driversettlement.DisputeCategory
	Description       string
}

type ResolveDisputeRequest struct {
	TenantInfo     pagination.TenantInfo
	DisputeID      pulid.ID
	Approve        bool
	ResolutionNote string
	Adjustment     *AdjustmentLineInput
}

func (s *Service) ListDisputesConnection(
	ctx context.Context,
	req *repositories.ListSettlementDisputeConnectionRequest,
) (*pagination.CursorListResult[*driversettlement.Dispute], error) {
	return s.disputeRepo.ListConnection(ctx, req)
}

func (s *Service) GetDispute(
	ctx context.Context,
	req repositories.GetSettlementDisputeByIDRequest,
) (*driversettlement.Dispute, error) {
	return s.disputeRepo.GetByID(ctx, req)
}

func (s *Service) CountOpenDisputes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (int, error) {
	return s.disputeRepo.CountOpen(ctx, tenantInfo)
}

func (s *Service) ListDisputesForWorker(
	ctx context.Context,
	req *repositories.ListSettlementDisputesForWorkerRequest,
) ([]*driversettlement.Dispute, error) {
	return s.disputeRepo.ListForWorker(ctx, req)
}

func (s *Service) CreateDispute(
	ctx context.Context,
	req *CreateDisputeRequest,
) (*driversettlement.Dispute, error) {
	if err := s.requireDisputesEnabled(ctx, req.TenantInfo); err != nil {
		return nil, err
	}

	settlement, err := s.settlementRepo.GetByID(ctx, repositories.GetDriverSettlementByIDRequest{
		ID:           req.SettlementID,
		TenantInfo:   req.TenantInfo,
		IncludeLines: req.SettlementLineID != nil,
	})
	if err != nil {
		return nil, err
	}
	if settlement.WorkerID != req.WorkerID {
		return nil, errortypes.NewValidationError(
			"settlementId",
			errortypes.ErrInvalid,
			"This settlement does not belong to you",
		)
	}
	if settlement.Status != driversettlement.StatusPosted &&
		settlement.Status != driversettlement.StatusPaid &&
		settlement.Status != driversettlement.StatusApproved {
		return nil, errortypes.NewValidationError(
			"settlementId",
			errortypes.ErrInvalidOperation,
			"Only issued settlements can be disputed",
		)
	}
	if req.SettlementLineID != nil && !lineBelongsToSettlement(settlement, *req.SettlementLineID) {
		return nil, errortypes.NewValidationError(
			"settlementLineId",
			errortypes.ErrInvalid,
			"The disputed line does not belong to this settlement",
		)
	}

	openDisputes, err := s.disputeRepo.ListForWorker(
		ctx,
		&repositories.ListSettlementDisputesForWorkerRequest{
			TenantInfo: req.TenantInfo,
			WorkerID:   req.WorkerID,
			Statuses: []driversettlement.DisputeStatus{
				driversettlement.DisputeStatusOpen,
				driversettlement.DisputeStatusInReview,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	for _, existing := range openDisputes {
		if existing.SettlementID != req.SettlementID {
			continue
		}
		sameLine := existing.SettlementLineID == nil && req.SettlementLineID == nil
		if existing.SettlementLineID != nil && req.SettlementLineID != nil {
			sameLine = *existing.SettlementLineID == *req.SettlementLineID
		}
		if sameLine {
			return nil, errortypes.NewValidationError(
				"settlementId",
				errortypes.ErrInvalidOperation,
				"You already have an open dispute for this item",
			)
		}
	}

	entity := &driversettlement.Dispute{
		BusinessUnitID:    req.TenantInfo.BuID,
		OrganizationID:    req.TenantInfo.OrgID,
		SettlementID:      req.SettlementID,
		SettlementLineID:  req.SettlementLineID,
		WorkerID:          req.WorkerID,
		Status:            driversettlement.DisputeStatusOpen,
		Category:          req.Category,
		Description:       req.Description,
		SubmittedByUserID: req.SubmittedByUserID,
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.disputeRepo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logDisputeAudit(
		ctx,
		created, nil, req.SubmittedByUserID, permission.OpCreate, "Dispute submitted")
	return created, nil
}

func (s *Service) WithdrawDispute(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	disputeID pulid.ID,
	workerID pulid.ID,
) (*driversettlement.Dispute, error) {
	dispute, err := s.disputeRepo.GetByID(ctx, repositories.GetSettlementDisputeByIDRequest{
		ID:         disputeID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if dispute.WorkerID != workerID {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalid,
			"This dispute does not belong to you",
		)
	}
	if dispute.Status.IsTerminal() {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalidOperation,
			"This dispute has already been resolved",
		)
	}

	previous := *dispute
	now := timeutils.NowUnix()
	dispute.Status = driversettlement.DisputeStatusWithdrawn
	dispute.ResolvedAt = &now

	updated, err := s.disputeRepo.Update(ctx, dispute)
	if err != nil {
		return nil, err
	}
	s.logDisputeAudit(
		ctx,
		updated,
		&previous,
		tenantInfo.UserID,
		permission.OpUpdate,
		"Dispute withdrawn by driver",
	)
	return updated, nil
}

func (s *Service) StartDisputeReview(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	disputeID pulid.ID,
	actor *serviceports.RequestActor,
) (*driversettlement.Dispute, error) {
	if err := requireActor(actor, "Dispute review"); err != nil {
		return nil, err
	}
	dispute, err := s.disputeRepo.GetByID(ctx, repositories.GetSettlementDisputeByIDRequest{
		ID:         disputeID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if dispute.Status != driversettlement.DisputeStatusOpen {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalidOperation,
			"Only open disputes can be moved to review",
		)
	}

	previous := *dispute
	dispute.Status = driversettlement.DisputeStatusInReview
	updated, err := s.disputeRepo.Update(ctx, dispute)
	if err != nil {
		return nil, err
	}
	s.logDisputeAudit(
		ctx,
		updated, &previous, actor.UserID, permission.OpUpdate, "Dispute under review")
	return updated, nil
}

func (s *Service) ResolveDispute(
	ctx context.Context,
	req *ResolveDisputeRequest,
	actor *serviceports.RequestActor,
) (*driversettlement.Dispute, error) {
	if err := requireActor(actor, "Dispute resolution"); err != nil {
		return nil, err
	}
	if req.ResolutionNote == "" {
		return nil, errortypes.NewValidationError(
			"resolutionNote",
			errortypes.ErrRequired,
			"A resolution note is required",
		)
	}
	if !req.Approve && req.Adjustment != nil {
		return nil, errortypes.NewValidationError(
			"adjustment",
			errortypes.ErrInvalid,
			"A denied dispute cannot include an adjustment",
		)
	}

	dispute, err := s.disputeRepo.GetByID(ctx, repositories.GetSettlementDisputeByIDRequest{
		ID:         req.DisputeID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if dispute.Status.IsTerminal() {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalidOperation,
			"This dispute has already been resolved",
		)
	}

	previous := *dispute
	now := timeutils.NowUnix()

	if req.Adjustment != nil {
		lineID, adjErr := s.applyDisputeAdjustment(ctx, req, dispute, actor)
		if adjErr != nil {
			return nil, adjErr
		}
		if !lineID.IsNil() {
			dispute.ResolutionLineID = &lineID
		}
	}

	if req.Approve {
		dispute.Status = driversettlement.DisputeStatusResolved
	} else {
		dispute.Status = driversettlement.DisputeStatusDenied
	}
	dispute.ResolutionNote = req.ResolutionNote
	dispute.ResolvedByID = &actor.UserID
	dispute.ResolvedAt = &now

	updated, err := s.disputeRepo.Update(ctx, dispute)
	if err != nil {
		return nil, err
	}
	comment := "Dispute denied"
	if req.Approve {
		comment = "Dispute resolved"
	}
	s.logDisputeAudit(
		ctx,
		updated, &previous, actor.UserID, permission.OpApprove, comment)
	if s.driverNotify != nil {
		title := "Your pay dispute was denied"
		message := "Dispatch reviewed your dispute and the original settlement stands. See the note in Dash."
		if req.Approve {
			title = "Your pay dispute was resolved"
			message = "Dispatch resolved your dispute in your favor. See the details in Dash."
		}
		s.driverNotify.Notify(ctx, &drivernotificationservice.DriverNotification{
			TenantInfo: req.TenantInfo,
			WorkerID:   updated.WorkerID,
			EventType:  "dash.dispute_resolved",
			Priority:   notification.PriorityHigh,
			Title:      title,
			Message:    message,
			Link:       "/dash/money",
			RelatedEntities: map[string]any{
				"disputeId": updated.ID.String(),
			},
		})
	}
	return updated, nil
}

func (s *Service) applyDisputeAdjustment(
	ctx context.Context,
	req *ResolveDisputeRequest,
	dispute *driversettlement.Dispute,
	actor *serviceports.RequestActor,
) (pulid.ID, error) {
	target, err := s.settlementRepo.GetOpenDraftForWorker(
		ctx,
		repositories.GetOpenDraftForWorkerRequest{
			TenantInfo: req.TenantInfo,
			WorkerID:   dispute.WorkerID,
		},
	)
	if err != nil {
		if !errortypes.IsNotFoundError(err) {
			return pulid.Nil, err
		}
		target, err = s.generateDisputeSettlement(ctx, req.TenantInfo, dispute.WorkerID, actor)
		if err != nil {
			return pulid.Nil, err
		}
	}

	updated, err := s.AddAdjustmentLine(ctx, req.TenantInfo, target.ID, req.Adjustment, actor)
	if err != nil {
		return pulid.Nil, err
	}
	for _, line := range updated.Lines {
		if line.Category == driversettlement.LineCategoryAdjustment &&
			line.Description == req.Adjustment.Description &&
			line.AmountMinor == req.Adjustment.AmountMinor {
			return line.ID, nil
		}
	}
	return pulid.Nil, nil
}

func (s *Service) requireDisputesEnabled(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) error {
	control, err := s.dashControlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return err
	}
	if !control.AllowSettlementDisputes {
		return errortypes.NewValidationError(
			"feature",
			errortypes.ErrInvalidOperation,
			"Your carrier handles pay questions outside Dash — call your fleet manager.",
		)
	}
	return nil
}

func (s *Service) generateDisputeSettlement(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	actor *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	control, err := s.settlementControl.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	bounds := ResolveCurrentPeriod(control, timeutils.NowUnix())
	settlement, err := s.GenerateOffCycle(ctx, &GenerateForWorkerRequest{
		TenantInfo:  tenantInfo,
		WorkerID:    workerID,
		PeriodStart: bounds.PeriodStart,
		PeriodEnd:   bounds.PeriodEnd,
		PayDate:     bounds.PayDate,
	}, actor)
	if err != nil {
		return nil, err
	}
	if settlement == nil {
		return nil, errortypes.NewValidationError(
			"adjustment",
			errortypes.ErrInvalidOperation,
			"No open settlement exists for this driver and one could not be created; generate an off-cycle settlement first",
		)
	}
	return settlement, nil
}

func lineBelongsToSettlement(
	settlement *driversettlement.Settlement,
	lineID pulid.ID,
) bool {
	for _, line := range settlement.Lines {
		if line.ID == lineID {
			return true
		}
	}
	return false
}

func (s *Service) logDisputeAudit(
	ctx context.Context,
	current *driversettlement.Dispute,
	previous *driversettlement.Dispute,
	userID pulid.ID,
	operation permission.Operation,
	comment string,
) {
	s.publishRealtimeInvalidation(
		ctx,
		permission.ResourceSettlementDispute.String(),
		operation,
		current.ID,
		current.OrganizationID,
		current.BusinessUnitID,
		userID,
	)
	params := &serviceports.LogActionParams{
		Resource:       permission.ResourceSettlementDispute,
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
		s.l.Error("failed to log settlement dispute audit action", zap.Error(err))
	}
}
