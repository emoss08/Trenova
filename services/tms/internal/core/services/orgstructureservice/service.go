// Package orgstructureservice owns the shape of the organisation: the job
// positions the roster is counted by, who each worker answers to, and who may
// approve in somebody else's place while they are away.
//
// The scoping half is the point. Every other HR area asks the same question —
// may this user act on this worker — and answering it in one place is what
// stops five areas each inventing their own idea of a manager.
package orgstructureservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	realtimePosition   = "job_position"
	realtimeDelegation = "approval_delegation"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.OrgStructureRepository
	AuditService services.AuditService
	Realtime     services.RealtimeService `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	repo         repositories.OrgStructureRepository
	auditService services.AuditService
	realtime     services.RealtimeService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.org-structure"),
		repo:         p.Repo,
		auditService: p.AuditService,
		realtime:     p.Realtime,
	}
}

// Deps is the constructor shape tests use to swap in fakes.
type Deps struct {
	Logger       *zap.Logger
	Repo         repositories.OrgStructureRepository
	AuditService services.AuditService
}

func NewWithDeps(d Deps) *Service {
	logger := d.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		l:            logger.Named("service.org-structure"),
		repo:         d.Repo,
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
		s.l.Warn("failed to publish org structure invalidation",
			zap.String("resource", resource),
			zap.Error(err))
	}
}

// Headcount is the roster counted three ways. They run together because the
// page shows them together and they touch the same two tables.
type Headcount struct {
	ByFleet      []repositories.HeadcountRow
	ByPosition   []repositories.HeadcountRow
	ByDepartment []repositories.HeadcountRow
	ActiveTotal  int
	DriverTotal  int
	Terminated   int
}

func (s *Service) Headcount(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*Headcount, error) {
	byFleet, err := s.repo.HeadcountByFleet(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	byPosition, err := s.repo.HeadcountByPosition(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	byDepartment, err := s.repo.HeadcountByDepartment(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	out := &Headcount{
		ByFleet:      byFleet,
		ByPosition:   byPosition,
		ByDepartment: byDepartment,
	}
	// The totals come off the terminal grouping rather than being counted a
	// fourth time: every worker has exactly one terminal slot, including the
	// empty one, so it is the grouping that sums to the roster.
	for _, row := range byFleet {
		out.ActiveTotal += row.Workers
		out.DriverTotal += row.Drivers
		out.Terminated += row.Terminated
	}

	return out, nil
}

// ListPositions reads the job catalog.
func (s *Service) ListPositions(
	ctx context.Context,
	req *repositories.ListJobPositionsRequest,
) ([]*worker.JobPosition, error) {
	return s.repo.ListPositions(ctx, req)
}

func (s *Service) GetPosition(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.JobPosition, error) {
	return s.repo.GetPositionByID(ctx, &repositories.GetJobPositionByIDRequest{
		ID:               id,
		TenantInfo:       tenantInfo,
		IncludeReportsTo: true,
	})
}

func (s *Service) CreatePosition(
	ctx context.Context,
	entity *worker.JobPosition,
	userID pulid.ID,
) (*worker.JobPosition, error) {
	entity.Normalise()

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	tenantInfo := positionTenant(entity)
	if err := s.checkReportingLine(ctx, tenantInfo, entity); err != nil {
		return nil, err
	}

	created, err := s.repo.CreatePosition(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource: permission.ResourceJobPosition, resourceID: created.GetResourceID(),
		operation: permission.OpCreate, userID: userID, tenant: tenantInfo,
		current: created, comment: "Position created: " + created.Title,
	})
	s.publish(ctx, tenantInfo, realtimePosition, permission.OpCreate, created.ID, userID)

	return created, nil
}

func (s *Service) UpdatePosition(
	ctx context.Context,
	entity *worker.JobPosition,
	userID pulid.ID,
) (*worker.JobPosition, error) {
	tenantInfo := positionTenant(entity)

	original, err := s.repo.GetPositionByID(ctx, &repositories.GetJobPositionByIDRequest{
		ID:         entity.ID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if entity.Version > 0 && original.Version != entity.Version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"Position was changed by someone else. Reload and try again",
		)
	}
	entity.CreatedAt = original.CreatedAt
	entity.Version = original.Version
	entity.Normalise()

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	if err = s.checkReportingLine(ctx, tenantInfo, entity); err != nil {
		return nil, err
	}
	// Archiving a position out from under the people holding it would leave a
	// roster nobody could group, so it is refused while anyone is in it.
	if original.Status != entity.Status {
		if err = s.checkArchivable(ctx, tenantInfo, entity); err != nil {
			return nil, err
		}
	}

	updated, err := s.repo.UpdatePosition(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource: permission.ResourceJobPosition, resourceID: updated.GetResourceID(),
		operation: permission.OpUpdate, userID: userID, tenant: tenantInfo,
		current: updated, previous: original, comment: "Position updated",
	})
	s.publish(ctx, tenantInfo, realtimePosition, permission.OpUpdate, updated.ID, userID)

	return updated, nil
}

func (s *Service) checkArchivable(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *worker.JobPosition,
) error {
	if entity.Status == domaintypes.StatusActive {
		return nil
	}
	holders, err := s.repo.CountPositionHolders(ctx, tenantInfo, entity.ID)
	if err != nil {
		return err
	}
	if holders > 0 {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Move the people in this position somewhere else before archiving it",
		)
	}
	return nil
}

// checkReportingLine walks the reports-to chain looking for the position being
// saved. A cycle in the org chart is not a shape anybody can draw, and it is
// an infinite loop for anything that walks it.
func (s *Service) checkReportingLine(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *worker.JobPosition,
) error {
	if entity.ReportsToPositionID.IsNil() || entity.ID.IsNil() {
		return nil
	}

	seen := map[pulid.ID]bool{entity.ID: true}
	next := entity.ReportsToPositionID
	for !next.IsNil() {
		if seen[next] {
			return errortypes.NewValidationError(
				"reportsToPositionId",
				errortypes.ErrInvalid,
				"That reporting line loops back on itself",
			)
		}
		seen[next] = true

		parent, err := s.repo.GetPositionByID(ctx, &repositories.GetJobPositionByIDRequest{
			ID:         next,
			TenantInfo: tenantInfo,
		})
		if err != nil {
			return err
		}
		next = parent.ReportsToPositionID
	}

	return nil
}

func positionTenant(entity *worker.JobPosition) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}
