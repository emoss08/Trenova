//nolint:gocritic // existing legacy workflow/API shape is intentionally kept stable
package workerptoservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/internal/core/services/ptoledgerservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/smsjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger          *zap.Logger
	DB              ports.DBConnection
	Repo            repositories.WorkerPTORepository
	UserRepo        repositories.UserRepository
	WorkerRepo      repositories.WorkerRepository
	Ledger          *ptoledgerservice.Service
	WorkflowStarter services.WorkflowStarter
	AuditService    services.AuditService
	DriverNotify    *drivernotificationservice.Service `optional:"true"`
	Realtime        services.RealtimeService           `optional:"true"`
}

type Service struct {
	l               *zap.Logger
	db              ports.DBConnection
	repo            repositories.WorkerPTORepository
	userRepo        repositories.UserRepository
	workerRepo      repositories.WorkerRepository
	ledger          *ptoledgerservice.Service
	workflowStarter services.WorkflowStarter
	auditService    services.AuditService
	driverNotify    *drivernotificationservice.Service
	realtime        services.RealtimeService
}

const maxPTOChartRangeSeconds int64 = 366 * 24 * 60 * 60

var _ services.WorkerPTOService = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		l:               p.Logger.Named("service.workerpto"),
		db:              p.DB,
		repo:            p.Repo,
		userRepo:        p.UserRepo,
		workerRepo:      p.WorkerRepo,
		ledger:          p.Ledger,
		workflowStarter: p.WorkflowStarter,
		auditService:    p.AuditService,
		driverNotify:    p.DriverNotify,
		realtime:        p.Realtime,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListPTORequest,
) (*pagination.CursorListResult[*worker.WorkerPTO], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetPTOByIDRequest,
) (*worker.WorkerPTO, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *worker.WorkerPTO,
	userID pulid.ID,
) (*worker.WorkerPTO, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("userID", userID.String()),
		zap.String("ptoID", entity.GetResourceID()),
	)

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	overlaps, err := s.repo.HasOverlap(ctx, &repositories.PTOOverlapRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		WorkerID:  entity.WorkerID,
		StartDate: entity.StartDate,
		EndDate:   entity.EndDate,
	})
	if err != nil {
		log.Error("failed to check PTO overlap", zap.Error(err))
		return nil, err
	}
	if overlaps {
		return nil, errortypes.NewValidationError(
			"startDate",
			errortypes.ErrInvalid,
			"This worker already has pending or approved time off that overlaps these dates",
		)
	}

	availability, err := s.prepareLedgerFields(ctx, entity, pulid.Nil)
	if err != nil {
		return nil, err
	}

	autoApprove := availability != nil && availability.Policy != nil &&
		!availability.Policy.Policy.RequiresApproval
	if autoApprove {
		entity.Status = worker.PTOStatusApproved
		entity.ApproverID = userID
		entity.AutoApproved = true
	}

	var createdEntity *worker.WorkerPTO
	err = s.inTx(ctx, func(txCtx context.Context) error {
		created, txErr := s.repo.Create(txCtx, entity)
		if txErr != nil {
			return txErr
		}
		if autoApprove {
			if txErr = s.bookUsage(txCtx, tenantOf(created), created, userID); txErr != nil {
				return txErr
			}
		}
		createdEntity = created
		return nil
	})
	if err != nil {
		log.Error("failed to create PTO", zap.Error(err))
		return nil, err
	}
	s.publishPTOInvalidation(ctx, createdEntity, permission.OpCreate, userID)
	if autoApprove {
		s.notifyDriverPTO(ctx, tenantOf(createdEntity), createdEntity, driverPTONotice{
			eventType: "dash.pto_reviewed",
			approved:  true,
		})
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceWorkerPTO,
		ResourceID:     createdEntity.GetResourceID(),
		Operation:      permission.OpCreate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.GetOrganizationID(),
		BusinessUnitID: createdEntity.GetBusinessUnitID(),
	}, auditservice.WithComment("PTO created")); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
		return nil, err
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *worker.WorkerPTO,
	userID pulid.ID,
) (*worker.WorkerPTO, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("userID", userID.String()),
		zap.String("ptoID", entity.GetResourceID()),
	)

	current, err := s.repo.GetByID(ctx, &repositories.GetPTOByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}

	if current.Status != worker.PTOStatusRequested {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"PTO is {0} and can no longer be edited", strings.ToLower(string(current.Status)),
		)
	}

	entity.WorkerID = current.WorkerID
	entity.Status = current.Status
	entity.ApproverID = current.ApproverID
	entity.RejectorID = current.RejectorID
	entity.CancelledByID = current.CancelledByID
	entity.CreatedAt = current.CreatedAt

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	overlaps, err := s.repo.HasOverlap(ctx, &repositories.PTOOverlapRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		WorkerID:  entity.WorkerID,
		StartDate: entity.StartDate,
		EndDate:   entity.EndDate,
		ExcludeID: entity.ID,
	})
	if err != nil {
		log.Error("failed to check PTO overlap", zap.Error(err))
		return nil, err
	}
	if overlaps {
		return nil, errortypes.NewValidationError(
			"startDate",
			errortypes.ErrInvalid,
			"This worker already has pending or approved time off that overlaps these dates",
		)
	}

	if _, err = s.prepareLedgerFields(ctx, entity, entity.ID); err != nil {
		return nil, err
	}

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update PTO", zap.Error(err))
		return nil, err
	}
	s.publishPTOInvalidation(ctx, updated, permission.OpUpdate, userID)

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceWorkerPTO,
		ResourceID:     updated.GetResourceID(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		PreviousState:  jsonutils.MustToJSON(current),
		CurrentState:   jsonutils.MustToJSON(updated),
		OrganizationID: updated.GetOrganizationID(),
		BusinessUnitID: updated.GetBusinessUnitID(),
	}, auditservice.WithComment("PTO updated"), auditservice.WithDiff(current, updated)); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updated, nil
}

