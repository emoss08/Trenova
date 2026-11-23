package workeremploymentservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/fleetcodeports"
	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driverpayservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/driverportalinvitationservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/ptopolicyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerchecklistports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workeremploymentports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workertrainingports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	realtimeResource        = "worker_employment_event"
	realtimeWorkers         = "workers"
	terminationCancelReason = "Employment ended"
	upcomingPTOPageSize     = 100
	teamChangeAction        = "worker.team_changed"
)

type WorkerUpdater interface {
	UpdateFromEmployment(
		ctx context.Context,
		entity *worker.Worker,
		actor *permissionports.RequestActor,
	) (*worker.Worker, error)
}

type PTOPolicyManager interface {
	ActiveAssignment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) (*pto.WorkerPTOPolicyAssignment, error)
	EndAssignment(
		ctx context.Context,
		req *ptopolicyservice.EndAssignmentRequest,
	) (*pto.WorkerPTOPolicyAssignment, error)
	DefaultPolicy(ctx context.Context, tenantInfo pagination.TenantInfo) (*pto.PTOPolicy, error)
	AssignAtHire(ctx context.Context, req *ptopolicyservice.HireAssignment) error
}

type PTOSettler interface {
	PreviewTerminationSettlement(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		effectiveAt int64,
		released []ptoledgerservice.ReleasedUsage,
	) (*ptoledgerservice.TerminationPlan, error)
	SettleOnTermination(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		effectiveAt int64,
		plan *ptoledgerservice.TerminationPlan,
		actor ptoledgerservice.Actor,
	) (*ptoledgerservice.TerminationSettlement, error)
}

type PayAssignmentManager interface {
	EndAssignment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		assignmentID pulid.ID,
		endDate int64,
		actor *permissionports.RequestActor,
	) (*payassignment.WorkerPayAssignment, error)
}

type PortalAccess interface {
	GetPortalStatus(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) (*driverportalinvitationservice.PortalStatus, error)
	RevokeAccess(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		actor *permissionports.RequestActor,
	) error
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	DB            kernelports.Transactor
	Repo          workeremploymentports.WorkerEmploymentEventRepository
	WorkerRepo    workerports.WorkerRepository
	FleetCodeRepo fleetcodeports.FleetCodeRepository
	DocumentRepo  documentports.DocumentRepository
	PayAssignRepo driverpayports.WorkerPayAssignmentRepository
	Users         tenantports.UserRepository
	Workers       *workerservice.Service
	PTOPolicies   *ptopolicyservice.Service
	PTO           workerptoports.WorkerPTOService
	PTOLedger     *ptoledgerservice.Service
	DriverPay     *driverpayservice.Service
	AuditService  auditports.AuditService
	Security      permissionports.SecurityAuditor
	Realtime      realtimeports.RealtimeService
	Checklists    workerchecklistports.ChecklistSpawner
	Training      workertrainingports.TrainingAssigner
}

type Service struct {
	l             *zap.Logger
	db            kernelports.Transactor
	repo          workeremploymentports.WorkerEmploymentEventRepository
	workerRepo    workerports.WorkerRepository
	fleetCodeRepo fleetcodeports.FleetCodeRepository
	documentRepo  documentports.DocumentRepository
	payAssignRepo driverpayports.WorkerPayAssignmentRepository
	users         tenantports.UserRepository
	workers       WorkerUpdater
	ptoPolicies   PTOPolicyManager
	pto           workerptoports.WorkerPTOService
	ptoLedger     PTOSettler
	driverPay     PayAssignmentManager
	auditService  auditports.AuditService
	security      permissionports.SecurityAuditor
	realtime      realtimeports.RealtimeService
	checklists    workerchecklistports.ChecklistSpawner
	training      workertrainingports.TrainingAssigner
	portal        PortalAccess
}

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.worker-employment"),
		db:            p.DB,
		repo:          p.Repo,
		workerRepo:    p.WorkerRepo,
		fleetCodeRepo: p.FleetCodeRepo,
		documentRepo:  p.DocumentRepo,
		payAssignRepo: p.PayAssignRepo,
		users:         p.Users,
		workers:       p.Workers,
		ptoPolicies:   p.PTOPolicies,
		pto:           p.PTO,
		ptoLedger:     p.PTOLedger,
		driverPay:     p.DriverPay,
		auditService:  p.AuditService,
		security:      p.Security,
		realtime:      p.Realtime,
		checklists:    p.Checklists,
		training:      p.Training,
	}
}

func (s *Service) SetPortalRevoker(portal PortalAccess) {
	s.portal = portal
}

func (s *Service) List(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	kinds []employment.EmploymentEventKind,
) ([]*employment.WorkerEmploymentEvent, error) {
	return s.repo.List(ctx, &workeremploymentports.ListWorkerEmploymentEventsRequest{
		TenantInfo:      tenantInfo,
		WorkerID:        workerID,
		Kinds:           kinds,
		IncludeDocument: true,
		IncludeActors:   true,
	})
}

func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*employment.WorkerEmploymentEvent, error) {
	return s.repo.GetByID(ctx, &workeremploymentports.GetWorkerEmploymentEventByIDRequest{
		ID:              id,
		TenantInfo:      tenantInfo,
		IncludeDocument: true,
		IncludeActors:   true,
	})
}

type RecordRequest struct {
	TenantInfo  pagination.TenantInfo
	WorkerID    pulid.ID
	Kind        employment.EmploymentEventKind
	EffectiveAt int64
	Reason      string
	Notes       string
	DocumentID  pulid.ID
	FleetCodeID *pulid.ID
	ManagerID   *pulid.ID
	DriverType  *worker.DriverType
	WorkerType  *worker.WorkerType
	Rate        string
	RateUnit    string
	LeaveType   *worker.LeaveType
	UserID      pulid.ID
}

