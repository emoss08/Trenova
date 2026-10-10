// Package workerchecklistservice runs onboarding and offboarding checklists:
// the per-org templates, the instances spawned for a worker (by an employment
// event or by hand), the auto-satisfaction pass that ticks Credential, Document
// and PortalAccess items when the evidence exists, and the DQF readiness flag
// that flips when an onboarding checklist closes.
package workerchecklistservice

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	realtimeResource = "worker_checklist"
	realtimeTemplate = "worker_checklist_template"
	realtimeWorkers  = "workers"
)

type Params struct {
	fx.In

	Logger         *zap.Logger
	Repo           workerports.WorkerChecklistRepository
	WorkerRepo     workerports.WorkerRepository
	CredentialRepo workerports.WorkerCredentialRepository
	DocumentRepo   documentports.DocumentRepository
	AuditService   auditports.AuditService
	Realtime       agentports.RealtimeService `optional:"true"`
}

type Service struct {
	l              *zap.Logger
	repo           workerports.WorkerChecklistRepository
	workerRepo     workerports.WorkerRepository
	credentialRepo workerports.WorkerCredentialRepository
	documentRepo   documentports.DocumentRepository
	auditService   auditports.AuditService
	realtime       agentports.RealtimeService
	refreshLocks   sync.Map
}

func New(p Params) *Service {
	return &Service{
		l:              p.Logger.Named("service.worker-checklist"),
		repo:           p.Repo,
		workerRepo:     p.WorkerRepo,
		credentialRepo: p.CredentialRepo,
		documentRepo:   p.DocumentRepo,
		auditService:   p.AuditService,
		realtime:       p.Realtime,
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
	params := &auditports.LogActionParams{
		Resource:       p.resource,
		ResourceID:     p.resourceID,
		Operation:      p.operation,
		UserID:         p.userID,
		CurrentState:   jsonutils.MustToJSON(p.current),
		OrganizationID: p.tenant.OrgID,
		BusinessUnitID: p.tenant.BuID,
	}
	opts := []auditports.LogOption{auditservice.WithComment(p.comment)}
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
	actorType := permissionports.PrincipalTypeUser
	if userID.IsNil() {
		actorType = permissionports.PrincipalTypeSystem
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
		s.l.Warn(
			"failed to publish checklist invalidation",
			zap.String("resource", resource),
			zap.Error(err),
		)
	}
}

func templateTenant(entity *worker.WorkerChecklistTemplate) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}

func checklistTenant(entity *worker.WorkerChecklist) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}