func (s *Service) Approve(
	ctx context.Context,
	req *repositories.UpdatePTOStatusRequest,
) (*worker.WorkerPTO, error) {
	return s.transition(ctx, transitionParams{
		req:          req,
		target:       worker.PTOStatusApproved,
		notifyWorker: true,
	})
}

func (s *Service) Reject(
	ctx context.Context,
	req *repositories.UpdatePTOStatusRequest,
) (*worker.WorkerPTO, error) {
	if strings.TrimSpace(req.Reason) == "" {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"A reason is required to reject a PTO request",
		)
	}

	return s.transition(ctx, transitionParams{
		req:          req,
		target:       worker.PTOStatusRejected,
		notifyWorker: true,
	})
}

func (s *Service) Cancel(
	ctx context.Context,
	req *repositories.UpdatePTOStatusRequest,
) (*worker.WorkerPTO, error) {
	return s.transition(ctx, transitionParams{
		req:          req,
		target:       worker.PTOStatusCancelled,
		notifyWorker: true,
	})
}

func (s *Service) CancelRequested(
	ctx context.Context,
	req *repositories.UpdatePTOStatusRequest,
) (*worker.WorkerPTO, error) {
	return s.transition(ctx, transitionParams{
		req:          req,
		target:       worker.PTOStatusCancelled,
		notifyWorker: false,
	})
}