type RecordResult struct {
	Event   *employment.WorkerEmploymentEvent
	Effects []*employment.Effect
}

func (s *Service) loadState(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	includeTeam bool,
) (*worker.Worker, []*employment.WorkerEmploymentEvent, error) {
	wrk, err := s.workerRepo.GetByID(ctx, workerports.GetWorkerByIDRequest{
		ID:               workerID,
		TenantInfo:       tenantInfo,
		IncludeProfile:   true,
		IncludeFleetCode: includeTeam,
		IncludeManager:   includeTeam,
	})
	if err != nil {
		return nil, nil, err
	}
	history, err := s.repo.List(ctx, &workeremploymentports.ListWorkerEmploymentEventsRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
		Ascending:  true,
	})
	if err != nil {
		return nil, nil, err
	}
	return wrk, history, nil
}

func (s *Service) Options(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*Options, error) {
	wrk, history, err := s.loadState(ctx, tenantInfo, workerID, false)
	if err != nil {
		return nil, err
	}
	return &Options{
		RecordableKinds:     employment.RecordableKinds(employment.EmploymentStateOf(wrk, history)),
		ReasonRequiredKinds: employment.ReasonRequiredKinds(),
	}, nil
}

type AmendRequest struct {
	ID            pulid.ID
	TenantInfo    pagination.TenantInfo
	EffectiveAt   *int64
	Reason        *string
	Notes         *string
	DocumentID    *pulid.ID
	AmendmentNote string
	Version       int64
	UserID        pulid.ID
}

func (s *Service) Amend(
	ctx context.Context,
	req *AmendRequest,
) (*employment.WorkerEmploymentEvent, error) {
	log := s.l.With(zap.String("operation", "Amend"), zap.String("id", req.ID.String()))

	original, err := s.repo.GetByID(ctx, &workeremploymentports.GetWorkerEmploymentEventByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if req.Version > 0 && original.Version != req.Version {
		return nil, errortypes.NewVersionMismatchError(
			"Event was changed by someone else. Reload and try again",
		)
	}
	if strings.TrimSpace(req.AmendmentNote) == "" {
		return nil, errortypes.NewValidationError(
			"amendmentNote",
			errortypes.ErrRequired,
			"Say why the event is being amended",
		)
	}

	updated := *original
	if req.EffectiveAt != nil {
		updated.EffectiveAt = *req.EffectiveAt
	}
	if req.Reason != nil {
		updated.Reason = strings.TrimSpace(*req.Reason)
	}
	if req.Notes != nil {
		updated.Notes = strings.TrimSpace(*req.Notes)
	}
	if req.DocumentID != nil {
		updated.DocumentID = *req.DocumentID
		if !updated.DocumentID.IsNil() {
			if err = s.requireWorkerDocument(
				ctx,
				req.TenantInfo,
				original.WorkerID,
				updated.DocumentID,
			); err != nil {
				return nil, err
			}
		}
	}
	now := timeutils.NowUnix()
	updated.AmendedAt = &now
	updated.AmendedByID = req.UserID
	updated.AmendmentNote = strings.TrimSpace(req.AmendmentNote)

	multiErr := errortypes.NewMultiError()
	updated.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	saved, err := s.repo.Update(ctx, &updated)
	if err != nil {
		log.Error("failed to amend employment event", zap.Error(err))
		return nil, err
	}

	if err = s.audit(ctx, saved, original, permission.OpUpdate, req.UserID, "Employment event amended"); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}
	s.publish(ctx, req.TenantInfo, saved, permission.OpUpdate, req.UserID)

	return saved, nil
}

func (s *Service) requireWorkerDocument(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID, documentID pulid.ID,
) error {
	doc, err := s.documentRepo.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if doc.ResourceType != "worker" || doc.ResourceID != workerID.String() {
		return errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document does not belong to this worker",
		)
	}
	return nil
}

func (s *Service) actor(
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}
}

func (s *Service) audit(
	ctx context.Context,
	current, previous *employment.WorkerEmploymentEvent,
	operation permission.Operation,
	userID pulid.ID,
	comment string,
) error {
	if userID.IsNil() {
		return nil
	}
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceWorkerEmploymentEvent,
		ResourceID:     current.GetResourceID(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	opts := []auditports.LogOption{
		auditservice.WithComment(comment),
		auditservice.WithSubjectOf(current),
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
		opts = append(opts, auditservice.WithDiff(previous, current))
	}
	return s.auditService.LogAction(ctx, params, opts...)
}

func (s *Service) publish(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	event *employment.WorkerEmploymentEvent,
	operation permission.Operation,
	userID pulid.ID,
) {
	if event == nil {
		return
	}
	for _, resource := range []string{realtimeResource, realtimeWorkers} {
		params := &realtimeinvalidation.PublishParams{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			ActorUserID:    userID,
			ActorType:      permissionports.PrincipalTypeUser,
			ActorID:        userID,
			Resource:       resource,
			Action:         string(operation),
			RecordID:       event.ID,
		}
		if resource == realtimeWorkers {
			params.RecordID = event.WorkerID
		}
		kernelports.AfterCommit(ctx, func(committed context.Context) {
			if err := realtimeinvalidation.Publish(committed, s.realtime, params); err != nil {
				s.l.Warn("failed to publish employment invalidation", zap.Error(err))
			}
		})
	}
}
