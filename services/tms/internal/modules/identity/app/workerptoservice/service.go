//nolint:gocritic // existing legacy workflow/API shape is intentionally kept stable
package workerptoservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/notificationtemplate"
	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/drivernotificationservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	DB            kernelports.Transactor
	Repo          workerptoports.WorkerPTORepository
	Ledger        *ptoledgerservice.Service
	Organizations tenantports.OrganizationRepository
	AuditService  auditports.AuditService
	DriverNotify  *drivernotificationservice.Service
	Realtime      realtimeports.RealtimeService
}

type Service struct {
	l             *zap.Logger
	db            kernelports.Transactor
	repo          workerptoports.WorkerPTORepository
	ledger        *ptoledgerservice.Service
	organizations tenantports.OrganizationReader
	auditService  auditports.AuditService
	driverNotify  *drivernotificationservice.Service
	realtime      realtimeports.RealtimeService
}

const maxPTOChartRangeSeconds int64 = 366 * 24 * 60 * 60

var _ workerptoports.WorkerPTOService = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.workerpto"),
		db:            p.DB,
		repo:          p.Repo,
		ledger:        p.Ledger,
		organizations: p.Organizations,
		auditService:  p.AuditService,
		driverNotify:  p.DriverNotify,
		realtime:      p.Realtime,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *workerptoports.ListPTORequest,
) (*pagination.CursorListResult[*worker.WorkerPTO], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *workerptoports.GetPTOByIDRequest,
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

	autoApprove, err := s.prepareCreate(ctx, entity, userID)
	if err != nil {
		return nil, err
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
			eventType: eventPTOReviewed,
			approved:  true,
		})
	}

	if err = s.auditService.LogAction(ctx, &auditports.LogActionParams{
		Resource:       permission.ResourceWorkerPTO,
		ResourceID:     createdEntity.GetResourceID(),
		Operation:      permission.OpCreate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.GetOrganizationID(),
		BusinessUnitID: createdEntity.GetBusinessUnitID(),
	}, auditservice.WithComment("PTO created"), auditservice.WithSubjectOf(createdEntity)); err != nil {
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

	current, err := s.prepareUpdate(ctx, entity)
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update PTO", zap.Error(err))
		return nil, err
	}
	s.publishPTOInvalidation(ctx, updated, permission.OpUpdate, userID)

	if err = s.auditService.LogAction(ctx, &auditports.LogActionParams{
		Resource:       permission.ResourceWorkerPTO,
		ResourceID:     updated.GetResourceID(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		PreviousState:  jsonutils.MustToJSON(current),
		CurrentState:   jsonutils.MustToJSON(updated),
		OrganizationID: updated.GetOrganizationID(),
		BusinessUnitID: updated.GetBusinessUnitID(),
	}, auditservice.WithComment("PTO updated"),
		auditservice.WithDiff(current, updated),
		auditservice.WithSubjectOf(updated)); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updated, nil
}

func (s *Service) Approve(
	ctx context.Context,
	req *workerptoports.UpdatePTOStatusRequest,
) (*worker.WorkerPTO, error) {
	return s.transition(ctx, transitionParams{
		req:          req,
		target:       worker.PTOStatusApproved,
		notifyWorker: true,
	})
}

func (s *Service) Reject(
	ctx context.Context,
	req *workerptoports.UpdatePTOStatusRequest,
) (*worker.WorkerPTO, error) {
	if err := requireRejectionReason(req); err != nil {
		return nil, err
	}

	return s.transition(ctx, transitionParams{
		req:          req,
		target:       worker.PTOStatusRejected,
		notifyWorker: true,
	})
}

func (s *Service) Cancel(
	ctx context.Context,
	req *workerptoports.UpdatePTOStatusRequest,
) (*worker.WorkerPTO, error) {
	return s.transition(ctx, transitionParams{
		req:          req,
		target:       worker.PTOStatusCancelled,
		notifyWorker: true,
	})
}

func (s *Service) CancelRequested(
	ctx context.Context,
	req *workerptoports.UpdatePTOStatusRequest,
) (*worker.WorkerPTO, error) {
	return s.transition(ctx, transitionParams{
		req:          req,
		target:       worker.PTOStatusCancelled,
		notifyWorker: false,
	})
}

