// Package workercredentialservice owns the credential registry: the per-org
// catalog of credential types, the credentials each worker holds, and the
// evaluation that turns expiries into a compliance picture.
//
// Six system types mirror the historical columns on worker_profiles
// (licence, hazmat, medical card, TWIC, physical, MVR). Those columns are still
// what the dispatch eligibility rules read, so the registry is authoritative
// for the office UI and copies every change back to the column, while a
// profile edit through the worker form is copied forward into the registry by
// SyncFromProfile. Neither side ever bumps the other's optimistic version.
package workercredentialservice

import (
	"context"
	"sync"

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
	realtimeResource     = "worker_credential"
	realtimeResourceType = "worker_credential_type"
	realtimeWorkers      = "workers"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.WorkerCredentialRepository
	WorkerRepo   repositories.WorkerRepository
	DocumentRepo repositories.DocumentRepository
	AuditService services.AuditService
	Realtime     services.RealtimeService `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	repo          repositories.WorkerCredentialRepository
	workerRepo    repositories.WorkerRepository
	documentRepo  repositories.DocumentRepository
	auditService  services.AuditService
	realtime      services.RealtimeService
	seededTenants sync.Map
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.worker-credential"),
		repo:         p.Repo,
		workerRepo:   p.WorkerRepo,
		documentRepo: p.DocumentRepo,
		auditService: p.AuditService,
		realtime:     p.Realtime,
	}
}

func (s *Service) auditType(
	current, previous *worker.WorkerCredentialType,
	operation permission.Operation,
	userID pulid.ID,
	comment string,
	log *zap.Logger,
) {
	s.audit(&auditParams{
		resource:   permission.ResourceWorkerCredentialType,
		resourceID: current.GetResourceID(),
		operation:  operation,
		userID:     userID,
		tenant:     typeTenant(current),
		current:    current,
		previous:   previous,
		comment:    comment,
		log:        log,
	})
}

func (s *Service) auditCredential(
	current, previous *worker.WorkerCredential,
	operation permission.Operation,
	userID pulid.ID,
	comment string,
	log *zap.Logger,
) {
	s.audit(&auditParams{
		resource:   permission.ResourceWorkerCredential,
		resourceID: current.GetResourceID(),
		operation:  operation,
		userID:     userID,
		tenant:     credentialTenant(current),
		current:    current,
		previous:   previous,
		comment:    comment,
		log:        log,
	})
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
	if p.userID.IsNil() {
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
		s.l.Warn("failed to publish credential invalidation",
			zap.String("resource", resource),
			zap.Error(err))
	}
}

func credentialTenant(entity *worker.WorkerCredential) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}