func (s *Service) BulkAction(
	ctx context.Context,
	req *services.PTOBulkActionRequest,
) (*services.PTOBulkActionPayload, error) {
	if len(req.PTOIDs) == 0 {
		return nil, errortypes.NewValidationError(
			"ptoIds",
			errortypes.ErrRequired,
			"Select at least one PTO request",
		)
	}
	if !req.Action.IsValid() {
		return nil, errortypes.NewValidationError(
			"action",
			errortypes.ErrInvalid,
			"Bulk action is invalid",
		)
	}
	if req.Action == services.PTOBulkActionReject && strings.TrimSpace(req.Reason) == "" {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"A reason is required to reject PTO requests",
		)
	}

	existing, err := s.repo.GetByIDs(ctx, &repositories.GetPTOsByIDsRequest{
		IDs:        req.PTOIDs,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	byID := make(map[pulid.ID]*worker.WorkerPTO, len(existing))
	for _, pto := range existing {
		byID[pto.ID] = pto
	}

	payload := &services.PTOBulkActionPayload{
		Results: make([]*services.PTOBulkActionResult, 0, len(req.PTOIDs)),
	}
	for _, ptoID := range req.PTOIDs {
		result := &services.PTOBulkActionResult{PTOID: ptoID}

		current, found := byID[ptoID]
		if !found {
			result.Error = "PTO request not found within your organization"
			payload.FailureCount++
			payload.Results = append(payload.Results, result)
			continue
		}

		_, actionErr := s.transition(ctx, transitionParams{
			req: &repositories.UpdatePTOStatusRequest{
				ID:              ptoID,
				TenantInfo:      req.TenantInfo,
				UserID:          req.UserID,
				Reason:          req.Reason,
				ExpectedVersion: current.Version,
			},
			target:       req.Action.TargetStatus(),
			notifyWorker: true,
			current:      current,
		})
		if actionErr != nil {
			result.Error = actionErr.Error()
			payload.FailureCount++
		} else {
			result.Success = true
			payload.SuccessCount++
		}
		payload.Results = append(payload.Results, result)
	}

	return payload, nil
}

type transitionParams struct {
	req          *repositories.UpdatePTOStatusRequest
	target       worker.PTOStatus
	notifyWorker bool
	current      *worker.WorkerPTO
}

func (s *Service) transition(
	ctx context.Context,
	params transitionParams,
) (*worker.WorkerPTO, error) {
	req := params.req
	log := s.l.With(
		zap.String("operation", "transition"),
		zap.String("ptoID", req.ID.String()),
		zap.String("target", string(params.target)),
	)

	current := params.current
	if current == nil {
		var err error
		current, err = s.repo.GetByID(ctx, &repositories.GetPTOByIDRequest{
			ID:         req.ID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return nil, err
		}
	}

	if !current.Status.CanTransitionTo(params.target) {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"PTO is {0} and cannot be {1}", strings.ToLower(string(current.Status)), strings.ToLower(string(params.target)),
		)
	}

	req.Status = params.target
	req.Reason = strings.TrimSpace(req.Reason)
	if req.ExpectedVersion == 0 {
		req.ExpectedVersion = current.Version
	}

	if params.target == worker.PTOStatusApproved {
		if err := s.requireAvailability(ctx, current, current.ID); err != nil {
			return nil, err
		}
	}

	var updated *worker.WorkerPTO
	err := s.inTx(ctx, func(txCtx context.Context) error {
		result, txErr := s.repo.UpdateStatus(txCtx, req)
		if txErr != nil {
			return txErr
		}
		switch params.target { //nolint:exhaustive // only these targets touch the ledger
		case worker.PTOStatusApproved:
			txErr = s.bookUsage(txCtx, req.TenantInfo, result, req.UserID)
		case worker.PTOStatusCancelled:
			if current.Status == worker.PTOStatusApproved {
				txErr = s.releaseUsage(txCtx, req.TenantInfo, result, req.UserID)
			}
		}
		if txErr != nil {
			return txErr
		}
		updated = result
		return nil
	})
	if err != nil {
		return nil, err
	}

	operation := transitionOperation(params.target)
	s.logTransitionAudit(current, updated, operation, req, log)
	s.publishPTOInvalidation(ctx, updated, operation, req.UserID)

	if params.notifyWorker {
		s.notifyTransition(ctx, req, updated, params.target, log)
	}

	return updated, nil
}

func transitionOperation(target worker.PTOStatus) permission.Operation {
	switch target { //nolint:exhaustive // Requested is never a transition target
	case worker.PTOStatusApproved:
		return permission.OpApprove
	case worker.PTOStatusRejected:
		return permission.OpReject
	case worker.PTOStatusCancelled:
		return permission.OpCancel
	default:
		return permission.OpUpdate
	}
}

func (s *Service) logTransitionAudit(
	previous, updated *worker.WorkerPTO,
	operation permission.Operation,
	req *repositories.UpdatePTOStatusRequest,
	log *zap.Logger,
) {
	if s.auditService == nil {
		return
	}

	comment := fmt.Sprintf("PTO %s", strings.ToLower(string(updated.Status)))
	// A decision taken under a delegation has to say so. Otherwise the trail
	// reads as the delegate's own call, and nobody can tell afterwards which
	// manager's authority was actually used.
	if !req.OnBehalfOf.IsNil() {
		comment = fmt.Sprintf("%s on behalf of %s", comment, req.OnBehalfOf)
	}
	if req.Reason != "" {
		comment = fmt.Sprintf("%s: %s", comment, req.Reason)
	}

	if err := s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceWorkerPTO,
		ResourceID:     updated.GetResourceID(),
		Operation:      operation,
		UserID:         req.UserID,
		PreviousState:  jsonutils.MustToJSON(previous),
		CurrentState:   jsonutils.MustToJSON(updated),
		OrganizationID: updated.GetOrganizationID(),
		BusinessUnitID: updated.GetBusinessUnitID(),
	}, auditservice.WithComment(comment), auditservice.WithDiff(previous, updated)); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}
}

