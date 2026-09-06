// Package benefitsservice owns what a carrier offers its people and who is on
// it.
//
// The load-bearing decision is that an employee contribution is taken through
// an ordinary recurring deduction rather than a second mechanism. The
// settlement builder already knows how to take money off a settlement — how to
// cap it, pause it, order it against a garnishment, and reverse it when a
// settlement is voided. A parallel benefits deduction would have to be taught
// all of that again and would drift from the original within a release.
package benefitsservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Repo          repositories.BenefitRepository
	DeductionRepo repositories.RecurringDeductionRepository
	AuditService  services.AuditService
}

type Service struct {
	l             *zap.Logger
	repo          repositories.BenefitRepository
	deductionRepo repositories.RecurringDeductionRepository
	auditService  services.AuditService
}

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.benefits"),
		repo:          p.Repo,
		deductionRepo: p.DeductionRepo,
		auditService:  p.AuditService,
	}
}

// Deps is the constructor shape tests use to swap in fakes.
type Deps struct {
	Logger        *zap.Logger
	Repo          repositories.BenefitRepository
	DeductionRepo repositories.RecurringDeductionRepository
	AuditService  services.AuditService
}

func NewWithDeps(d Deps) *Service {
	logger := d.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		l:             logger.Named("service.benefits"),
		repo:          d.Repo,
		deductionRepo: d.DeductionRepo,
		auditService:  d.AuditService,
	}
}

func (s *Service) audit(
	resourceID string,
	operation permission.Operation,
	userID pulid.ID,
	tenantInfo pagination.TenantInfo,
	current, previous any,
	comment string,
) {
	if s.auditService == nil || userID.IsNil() {
		return
	}
	params := &services.LogActionParams{
		Resource:       permission.ResourceBenefitPlan,
		ResourceID:     resourceID,
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}
	opts := []services.LogOption{auditservice.WithComment(comment)}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
		opts = append(opts, auditservice.WithDiff(previous, current))
	}
	if err := s.auditService.LogAction(params, opts...); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}
}

func (s *Service) ListPlans(
	ctx context.Context,
	req *repositories.ListBenefitPlansRequest,
) ([]*driverpay.BenefitPlan, error) {
	return s.repo.ListPlans(ctx, req)
}

func (s *Service) GetPlan(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*driverpay.BenefitPlan, error) {
	return s.repo.GetPlanByID(ctx, &repositories.GetBenefitPlanByIDRequest{
		ID:             id,
		TenantInfo:     tenantInfo,
		IncludePayCode: true,
	})
}

func (s *Service) CreatePlan(
	ctx context.Context,
	entity *driverpay.BenefitPlan,
	userID pulid.ID,
) (*driverpay.BenefitPlan, error) {
	entity.Normalise()

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.repo.CreatePlan(ctx, entity)
	if err != nil {
		return nil, err
	}

	tenantInfo := planTenant(created)
	s.audit(
		created.GetResourceID(), permission.OpCreate, userID, tenantInfo,
		created, nil, "Benefit plan created: "+created.Name,
	)

	return created, nil
}

func (s *Service) UpdatePlan(
	ctx context.Context,
	entity *driverpay.BenefitPlan,
	userID pulid.ID,
) (*driverpay.BenefitPlan, error) {
	tenantInfo := planTenant(entity)

	original, err := s.repo.GetPlanByID(ctx, &repositories.GetBenefitPlanByIDRequest{
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
			"Plan was changed by someone else. Reload and try again",
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

	// Archiving a plan out from under the people on it would leave a deduction
	// nobody could explain, so it is refused while anyone is enrolled.
	if original.Status != entity.Status && entity.Status != domaintypes.StatusActive {
		enrolled, cErr := s.repo.CountPlanEnrollments(ctx, tenantInfo, entity.ID)
		if cErr != nil {
			return nil, cErr
		}
		if enrolled > 0 {
			return nil, errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalidOperation,
				"End the enrollments on this plan before archiving it",
			)
		}
	}

	updated, err := s.repo.UpdatePlan(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(
		updated.GetResourceID(), permission.OpUpdate, userID, tenantInfo,
		updated, original, "Benefit plan updated",
	)

	return updated, nil
}

func (s *Service) ListEnrollments(
	ctx context.Context,
	req *repositories.ListBenefitEnrollmentsRequest,
) ([]*driverpay.WorkerBenefitEnrollment, error) {
	return s.repo.ListEnrollments(ctx, req)
}

func (s *Service) BenefitCosts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	planYear int16,
) ([]repositories.BenefitCostRow, error) {
	return s.repo.BenefitCosts(ctx, tenantInfo, planYear)
}

func planTenant(entity *driverpay.BenefitPlan) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}

func deductionDescription(plan *driverpay.BenefitPlan, tier driverpay.CoverageTier) string {
	return fmt.Sprintf("%s — %s", plan.Name, tier)
}
