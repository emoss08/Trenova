package driverportalservice

import (
	"context"
	"mime/multipart"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/dashcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementaccrualservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementreviewservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementworkspaceservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payexpense"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// --- PTO ---

func (s *Service) MyPTO(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*worker.WorkerPTO, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	return s.portalRepo.ListWorkerPTO(ctx, tenantInfo, wrk.ID)
}

type RequestMyPTORequest struct {
	Type      worker.PTOType
	StartDate int64
	EndDate   int64
	Reason    string
}

func (s *Service) RequestMyPTO(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req *RequestMyPTORequest,
) (*worker.WorkerPTO, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if _, err = s.RequireFeature(ctx, tenantInfo,
		func(control *dashcontrol.DashControl) bool { return control.AllowPtoRequests },
		"Your carrier handles time-off requests outside Dash — talk to your fleet manager.",
	); err != nil {
		return nil, err
	}

	entity := &worker.WorkerPTO{
		BusinessUnitID: tenantInfo.BuID,
		OrganizationID: tenantInfo.OrgID,
		WorkerID:       wrk.ID,
		Status:         worker.PTOStatusRequested,
		Type:           req.Type,
		StartDate:      req.StartDate,
		EndDate:        req.EndDate,
		Reason:         strings.TrimSpace(req.Reason),
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.ptoService.Create(ctx, entity, tenantInfo.UserID)
	if err != nil {
		return nil, err
	}
	s.NotifyDispatch(
		ctx,
		tenantInfo,
		"dash.pto_requested",
		"Time-off request",
		strings.TrimSpace(wrk.FirstName+" "+wrk.LastName)+" requested time off.",
		"/dispatch-management/workers",
		map[string]any{"workerId": wrk.ID.String(), "ptoId": created.ID.String()},
	)
	return created, nil
}

func (s *Service) MyPTOBalances(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*ptoledgerservice.BalanceView, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if s.ptoLedger == nil {
		return []*ptoledgerservice.BalanceView{}, nil
	}

	views, err := s.ptoLedger.GetBalances(ctx, tenantInfo, wrk.ID)
	if err != nil {
		return nil, err
	}
	tracked := make([]*ptoledgerservice.BalanceView, 0, len(views))
	for _, view := range views {
		if view.Tracked {
			tracked = append(tracked, view)
		}
	}
	return tracked, nil
}

func (s *Service) CancelMyPTO(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ptoID pulid.ID,
) (*worker.WorkerPTO, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	pto, err := s.ptoRepo.GetByID(ctx, &workerptoports.GetPTOByIDRequest{
		ID:         ptoID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if pto.WorkerID != wrk.ID {
		return nil, errortypes.NewNotFoundError("Time-off request not found")
	}
	if pto.Status != worker.PTOStatusRequested {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalidOperation,
			"Only pending requests can be cancelled",
		)
	}

	return s.ptoService.CancelRequested(ctx, &workerptoports.UpdatePTOStatusRequest{
		ID:         ptoID,
		TenantInfo: tenantInfo,
		Status:     worker.PTOStatusCancelled,
		UserID:     tenantInfo.UserID,
	})
}

// --- Expenses ---

type SubmitMyExpenseRequest struct {
	ShipmentID   *pulid.ID
	PayCodeID    *pulid.ID
	AmountMinor  int64
	Description  string
	IncurredDate int64
}

func (s *Service) MyExpenses(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*payexpense.Expense, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	return s.expenseService.ListExpensesForWorker(
		ctx,
		&driverpayports.ListDriverExpensesForWorkerRequest{
			TenantInfo: tenantInfo,
			WorkerID:   wrk.ID,
		},
	)
}

func (s *Service) SubmitMyExpense(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req *SubmitMyExpenseRequest,
) (*payexpense.Expense, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if _, err = s.RequireFeature(ctx, tenantInfo,
		func(control *dashcontrol.DashControl) bool { return control.AllowExpenseSubmission },
		"Your carrier handles expense reimbursements outside Dash — see your fleet manager.",
	); err != nil {
		return nil, err
	}
	if req.ShipmentID != nil && !req.ShipmentID.IsNil() {
		assigned, assignedErr := s.portalRepo.WorkerAssignedToShipment(
			ctx,
			tenantInfo,
			wrk.ID,
			*req.ShipmentID,
		)
		if assignedErr != nil {
			return nil, assignedErr
		}
		if !assigned {
			return nil, errortypes.NewNotFoundError("Load not found")
		}
	}

	return s.expenseService.SubmitExpense(ctx, &driversettlementreviewservice.SubmitExpenseRequest{
		TenantInfo:        tenantInfo,
		WorkerID:          wrk.ID,
		SubmittedByUserID: tenantInfo.UserID,
		ShipmentID:        req.ShipmentID,
		PayCodeID:         req.PayCodeID,
		AmountMinor:       req.AmountMinor,
		Description:       req.Description,
		IncurredDate:      req.IncurredDate,
	})
}

func (s *Service) CancelMyExpense(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	expenseID pulid.ID,
) (*payexpense.Expense, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	return s.expenseService.CancelExpense(ctx, tenantInfo, expenseID, wrk.ID)
}

// UploadMyExpenseReceipt stores a receipt image against the driver's own
// expense and links it for the reviewer.
func (s *Service) UploadMyExpenseReceipt(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	expenseID pulid.ID,
	file *multipart.FileHeader,
	actor *permissionports.RequestActor,
) (*payexpense.Expense, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Receipt upload requires an authenticated user",
		)
	}

	expense, err := s.expenseService.GetExpense(
		ctx,
		driverpayports.GetDriverExpenseByIDRequest{ID: expenseID, TenantInfo: tenantInfo},
	)
	if err != nil {
		return nil, err
	}
	if expense.WorkerID != wrk.ID {
		return nil, errortypes.NewNotFoundError("Expense not found")
	}

	result, err := s.documentService.Upload(ctx, &documentservice.UploadRequest{
		TenantInfo:   tenantInfo,
		Actor:        *actor,
		File:         file,
		ResourceID:   expenseID.String(),
		ResourceType: "driver_expense",
	})
	if err != nil {
		return nil, err
	}
	return s.expenseService.SetExpenseReceipt(
		ctx,
		tenantInfo,
		expenseID,
		wrk.ID,
		result.Document.ID,
	)
}

// --- Assignment acknowledgment ---

type RespondToAssignmentRequest struct {
	AssignmentID pulid.ID
	Accept       bool
	Reason       string
}

func (s *Service) RespondToMyAssignment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req *RespondToAssignmentRequest,
) error {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return err
	}
	control, err := s.RequireFeature(ctx, tenantInfo,
		func(dc *dashcontrol.DashControl) bool { return dc.RequireLoadAcknowledgment },
		"Your carrier doesn't use load acceptance in Dash.",
	)
	if err != nil {
		return err
	}

	ack := shipmentvocab.AssignmentAckAccepted
	if !req.Accept {
		if !control.AllowLoadRefusals {
			return errortypes.NewValidationError(
				"accept",
				errortypes.ErrInvalidOperation,
				"Your carrier doesn't allow declining loads in Dash — call your dispatcher instead.",
			)
		}
		ack = shipmentvocab.AssignmentAckDeclined
		if strings.TrimSpace(req.Reason) == "" {
			return errortypes.NewValidationError(
				"reason",
				errortypes.ErrRequired,
				"Tell dispatch why you're declining so they can replan",
			)
		}
	}

	updated, err := s.portalRepo.UpdateAssignmentAck(
		ctx,
		tenantInfo,
		wrk.ID,
		req.AssignmentID,
		ack,
		strings.TrimSpace(req.Reason),
	)
	if err != nil {
		return err
	}

	if !req.Accept {
		s.NotifyDispatch(
			ctx,
			tenantInfo,
			"dash.load_declined",
			"Driver declined a load",
			strings.TrimSpace(wrk.FirstName+" "+wrk.LastName)+" declined a load: "+req.Reason,
			"/shipment-management/shipments",
			map[string]any{
				"assignmentId": updated.ID.String(),
				"moveId":       updated.ShipmentMoveID.String(),
				"workerId":     wrk.ID.String(),
			},
		)
	}
	return nil
}