func (s *Service) notifyTransition(
	ctx context.Context,
	req *repositories.UpdatePTOStatusRequest,
	updated *worker.WorkerPTO,
	target worker.PTOStatus,
	log *zap.Logger,
) {
	switch target { //nolint:exhaustive // Requested is never a transition target
	case worker.PTOStatusApproved:
		s.notifyDriverPTO(ctx, req.TenantInfo, updated, driverPTONotice{
			eventType: "dash.pto_reviewed",
			approved:  true,
		})
	case worker.PTOStatusRejected:
		s.notifyDriverPTO(ctx, req.TenantInfo, updated, driverPTONotice{
			eventType: "dash.pto_reviewed",
			reason:    req.Reason,
		})
	case worker.PTOStatusCancelled:
		s.notifyDriverPTO(ctx, req.TenantInfo, updated, driverPTONotice{
			eventType: "dash.pto_cancelled",
			reason:    req.Reason,
		})
	}

	s.sendTransitionSMS(ctx, req, updated, target, log)
}

func (s *Service) sendTransitionSMS(
	ctx context.Context,
	req *repositories.UpdatePTOStatusRequest,
	updated *worker.WorkerPTO,
	target worker.PTOStatus,
	log *zap.Logger,
) {
	if s.workflowStarter == nil || !s.workflowStarter.Enabled() {
		log.Warn("workflow starter disabled; skipping PTO SMS")
		return
	}

	user, err := s.userRepo.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  req.TenantInfo.OrgID,
			BuID:   req.TenantInfo.BuID,
			UserID: req.UserID,
		},
	})
	if err != nil {
		log.Error("failed to get acting user for PTO SMS", zap.Error(err))
		return
	}

	wrk, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:         updated.WorkerID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		log.Error("failed to get worker for PTO SMS", zap.Error(err))
		return
	}
	if wrk.PhoneNumber == "" {
		return
	}

	message, ok := transitionSMSMessage(user.Name, updated, target, req.Reason)
	if !ok {
		return
	}

	payload := &smsjobs.SendSMSPayload{
		PhoneNumber:    wrk.PhoneNumber,
		Message:        message,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}

	if _, err = s.workflowStarter.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID: fmt.Sprintf(
			"pto-%s-%s-%d",
			strings.ToLower(string(target)),
			updated.ID.String(),
			timeutils.NowUnix(),
		),
		TaskQueue: temporaltype.SMSTaskQueue,
		StaticSummary: fmt.Sprintf(
			"Sending SMS to worker %s for %s PTO",
			updated.WorkerID.String(),
			strings.ToLower(string(target)),
		),
	}, "SendSMSWorkflow", payload); err != nil {
		log.Error("failed to start PTO SMS workflow", zap.Error(err))
	}
}

