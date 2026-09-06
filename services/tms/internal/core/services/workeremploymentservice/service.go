// Package workeremploymentservice keeps a worker's employment timeline and is
// the only path that moves employment state. Recording an event validates it
// against the worker's current state, applies its effect to the worker record
// through the worker service (so audit, mirroring and realtime fan-out are the
// same as an edit), runs the cascade the event implies (termination ends the
// PTO and pay assignments and cancels upcoming time off; a rehire re-enrols the
// worker in the default PTO policy), then appends the event. Amendments correct
// what was written without replaying effects.
package workeremploymentservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/driverpayservice"
	"github.com/emoss08/trenova/internal/core/services/ptoledgerservice"
	"github.com/emoss08/trenova/internal/core/services/ptopolicyservice"
	"github.com/emoss08/trenova/internal/core/services/workerservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	realtimeResource        = "worker_employment_event"
	realtimeWorkers         = "workers"
	terminationCancelReason = "Employment ended"
	upcomingPTOPageSize     = 100
)

// WorkerUpdater is the slice of the worker service the timeline needs.
type WorkerUpdater interface {
	UpdateFromEmployment(
		ctx context.Context,
		entity *worker.Worker,
		actor *services.RequestActor,
	) (*worker.Worker, error)
}

// PTOPolicyManager is the slice of the PTO policy service the cascade needs.
type PTOPolicyManager interface {
	ListAssignments(
		ctx context.Context,
		req *repositories.ListPTOAssignmentsRequest,
	) ([]*worker.WorkerPTOPolicyAssignment, error)
	EndAssignment(
		ctx context.Context,
		req *ptopolicyservice.EndAssignmentRequest,
	) (*worker.WorkerPTOPolicyAssignment, error)
	AssignDefaultPolicy(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		effectiveFrom int64,
		userID pulid.ID,
	) error
}

// PTOSettler closes out PTO balances when employment ends.
type PTOSettler interface {
	SettleOnTermination(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		effectiveAt int64,
		actor ptoledgerservice.Actor,
	) (*ptoledgerservice.TerminationSettlement, error)
}

