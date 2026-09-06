// Package workersafetyservice owns a worker's safety record: accidents,
// incidents, near misses, citations and roadside inspections with the points
// they carry, the progressive-discipline ladder built on top of them, the
// recognition that balances the picture, and the scorecard that rolls it all
// up for the office and the driver.
package workersafetyservice

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
	"github.com/emoss08/trenova/internal/core/services/workeremploymentservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	realtimeSafetyEvent  = "worker_safety_event"
	realtimeDiscipline   = "worker_disciplinary_action"
	realtimeRecognition  = "worker_recognition"
	workerResourceType   = "worker"
	safetyResourceType   = "worker_safety_event"
	eventRecognition     = "dash.recognition"
	eventDisciplinary    = "dash.disciplinary_issued"
	dashProfileLink      = "/dash/profile"
	disciplineReasonNote = "Issued from the discipline ladder"
)

// EmploymentRecorder is the slice of the employment service a suspension or
// termination needs so the timeline stays the only path that moves status.
type EmploymentRecorder interface {
	Record(
		ctx context.Context,
		req *workeremploymentservice.RecordRequest,
	) (*workeremploymentservice.RecordResult, error)
}

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.WorkerSafetyRepository
	WorkerRepo   repositories.WorkerRepository
	DocumentRepo repositories.DocumentRepository
	Employment   *workeremploymentservice.Service
	AuditService services.AuditService
	Realtime     services.RealtimeService           `optional:"true"`
	DriverNotify *drivernotificationservice.Service `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	repo         repositories.WorkerSafetyRepository
	workerRepo   repositories.WorkerRepository
	documentRepo repositories.DocumentRepository
	employment   EmploymentRecorder
	auditService services.AuditService
	realtime     services.RealtimeService
	driverNotify *drivernotificationservice.Service
}

func New(p Params) *Service {
	svc := &Service{
		l:            p.Logger.Named("service.worker-safety"),
		repo:         p.Repo,
		workerRepo:   p.WorkerRepo,
		documentRepo: p.DocumentRepo,
		auditService: p.AuditService,
		realtime:     p.Realtime,
		driverNotify: p.DriverNotify,
	}
	if p.Employment != nil {
		svc.employment = p.Employment
	}
	return svc
}

// Deps is the constructor shape tests use to swap in fakes.
type Deps struct {
	Logger       *zap.Logger
	Repo         repositories.WorkerSafetyRepository
	WorkerRepo   repositories.WorkerRepository
	DocumentRepo repositories.DocumentRepository
	Employment   EmploymentRecorder
	AuditService services.AuditService
}

func NewWithDeps(d Deps) *Service {
	logger := d.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		l:            logger.Named("service.worker-safety"),
		repo:         d.Repo,
		workerRepo:   d.WorkerRepo,
		documentRepo: d.DocumentRepo,
		employment:   d.Employment,
		auditService: d.AuditService,
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
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ActorUserID:    userID,
		ActorType:      services.PrincipalTypeUser,
		ActorID:        userID,
		Resource:       resource,
		Action:         string(operation),
		RecordID:       recordID,
	}); err != nil {
		s.l.Warn("failed to publish safety invalidation",
			zap.String("resource", resource),
			zap.Error(err))
	}
}

func (s *Service) notifyDriver(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	eventType string,
	priority notification.Priority,
	context documenttemplate.DriverNotificationContext,
	correlation string,
) {
	if s.driverNotify == nil {
		return
	}
	s.driverNotify.NotifyWithCorrelation(ctx, &drivernotificationservice.DriverNotification{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
		EventType:  eventType,
		Priority:   priority,
		Context:    context,
		Link:       dashProfileLink,
	}, correlation)
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

// Scorecard rolls a worker's whole record into one picture.
// RefreshRollup recomputes the scorecard and caches the rating and score on
// the worker's profile so the roster can filter and sort on safety without
// replaying every event for every row. Best-effort: a stale cache is repaired
// by the nightly sweep, and must never fail the write that triggered it.
func (s *Service) RefreshRollup(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.SafetyScorecard, error) {
	card, err := s.Scorecard(ctx, tenantInfo, workerID)
	if err != nil {
		return nil, err
	}
	if err = s.workerRepo.UpdateProfileSafetyRollup(
		ctx,
		&repositories.UpdateProfileSafetyRollupRequest{
			TenantInfo: tenantInfo,
			WorkerID:   workerID,
			Rating:     card.Rating,
			Score:      int16(card.Score),
		},
	); err != nil {
		return nil, err
	}
	return card, nil
}

func (s *Service) Scorecard(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.SafetyScorecard, error) {
	if _, err := s.loadWorker(ctx, tenantInfo, workerID); err != nil {
		return nil, err
	}
	events, err := s.repo.ListEvents(ctx, &repositories.ListWorkerSafetyEventsRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
	})
	if err != nil {
		return nil, err
	}
	actions, err := s.repo.ListActions(ctx, &repositories.ListWorkerDisciplinaryActionsRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
	})
	if err != nil {
		return nil, err
	}
	recognitions, err := s.repo.ListRecognitions(ctx, &repositories.ListWorkerRecognitionsRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
	})
	if err != nil {
		return nil, err
	}
	return worker.BuildSafetyScorecard(workerID, events, actions, recognitions, timeutils.NowUnix()), nil
}

func eventTenant(entity *worker.WorkerSafetyEvent) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}

func actionTenant(entity *worker.WorkerDisciplinaryAction) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}

func recognitionTenant(entity *worker.WorkerRecognition) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}

// refreshRollupQuietly caches the worker's safety standing on their profile so
// the roster can filter and sort on it. Best-effort on purpose: the cache is
// re-checked by the nightly sweep, and a cache write must never fail the
// change that triggered it.
func (s *Service) refreshRollupQuietly(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) {
	if _, err := s.RefreshRollup(ctx, tenantInfo, workerID); err != nil {
		s.l.Warn("failed to refresh worker safety rollup",
			zap.String("workerId", workerID.String()),
			zap.Error(err))
	}
}
