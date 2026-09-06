// Package workerleaveservice owns leave of absence: the cases, the days taken
// against them, and the FMLA entitlement those days draw down.
//
// The balance is derived on read from the entries inside the measurement
// window, never stored. The window itself moves — under the rolling method it
// moves every day — so a stored balance would be wrong by tomorrow.
package workerleaveservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
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
	realtimeCase    = "worker_leave_case"
	realtimeEntry   = "worker_leave_entry"
	realtimeControl = "leave_control"
	secondsPerDay   = int64(86400)
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.WorkerLeaveRepository
	WorkerRepo   repositories.WorkerRepository
	AuditService services.AuditService
	Realtime     services.RealtimeService `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	repo         repositories.WorkerLeaveRepository
	workerRepo   repositories.WorkerRepository
	auditService services.AuditService
	realtime     services.RealtimeService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.worker-leave"),
		repo:         p.Repo,
		workerRepo:   p.WorkerRepo,
		auditService: p.AuditService,
		realtime:     p.Realtime,
	}
}

// Deps is the constructor shape tests use to swap in fakes.
type Deps struct {
	Logger       *zap.Logger
	Repo         repositories.WorkerLeaveRepository
	WorkerRepo   repositories.WorkerRepository
	AuditService services.AuditService
}

func NewWithDeps(d Deps) *Service {
	logger := d.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		l:            logger.Named("service.worker-leave"),
		repo:         d.Repo,
		workerRepo:   d.WorkerRepo,
		auditService: d.AuditService,
	}
}

type auditParams struct {
	resourceID string
	operation  permission.Operation
	userID     pulid.ID
	tenant     pagination.TenantInfo
	current    any
	previous   any
	comment    string
}

func (s *Service) audit(p *auditParams) {
	if s.auditService == nil || p.userID.IsNil() {
		return
	}
	params := &services.LogActionParams{
		Resource:       permission.ResourceWorkerLeave,
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
		s.l.Error("failed to log audit action", zap.Error(err))
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
		s.l.Warn("failed to publish leave invalidation",
			zap.String("resource", resource),
			zap.Error(err))
	}
}

func (s *Service) GetControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*worker.LeaveControl, error) {
	return s.repo.GetControl(ctx, tenantInfo)
}

// UpdateControlRequest carries the organisation's leave settings. Every field
// is optional so a caller can change one without restating the rest.
type UpdateControlRequest struct {
	TenantInfo             pagination.TenantInfo
	MeasurementMethod      *worker.LeaveMeasurementMethod
	EntitlementWeeks       *decimal.Decimal
	MilitaryCaregiverWeeks *decimal.Decimal
	WorkweekHours          *decimal.Decimal
	EligibilityMonths      *int32
	EligibilityHours       *int32
	CertificationDueDays   *int32
	UserID                 pulid.ID
}

func (s *Service) UpdateControl(
	ctx context.Context,
	req *UpdateControlRequest,
) (*worker.LeaveControl, error) {
	entity, err := s.repo.GetControl(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	previous := *entity
	applyControlUpdate(entity, req)

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	updated, err := s.repo.UpdateControl(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resourceID: updated.ID.String(),
		operation:  permission.OpUpdate,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    updated,
		previous:   &previous,
		comment: "Leave is now measured by " +
			strings.ToLower(updated.MeasurementMethod.Label()),
	})
	s.publish(ctx, req.TenantInfo, realtimeControl, permission.OpUpdate, updated.ID, req.UserID)

	return updated, nil
}

func applyControlUpdate(entity *worker.LeaveControl, req *UpdateControlRequest) {
	if req.MeasurementMethod != nil {
		entity.MeasurementMethod = *req.MeasurementMethod
	}
	if req.EntitlementWeeks != nil {
		entity.EntitlementWeeks = *req.EntitlementWeeks
	}
	if req.MilitaryCaregiverWeeks != nil {
		entity.MilitaryCaregiverWeeks = *req.MilitaryCaregiverWeeks
	}
	if req.WorkweekHours != nil {
		entity.WorkweekHours = *req.WorkweekHours
	}
	if req.EligibilityMonths != nil {
		entity.EligibilityMonths = *req.EligibilityMonths
	}
	if req.EligibilityHours != nil {
		entity.EligibilityHours = *req.EligibilityHours
	}
	if req.CertificationDueDays != nil {
		entity.CertificationDueDays = *req.CertificationDueDays
	}
}

// Entitlement reads a worker's FMLA standing: what they are entitled to, what
// they have used inside the measurement window, and what is left.
func (s *Service) Entitlement(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	asOf int64,
) (worker.LeaveEntitlement, error) {
	if asOf <= 0 {
		asOf = timeutils.NowUnix()
	}

	wrk, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:             workerID,
		TenantInfo:     tenantInfo,
		IncludeProfile: true,
	})
	if err != nil {
		return worker.LeaveEntitlement{}, err
	}

	control, err := s.repo.GetControl(ctx, tenantInfo)
	if err != nil {
		return worker.LeaveEntitlement{}, err
	}

	cases, err := s.repo.ListCases(ctx, &repositories.ListLeaveCasesRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
	})
	if err != nil {
		return worker.LeaveEntitlement{}, err
	}

	// The forward method needs the first day ever taken to know where the
	// period starts, so the entries cannot be trimmed to a window that has not
	// been worked out yet.
	entries, err := s.repo.ListEntries(ctx, &repositories.ListLeaveEntriesRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
	})
	if err != nil {
		return worker.LeaveEntitlement{}, err
	}

	in := worker.LeaveEntitlementInput{
		Control: control,
		Cases:   cases,
		Entries: entries,
		AsOf:    asOf,
	}
	if wrk.Profile != nil {
		in.HireDate = wrk.Profile.HireDate
	}

	return worker.BuildLeaveEntitlement(in), nil
}

func caseTenant(entity *worker.WorkerLeaveCase) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}
