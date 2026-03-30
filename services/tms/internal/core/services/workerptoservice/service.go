package workerptoservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/smsjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger          *zap.Logger
	Repo            repositories.WorkerPTORepository
	UserRepo        repositories.UserRepository
	WorkerRepo      repositories.WorkerRepository
	WorkflowStarter services.WorkflowStarter
	AuditService    services.AuditService
}

type Service struct {
	l               *zap.Logger
	repo            repositories.WorkerPTORepository
	userRepo        repositories.UserRepository
	workerRepo      repositories.WorkerRepository
	workflowStarter services.WorkflowStarter
	auditService    services.AuditService
}

const maxPTOChartRangeSeconds int64 = 366 * 24 * 60 * 60

func New(p Params) *Service {
	return &Service{
		l:               p.Logger.Named("service.workerpto"),
		repo:            p.Repo,
		userRepo:        p.UserRepo,
		workerRepo:      p.WorkerRepo,
		workflowStarter: p.WorkflowStarter,
		auditService:    p.AuditService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListPTORequest,
) (*pagination.ListResult[*worker.WorkerPTO], error) {
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

	// TODO: Validate PTO data

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create PTO", zap.Error(err))
		return nil, err
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

func (s *Service) Approve(
	ctx context.Context,
	req *repositories.UpdatePTOStatusRequest,
) (*worker.WorkerPTO, error) {
	log := s.l.With(
		zap.String("operation", "Approve"),
		zap.Any("request", req),
	)

	updatedEntity, err := s.repo.UpdateStatus(ctx, req)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  req.TenantInfo.OrgID,
			BuID:   req.TenantInfo.BuID,
			UserID: req.UserID,
		},
	})
	if err != nil {
		log.Error("failed to get user", zap.Error(err))
		return nil, err
	}

	wrk, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:             updatedEntity.WorkerID,
		TenantInfo:     req.TenantInfo,
		IncludeProfile: true,
	})
	if err != nil {
		log.Error("failed to get worker", zap.Error(err))
		return nil, err
	}

	payload := &smsjobs.SendSMSPayload{
		PhoneNumber: wrk.PhoneNumber,
		Message: fmt.Sprintf(
			"%s has approved your PTO request for dates %s to %s.",
			user.Name,
			timeutils.UnixToHumanReadable(updatedEntity.StartDate),
			timeutils.UnixToHumanReadable(updatedEntity.EndDate),
		),
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}

	if !s.workflowStarter.Enabled() {
		return nil, fmt.Errorf("failed to send SMS: %w", services.ErrWorkflowStarterDisabled)
	}

	_, err = s.workflowStarter.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID: fmt.Sprintf(
			"pto-approved-%s-%d",
			updatedEntity.ID.String(),
			timeutils.NowUnix(),
		),
		TaskQueue: temporaltype.SMSTaskQueue,
		StaticSummary: fmt.Sprintf(
			"Sending SMS to worker %s for approved PTO",
			updatedEntity.WorkerID.String(),
		),
	}, "SendSMSWorkflow", payload)
	if err != nil {
		log.Error("failed to send SMS", zap.Error(err))
		return nil, fmt.Errorf("failed to send SMS: %w", err)
	}

	return updatedEntity, nil
}

func (s *Service) Reject(
	ctx context.Context,
	req *repositories.UpdatePTOStatusRequest,
) (*worker.WorkerPTO, error) {
	log := s.l.With(
		zap.String("operation", "Reject"),
		zap.Any("request", req),
	)

	updatedEntity, err := s.repo.UpdateStatus(ctx, req)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  req.TenantInfo.OrgID,
			BuID:   req.TenantInfo.BuID,
			UserID: req.UserID,
		},
	})
	if err != nil {
		log.Error("failed to get user", zap.Error(err))
		return nil, err
	}

	wrk, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:             updatedEntity.WorkerID,
		TenantInfo:     req.TenantInfo,
		IncludeProfile: true,
	})
	if err != nil {
		log.Error("failed to get worker", zap.Error(err))
		return nil, err
	}

	payload := &smsjobs.SendSMSPayload{
		PhoneNumber: wrk.PhoneNumber,
		Message: fmt.Sprintf(
			"%s has rejected your PTO request for dates %s to %s. Reason: %s",
			user.Name,
			timeutils.UnixToHumanReadable(updatedEntity.StartDate),
			timeutils.UnixToHumanReadable(updatedEntity.EndDate),
			updatedEntity.Reason,
		),
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}

	if !s.workflowStarter.Enabled() {
		return nil, fmt.Errorf("failed to send SMS: %w", services.ErrWorkflowStarterDisabled)
	}

	_, err = s.workflowStarter.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID: fmt.Sprintf(
			"pto-rejected-%s-%d",
			updatedEntity.ID.String(),
			timeutils.NowUnix(),
		),
		TaskQueue: temporaltype.SMSTaskQueue,
		StaticSummary: fmt.Sprintf(
			"Sending SMS to worker %s for rejected PTO",
			updatedEntity.WorkerID.String(),
		),
	},
		"SendSMSWorkflow",
		payload,
	)
	if err != nil {
		log.Error("failed to send SMS", zap.Error(err))
		return nil, fmt.Errorf("failed to send SMS: %w", err)
	}

	return s.repo.UpdateStatus(ctx, req)
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
) (*pagination.ListResult[*worker.WorkerPTO], error) {
	return s.repo.ListUpcoming(ctx, req)
}
