// Package workertrainingservice owns the training catalog and the records
// that put workers through it: the required-course matrix per driver type,
// assignments with due dates, completions with scores and expiries, waivers,
// and the driver-portal path where a course is opened and acknowledged.
package workertrainingservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	realtimeResource       = "worker_training"
	realtimeResourceCourse = "training_course"
	workerResourceType     = "worker"
	trainingResourceType   = "worker_training"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.WorkerTrainingRepository
	WorkerRepo   repositories.WorkerRepository
	DocumentRepo repositories.DocumentRepository
	AuditService services.AuditService
	Realtime     services.RealtimeService `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	repo         repositories.WorkerTrainingRepository
	workerRepo   repositories.WorkerRepository
	documentRepo repositories.DocumentRepository
	auditService services.AuditService
	realtime     services.RealtimeService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.worker-training"),
		repo:         p.Repo,
		workerRepo:   p.WorkerRepo,
		documentRepo: p.DocumentRepo,
		auditService: p.AuditService,
		realtime:     p.Realtime,
	}
}

var _ services.TrainingAssigner = (*Service)(nil)

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

func (s *Service) auditCourse(
	current, previous *worker.TrainingCourse,
	operation permission.Operation,
	userID pulid.ID,
	comment string,
	log *zap.Logger,
) {
	s.audit(&auditParams{
		resource:   permission.ResourceTrainingCourse,
		resourceID: current.GetResourceID(),
		operation:  operation,
		userID:     userID,
		tenant:     courseTenant(current),
		current:    current,
		previous:   previous,
		comment:    comment,
		log:        log,
	})
}

func (s *Service) auditRecord(
	current, previous *worker.WorkerTrainingRecord,
	operation permission.Operation,
	userID pulid.ID,
	comment string,
	log *zap.Logger,
) {
	s.audit(&auditParams{
		resource:   permission.ResourceWorkerTraining,
		resourceID: current.GetResourceID(),
		operation:  operation,
		userID:     userID,
		tenant:     recordTenant(current),
		current:    current,
		previous:   previous,
		comment:    comment,
		log:        log,
	})
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
		s.l.Warn("failed to publish training invalidation",
			zap.String("resource", resource),
			zap.Error(err))
	}
}

func courseTenant(entity *worker.TrainingCourse) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}

func recordTenant(entity *worker.WorkerTrainingRecord) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
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

// refreshRollupQuietly caches the worker's training standing on their profile
// so the roster can filter and sort on it. Best-effort on purpose: the cache
// is re-checked by the nightly sweep, and a cache write must never fail the
// change that triggered it.
func (s *Service) refreshRollupQuietly(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) {
	if _, err := s.RefreshRollup(ctx, tenantInfo, workerID); err != nil {
		s.l.Warn("failed to refresh worker training rollup",
			zap.String("workerId", workerID.String()),
			zap.Error(err))
	}
}