// --- Pay estimate & YTD ---

func (s *Service) MyLoadPayEstimate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	moveID pulid.ID,
) (*driversettlementaccrualservice.MovePayEstimate, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if _, err = s.RequireFeature(ctx, tenantInfo,
		func(control *dashcontrol.DashControl) bool {
			return control.ShowLoadPay && control.ShowPayEstimates
		},
		"Pay estimates aren't enabled for your carrier.",
	); err != nil {
		return nil, err
	}
	assigned, err := s.portalRepo.WorkerAssignedToMove(ctx, tenantInfo, wrk.ID, moveID)
	if err != nil {
		return nil, err
	}
	if !assigned {
		return nil, errortypes.NewNotFoundError("Load not found")
	}
	return s.payEstimator.EstimateWorkerMovePay(ctx, tenantInfo, wrk.ID, shipmentID, moveID)
}

func (s *Service) MyYtdPay(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	year int,
) (*driversettlementworkspaceservice.WorkerYTDPaySummary, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	summaries, err := s.payHistory.GetYTDPaySummaries(ctx, tenantInfo, year, "")
	if err != nil {
		return nil, err
	}
	for _, summary := range summaries {
		if summary.WorkerID == wrk.ID {
			return summary, nil
		}
	}
	return nil, errortypes.NewNotFoundError("No pay recorded for this year")
}

// --- shared helpers ---

func (s *Service) NotifyDispatch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	eventType, title, message, link string,
	related map[string]any,
) {
	if s.notificationService == nil {
		return
	}
	buID := tenantInfo.BuID
	entity := &notification.Notification{
		OrganizationID:  tenantInfo.OrgID,
		BusinessUnitID:  &buID,
		EventType:       eventType,
		Priority:        notification.PriorityHigh,
		Channel:         notification.ChannelGlobal,
		Title:           title,
		Message:         message,
		Data:            map[string]any{"link": link},
		RelatedEntities: related,
		Source:          "driver_portal",
	}
	if _, err := s.notificationService.Create(ctx, entity); err != nil {
		s.l.Warn("failed to notify dispatch", zap.String("eventType", eventType), zap.Error(err))
	}
}