func transitionSMSMessage(
	actorName string,
	pto *worker.WorkerPTO,
	target worker.PTOStatus,
	reason string,
) (string, bool) {
	dates := fmt.Sprintf(
		"%s to %s",
		timeutils.UnixToHumanReadable(pto.StartDate),
		timeutils.UnixToHumanReadable(pto.EndDate),
	)

	switch target { //nolint:exhaustive // Requested is never a transition target
	case worker.PTOStatusApproved:
		return fmt.Sprintf("%s has approved your PTO request for dates %s.", actorName, dates), true
	case worker.PTOStatusRejected:
		return fmt.Sprintf(
			"%s has rejected your PTO request for dates %s. Reason: %s",
			actorName,
			dates,
			reason,
		), true
	case worker.PTOStatusCancelled:
		if reason == "" {
			return fmt.Sprintf(
				"%s has cancelled your PTO request for dates %s.",
				actorName,
				dates,
			), true
		}
		return fmt.Sprintf(
			"%s has cancelled your PTO request for dates %s. Reason: %s",
			actorName,
			dates,
			reason,
		), true
	default:
		return "", false
	}
}

func (s *Service) GetChartData(
	ctx context.Context,
	req *repositories.PTOChartRequest,
) ([]*repositories.PTOChartDataPoint, error) {
	if err := s.validateChartRequest(req); err != nil {
		return nil, err
	}

	return s.repo.GetChartData(ctx, req)
}

func (s *Service) validateChartRequest(req *repositories.PTOChartRequest) error {
	multiErr := errortypes.NewMultiError()

	if req == nil {
		multiErr.Add("request", errortypes.ErrRequired, "request is required")
		return multiErr
	}

	if req.Filter == nil {
		multiErr.Add("filter", errortypes.ErrRequired, "filter is required")
	}

	if req.StartDateFrom <= 0 {
		multiErr.Add("startDateFrom", errortypes.ErrInvalid, "startDateFrom must be greater than 0")
	}

	if req.StartDateTo <= 0 {
		multiErr.Add("startDateTo", errortypes.ErrInvalid, "startDateTo must be greater than 0")
	}

	if req.StartDateFrom > 0 && req.StartDateTo > 0 {
		if req.StartDateFrom > req.StartDateTo {
			multiErr.Add(
				"dateRange",
				errortypes.ErrInvalid,
				"startDateFrom must be less than or equal to startDateTo",
			)
		}

		if req.StartDateTo-req.StartDateFrom > maxPTOChartRangeSeconds {
			multiErr.Add(
				"dateRange",
				errortypes.ErrInvalidLength,
				"date range cannot exceed 366 days",
			)
		}
	}

	if req.Type != "" && !strings.EqualFold(req.Type, "all") {
		if _, err := worker.PTOTypeFromString(req.Type); err != nil {
			multiErr.Add(
				"type",
				errortypes.ErrInvalid,
				"type must be one of: all, Personal, Vacation, Sick, Holiday, Bereavement, Maternity, Paternity",
			)
		}
	}

	if req.WorkerID != "" {
		if _, err := pulid.MustParse(req.WorkerID); err != nil {
			multiErr.Add("workerId", errortypes.ErrInvalidFormat, "workerId must be a valid ID")
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) ListUpcoming(
	ctx context.Context,
	req *repositories.ListUpcomingPTORequest,
) (*pagination.CursorListResult[*worker.WorkerPTO], error) {
	return s.repo.ListUpcoming(ctx, req)
}

func (s *Service) publishPTOInvalidation(
	ctx context.Context,
	pto *worker.WorkerPTO,
	action permission.Operation,
	userID pulid.ID,
) {
	if s.realtime == nil || pto == nil {
		return
	}
	err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: pto.OrganizationID,
		BusinessUnitID: pto.BusinessUnitID,
		ActorUserID:    userID,
		ActorType:      services.PrincipalTypeUser,
		ActorID:        userID,
		Resource:       "worker_pto",
		Action:         string(action),
		RecordID:       pto.ID,
	})
	if err != nil {
		s.l.Warn("failed to publish PTO invalidation", zap.Error(err))
	}
}

type driverPTONotice struct {
	eventType string
	approved  bool
	reason    string
}

