package reportservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/pkg/utils"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger         *zap.Logger
	Repo           repositories.ReportRepository
	TemporalClient client.Client
}

type Service struct {
	l              *zap.Logger
	repo           repositories.ReportRepository
	temporalClient client.Client
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:              p.Logger.Named("service.report"),
		repo:           p.Repo,
		temporalClient: p.TemporalClient,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListReportRequest,
) (*pagination.ListResult[*report.Report], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetReportByIDRequest,
) (*report.Report, error) {
	return s.repo.Get(ctx, req)
}

type GenerateReportRequest struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	UserID         pulid.ID
	UserEmail      string
	ResourceType   string
	Name           string
	Format         report.Format
	DeliveryMethod report.DeliveryMethod
	FilterState    pagination.QueryOptions
	EmailProfileID *pulid.ID
}

func (s *Service) GenerateReport(
	ctx context.Context,
	req *GenerateReportRequest,
) (*report.Report, error) {
	log := s.l.With(
		zap.String("operation", "GenerateReport"),
		zap.String("resourceType", req.ResourceType),
		zap.String("format", req.Format.String()),
		zap.String("userID", req.UserID.String()),
	)

	if !req.Format.IsValid() {
		return nil, fmt.Errorf("invalid format: %s", req.Format)
	}

	if !req.DeliveryMethod.IsValid() {
		return nil, fmt.Errorf("invalid delivery method: %s", req.DeliveryMethod)
	}

	rpt := &report.Report{
		ID:             pulid.MustNew("report_"),
		OrganizationID: req.OrganizationID,
		BusinessUnitID: req.BusinessUnitID,
		UserID:         req.UserID,
		ResourceType:   req.ResourceType,
		Name:           req.Name,
		Format:         req.Format,
		DeliveryMethod: req.DeliveryMethod,
		Status:         report.StatusPending,
		FilterState:    req.FilterState,
	}

	if err := s.repo.Create(ctx, rpt); err != nil {
		log.Error("failed to create report record", zap.Error(err))
		return nil, fmt.Errorf("failed to create report: %w", err)
	}

	payload := &temporaltype.GenerateReportPayload{
		ReportID:       rpt.ID,
		OrganizationID: req.OrganizationID,
		BusinessUnitID: req.BusinessUnitID,
		UserID:         req.UserID,
		UserEmail:      req.UserEmail,
		ResourceType:   req.ResourceType,
		Format:         req.Format,
		DeliveryMethod: req.DeliveryMethod,
		FilterState:    req.FilterState,
		EmailProfileID: req.EmailProfileID,
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("report-%s", rpt.ID),
		TaskQueue: temporaltype.ReportTaskQueue,
	}

	_, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"GenerateReportWorkflow",
		payload,
	)
	if err != nil {
		log.Error("failed to start workflow", zap.Error(err))

		rpt.Status = report.StatusFailed
		rpt.ErrorMessage = fmt.Sprintf("failed to start workflow: %v", err)
		rpt.CompletedAt = utils.Int64ToPointer(time.Now().Unix())
		_ = s.repo.Update(ctx, rpt)

		return nil, fmt.Errorf("failed to start report generation workflow: %w", err)
	}

	log.Info("report generation workflow started",
		zap.String("reportID", rpt.ID.String()),
		zap.String("workflowID", fmt.Sprintf("report-%s", rpt.ID)),
	)

	return rpt, nil
}

func (s *Service) Delete(
	ctx context.Context,
	id pulid.ID,
) error {
	return s.repo.Delete(ctx, id)
}