// PayAssignmentManager is the slice of the driver pay service the cascade needs.
type PayAssignmentManager interface {
	EndAssignment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		assignmentID pulid.ID,
		endDate int64,
		actor *services.RequestActor,
	) (*driverpay.WorkerPayAssignment, error)
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	Repo          repositories.WorkerEmploymentEventRepository
	WorkerRepo    repositories.WorkerRepository
	FleetCodeRepo repositories.FleetCodeRepository
	DocumentRepo  repositories.DocumentRepository
	PayAssignRepo repositories.WorkerPayAssignmentRepository
	Workers       *workerservice.Service
	PTOPolicies   *ptopolicyservice.Service
	PTO           services.WorkerPTOService
	PTOLedger     *ptoledgerservice.Service
	DriverPay     *driverpayservice.Service
	AuditService  services.AuditService
	Realtime      services.RealtimeService  `optional:"true"`
	Checklists    services.ChecklistSpawner `optional:"true"`
	Training      services.TrainingAssigner `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	repo          repositories.WorkerEmploymentEventRepository
	workerRepo    repositories.WorkerRepository
	fleetCodeRepo repositories.FleetCodeRepository
	documentRepo  repositories.DocumentRepository
	payAssignRepo repositories.WorkerPayAssignmentRepository
	workers       WorkerUpdater
	ptoPolicies   PTOPolicyManager
	pto           services.WorkerPTOService
	ptoLedger     PTOSettler
	driverPay     PayAssignmentManager
	auditService  services.AuditService
	realtime      services.RealtimeService
	checklists    services.ChecklistSpawner
	training      services.TrainingAssigner
	portal        services.PortalAccessRevoker
}

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.worker-employment"),
		repo:          p.Repo,
		workerRepo:    p.WorkerRepo,
		fleetCodeRepo: p.FleetCodeRepo,
		documentRepo:  p.DocumentRepo,
		payAssignRepo: p.PayAssignRepo,
		workers:       p.Workers,
		ptoPolicies:   p.PTOPolicies,
		pto:           p.PTO,
		ptoLedger:     p.PTOLedger,
		driverPay:     p.DriverPay,
		auditService:  p.AuditService,
		realtime:      p.Realtime,
		checklists:    p.Checklists,
		training:      p.Training,
	}
}

// Deps lets tests assemble a service from narrow fakes without Uber FX.
type Deps struct {
	Logger        *zap.Logger
	Repo          repositories.WorkerEmploymentEventRepository
	WorkerRepo    repositories.WorkerRepository
	FleetCodeRepo repositories.FleetCodeRepository
	DocumentRepo  repositories.DocumentRepository
	PayAssignRepo repositories.WorkerPayAssignmentRepository
	Workers       WorkerUpdater
	PTOPolicies   PTOPolicyManager
	PTO           services.WorkerPTOService
	PTOLedger     PTOSettler
	DriverPay     PayAssignmentManager
	AuditService  services.AuditService
	Realtime      services.RealtimeService
	Checklists    services.ChecklistSpawner
	Training      services.TrainingAssigner
	Portal        services.PortalAccessRevoker
}

func NewWithDeps(d Deps) *Service {
	logger := d.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		l:             logger.Named("service.worker-employment"),
		repo:          d.Repo,
		workerRepo:    d.WorkerRepo,
		fleetCodeRepo: d.FleetCodeRepo,
		documentRepo:  d.DocumentRepo,
		payAssignRepo: d.PayAssignRepo,
		workers:       d.Workers,
		ptoPolicies:   d.PTOPolicies,
		pto:           d.PTO,
		ptoLedger:     d.PTOLedger,
		driverPay:     d.DriverPay,
		auditService:  d.AuditService,
		realtime:      d.Realtime,
		checklists:    d.Checklists,
		training:      d.Training,
		portal:        d.Portal,
	}
}

// SetPortalRevoker breaks the construction cycle between this service and the
// driver portal service, which reaches back here through the safety service to
// record a termination. The revoker arrives after both are built.
func (s *Service) SetPortalRevoker(revoker services.PortalAccessRevoker) {
	s.portal = revoker
}

func (s *Service) List(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	kinds []worker.EmploymentEventKind,
) ([]*worker.WorkerEmploymentEvent, error) {
	return s.repo.List(ctx, &repositories.ListWorkerEmploymentEventsRequest{
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
) (*worker.WorkerEmploymentEvent, error) {
	return s.repo.GetByID(ctx, &repositories.GetWorkerEmploymentEventByIDRequest{
		ID:              id,
		TenantInfo:      tenantInfo,
		IncludeDocument: true,
		IncludeActors:   true,
	})
}

// RecordRequest carries the typed inputs a kind can use; the service turns them
// into the event's from/to values so the timeline renders what actually moved.
type RecordRequest struct {
	TenantInfo  pagination.TenantInfo
	WorkerID    pulid.ID
	Kind        worker.EmploymentEventKind
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

// CascadeSummary reports what a status-changing event did beyond the worker
// row, so the office sees the consequences in one place.
type CascadeSummary struct {
	PTOAssignmentEnded   bool
	PayAssignmentEnded   bool
	UpcomingPTOCancelled int
	DefaultPolicyApplied bool
	ChecklistStarted     bool
	PTOPaidOutDays       decimal.Decimal
	PTOForfeitedDays     decimal.Decimal
	TrainingAssigned     int
	// PortalAccessRevoked reports that a termination shut off the worker's
	// Dash sign-in. PortalRevocationError carries why it could not be shut
	// off, because a termination that is already recorded must still be
	// reported honestly rather than rolled back.
	PortalAccessRevoked   bool
	PortalRevocationError string
}

type RecordResult struct {
	Event   *worker.WorkerEmploymentEvent
	Cascade CascadeSummary
}

func (s *Service) Record(ctx context.Context, req *RecordRequest) (*RecordResult, error) {
	log := s.l.With(
		zap.String("operation", "Record"),
		zap.String("workerId", req.WorkerID.String()),
		zap.String("kind", req.Kind.String()),
	)

	wrk, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:             req.WorkerID,
		TenantInfo:     req.TenantInfo,
		IncludeProfile: true,
		IncludeState:   true,
	})
	if err != nil {
		return nil, err
	}

	history, err := s.repo.List(ctx, &repositories.ListWorkerEmploymentEventsRequest{
		TenantInfo: req.TenantInfo,
		WorkerID:   req.WorkerID,
		Ascending:  true,
	})
	if err != nil {
		return nil, err
	}

	event := &worker.WorkerEmploymentEvent{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		WorkerID:       req.WorkerID,
		Kind:           req.Kind,
		EffectiveAt:    req.EffectiveAt,
		Reason:         strings.TrimSpace(req.Reason),
		Notes:          strings.TrimSpace(req.Notes),
		DocumentID:     req.DocumentID,
		RecordedByID:   req.UserID,
		FromValues:     map[string]string{},
		ToValues:       map[string]string{},
	}

	multiErr := errortypes.NewMultiError()
	event.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	if err = req.Kind.CanRecord(worker.EmploymentStateOf(wrk, history)); err != nil {
		return nil, err
	}
	if err = s.populateValues(ctx, req, wrk, event); err != nil {
		return nil, err
	}
	if !event.DocumentID.IsNil() {
		if err = s.requireWorkerDocument(ctx, req.TenantInfo, wrk.ID, event.DocumentID); err != nil {
			return nil, err
		}
	}

	actor := s.actor(req.TenantInfo, req.UserID)
	if event.Apply(wrk) {
		if _, err = s.workers.UpdateFromEmployment(ctx, wrk, actor); err != nil {
			log.Error("failed to apply employment event to worker", zap.Error(err))
			return nil, err
		}
	}

	cascade, err := s.cascade(ctx, req, wrk, actor, log)
	if err != nil {
		return nil, err
	}

	created, err := s.repo.Create(ctx, event)
	if err != nil {
		log.Error("failed to record employment event", zap.Error(err))
		return nil, err
	}

	s.audit(created, nil, permission.OpCreate, req.UserID, "Employment event recorded", log)
	s.publish(ctx, req.TenantInfo, created, permission.OpCreate, req.UserID)
	cascade.ChecklistStarted = s.spawnChecklist(ctx, created, wrk, req.UserID, log)

	return &RecordResult{Event: created, Cascade: cascade}, nil
}

// RecordHired opens a timeline for a freshly created worker. It is best-effort
// from the worker service's point of view and never applies effects, because
// the worker row was just written with the hire date already on it.
func (s *Service) RecordHired(ctx context.Context, wrk *worker.Worker, userID pulid.ID) error {
	if wrk == nil || wrk.Profile == nil || wrk.Profile.HireDate <= 0 {
		return nil
	}
	tenantInfo := pagination.TenantInfo{OrgID: wrk.OrganizationID, BuID: wrk.BusinessUnitID}
	event := &worker.WorkerEmploymentEvent{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		WorkerID:       wrk.ID,
		Kind:           worker.EmploymentEventHired,
		EffectiveAt:    wrk.Profile.HireDate,
		RecordedByID:   userID,
		FromValues:     map[string]string{},
		ToValues: map[string]string{
			worker.EmploymentValueHireDate: fmt.Sprintf("%d", wrk.Profile.HireDate),
		},
	}
	if !wrk.FleetCodeID.IsNil() {
		event.ToValues[worker.EmploymentValueFleetCodeID] = wrk.FleetCodeID.String()
	}
	created, err := s.repo.Create(ctx, event)
	if err != nil {
		return err
	}
	s.publish(ctx, tenantInfo, created, permission.OpCreate, userID)
	s.spawnChecklist(ctx, created, wrk, userID, s.l.With(zap.String("operation", "RecordHired")))
	return nil
}

// spawnChecklist starts the default onboarding/offboarding checklist for the
// event. It is best-effort: a checklist that fails to start must not undo an
// employment change that has already been recorded.
func (s *Service) spawnChecklist(
	ctx context.Context,
	event *worker.WorkerEmploymentEvent,
	wrk *worker.Worker,
	userID pulid.ID,
	log *zap.Logger,
) bool {
	if s.checklists == nil {
		return false
	}
	checklist, err := s.checklists.SpawnForEvent(ctx, event, wrk, userID)
	if err != nil {
		log.Warn("failed to start checklist for employment event", zap.Error(err))
		return false
	}
	return checklist != nil
}

func (s *Service) populateValues(
	ctx context.Context,
	req *RecordRequest,
	wrk *worker.Worker,
	event *worker.WorkerEmploymentEvent,
) error {
	switch req.Kind {
	case worker.EmploymentEventHired, worker.EmploymentEventRehired:
		if wrk.Profile != nil && wrk.Profile.HireDate > 0 {
			event.FromValues[worker.EmploymentValueHireDate] = fmt.Sprintf("%d", wrk.Profile.HireDate)
		}
		event.ToValues[worker.EmploymentValueHireDate] = fmt.Sprintf("%d", req.EffectiveAt)
		event.FromValues[worker.EmploymentValueStatus] = wrk.Status.String()
		event.ToValues[worker.EmploymentValueStatus] = "Active"
	case worker.EmploymentEventTerminated:
		event.FromValues[worker.EmploymentValueStatus] = wrk.Status.String()
		event.ToValues[worker.EmploymentValueStatus] = "Inactive"
		event.ToValues[worker.EmploymentValueTerminationDate] = fmt.Sprintf("%d", req.EffectiveAt)
	case worker.EmploymentEventSuspended:
		event.FromValues[worker.EmploymentValueCanBeAssigned] = fmt.Sprintf("%t", wrk.CanBeAssigned)
		event.ToValues[worker.EmploymentValueCanBeAssigned] = "false"
	case worker.EmploymentEventLeaveStarted:
		if req.LeaveType == nil || !req.LeaveType.IsValid() {
			return errortypes.NewValidationError(
				"leaveType",
				errortypes.ErrRequired,
				"Choose the kind of leave",
			)
		}
		event.FromValues[worker.EmploymentValueCanBeAssigned] = fmt.Sprintf("%t", wrk.CanBeAssigned)
		event.ToValues[worker.EmploymentValueCanBeAssigned] = "false"
		event.ToValues[worker.EmploymentValueLeaveType] = req.LeaveType.String()
	case worker.EmploymentEventReinstated:
		event.FromValues[worker.EmploymentValueCanBeAssigned] = fmt.Sprintf("%t", wrk.CanBeAssigned)
		event.ToValues[worker.EmploymentValueCanBeAssigned] = "true"
	case worker.EmploymentEventLeaveEnded:
		event.FromValues[worker.EmploymentValueCanBeAssigned] = fmt.Sprintf("%t", wrk.CanBeAssigned)
		event.ToValues[worker.EmploymentValueCanBeAssigned] = "true"
		if wrk.LeaveType != "" {
			event.FromValues[worker.EmploymentValueLeaveType] = wrk.LeaveType.String()
		}
	case worker.EmploymentEventTransferred:
		return s.populateTransfer(ctx, req, wrk, event)
	case worker.EmploymentEventPromoted:
		return populatePromotion(req, wrk, event)
	case worker.EmploymentEventRateChanged:
		rate := strings.TrimSpace(req.Rate)
		if rate == "" {
			return errortypes.NewValidationError("rate", errortypes.ErrRequired, "Enter the new rate")
		}
		event.ToValues[worker.EmploymentValueRate] = rate
		if unit := strings.TrimSpace(req.RateUnit); unit != "" {
			event.ToValues[worker.EmploymentValueRateUnit] = unit
		}
	case worker.EmploymentEventProbationEnded:
	}
	return nil
}

func (s *Service) populateTransfer(
	ctx context.Context,
	req *RecordRequest,
	wrk *worker.Worker,
	event *worker.WorkerEmploymentEvent,
) error {
	if req.FleetCodeID == nil && req.ManagerID == nil {
		return errortypes.NewValidationError(
			"fleetCodeId",
			errortypes.ErrRequired,
			"A transfer needs a new fleet code or manager",
		)
	}
	if req.FleetCodeID != nil {
		if *req.FleetCodeID == wrk.FleetCodeID {
			return errortypes.NewValidationError(
				"fleetCodeId",
				errortypes.ErrInvalid,
				"The worker is already in that fleet",
			)
		}
		if !wrk.FleetCodeID.IsNil() {
			event.FromValues[worker.EmploymentValueFleetCodeID] = wrk.FleetCodeID.String()
			if wrk.FleetCode != nil {
				event.FromValues[worker.EmploymentValueFleetCode] = wrk.FleetCode.Code
			}
		}
		if req.FleetCodeID.IsNil() {
			event.ToValues[worker.EmploymentValueFleetCodeID] = ""
		} else {
			tenantInfo := req.TenantInfo
			fleet, err := s.fleetCodeRepo.GetByID(ctx, repositories.GetFleetCodeByIDRequest{
				ID:         *req.FleetCodeID,
				TenantInfo: &tenantInfo,
			})
			if err != nil {
				return err
			}
			event.ToValues[worker.EmploymentValueFleetCodeID] = fleet.ID.String()
			event.ToValues[worker.EmploymentValueFleetCode] = fleet.Code
		}
	}
	if req.ManagerID != nil {
		if !wrk.ManagerID.IsNil() {
			event.FromValues[worker.EmploymentValueManagerID] = wrk.ManagerID.String()
		}
		event.ToValues[worker.EmploymentValueManagerID] = req.ManagerID.String()
	}
	return nil
}

func populatePromotion(
	req *RecordRequest,
	wrk *worker.Worker,
	event *worker.WorkerEmploymentEvent,
) error {
	if req.DriverType == nil && req.WorkerType == nil {
		return errortypes.NewValidationError(
			"driverType",
			errortypes.ErrRequired,
			"A promotion needs a new driver type or worker type",
		)
	}
	changed := false
	if req.DriverType != nil {
		if !req.DriverType.IsValid() {
			return errortypes.NewValidationError("driverType", errortypes.ErrInvalid, "Unknown driver type")
		}
		if *req.DriverType != wrk.DriverType {
			event.FromValues[worker.EmploymentValueDriverType] = wrk.DriverType.String()
			event.ToValues[worker.EmploymentValueDriverType] = req.DriverType.String()
			changed = true
		}
	}
	if req.WorkerType != nil {
		if !req.WorkerType.IsValid() {
			return errortypes.NewValidationError("workerType", errortypes.ErrInvalid, "Unknown worker type")
		}
		if *req.WorkerType != wrk.Type {
			event.FromValues[worker.EmploymentValueWorkerType] = wrk.Type.String()
			event.ToValues[worker.EmploymentValueWorkerType] = req.WorkerType.String()
			changed = true
		}
	}
	if !changed {
		return errortypes.NewValidationError(
			"driverType",
			errortypes.ErrInvalid,
			"Nothing changes — the worker already has that type",
		)
	}
	return nil
}

func (s *Service) cascade(
	ctx context.Context,
	req *RecordRequest,
	wrk *worker.Worker,
	actor *services.RequestActor,
	log *zap.Logger,
) (CascadeSummary, error) {
	var summary CascadeSummary
	switch req.Kind {
	case worker.EmploymentEventTerminated:
		ended, err := s.endPTOAssignment(ctx, req)
		if err != nil {
			return summary, err
		}
		summary.PTOAssignmentEnded = ended
		payEnded, err := s.endPayAssignment(ctx, req, actor)
		if err != nil {
			return summary, err
		}
		summary.PayAssignmentEnded = payEnded
		cancelled, err := s.cancelUpcomingPTO(ctx, req, log)
		if err != nil {
			return summary, err
		}
		summary.UpcomingPTOCancelled = cancelled
		if s.ptoLedger != nil {
			settlement, settleErr := s.ptoLedger.SettleOnTermination(
				ctx, req.TenantInfo, req.WorkerID, req.EffectiveAt,
				ptoledgerservice.UserActor(req.UserID),
			)
			if settleErr != nil {
				return summary, fmt.Errorf("settle PTO balances: %w", settleErr)
			}
			summary.PTOPaidOutDays = settlement.PaidOutDays
			summary.PTOForfeitedDays = settlement.ForfeitedDays
		}
		summary.PortalAccessRevoked, summary.PortalRevocationError = s.revokePortalAccess(
			ctx, req, actor, log,
		)
	case worker.EmploymentEventRehired:
		if err := s.ptoPolicies.AssignDefaultPolicy(
			ctx, req.TenantInfo, wrk.ID, req.EffectiveAt, req.UserID,
		); err != nil {
			log.Warn("failed to re-enrol rehired worker in default PTO policy", zap.Error(err))
		} else {
			summary.DefaultPolicyApplied = true
		}
	case worker.EmploymentEventHired, worker.EmploymentEventProbationEnded,
		worker.EmploymentEventPromoted, worker.EmploymentEventTransferred,
		worker.EmploymentEventLeaveStarted, worker.EmploymentEventLeaveEnded,
		worker.EmploymentEventSuspended, worker.EmploymentEventReinstated,
		worker.EmploymentEventRateChanged:
	}
	switch req.Kind {
	case worker.EmploymentEventHired, worker.EmploymentEventRehired, worker.EmploymentEventPromoted:
		summary.TrainingAssigned = s.assignTraining(ctx, req, log)
	default:
	}
	return summary, nil
}

// revokePortalAccess shuts off Dash for a worker whose employment has ended.
// The worker row is already Inactive by the time the cascade runs, so a portal
// failure is reported rather than raised: aborting here would lose the
// timeline event for a termination that has already taken effect. The
// offboarding checklist keeps its item as the human backstop.
func (s *Service) revokePortalAccess(
	ctx context.Context,
	req *RecordRequest,
	actor *services.RequestActor,
	log *zap.Logger,
) (bool, string) {
	if s.portal == nil {
		return false, ""
	}
	revoked, err := s.portal.RevokeOnTermination(ctx, req.TenantInfo, req.WorkerID, actor)
	if err != nil {
		log.Error("failed to revoke portal access on termination", zap.Error(err))
		return false, err.Error()
	}
	return revoked, ""
}

// assignTraining opens the required courses the worker is missing after a
// hire, rehire or driver-type change. Best-effort: a training gap must not
// undo an employment change that has already been recorded.
func (s *Service) assignTraining(ctx context.Context, req *RecordRequest, log *zap.Logger) int {
	if s.training == nil {
		return 0
	}
	records, err := s.training.AssignRequired(ctx, req.TenantInfo, req.WorkerID, req.UserID)
	if err != nil {
		log.Warn("failed to assign required training", zap.Error(err))
		return 0
	}
	return len(records)
}

func (s *Service) endPTOAssignment(ctx context.Context, req *RecordRequest) (bool, error) {
	assignments, err := s.ptoPolicies.ListAssignments(ctx, &repositories.ListPTOAssignmentsRequest{
		TenantInfo: req.TenantInfo,
		WorkerID:   req.WorkerID,
	})
	if err != nil {
		return false, err
	}
	for _, assignment := range assignments {
		if !assignment.IsOpen() {
			continue
		}
		effectiveTo := req.EffectiveAt
		if effectiveTo <= assignment.EffectiveFrom {
			effectiveTo = assignment.EffectiveFrom + 1
		}
		if _, err = s.ptoPolicies.EndAssignment(ctx, &ptopolicyservice.EndAssignmentRequest{
			TenantInfo:   req.TenantInfo,
			AssignmentID: assignment.ID,
			EffectiveTo:  effectiveTo,
			Version:      assignment.Version,
			UserID:       req.UserID,
		}); err != nil {
			return false, fmt.Errorf("end PTO policy assignment: %w", err)
		}
		return true, nil
	}
	return false, nil
}

func (s *Service) endPayAssignment(
	ctx context.Context,
	req *RecordRequest,
	actor *services.RequestActor,
) (bool, error) {
	assignment, err := s.payAssignRepo.GetEffectiveForWorker(
		ctx,
		repositories.GetWorkerPayAssignmentRequest{
			TenantInfo: req.TenantInfo,
			WorkerID:   req.WorkerID,
			AsOf:       req.EffectiveAt,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	if assignment == nil || assignment.EffectiveTo != nil {
		return false, nil
	}
	endDate := req.EffectiveAt
	if endDate <= assignment.EffectiveFrom {
		endDate = assignment.EffectiveFrom + 1
	}
	if _, err = s.driverPay.EndAssignment(
		ctx, req.TenantInfo, assignment.ID, endDate, actor,
	); err != nil {
		return false, fmt.Errorf("end pay assignment: %w", err)
	}
	return true, nil
}

func (s *Service) cancelUpcomingPTO(
	ctx context.Context,
	req *RecordRequest,
	log *zap.Logger,
) (int, error) {
	cancelled := 0
	for _, status := range []worker.PTOStatus{worker.PTOStatusApproved, worker.PTOStatusRequested} {
		result, err := s.pto.List(ctx, &repositories.ListPTORequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: req.TenantInfo,
				Pagination: pagination.Info{Limit: upcomingPTOPageSize},
			},
			Status:        status.String(),
			StartDateFrom: req.EffectiveAt,
			WorkerID:      req.WorkerID,
		})
		if err != nil {
			return cancelled, err
		}
		for _, pto := range result.Items {
			if _, err = s.pto.Cancel(ctx, &repositories.UpdatePTOStatusRequest{
				ID:              pto.ID,
				TenantInfo:      req.TenantInfo,
				Status:          worker.PTOStatusCancelled,
				UserID:          req.UserID,
				Reason:          terminationCancelReason,
				ExpectedVersion: pto.Version,
			}); err != nil {
				log.Warn("failed to cancel upcoming PTO on termination",
					zap.String("ptoId", pto.ID.String()),
					zap.Error(err))
				continue
			}
			cancelled++
		}
	}
	return cancelled, nil
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

// Amend corrects the narrative of an event. Effects are never replayed: a
// corrected termination date does not move the worker's termination date,
// because that would silently rewrite what the cascade already did.
func (s *Service) Amend(
	ctx context.Context,
	req *AmendRequest,
) (*worker.WorkerEmploymentEvent, error) {
	log := s.l.With(zap.String("operation", "Amend"), zap.String("id", req.ID.String()))

	original, err := s.repo.GetByID(ctx, &repositories.GetWorkerEmploymentEventByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if req.Version > 0 && original.Version != req.Version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
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
			if err = s.requireWorkerDocument(ctx, req.TenantInfo, original.WorkerID, updated.DocumentID); err != nil {
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

	s.audit(saved, original, permission.OpUpdate, req.UserID, "Employment event amended", log)
	s.publish(ctx, req.TenantInfo, saved, permission.OpUpdate, req.UserID)

	return saved, nil
}

func (s *Service) requireWorkerDocument(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID, documentID pulid.ID,
) error {
	doc, err := s.documentRepo.GetByID(ctx, repositories.GetDocumentByIDRequest{
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

func (s *Service) actor(tenantInfo pagination.TenantInfo, userID pulid.ID) *services.RequestActor {
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}
}

func (s *Service) audit(
	current, previous *worker.WorkerEmploymentEvent,
	operation permission.Operation,
	userID pulid.ID,
	comment string,
	log *zap.Logger,
) {
	if s.auditService == nil || userID.IsNil() {
		return
	}
	params := &services.LogActionParams{
		Resource:       permission.ResourceWorkerEmploymentEvent,
		ResourceID:     current.GetResourceID(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	opts := []services.LogOption{auditservice.WithComment(comment)}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
		opts = append(opts, auditservice.WithDiff(previous, current))
	}
	if err := s.auditService.LogAction(params, opts...); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}
}

func (s *Service) publish(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	event *worker.WorkerEmploymentEvent,
	operation permission.Operation,
	userID pulid.ID,
) {
	if s.realtime == nil || event == nil {
		return
	}
	for _, resource := range []string{realtimeResource, realtimeWorkers} {
		recordID := event.ID
		if resource == realtimeWorkers {
			recordID = event.WorkerID
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
			s.l.Warn("failed to publish employment invalidation", zap.Error(err))
		}
	}
}