func (s *Service) notifyDriverPTO(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	pto *worker.WorkerPTO,
	notice driverPTONotice,
) {
	if s.driverNotify == nil || pto == nil {
		return
	}
	s.driverNotify.Notify(ctx, &drivernotificationservice.DriverNotification{
		TenantInfo: tenantInfo,
		WorkerID:   pto.WorkerID,
		EventType:  notice.eventType,
		Priority:   notification.PriorityHigh,
		Context: documenttemplate.DriverNotificationContext{
			Approved: notice.approved,
			Reason:   notice.reason,
		},
		Link: "/dash/profile",
		RelatedEntities: map[string]any{
			"ptoId": pto.ID.String(),
		},
	})
}

func tenantOf(pto *worker.WorkerPTO) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pto.OrganizationID, BuID: pto.BusinessUnitID}
}

func (s *Service) inTx(ctx context.Context, fn func(context.Context) error) error {
	if s.db == nil {
		return fn(ctx)
	}
	return s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		return fn(txCtx)
	})
}

func (s *Service) prepareLedgerFields(
	ctx context.Context,
	entity *worker.WorkerPTO,
	excludeID pulid.ID,
) (*ptoledgerservice.AvailabilityResult, error) {
	if s.ledger == nil {
		entity.Days = worker.ComputePTODays(entity.StartDate, entity.EndDate, nil, true, nil)
		return nil, nil
	}

	tenantInfo := tenantOf(entity)
	resolved, err := s.ledger.ResolvePolicy(ctx, tenantInfo, entity.WorkerID, entity.StartDate)
	if err != nil {
		return nil, err
	}
	loc, err := s.ledger.OrgLocation(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	countWeekends := true
	if resolved != nil {
		countWeekends = resolved.Policy.CountWeekends
	}
	holidays, err := s.ledger.HolidayCalendar(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if blocked := holidays.BlackoutsBetween(entity.StartDate, entity.EndDate, loc); len(blocked) > 0 {
		return nil, errortypes.NewValidationError(
			"startDate",
			errortypes.ErrInvalid,
			"These dates fall on a blackout: {0}", blocked[0].Name,
		)
	}
	entity.Days = worker.ComputePTODays(entity.StartDate, entity.EndDate, loc, countWeekends, holidays)

	availability, err := s.ledger.CheckAvailability(ctx, &ptoledgerservice.AvailabilityRequest{
		TenantInfo:   tenantInfo,
		WorkerID:     entity.WorkerID,
		PTOType:      entity.Type,
		Days:         entity.Days,
		StartDate:    entity.StartDate,
		ExcludePTOID: excludeID,
	})
	if err != nil {
		return nil, err
	}
	if validationErr := availability.ValidationError(); validationErr != nil {
		return nil, validationErr
	}
	return availability, nil
}

func (s *Service) requireAvailability(
	ctx context.Context,
	pto *worker.WorkerPTO,
	excludeID pulid.ID,
) error {
	if s.ledger == nil {
		return nil
	}
	availability, err := s.ledger.CheckAvailability(ctx, &ptoledgerservice.AvailabilityRequest{
		TenantInfo:   tenantOf(pto),
		WorkerID:     pto.WorkerID,
		PTOType:      pto.Type,
		Days:         pto.Days,
		StartDate:    pto.StartDate,
		ExcludePTOID: excludeID,
	})
	if err != nil {
		return err
	}
	return availability.ValidationError()
}

func (s *Service) bookUsage(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	pto *worker.WorkerPTO,
	userID pulid.ID,
) error {
	if s.ledger == nil {
		return nil
	}
	entry, err := s.ledger.PostUsage(ctx, tenantInfo, pto, ptoledgerservice.UserActor(userID))
	if err != nil || entry == nil {
		return err
	}
	pto.BalanceAfterDays = decimal.NewNullDecimal(entry.BalanceAfterDays)
	return s.repo.SetBalanceAfter(ctx, &repositories.SetPTOBalanceAfterRequest{
		ID:               pto.ID,
		TenantInfo:       tenantInfo,
		BalanceAfterDays: entry.BalanceAfterDays,
	})
}

func (s *Service) releaseUsage(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	pto *worker.WorkerPTO,
	userID pulid.ID,
) error {
	if s.ledger == nil {
		return nil
	}
	_, err := s.ledger.PostReversal(ctx, tenantInfo, pto, ptoledgerservice.UserActor(userID))
	return err
}
