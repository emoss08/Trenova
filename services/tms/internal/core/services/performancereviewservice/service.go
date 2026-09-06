// Package performancereviewservice runs the review cycle: templates that say
// what gets rated, reviews drafted by a manager, submitted to the worker,
// signed off in Dash and closed with the next date set from the cadence.
package performancereviewservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	realtimeReview        = "performance_review"
	realtimeTemplate      = "performance_review_template"
	eventReviewSubmitted  = "dash.review_submitted"
	dashProfileLink       = "/dash/profile"
	reviewSubmittedPrefix = "review-submitted-"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.PerformanceReviewRepository
	WorkerRepo   repositories.WorkerRepository
	UserRepo     repositories.UserRepository
	AuditService services.AuditService
	Realtime     services.RealtimeService           `optional:"true"`
	DriverNotify *drivernotificationservice.Service `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	repo         repositories.PerformanceReviewRepository
	workerRepo   repositories.WorkerRepository
	userRepo     repositories.UserRepository
	auditService services.AuditService
	realtime     services.RealtimeService
	driverNotify *drivernotificationservice.Service
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.performance-review"),
		repo:         p.Repo,
		workerRepo:   p.WorkerRepo,
		userRepo:     p.UserRepo,
		auditService: p.AuditService,
		realtime:     p.Realtime,
		driverNotify: p.DriverNotify,
	}
}

type auditParams struct {
	resource   permission.Resource
	resourceID string
	operation  permission.Operation
	userID     pulid.ID
	tenant     pagination.TenantInfo
	current    any
	previous   any
	comment    string
	log        *zap.Logger
}

func (s *Service) audit(p *auditParams) {
	if s.auditService == nil || p.userID.IsNil() {
		return
	}
	params := &services.LogActionParams{
		Resource:       p.resource,
		ResourceID:     p.resourceID,
		Operation:      p.operation,
		UserID:         p.userID,
		CurrentState:   jsonutils.MustToJSON(p.current),
		OrganizationID: p.tenant.OrgID,
		BusinessUnitID: p.tenant.BuID,
	}
	opts := []services.LogOption{auditservice.WithComment(p.comment)}
	if p.previous != nil {
		params.PreviousState = jsonutils.MustToJSON(p.previous)
		opts = append(opts, auditservice.WithDiff(p.previous, p.current))
	}
	if err := s.auditService.LogAction(params, opts...); err != nil {
		p.log.Error("failed to log audit action", zap.Error(err))
	}
}

func (s *Service) publish(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resource string,
	operation permission.Operation,
	recordID pulid.ID,
	userID pulid.ID,
) {
	if s.realtime == nil {
		return
	}
	actorType := services.PrincipalTypeUser
	if userID.IsNil() {
		actorType = services.PrincipalTypeSystem
	}
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ActorUserID:    userID,
		ActorType:      actorType,
		ActorID:        userID,
		Resource:       resource,
		Action:         string(operation),
		RecordID:       recordID,
	}); err != nil {
		s.l.Warn("failed to publish review invalidation",
			zap.String("resource", resource),
			zap.Error(err))
	}
}

func (s *Service) notifyDriver(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	review *worker.PerformanceReview,
	reviewerName string,
) {
	if s.driverNotify == nil {
		return
	}
	s.driverNotify.NotifyWithCorrelation(ctx, &drivernotificationservice.DriverNotification{
		TenantInfo: tenantInfo,
		WorkerID:   review.WorkerID,
		EventType:  eventReviewSubmitted,
		Priority:   notification.PriorityMedium,
		Context: documenttemplate.DriverNotificationContext{
			ReviewTitle:  review.Title,
			ReviewerName: reviewerName,
		},
		Link: dashProfileLink,
	}, reviewSubmittedPrefix+review.ID.String())
}

func (s *Service) loadWorker(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.Worker, error) {
	return s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:             workerID,
		TenantInfo:     tenantInfo,
		IncludeProfile: true,
	})
}

func templateTenant(entity *worker.PerformanceReviewTemplate) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}

func reviewTenant(entity *worker.PerformanceReview) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}