func (s *Service) BulkAction(
	ctx context.Context,
	req *workerptoports.PTOBulkActionRequest,
) (*workerptoports.PTOBulkActionPayload, error) {
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
	if req.Action == workerptoports.PTOBulkActionReject && strings.TrimSpace(req.Reason) == "" {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"A reason is required to reject PTO requests",
		)
	}

	existing, err := s.repo.GetByIDs(ctx, &workerptoports.GetPTOsByIDsRequest{
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

	payload := &workerptoports.PTOBulkActionPayload{
		Results: make([]*workerptoports.PTOBulkActionResult, 0, len(req.PTOIDs)),
	}
	for _, ptoID := range req.PTOIDs {
		result := &workerptoports.PTOBulkActionResult{PTOID: ptoID}

		current, found := byID[ptoID]
		if !found {
			result.Error = "PTO request not found within your organization"
			payload.FailureCount++
			payload.Results = append(payload.Results, result)
			continue
		}

		_, actionErr := s.transition(ctx, transitionParams{
			req: &workerptoports.UpdatePTOStatusRequest{
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
	req          *workerptoports.UpdatePTOStatusRequest
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

	current, err := s.planTransition(ctx, params)
	if err != nil {
		return nil, err
	}

	var updated *worker.WorkerPTO
	err = s.inTx(ctx, func(txCtx context.Context) error {
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
	s.logTransitionAudit(ctx, current, updated, operation, req, log)
	s.publishPTOInvalidation(ctx, updated, operation, req.UserID)

	if params.notifyWorker {
		if notice, ok := transitionNotice(params.target, req.Reason); ok {
			s.notifyDriverPTO(ctx, req.TenantInfo, updated, notice)
		}
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
	ctx context.Context,
	previous, updated *worker.WorkerPTO,
	operation permission.Operation,
	req *workerptoports.UpdatePTOStatusRequest,
	log *zap.Logger,
) {
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

	if err := s.auditService.LogAction(ctx, &auditports.LogActionParams{
		Resource:       permission.ResourceWorkerPTO,
		ResourceID:     updated.GetResourceID(),
		Operation:      operation,
		UserID:         req.UserID,
		PreviousState:  jsonutils.MustToJSON(previous),
		CurrentState:   jsonutils.MustToJSON(updated),
		OrganizationID: updated.GetOrganizationID(),
		BusinessUnitID: updated.GetBusinessUnitID(),
	}, auditservice.WithComment(comment),
		auditservice.WithDiff(previous, updated),
		auditservice.WithSubjectOf(updated)); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}
}

func (s *Service) GetChartData(
	ctx context.Context,
	req *workerptoports.PTOChartRequest,
) ([]*workerptoports.PTOChartDataPoint, error) {
	if err := s.validateChartRequest(req); err != nil {
		return nil, err
	}

	return s.repo.GetChartData(ctx, req)
}

func (s *Service) validateChartRequest(req *workerptoports.PTOChartRequest) error {
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

	return multiErr.Err()
}

func (s *Service) ListUpcoming(
	ctx context.Context,
	req *workerptoports.ListUpcomingPTORequest,
) (*pagination.CursorListResult[*worker.WorkerPTO], error) {
	return s.repo.ListUpcoming(ctx, req)
}

func (s *Service) publishPTOInvalidation(
	ctx context.Context,
	pto *worker.WorkerPTO,
	action permission.Operation,
	userID pulid.ID,
) {
	if pto == nil {
		return
	}
	err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: pto.OrganizationID,
		BusinessUnitID: pto.BusinessUnitID,
		ActorUserID:    userID,
		ActorType:      permissionports.PrincipalTypeUser,
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
	if pto == nil {
		return
	}
	s.driverNotify.Notify(ctx, ptoDriverNotification(tenantInfo, pto, notice))
}

func ptoDriverNotification(
	tenantInfo pagination.TenantInfo,
	pto *worker.WorkerPTO,
	notice driverPTONotice,
) *drivernotificationservice.DriverNotification {
	return &drivernotificationservice.DriverNotification{
		TenantInfo: tenantInfo,
		WorkerID:   pto.WorkerID,
		EventType:  notice.eventType,
		Priority:   notification.PriorityHigh,
		Context: notificationtemplate.DriverNotificationContext{
			Approved: notice.approved,
			Reason:   notice.reason,
		},
		Link: "/dash/profile",
		RelatedEntities: map[string]any{
			"ptoId": pto.ID.String(),
		},
	}
}

func tenantOf(pto *worker.WorkerPTO) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pto.OrganizationID, BuID: pto.BusinessUnitID}
}

func (s *Service) inTx(ctx context.Context, fn func(context.Context) error) error {
	return s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		return fn(txCtx)
	})
}

func (s *Service) prepareLedgerFields(
	ctx context.Context,
	entity *worker.WorkerPTO,
	excludeID pulid.ID,
) (*ptoledgerservice.AvailabilityResult, error) {
	tenantInfo := tenantOf(entity)
	resolved, err := s.ledger.ResolvePolicy(ctx, tenantInfo, entity.WorkerID, entity.StartDate)
	if err != nil {
		return nil, err
	}
	loc, err := tenantports.OrganizationLocation(ctx, s.organizations, tenantInfo)
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
	if blocked := holidays.BlackoutsBetween(entity.StartDate, entity.EndDate, loc); len(
		blocked,
	) > 0 {
		return nil, errortypes.NewValidationError(
			"startDate",
			errortypes.ErrInvalid,
			"These dates fall on a blackout: {0}", blocked[0].Name,
		)
	}
	entity.Days = pto.ComputePTODays(
		entity.StartDate,
		entity.EndDate,
		loc,
		countWeekends,
		holidays,
	)

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
	availability, err := s.checkAvailability(ctx, pto, excludeID)
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
	entry, err := s.ledger.PostUsage(ctx, tenantInfo, pto, ptoledgerservice.UserActor(userID))
	if err != nil || entry == nil {
		return err
	}
	pto.BalanceAfterDays = decimal.NewNullDecimal(entry.BalanceAfterDays)
	return s.repo.SetBalanceAfter(ctx, &workerptoports.SetPTOBalanceAfterRequest{
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
	_, err := s.ledger.PostReversal(ctx, tenantInfo, pto, ptoledgerservice.UserActor(userID))
	return err
}
